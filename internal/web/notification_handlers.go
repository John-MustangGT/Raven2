// internal/web/notification_handlers.go - New file for notification-related API endpoints
package web

import (
    "net/http"
    "context"
    "time"

    "raven2/internal/config"
    "raven2/internal/notifications"
    "github.com/gin-gonic/gin"
    "github.com/sirupsen/logrus"
)

// Add these routes to your setupRoutes() method in server.go:
func (s *Server) setupNotificationRoutes() {
    api := s.router.Group("/api")
    
    // Notification endpoints
    notifications := api.Group("/notifications")
    {
        notifications.GET("/status", s.getNotificationStatus)
        notifications.POST("/pushover/test", s.testPushoverConfig)
        notifications.GET("/pushover/config", s.getPushoverConfig)
        notifications.PUT("/pushover/config", s.updatePushoverConfig)
    }
}

// NotificationStatusResponse represents the current notification status
type NotificationStatusResponse struct {
    Enabled           bool              `json:"enabled"`
    PushoverEnabled   bool              `json:"pushover_enabled"`
    PushoverConfigured bool             `json:"pushover_configured"`
    Services          []ServiceStatus   `json:"services"`
}

type ServiceStatus struct {
    Name      string `json:"name"`
    Enabled   bool   `json:"enabled"`
    Configured bool  `json:"configured"`
    LastTest  string `json:"last_test,omitempty"`
    Status    string `json:"status"` // "ok", "error", "untested"
}

// PushoverConfigRequest represents the request to update Pushover config
type PushoverConfigRequest struct {
    Enabled         bool                     `json:"enabled"`
    UserKey         string                   `json:"user_key"`
    APIToken        string                   `json:"api_token"`
    Device          string                   `json:"device,omitempty"`
    Priority        int                      `json:"priority"`
    Sound           string                   `json:"sound,omitempty"`
    QuietHours      *QuietHoursRequest       `json:"quiet_hours,omitempty"`
    RealertInterval string                   `json:"realert_interval"`
    MaxRealerts     int                      `json:"max_realerts"`
    SendRecovery    bool                     `json:"send_recovery"`
    Title           string                   `json:"title,omitempty"`
    URLTitle        string                   `json:"url_title,omitempty"`
    URL             string                   `json:"url,omitempty"`
    TestOnSave      bool                     `json:"test_on_save"`
}

type QuietHoursRequest struct {
    Enabled   bool   `json:"enabled"`
    StartHour int    `json:"start_hour"`
    EndHour   int    `json:"end_hour"`
    Timezone  string `json:"timezone"`
}

// GET /api/notifications/status - Get notification system status
func (s *Server) getNotificationStatus(c *gin.Context) {
    status := NotificationStatusResponse{
        Enabled:           s.config.Pushover.Enabled,
        PushoverEnabled:   s.config.Pushover.Enabled,
        PushoverConfigured: s.config.Pushover.UserKey != "" && s.config.Pushover.APIToken != "",
        Services: []ServiceStatus{
            {
                Name:       "pushover",
                Enabled:    s.config.Pushover.Enabled,
                Configured: s.config.Pushover.UserKey != "" && s.config.Pushover.APIToken != "",
                Status:     "untested", // TODO: Track last test results
            },
        },
    }

    c.JSON(http.StatusOK, gin.H{"data": status})
}

// POST /api/notifications/pushover/test - Test Pushover configuration
func (s *Server) testPushoverConfig(c *gin.Context) {
    if !s.config.Pushover.Enabled {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Pushover is not enabled"})
        return
    }

    ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
    defer cancel()

    if err := s.engine.TestPushoverConfig(ctx); err != nil {
        logrus.WithError(err).Error("Pushover test failed")
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "Pushover test failed",
            "details": err.Error(),
        })
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "message": "Test notification sent successfully",
        "timestamp": time.Now(),
    })
}

// GET /api/notifications/pushover/config - Get Pushover configuration (sanitized)
func (s *Server) getPushoverConfig(c *gin.Context) {
    config := gin.H{
        "enabled":           s.config.Pushover.Enabled,
        "user_key_set":      s.config.Pushover.UserKey != "",
        "api_token_set":     s.config.Pushover.APIToken != "",
        "device":            s.config.Pushover.Device,
        "priority":          s.config.Pushover.Priority,
        "sound":             s.config.Pushover.Sound,
        "quiet_hours":       s.config.Pushover.QuietHours,
        "realert_interval":  s.config.Pushover.RealertInterval.String(),
        "max_realerts":      s.config.Pushover.MaxRealerts,
        "send_recovery":     s.config.Pushover.SendRecovery,
        "title":             s.config.Pushover.Title,
        "url_title":         s.config.Pushover.URLTitle,
        "url":               s.config.Pushover.URL,
        "overrides_count":   len(s.config.Pushover.Overrides),
    }

    c.JSON(http.StatusOK, gin.H{"data": config})
}

// PUT /api/notifications/pushover/config - Update Pushover configuration
func (s *Server) updatePushoverConfig(c *gin.Context) {
    var req PushoverConfigRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    // Validate the request
    if req.Enabled {
        if req.UserKey == "" {
            c.JSON(http.StatusBadRequest, gin.H{"error": "User key is required when Pushover is enabled"})
            return
        }
        if req.APIToken == "" {
            c.JSON(http.StatusBadRequest, gin.H{"error": "API token is required when Pushover is enabled"})
            return
        }
        if req.Priority < -2 || req.Priority > 2 {
            c.JSON(http.StatusBadRequest, gin.H{"error": "Priority must be between -2 and 2"})
            return
        }
        if req.MaxRealerts < 0 {
            c.JSON(http.StatusBadRequest, gin.H{"error": "Max realerts must be non-negative"})
            return
        }
    }

    // Parse realert interval
    realertInterval, err := time.ParseDuration(req.RealertInterval)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "Invalid realert interval format: " + req.RealertInterval,
        })
        return
    }

    // Update configuration (in a real application, you'd want to persist this)
    s.config.Pushover.Enabled = req.Enabled
    s.config.Pushover.UserKey = req.UserKey
    s.config.Pushover.APIToken = req.APIToken
    s.config.Pushover.Device = req.Device
    s.config.Pushover.Priority = req.Priority
    s.config.Pushover.Sound = req.Sound
    s.config.Pushover.RealertInterval = realertInterval
    s.config.Pushover.MaxRealerts = req.MaxRealerts
    s.config.Pushover.SendRecovery = req.SendRecovery
    s.config.Pushover.Title = req.Title
    s.config.Pushover.URLTitle = req.URLTitle
    s.config.Pushover.URL = req.URL

    if req.QuietHours != nil {
        s.config.Pushover.QuietHours = &config.QuietHours{
            Enabled:   req.QuietHours.Enabled,
            StartHour: req.QuietHours.StartHour,
            EndHour:   req.QuietHours.EndHour,
            Timezone:  req.QuietHours.Timezone,
        }
    }

    // Validate the updated configuration
    if err := s.config.Pushover.Validate(); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "Invalid configuration: " + err.Error(),
        })
        return
    }

    // Reinitialize the engine with new configuration
    if err := s.engine.RefreshConfig(); err != nil {
        logrus.WithError(err).Error("Failed to refresh engine configuration")
        c.JSON(http.StatusInternalServerError, gin.H{
            "error": "Failed to apply new configuration",
        })
        return
    }

    response := gin.H{
        "message": "Pushover configuration updated successfully",
        "timestamp": time.Now(),
    }

    // Test configuration if requested
    if req.TestOnSave && req.Enabled {
        ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
        defer cancel()

        if err := s.engine.TestPushoverConfig(ctx); err != nil {
            response["test_result"] = "failed"
            response["test_error"] = err.Error()
            logrus.WithError(err).Warn("Pushover test failed after configuration update")
        } else {
            response["test_result"] = "success"
            response["test_message"] = "Test notification sent successfully"
        }
    }

    c.JSON(http.StatusOK, response)
}

// Add this method to your existing handlers.go or create notification-specific handlers

// PushoverTestRequest represents a test notification request
type PushoverTestRequest struct {
    UserKey  string `json:"user_key"`
    APIToken string `json:"api_token"`
    Device   string `json:"device,omitempty"`
    Message  string `json:"message,omitempty"`
    Priority int    `json:"priority,omitempty"`
    Sound    string `json:"sound,omitempty"`
}

// POST /api/notifications/pushover/test-config - Test Pushover with custom config
func (s *Server) testPushoverCustomConfig(c *gin.Context) {
    var req PushoverTestRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    if req.UserKey == "" || req.APIToken == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "User key and API token are required"})
        return
    }

    if req.Message == "" {
        req.Message = "Test notification from Raven monitoring system"
    }

    // Create a temporary Pushover client for testing
    testConfig := &config.PushoverConfig{
        Enabled:  true,
        UserKey:  req.UserKey,
        APIToken: req.APIToken,
        Device:   req.Device,
        Priority: req.Priority,
        Sound:    req.Sound,
    }

    testClient := notifications.NewPushoverClient(testConfig, nil)

    ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
    defer cancel()

    if err := testClient.TestConnection(ctx); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "Test failed",
            "details": err.Error(),
        })
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "message": "Test notification sent successfully",
        "timestamp": time.Now(),
    })
}
