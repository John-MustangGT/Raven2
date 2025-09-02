// internal/notifications/pushover.go - Pushover notification service
package notifications

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "sync"
    "text/template"
    "time"

    "github.com/sirupsen/logrus"
    "raven2/internal/config"
    "raven2/internal/database"
)

const (
    PushoverAPIURL = "https://api.pushover.net/1/messages.json"
    UserAgent      = "Raven Network Monitor/2.0"
)

// NotificationEvent represents a monitoring event that can trigger notifications
type NotificationEvent struct {
    Host         *database.Host
    Check        *database.Check
    Status       *database.Status
    PreviousExit int
    Timestamp    time.Time
    IsRecovery   bool
}

// NotificationService handles sending notifications via various channels
type NotificationService struct {
    config      *config.NotificationConfig
    pushover    *PushoverService
    httpClient  *http.Client
    throttler   *NotificationThrottler
    templates   map[string]*template.Template
    mu          sync.RWMutex
}

// PushoverService handles Pushover-specific notifications
type PushoverService struct {
    config     *config.PushoverConfig
    httpClient *http.Client
    templates  map[string]*template.Template
}

// NotificationThrottler implements rate limiting for notifications
type NotificationThrottler struct {
    config      *config.ThrottleConfig
    hostCounts  map[string][]time.Time
    totalCounts []time.Time
    mu          sync.RWMutex
}

// PushoverMessage represents a message sent to Pushover API
type PushoverMessage struct {
    Token     string `json:"token"`
    User      string `json:"user"`
    Message   string `json:"message"`
    Title     string `json:"title,omitempty"`
    Priority  int    `json:"priority,omitempty"`
    Retry     int    `json:"retry,omitempty"`
    Expire    int    `json:"expire,omitempty"`
    Sound     string `json:"sound,omitempty"`
    Device    string `json:"device,omitempty"`
    Timestamp int64  `json:"timestamp,omitempty"`
    HTML      int    `json:"html,omitempty"`
}

// PushoverResponse represents the API response
type PushoverResponse struct {
    Status int      `json:"status"`
    Errors []string `json:"errors,omitempty"`
}

// NewNotificationService creates a new notification service
func NewNotificationService(cfg *config.NotificationConfig) (*NotificationService, error) {
    service := &NotificationService{
        config: cfg,
        httpClient: &http.Client{
            Timeout: 30 * time.Second,
        },
        templates: make(map[string]*template.Template),
    }

    // Initialize Pushover service if enabled
    if cfg.Enabled && cfg.Pushover.Enabled {
        pushoverService, err := NewPushoverService(&cfg.Pushover, service.httpClient)
        if err != nil {
            return nil, fmt.Errorf("failed to initialize Pushover service: %w", err)
        }
        service.pushover = pushoverService

        // Initialize throttler
        if cfg.Pushover.Throttle.Enabled {
            service.throttler = NewNotificationThrottler(&cfg.Pushover.Throttle)
        }
    }

    logrus.WithFields(logrus.Fields{
        "notifications_enabled": cfg.Enabled,
        "pushover_enabled":     cfg.Pushover.Enabled,
        "throttle_enabled":     cfg.Pushover.Throttle.Enabled,
    }).Info("Notification service initialized")

    return service, nil
}

// NewPushoverService creates a new Pushover service
func NewPushoverService(cfg *config.PushoverConfig, httpClient *http.Client) (*PushoverService, error) {
    service := &PushoverService{
        config:     cfg,
        httpClient: httpClient,
        templates:  make(map[string]*template.Template),
    }

    // Parse templates
    if err := service.parseTemplates(); err != nil {
        return nil, fmt.Errorf("failed to parse templates: %w", err)
    }

    return service, nil
}

// NewNotificationThrottler creates a new throttler
func NewNotificationThrottler(cfg *config.ThrottleConfig) *NotificationThrottler {
    return &NotificationThrottler{
        config:      cfg,
        hostCounts:  make(map[string][]time.Time),
        totalCounts: make([]time.Time, 0),
    }
}

// SendNotification sends a notification through all enabled channels
func (ns *NotificationService) SendNotification(ctx context.Context, event *NotificationEvent) error {
    if !ns.config.Enabled {
        return nil
    }

    // Apply throttling if enabled
    if ns.throttler != nil {
        if ns.throttler.IsThrottled(event.Host.ID) {
            logrus.WithFields(logrus.Fields{
                "host":  event.Host.Name,
                "check": event.Check.Name,
            }).Debug("Notification throttled")
            return nil
        }
    }

    // Send via Pushover if enabled
    if ns.pushover != nil {
        if err := ns.pushover.SendMessage(ctx, event); err != nil {
            logrus.WithError(err).Error("Failed to send Pushover notification")
            return err
        }
        
        // Record the notification for throttling
        if ns.throttler != nil {
            ns.throttler.RecordNotification(event.Host.ID)
        }
    }

    return nil
}

// SendMessage sends a message via Pushover
func (ps *PushoverService) SendMessage(ctx context.Context, event *NotificationEvent) error {
    // Check if we should notify for this state
    if !ps.shouldNotify(event) {
        logrus.WithFields(logrus.Fields{
            "host":         event.Host.Name,
            "check":        event.Check.Name,
            "status":       getStatusName(event.Status.ExitCode),
            "is_recovery":  event.IsRecovery,
        }).Debug("Skipping notification based on state filter")
        return nil
    }

    // Build message
    message, err := ps.buildMessage(event)
    if err != nil {
        return fmt.Errorf("failed to build message: %w", err)
    }

    // Send to Pushover API
    return ps.sendToPushover(ctx, message)
}

// shouldNotify determines if we should send a notification for this event
func (ps *PushoverService) shouldNotify(event *NotificationEvent) bool {
    if len(ps.config.OnlyOnState) == 0 {
        return true // Notify for all states if no filter is set
    }

    currentStatus := getStatusName(event.Status.ExitCode)
    
    for _, state := range ps.config.OnlyOnState {
        if state == currentStatus {
            return true
        }
        if state == "recovery" && event.IsRecovery {
            return true
        }
    }

    return false
}

// buildMessage creates a Pushover message from the event
func (ps *PushoverService) buildMessage(event *NotificationEvent) (*PushoverMessage, error) {
    // Prepare template data
    templateData := map[string]interface{}{
        "Host":         event.Host.Name,
        "HostDisplay":  event.Host.DisplayName,
        "Check":        event.Check.Name,
        "Status":       getStatusName(event.Status.ExitCode),
        "Output":       event.Status.Output,
        "LongOutput":   event.Status.LongOutput,
        "Duration":     fmt.Sprintf("%.2fms", event.Status.Duration),
        "Timestamp":    event.Timestamp.Format("2006-01-02 15:04:05"),
        "IsRecovery":   event.IsRecovery,
        "PreviousStatus": getStatusName(event.PreviousExit),
        "Group":        event.Host.Group,
        "IPv4":         event.Host.IPv4,
        "Hostname":     event.Host.Hostname,
    }

    // Add emoji and formatting based on status
    statusEmoji := getStatusEmoji(event.Status.ExitCode, event.IsRecovery)
    templateData["StatusEmoji"] = statusEmoji

    // Render title
    title, err := ps.renderTemplate("title", ps.config.Title, templateData)
    if err != nil {
        return nil, fmt.Errorf("failed to render title: %w", err)
    }

    // Render message body
    messageText, err := ps.renderTemplate("message", ps.config.Template, templateData)
    if err != nil {
        return nil, fmt.Errorf("failed to render message: %w", err)
    }

    // Create Pushover message
    message := &PushoverMessage{
        Token:     ps.config.APIToken,
        User:      ps.config.UserKey,
        Title:     title,
        Message:   statusEmoji + " " + messageText,
        Priority:  ps.config.Priority,
        Sound:     ps.config.Sound,
        Device:    ps.config.Device,
        Timestamp: event.Timestamp.Unix(),
    }

    // Set emergency priority options
    if ps.config.Priority == 2 {
        message.Retry = ps.config.Retry
        message.Expire = ps.config.Expire
    }

    return message, nil
}

// renderTemplate renders a template with the given data
func (ps *PushoverService) renderTemplate(name, templateText string, data map[string]interface{}) (string, error) {
    tmpl, exists := ps.templates[name]
    if !exists {
        var err error
        tmpl, err = template.New(name).Parse(templateText)
        if err != nil {
            return "", fmt.Errorf("failed to parse template %s: %w", name, err)
        }
        ps.templates[name] = tmpl
    }

    var buf bytes.Buffer
    if err := tmpl.Execute(&buf, data); err != nil {
        return "", fmt.Errorf("failed to execute template %s: %w", name, err)
    }

    return buf.String(), nil
}

// sendToPushover sends the message to Pushover API
func (ps *PushoverService) sendToPushover(ctx context.Context, message *PushoverMessage) error {
    jsonData, err := json.Marshal(message)
    if err != nil {
        return fmt.Errorf("failed to marshal message: %w", err)
    }

    req, err := http.NewRequestWithContext(ctx, "POST", PushoverAPIURL, bytes.NewBuffer(jsonData))
    if err != nil {
        return fmt.Errorf("failed to create request: %w", err)
    }

    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("User-Agent", UserAgent)

    resp, err := ps.httpClient.Do(req)
    if err != nil {
        return fmt.Errorf("failed to send request: %w", err)
    }
    defer resp.Body.Close()

    var pushoverResp PushoverResponse
    if err := json.NewDecoder(resp.Body).Decode(&pushoverResp); err != nil {
        return fmt.Errorf("failed to decode response: %w", err)
    }

    if pushoverResp.Status != 1 {
        return fmt.Errorf("pushover API error: %v", pushoverResp.Errors)
    }

    logrus.WithFields(logrus.Fields{
        "host":     message.Title,
        "priority": message.Priority,
        "sound":    message.Sound,
    }).Info("Pushover notification sent successfully")

    return nil
}

// parseTemplates parses and validates all templates
func (ps *PushoverService) parseTemplates() error {
    // Parse title template
    titleTemplate, err := template.New("title").Parse(ps.config.Title)
    if err != nil {
        return fmt.Errorf("failed to parse title template: %w", err)
    }
    ps.templates["title"] = titleTemplate

    // Parse message template
    messageTemplate, err := template.New("message").Parse(ps.config.Template)
    if err != nil {
        return fmt.Errorf("failed to parse message template: %w", err)
    }
    ps.templates["message"] = messageTemplate

    return nil
}

// IsThrottled checks if notifications for a host should be throttled
func (nt *NotificationThrottler) IsThrottled(hostID string) bool {
    if !nt.config.Enabled {
        return false
    }

    nt.mu.RLock()
    defer nt.mu.RUnlock()

    now := time.Now()
    windowStart := now.Add(-nt.config.Window)

    // Check per-host throttling
    hostTimes := nt.hostCounts[hostID]
    recentHostCount := 0
    for _, t := range hostTimes {
        if t.After(windowStart) {
            recentHostCount++
        }
    }

    if recentHostCount >= nt.config.MaxPerHost {
        return true
    }

    // Check total throttling
    recentTotalCount := 0
    for _, t := range nt.totalCounts {
        if t.After(windowStart) {
            recentTotalCount++
        }
    }

    return recentTotalCount >= nt.config.MaxTotal
}

// RecordNotification records a notification for throttling purposes
func (nt *NotificationThrottler) RecordNotification(hostID string) {
    if !nt.config.Enabled {
        return
    }

    nt.mu.Lock()
    defer nt.mu.Unlock()

    now := time.Now()

    // Record for host
    nt.hostCounts[hostID] = append(nt.hostCounts[hostID], now)

    // Record for total
    nt.totalCounts = append(nt.totalCounts, now)

    // Clean up old entries periodically
    nt.cleanup()
}

// cleanup removes old throttling entries
func (nt *NotificationThrottler) cleanup() {
    windowStart := time.Now().Add(-nt.config.Window)

    // Clean host counts
    for hostID, times := range nt.hostCounts {
        validTimes := make([]time.Time, 0)
        for _, t := range times {
            if t.After(windowStart) {
                validTimes = append(validTimes, t)
            }
        }
        if len(validTimes) == 0 {
            delete(nt.hostCounts, hostID)
        } else {
            nt.hostCounts[hostID] = validTimes
        }
    }

    // Clean total counts
    validTotalTimes := make([]time.Time, 0)
    for _, t := range nt.totalCounts {
        if t.After(windowStart) {
            validTotalTimes = append(validTotalTimes, t)
        }
    }
    nt.totalCounts = validTotalTimes
}

// TestNotification sends a test notification
func (ns *NotificationService) TestNotification(ctx context.Context, message string) error {
    if !ns.config.Enabled || ns.pushover == nil {
        return fmt.Errorf("notifications are not enabled or configured")
    }

    testMessage := &PushoverMessage{
        Token:   ns.pushover.config.APIToken,
        User:    ns.pushover.config.UserKey,
        Title:   "Raven Test Notification",
        Message: "🧪 " + message,
        Priority: 0, // Normal priority for tests
        Sound:   ns.pushover.config.Sound,
    }

    return ns.pushover.sendToPushover(ctx, testMessage)
}

// Helper functions

func getStatusName(exitCode int) string {
    switch exitCode {
    case 0:
        return "ok"
    case 1:
        return "warning"
    case 2:
        return "critical"
    default:
        return "unknown"
    }
}

func getStatusEmoji(exitCode int, isRecovery bool) string {
    if isRecovery {
        return "✅"
    }

    switch exitCode {
    case 0:
        return "✅"
    case 1:
        return "⚠️"
    case 2:
        return "🚨"
    default:
        return "❓"
    }
}

// GetStats returns notification statistics
func (ns *NotificationService) GetStats() map[string]interface{} {
    stats := map[string]interface{}{
        "enabled":         ns.config.Enabled,
        "pushover_enabled": false,
        "throttle_enabled": false,
    }

    if ns.pushover != nil {
        stats["pushover_enabled"] = ns.pushover.config.Enabled
        stats["pushover_priority"] = ns.pushover.config.Priority
        stats["pushover_sound"] = ns.pushover.config.Sound
    }

    if ns.throttler != nil {
        stats["throttle_enabled"] = ns.throttler.config.Enabled
        stats["throttle_window"] = ns.throttler.config.Window.String()
        stats["throttle_max_per_host"] = ns.throttler.config.MaxPerHost
        stats["throttle_max_total"] = ns.throttler.config.MaxTotal

        ns.throttler.mu.RLock()
        stats["throttle_host_count"] = len(ns.throttler.hostCounts)
        stats["throttle_total_recent"] = len(ns.throttler.totalCounts)
        ns.throttler.mu.RUnlock()
    }

    return stats
}

// ValidateConfig validates the notification configuration
func ValidateConfig(cfg *config.NotificationConfig) error {
    if !cfg.Enabled {
        return nil
    }

    if cfg.Pushover.Enabled {
        if cfg.Pushover.APIToken == "" {
            return fmt.Errorf("pushover API token is required")
        }
        if cfg.Pushover.UserKey == "" {
            return fmt.Errorf("pushover user key is required")
        }
        
        // Test template parsing
        _, err := template.New("test").Parse(cfg.Pushover.Title)
        if err != nil {
            return fmt.Errorf("invalid title template: %w", err)
        }
        
        _, err = template.New("test").Parse(cfg.Pushover.Template)
        if err != nil {
            return fmt.Errorf("invalid message template: %w", err)
        }
    }

    return nil
}
