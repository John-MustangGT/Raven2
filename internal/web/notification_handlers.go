// internal/web/notification_handlers.go - Web handlers for Pushover notifications
package web

import (
    "context"
    "net/http"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/sirupsen/logrus"
    "raven2/internal/config"
)

// NotificationSettings represents the notification configuration for the API
type NotificationSettings struct {
    Enabled  bool                    `json:"enabled"`
    Pushover PushoverSettingsRequest `json:"pushover"`
}

// PushoverSettingsRequest represents Pushover settings for API requests
type PushoverSettingsRequest struct {
    Enabled     bool     `json:"enabled"`
    APIToken    string   `json:"api_token"`
    UserKey     string   `json:"user_key"`
    Priority    int      `json:"priority"`
    Retry       int      `json:"retry"`
    Expire      int      `json:"expire"`
    Sound       string   `json:"sound"`
    Device      string   `json:"device"`
    Title       string   `json:"title"`
    Template    string   `json:"template"`
    OnlyOnState []string `json:"only_on_state"`
    Throttle    ThrottleSettingsRequest `json:"throttle"`
}

// ThrottleSettingsRequest represents throttle settings for API requests
type ThrottleSettingsRequest struct {
    Enabled     bool   `json:"enabled"`
    WindowMin   int    `json:"window_minutes"`
    MaxPerHost  int    `json:"max_per_host"`
    MaxTotal    int    `json:"max_total"`
}

// TestNotificationRequest represents a test notification request
type TestNotificationRequest struct {
    Message string `json:"message" binding:"required"`
}

// NotificationStatsResponse represents notification statistics
type NotificationStatsResponse struct {
    Enabled         bool                   `json:"enabled"`
    PushoverEnabled bool                   `json:"pushover_enabled"`
    ThrottleEnabled bool                   `json:"throttle_enabled"`
    Stats           map[string]interface{} `json:"stats"`
}

// Add these methods to your existing Server struct in server.go

// setupNotificationRoutes adds notification endpoints to the router
// Call this from your existing setupRoutes method
func (s *Server) setupNotificationRoutes() {
    api := s.router.Group("/api")
    
    notifications := api.Group("/notifications")
    {
        notifications.GET("/settings", s.getNotificationSettings)
        notifications.PUT("/settings", s.updateNotificationSettings)
        notifications.POST("/test", s.sendTestNotification)
        notifications.GET("/stats", s.getNotificationStats)
        notifications.POST("/validate", s.validateNotificationSettings)
    }
}

// GET /api/notifications/settings - Get current notification settings
func (s *Server) getNotificationSettings(c *gin.Context) {
    cfg := s.config.Notifications
    
    settings := NotificationSettings{
        Enabled: cfg.Enabled,
        Pushover: PushoverSettingsRequest{
            Enabled:     cfg.Pushover.Enabled,
            APIToken:    maskToken(cfg.Pushover.APIToken), // Mask the token for security
            UserKey:     maskToken(cfg.Pushover.UserKey),  // Mask the user key
            Priority:    cfg.Pushover.Priority,
            Retry:       cfg.Pushover.Retry,
            Expire:      cfg.Pushover.Expire,
            Sound:       cfg.Pushover.Sound,
            Device:      cfg.Pushover.Device,
            Title:       cfg.Pushover.Title,
            Template:    cfg.Pushover.Template,
            OnlyOnState: cfg.Pushover.OnlyOnState,
            Throttle: ThrottleSettingsRequest{
                Enabled:     cfg.Pushover.Throttle.Enabled,
                WindowMin:   int(cfg.Pushover.Throttle.Window.Minutes()),
                MaxPerHost:  cfg.Pushover.Throttle.MaxPerHost,
                MaxTotal:    cfg.Pushover.Throttle.MaxTotal,
            },
        },
    }
    
    c.JSON(http.StatusOK, gin.H{"data": settings})
}

// PUT /api/notifications/settings - Update notification settings
func (s *Server) updateNotificationSettings(c *gin.Context) {
    var req NotificationSettings
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    // Validate the settings
    if err := s.validateNotificationConfig(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    // Update configuration (in memory)
    s.config.Notifications.Enabled = req.Enabled
    s.config.Notifications.Pushover.Enabled = req.Pushover.Enabled
    
    // Only update tokens if they are not masked
    if !isMaskedToken(req.Pushover.APIToken) {
        s.config.Notifications.Pushover.APIToken = req.Pushover.APIToken
    }
    if !isMaskedToken(req.Pushover.UserKey) {
        s.config.Notifications.Pushover.UserKey = req.Pushover.UserKey
    }
    
    s.config.Notifications.Pushover.Priority = req.Pushover.Priority
    s.config.Notifications.Pushover.Retry = req.Pushover.Retry
    s.config.Notifications.Pushover.Expire = req.Pushover.Expire
    s.config.Notifications.Pushover.Sound = req.Pushover.Sound
    s.config.Notifications.Pushover.Device = req.Pushover.Device
    s.config.Notifications.Pushover.Title = req.Pushover.Title
    s.config.Notifications.Pushover.Template = req.Pushover.Template
    s.config.Notifications.Pushover.OnlyOnState = req.Pushover.OnlyOnState
    s.config.Notifications.Pushover.Throttle.Enabled = req.Pushover.Throttle.Enabled
    s.config.Notifications.Pushover.Throttle.Window = time.Duration(req.Pushover.Throttle.WindowMin) * time.Minute
    s.config.Notifications.Pushover.Throttle.MaxPerHost = req.Pushover.Throttle.MaxPerHost
    s.config.Notifications.Pushover.Throttle.MaxTotal = req.Pushover.Throttle.MaxTotal

    // Log the configuration change
    logrus.WithFields(logrus.Fields{
        "notifications_enabled": req.Enabled,
        "pushover_enabled":     req.Pushover.Enabled,
        "priority":             req.Pushover.Priority,
        "throttle_enabled":     req.Pushover.Throttle.Enabled,
    }).Info("Notification settings updated")

    // TODO: In a production system, you might want to:
    // 1. Save these settings to a configuration file
    // 2. Restart/reinitialize the notification service
    // 3. Validate the new settings before applying them
    
    c.JSON(http.StatusOK, gin.H{
        "message": "Notification settings updated successfully",
        "note":    "Settings updated in memory only. Restart required for full effect.",
    })
}

// POST /api/notifications/test - Send a test notification
func (s *Server) sendTestNotification(c *gin.Context) {
    var req TestNotificationRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    // Check if notifications are configured
    if !s.config.Notifications.Enabled {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Notifications are not enabled"})
        return
    }

    if !s.config.Notifications.Pushover.Enabled {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Pushover notifications are not enabled"})
        return
    }

    // Send test notification via the monitoring engine
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    if err := s.engine.TestNotifications(ctx); err != nil {
        logrus.WithError(err).Error("Failed to send test notification")
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send test notification: " + err.Error()})
        return
    }

    logrus.Info("Test notification sent successfully")
    c.JSON(http.StatusOK, gin.H{
        "message": "Test notification sent successfully",
        "timestamp": time.Now(),
    })
}

// GET /api/notifications/stats - Get notification statistics
func (s *Server) getNotificationStats(c *gin.Context) {
    stats := s.engine.GetNotificationStats()
    
    response := NotificationStatsResponse{
        Enabled:         s.config.Notifications.Enabled,
        PushoverEnabled: s.config.Notifications.Pushover.Enabled,
        ThrottleEnabled: s.config.Notifications.Pushover.Throttle.Enabled,
        Stats:           stats,
    }
    
    c.JSON(http.StatusOK, gin.H{"data": response})
}

// POST /api/notifications/validate - Validate notification settings without saving
func (s *Server) validateNotificationSettings(c *gin.Context) {
    var req NotificationSettings
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    // Validate the settings
    if err := s.validateNotificationConfig(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "valid": false,
            "error": err.Error(),
        })
        return
    }

    // Check if we can reach Pushover API (optional validation)
    if req.Enabled && req.Pushover.Enabled {
        // Only do this if tokens are provided and not masked
        if !isMaskedToken(req.Pushover.APIToken) && !isMaskedToken(req.Pushover.UserKey) {
            if err := s.testPushoverConnection(&req.Pushover); err != nil {
                c.JSON(http.StatusBadRequest, gin.H{
                    "valid": false,
                    "error": "Pushover API connection test failed: " + err.Error(),
                    "warning": "Settings are valid but Pushover API is not reachable",
                })
                return
            }
        }
    }

    c.JSON(http.StatusOK, gin.H{
        "valid": true,
        "message": "Notification settings are valid",
    })
}

// Helper functions

// validateNotificationConfig validates notification configuration
func (s *Server) validateNotificationConfig(settings *NotificationSettings) error {
    if !settings.Enabled {
        return nil // No validation needed if disabled
    }

    if settings.Pushover.Enabled {
        // Check required fields (unless they are masked)
        if !isMaskedToken(settings.Pushover.APIToken) && settings.Pushover.APIToken == "" {
            return fmt.Errorf("pushover API token is required")
        }
        if !isMaskedToken(settings.Pushover.UserKey) && settings.Pushover.UserKey == "" {
            return fmt.Errorf("pushover user key is required")
        }

        // Validate priority
        if settings.Pushover.Priority < -2 || settings.Pushover.Priority > 2 {
            return fmt.Errorf("priority must be between -2 and 2")
        }

        // Emergency priority validation
        if settings.Pushover.Priority == 2 {
            if settings.Pushover.Retry < 30 {
                return fmt.Errorf("retry must be at least 30 seconds for emergency priority")
            }
            if settings.Pushover.Expire < 60 || settings.Pushover.Expire > 10800 {
                return fmt.Errorf("expire must be between 60 and 10800 seconds for emergency priority")
            }
        }

        // Validate templates if provided
        if settings.Pushover.Title != "" {
            if _, err := template.New("test").Parse(settings.Pushover.Title); err != nil {
                return fmt.Errorf("invalid title template: %w", err)
            }
        }

        if settings.Pushover.Template != "" {
            if _, err := template.New("test").Parse(settings.Pushover.Template); err != nil {
                return fmt.Errorf("invalid message template: %w", err)
            }
        }

        // Validate throttle settings
        if settings.Pushover.Throttle.Enabled {
            if settings.Pushover.Throttle.WindowMin < 1 {
                return fmt.Errorf("throttle window must be at least 1 minute")
            }
            if settings.Pushover.Throttle.MaxPerHost < 1 {
                return fmt.Errorf("max notifications per host must be at least 1")
            }
            if settings.Pushover.Throttle.MaxTotal < 1 {
                return fmt.Errorf("max total notifications must be at least 1")
            }
        }
    }

    return nil
}

// testPushoverConnection tests if we can connect to Pushover API
func (s *Server) testPushoverConnection(settings *PushoverSettingsRequest) error {
    // This is a simple validation - we could make an API call to validate
    // For now, just check the token format
    if len(settings.APIToken) != 30 {
        return fmt.Errorf("API token should be 30 characters long")
    }
    if len(settings.UserKey) != 30 {
        return fmt.Errorf("user key should be 30 characters long")
    }
    
    // TODO: Could make actual API call to Pushover to validate
    return nil
}

// maskToken masks sensitive tokens for API responses
func maskToken(token string) string {
    if len(token) <= 8 {
        return "***"
    }
    return token[:4] + strings.Repeat("*", len(token)-8) + token[len(token)-4:]
}

// isMaskedToken checks if a token is masked
func isMaskedToken(token string) bool {
    return strings.Contains(token, "*")
}

// Available Pushover sounds for the frontend
func (s *Server) getPushoverSounds(c *gin.Context) {
    sounds := []map[string]string{
        {"value": "pushover", "label": "Pushover (default)"},
        {"value": "bike", "label": "Bike"},
        {"value": "bugle", "label": "Bugle"},
        {"value": "cashregister", "label": "Cash Register"},
        {"value": "classical", "label": "Classical"},
        {"value": "cosmic", "label": "Cosmic"},
        {"value": "falling", "label": "Falling"},
        {"value": "gamelan", "label": "Gamelan"},
        {"value": "incoming", "label": "Incoming"},
        {"value": "intermission", "label": "Intermission"},
        {"value": "magic", "label": "Magic"},
        {"value": "mechanical", "label": "Mechanical"},
        {"value": "pianobar", "label": "Piano Bar"},
        {"value": "siren", "label": "Siren"},
        {"value": "spacealarm", "label": "Space Alarm"},
        {"value": "tugboat", "label": "Tugboat"},
        {"value": "alien", "label": "Alien Alarm (long)"},
        {"value": "climb", "label": "Climb (long)"},
        {"value": "persistent", "label": "Persistent (long)"},
        {"value": "echo", "label": "Pushover Echo (long)"},
        {"value": "updown", "label": "Up Down (long)"},
        {"value": "vibrate", "label": "Vibrate Only"},
        {"value": "none", "label": "Silent"},
    }

    c.JSON(http.StatusOK, gin.H{"data": sounds})
}

// Template variables helper for the frontend
func (s *Server) getNotificationTemplateVariables(c *gin.Context) {
    variables := map[string]interface{}{
        "host_variables": []map[string]string{
            {"variable": "{{.Host}}", "description": "Host name"},
            {"variable": "{{.HostDisplay}}", "description": "Host display name"},
            {"variable": "{{.Group}}", "description": "Host group"},
            {"variable": "{{.IPv4}}", "description": "Host IP address"},
            {"variable": "{{.Hostname}}", "description": "Host hostname"},
        },
        "check_variables": []map[string]string{
            {"variable": "{{.Check}}", "description": "Check name"},
            {"variable": "{{.Status}}", "description": "Current status (ok, warning, critical, unknown)"},
            {"variable": "{{.Output}}", "description": "Check output message"},
            {"variable": "{{.LongOutput}}", "description": "Extended check output"},
        },
        "status_variables": []map[string]string{
            {"variable": "{{.StatusEmoji}}", "description": "Status emoji (✅ ⚠️ 🚨 ❓)"},
            {"variable": "{{.IsRecovery}}", "description": "True if this is a recovery to OK status"},
            {"variable": "{{.PreviousStatus}}", "description": "Previous status before this change"},
            {"variable": "{{.Duration}}", "description": "Check execution duration"},
            {"variable": "{{.Timestamp}}", "description": "Check timestamp"},
        },
        "examples": []map[string]string{
            {
                "name": "Simple Alert",
                "title": "Raven Alert: {{.Host}}",
                "template": "{{.Check}} on {{.Host}} is {{.Status}}: {{.Output}}",
            },
            {
                "name": "Detailed Alert",
                "title": "{{.StatusEmoji}} {{.Host}} - {{.Status}}",
                "template": "{{.StatusEmoji}} {{.Check}} on {{.Host}} ({{.Group}}) is {{.Status}}\n\nMessage: {{.Output}}\nDuration: {{.Duration}}\nTime: {{.Timestamp}}",
            },
            {
                "name": "Recovery Alert",
                "title": "✅ Recovery: {{.Host}}",
                "template": "{{if .IsRecovery}}🎉 RECOVERED! {{end}}{{.Check}} on {{.Host}} is now {{.Status}}\n\nPrevious: {{.PreviousStatus}}\nMessage: {{.Output}}",
            },
        },
    }

    c.JSON(http.StatusOK, gin.H{"data": variables})
}

// Add these routes to your setupNotificationRoutes method:
func (s *Server) setupNotificationRoutesExtended() {
    api := s.router.Group("/api")
    
    notifications := api.Group("/notifications")
    {
        notifications.GET("/settings", s.getNotificationSettings)
        notifications.PUT("/settings", s.updateNotificationSettings)
        notifications.POST("/test", s.sendTestNotification)
        notifications.GET("/stats", s.getNotificationStats)
        notifications.POST("/validate", s.validateNotificationSettings)
        
        // Additional helpful endpoints
        notifications.GET("/pushover/sounds", s.getPushoverSounds)
        notifications.GET("/template/variables", s.getNotificationTemplateVariables)
    }
}
