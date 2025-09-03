// internal/notifications/pushover.go - Pushover API client
package notifications

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "net/url"
    "strings"
    "time"

    "github.com/sirupsen/logrus"
    "raven2/internal/config"
    "raven2/internal/database"
)

const (
    PushoverAPIURL = "https://api.pushover.net/1/messages.json"
)

// PushoverClient handles sending notifications via Pushover API
type PushoverClient struct {
    client     *http.Client
    config     *config.PushoverConfig
    store      database.ExtendedStore
    sentAlerts map[string]*SentAlert // Track sent alerts for realert logic
}

// SentAlert tracks when an alert was last sent for realert logic
type SentAlert struct {
    HostID      string
    CheckID     string
    Severity    string
    FirstSent   time.Time
    LastSent    time.Time
    SendCount   int
    Resolved    bool
}

// PushoverMessage represents a message to send via Pushover
type PushoverMessage struct {
    Token     string `json:"token"`
    User      string `json:"user"`
    Device    string `json:"device,omitempty"`
    Title     string `json:"title,omitempty"`
    Message   string `json:"message"`
    HTML      int    `json:"html,omitempty"`
    Priority  int    `json:"priority"`
    Sound     string `json:"sound,omitempty"`
    URL       string `json:"url,omitempty"`
    URLTitle  string `json:"url_title,omitempty"`
    Timestamp int64  `json:"timestamp,omitempty"`
}

// PushoverResponse represents the API response
type PushoverResponse struct {
    Status  int      `json:"status"`
    Request string   `json:"request"`
    Errors  []string `json:"errors,omitempty"`
}

// NewPushoverClient creates a new Pushover client
func NewPushoverClient(cfg *config.PushoverConfig, store database.ExtendedStore) *PushoverClient {
    return &PushoverClient{
        client: &http.Client{
            Timeout: 30 * time.Second,
        },
        config:     cfg,
        store:      store,
        sentAlerts: make(map[string]*SentAlert),
    }
}

// SendNotification sends a notification for a status change
func (p *PushoverClient) SendNotification(ctx context.Context, host *database.Host, check *database.Check, status *database.Status) error {
    if !p.config.Enabled {
        return nil
    }

    // Get severity name
    severity := getSeverityName(status.ExitCode)
    
    // Get effective configuration for this host/check combination
    effective := p.config.GetEffectiveConfig(
        host.ID, check.ID, host.Name, check.Name, severity,
    )

    if effective == nil || !effective.Enabled {
        return nil
    }

    // Check if we're in quiet hours
    if effective.IsQuietTime() {
        logrus.WithFields(logrus.Fields{
            "host":  host.Name,
            "check": check.Name,
        }).Debug("Skipping notification due to quiet hours")
        return nil
    }

    alertKey := fmt.Sprintf("%s:%s", host.ID, check.ID)
    
    // Handle recovery notifications
    if status.ExitCode == 0 {
        return p.handleRecoveryNotification(ctx, alertKey, host, check, status, effective)
    }

    // Handle problem notifications
    return p.handleProblemNotification(ctx, alertKey, host, check, status, effective)
}

// handleRecoveryNotification handles OK status notifications
func (p *PushoverClient) handleRecoveryNotification(ctx context.Context, alertKey string, host *database.Host, check *database.Check, status *database.Status, config *config.EffectivePushoverConfig) error {
    sentAlert, exists := p.sentAlerts[alertKey]
    
    if !exists || sentAlert.Resolved {
        // No previous problem or already resolved
        return nil
    }

    if !config.SendRecovery {
        // Just mark as resolved without sending
        sentAlert.Resolved = true
        return nil
    }

    // Send recovery notification
    message := p.buildRecoveryMessage(host, check, status, sentAlert)
    
    pushoverMsg := &PushoverMessage{
        Token:     config.APIToken,
        User:      config.UserKey,
        Device:    config.Device,
        Title:     p.buildTitle(config, host, check, "RECOVERED"),
        Message:   message,
        Priority:  0, // Always normal priority for recovery
        Sound:     config.Sound,
        URL:       p.buildURL(config, host, check),
        URLTitle:  config.URLTitle,
        Timestamp: status.Timestamp.Unix(),
    }

    if err := p.sendMessage(ctx, pushoverMsg); err != nil {
        return fmt.Errorf("failed to send recovery notification: %w", err)
    }

    // Mark as resolved
    sentAlert.Resolved = true
    
    logrus.WithFields(logrus.Fields{
        "host":      host.Name,
        "check":     check.Name,
        "duration":  time.Since(sentAlert.FirstSent),
    }).Info("Sent recovery notification")

    return nil
}

// handleProblemNotification handles non-OK status notifications
func (p *PushoverClient) handleProblemNotification(ctx context.Context, alertKey string, host *database.Host, check *database.Check, status *database.Status, config *config.EffectivePushoverConfig) error {
    now := time.Now()
    sentAlert, exists := p.sentAlerts[alertKey]

    if !exists {
        // First time seeing this alert
        sentAlert = &SentAlert{
            HostID:    host.ID,
            CheckID:   check.ID,
            Severity:  getSeverityName(status.ExitCode),
            FirstSent: now,
            SendCount: 0,
            Resolved:  false,
        }
        p.sentAlerts[alertKey] = sentAlert
    } else if sentAlert.Resolved {
        // Alert was resolved but now failing again
        sentAlert.FirstSent = now
        sentAlert.SendCount = 0
        sentAlert.Resolved = false
    } else {
        // Existing unresolved alert - check if we should realert
        if config.RealertInterval > 0 && config.MaxRealerts > 0 {
            if sentAlert.SendCount >= config.MaxRealerts {
                return nil // Max realerts reached
            }
            
            if now.Sub(sentAlert.LastSent) < config.RealertInterval {
                return nil // Too soon to realert
            }
        } else {
            return nil // No realert configured
        }
    }

    // Build and send the notification
    message := p.buildProblemMessage(host, check, status, sentAlert, config)
    
    pushoverMsg := &PushoverMessage{
        Token:     config.APIToken,
        User:      config.UserKey,
        Device:    config.Device,
        Title:     p.buildTitle(config, host, check, getSeverityName(status.ExitCode)),
        Message:   message,
        Priority:  config.Priority,
        Sound:     config.Sound,
        URL:       p.buildURL(config, host, check),
        URLTitle:  config.URLTitle,
        Timestamp: status.Timestamp.Unix(),
    }

    if err := p.sendMessage(ctx, pushoverMsg); err != nil {
        return fmt.Errorf("failed to send problem notification: %w", err)
    }

    // Update sent alert tracking
    sentAlert.LastSent = now
    sentAlert.SendCount++

    logType := "Sent"
    if sentAlert.SendCount > 1 {
        logType = "Resent"
    }

    logrus.WithFields(logrus.Fields{
        "host":       host.Name,
        "check":      check.Name,
        "severity":   getSeverityName(status.ExitCode),
        "send_count": sentAlert.SendCount,
        "type":       logType,
    }).Info("Sent problem notification")

    return nil
}

// buildProblemMessage creates the message text for problem notifications
func (p *PushoverClient) buildProblemMessage(host *database.Host, check *database.Check, status *database.Status, sentAlert *SentAlert, config *config.EffectivePushoverConfig) string {
    if config.MessageTemplate != "" {
        return p.expandTemplate(config.MessageTemplate, host, check, status, sentAlert)
    }

    severity := getSeverityName(status.ExitCode)
    hostName := host.DisplayName
    if hostName == "" {
        hostName = host.Name
    }

    var message strings.Builder
    message.WriteString(fmt.Sprintf("%s: %s on %s\n", severity, check.Name, hostName))
    
    if status.Output != "" {
        message.WriteString(fmt.Sprintf("Output: %s\n", status.Output))
    }
    
    if host.IPv4 != "" {
        message.WriteString(fmt.Sprintf("Address: %s\n", host.IPv4))
    } else if host.Hostname != "" {
        message.WriteString(fmt.Sprintf("Hostname: %s\n", host.Hostname))
    }
    
    if sentAlert.SendCount > 0 {
        message.WriteString(fmt.Sprintf("Duration: %s\n", 
            time.Since(sentAlert.FirstSent).Truncate(time.Second)))
        
        if sentAlert.SendCount > 1 {
            message.WriteString(fmt.Sprintf("Alert #%d\n", sentAlert.SendCount))
        }
    }
    
    message.WriteString(fmt.Sprintf("Time: %s", status.Timestamp.Format("2006-01-02 15:04:05")))

    return message.String()
}

// buildRecoveryMessage creates the message text for recovery notifications
func (p *PushoverClient) buildRecoveryMessage(host *database.Host, check *database.Check, status *database.Status, sentAlert *SentAlert) string {
    hostName := host.DisplayName
    if hostName == "" {
        hostName = host.Name
    }

    var message strings.Builder
    message.WriteString(fmt.Sprintf("RECOVERED: %s on %s\n", check.Name, hostName))
    
    if status.Output != "" {
        message.WriteString(fmt.Sprintf("Output: %s\n", status.Output))
    }
    
    duration := time.Since(sentAlert.FirstSent).Truncate(time.Second)
    message.WriteString(fmt.Sprintf("Downtime: %s\n", duration))
    
    if sentAlert.SendCount > 1 {
        message.WriteString(fmt.Sprintf("Alerts sent: %d\n", sentAlert.SendCount))
    }
    
    message.WriteString(fmt.Sprintf("Time: %s", status.Timestamp.Format("2006-01-02 15:04:05")))

    return message.String()
}

// buildTitle creates the notification title
func (p *PushoverClient) buildTitle(config *config.EffectivePushoverConfig, host *database.Host, check *database.Check, severity string) string {
    if config.Title != "" {
        return fmt.Sprintf("%s - %s", config.Title, severity)
    }
    
    hostName := host.DisplayName
    if hostName == "" {
        hostName = host.Name
    }
    
    return fmt.Sprintf("Raven: %s - %s", hostName, severity)
}

// buildURL creates the notification URL
func (p *PushoverClient) buildURL(config *config.EffectivePushoverConfig, host *database.Host, check *database.Check) string {
    if config.URL == "" {
        return ""
    }
    
    // Replace placeholders in URL
    url := config.URL
    url = strings.ReplaceAll(url, "{HOST_ID}", host.ID)
    url = strings.ReplaceAll(url, "{CHECK_ID}", check.ID)
    url = strings.ReplaceAll(url, "{HOST_NAME}", host.Name)
    url = strings.ReplaceAll(url, "{CHECK_NAME}", check.Name)
    
    return url
}

// expandTemplate expands a custom message template
func (p *PushoverClient) expandTemplate(template string, host *database.Host, check *database.Check, status *database.Status, sentAlert *SentAlert) string {
    result := template
    
    // Host variables
    result = strings.ReplaceAll(result, "{HOST_NAME}", host.Name)
    result = strings.ReplaceAll(result, "{HOST_DISPLAY_NAME}", host.DisplayName)
    result = strings.ReplaceAll(result, "{HOST_IP}", host.IPv4)
    result = strings.ReplaceAll(result, "{HOST_HOSTNAME}", host.Hostname)
    result = strings.ReplaceAll(result, "{HOST_GROUP}", host.Group)
    
    // Check variables
    result = strings.ReplaceAll(result, "{CHECK_NAME}", check.Name)
    result = strings.ReplaceAll(result, "{CHECK_TYPE}", check.Type)
    
    // Status variables
    result = strings.ReplaceAll(result, "{SEVERITY}", getSeverityName(status.ExitCode))
    result = strings.ReplaceAll(result, "{OUTPUT}", status.Output)
    result = strings.ReplaceAll(result, "{TIMESTAMP}", status.Timestamp.Format("2006-01-02 15:04:05"))
    
    // Alert tracking variables
    if sentAlert != nil {
        result = strings.ReplaceAll(result, "{DURATION}", time.Since(sentAlert.FirstSent).Truncate(time.Second).String())
        result = strings.ReplaceAll(result, "{ALERT_COUNT}", fmt.Sprintf("%d", sentAlert.SendCount))
    }
    
    return result
}

// sendMessage sends a message via the Pushover API
func (p *PushoverClient) sendMessage(ctx context.Context, msg *PushoverMessage) error {
    data := url.Values{}
    data.Set("token", msg.Token)
    data.Set("user", msg.User)
    data.Set("message", msg.Message)
    
    if msg.Device != "" {
        data.Set("device", msg.Device)
    }
    if msg.Title != "" {
        data.Set("title", msg.Title)
    }
    if msg.Priority != 0 {
        data.Set("priority", fmt.Sprintf("%d", msg.Priority))
    }
    if msg.Sound != "" {
        data.Set("sound", msg.Sound)
    }
    if msg.URL != "" {
        data.Set("url", msg.URL)
    }
    if msg.URLTitle != "" {
        data.Set("url_title", msg.URLTitle)
    }
    if msg.Timestamp > 0 {
        data.Set("timestamp", fmt.Sprintf("%d", msg.Timestamp))
    }

    req, err := http.NewRequestWithContext(ctx, "POST", PushoverAPIURL, strings.NewReader(data.Encode()))
    if err != nil {
        return fmt.Errorf("failed to create request: %w", err)
    }
    
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

    resp, err := p.client.Do(req)
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

    return nil
}

// TestConnection tests the Pushover configuration by sending a test message
func (p *PushoverClient) TestConnection(ctx context.Context) error {
    if !p.config.Enabled {
        return fmt.Errorf("pushover is not enabled")
    }

    msg := &PushoverMessage{
        Token:    p.config.APIToken,
        User:     p.config.UserKey,
        Device:   p.config.Device,
        Title:    "Raven Test",
        Message:  "This is a test notification from Raven monitoring system.",
        Priority: 0,
        Sound:    p.config.Sound,
    }

    return p.sendMessage(ctx, msg)
}

// CleanupResolvedAlerts removes resolved alerts from tracking after a period
func (p *PushoverClient) CleanupResolvedAlerts(maxAge time.Duration) {
    cutoff := time.Now().Add(-maxAge)
    
    for key, alert := range p.sentAlerts {
        if alert.Resolved && alert.LastSent.Before(cutoff) {
            delete(p.sentAlerts, key)
        }
    }
}

// getSeverityName converts exit code to severity name
func getSeverityName(exitCode int) string {
    switch exitCode {
    case 0:
        return "OK"
    case 1:
        return "WARNING"
    case 2:
        return "CRITICAL"
    default:
        return "UNKNOWN"
    }
}
