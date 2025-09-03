// internal/config/pushover.go - Pushover configuration structures
package config

import (
    "fmt"
    "time"
)

// PushoverConfig holds global Pushover notification settings
type PushoverConfig struct {
    Enabled           bool              `yaml:"enabled"`
    UserKey           string            `yaml:"user_key"`
    APIToken          string            `yaml:"api_token"`
    Device            string            `yaml:"device,omitempty"`
    Priority          int               `yaml:"priority"`           // -2 to 2
    Sound             string            `yaml:"sound,omitempty"`    // pushover sound name
    QuietHours        *QuietHours       `yaml:"quiet_hours,omitempty"`
    RealertInterval   time.Duration     `yaml:"realert_interval"`   // how often to resend
    MaxRealerts       int               `yaml:"max_realerts"`       // max number of realerts
    SendRecovery      bool              `yaml:"send_recovery"`      // send when status goes OK
    Title             string            `yaml:"title,omitempty"`    // default title prefix
    URLTitle          string            `yaml:"url_title,omitempty"` // title for URL
    URL               string            `yaml:"url,omitempty"`      // URL to include
    Overrides         []PushoverOverride `yaml:"overrides,omitempty"` // per-host/check overrides
}

// QuietHours defines when notifications should be suppressed
type QuietHours struct {
    Enabled   bool   `yaml:"enabled"`
    StartHour int    `yaml:"start_hour"` // 0-23
    EndHour   int    `yaml:"end_hour"`   // 0-23  
    Timezone  string `yaml:"timezone"`   // IANA timezone, e.g., "America/New_York"
}

// PushoverOverride allows per-host, per-check, or per-host-check customization
type PushoverOverride struct {
    Name              string        `yaml:"name"`                          // descriptive name
    HostID            string        `yaml:"host_id,omitempty"`            // specific host
    CheckID           string        `yaml:"check_id,omitempty"`           // specific check
    HostPattern       string        `yaml:"host_pattern,omitempty"`       // regex pattern for hosts
    CheckPattern      string        `yaml:"check_pattern,omitempty"`      // regex pattern for checks
    Severity          []string      `yaml:"severity,omitempty"`           // only for these severities
    Enabled           *bool         `yaml:"enabled,omitempty"`            // override global enabled
    UserKey           string        `yaml:"user_key,omitempty"`           // different user/group
    Priority          *int          `yaml:"priority,omitempty"`           // override priority
    Sound             string        `yaml:"sound,omitempty"`              // override sound
    QuietHours        *QuietHours   `yaml:"quiet_hours,omitempty"`        // override quiet hours
    RealertInterval   time.Duration `yaml:"realert_interval,omitempty"`   // override realert interval
    MaxRealerts       *int          `yaml:"max_realerts,omitempty"`       // override max realerts
    SendRecovery      *bool         `yaml:"send_recovery,omitempty"`      // override recovery sending
    Title             string        `yaml:"title,omitempty"`              // custom title
    MessageTemplate   string        `yaml:"message_template,omitempty"`   // custom message format
}

// Validate ensures the Pushover configuration is valid
func (p *PushoverConfig) Validate() error {
    if !p.Enabled {
        return nil
    }

    if p.UserKey == "" {
        return fmt.Errorf("pushover user_key is required when enabled")
    }

    if p.APIToken == "" {
        return fmt.Errorf("pushover api_token is required when enabled")
    }

    if p.Priority < -2 || p.Priority > 2 {
        return fmt.Errorf("pushover priority must be between -2 and 2")
    }

    if p.QuietHours != nil && p.QuietHours.Enabled {
        if p.QuietHours.StartHour < 0 || p.QuietHours.StartHour > 23 {
            return fmt.Errorf("quiet hours start_hour must be between 0 and 23")
        }
        if p.QuietHours.EndHour < 0 || p.QuietHours.EndHour > 23 {
            return fmt.Errorf("quiet hours end_hour must be between 0 and 23")
        }
        if p.QuietHours.Timezone == "" {
            p.QuietHours.Timezone = "UTC"
        }
    }

    if p.RealertInterval < 0 {
        return fmt.Errorf("realert_interval must be positive")
    }

    if p.MaxRealerts < 0 {
        return fmt.Errorf("max_realerts must be non-negative")
    }

    return nil
}

// GetEffectiveConfig returns the effective Pushover configuration for a specific host/check
func (p *PushoverConfig) GetEffectiveConfig(hostID, checkID, hostName, checkName, severity string) *EffectivePushoverConfig {
    if !p.Enabled {
        return nil
    }

    effective := &EffectivePushoverConfig{
        Enabled:         p.Enabled,
        UserKey:         p.UserKey,
        APIToken:        p.APIToken,
        Device:          p.Device,
        Priority:        p.Priority,
        Sound:           p.Sound,
        QuietHours:      p.QuietHours,
        RealertInterval: p.RealertInterval,
        MaxRealerts:     p.MaxRealerts,
        SendRecovery:    p.SendRecovery,
        Title:           p.Title,
        URLTitle:        p.URLTitle,
        URL:             p.URL,
    }

    // Apply overrides in order, with later ones taking precedence
    for _, override := range p.Overrides {
        if override.Matches(hostID, checkID, hostName, checkName, severity) {
            effective.ApplyOverride(&override)
        }
    }

    return effective
}

// Matches determines if an override applies to the given host/check combination
func (o *PushoverOverride) Matches(hostID, checkID, hostName, checkName, severity string) bool {
    // Check severity filter first
    if len(o.Severity) > 0 {
        found := false
        for _, s := range o.Severity {
            if s == severity {
                found = true
                break
            }
        }
        if !found {
            return false
        }
    }

    // Check specific host/check IDs
    if o.HostID != "" && o.HostID != hostID {
        return false
    }
    if o.CheckID != "" && o.CheckID != checkID {
        return false
    }

    // Check patterns (would need regex matching in real implementation)
    if o.HostPattern != "" {
        // TODO: Implement regex matching for hostName
    }
    if o.CheckPattern != "" {
        // TODO: Implement regex matching for checkName
    }

    return true
}

// EffectivePushoverConfig represents the final configuration after applying overrides
type EffectivePushoverConfig struct {
    Enabled         bool
    UserKey         string
    APIToken        string
    Device          string
    Priority        int
    Sound           string
    QuietHours      *QuietHours
    RealertInterval time.Duration
    MaxRealerts     int
    SendRecovery    bool
    Title           string
    URLTitle        string
    URL             string
    MessageTemplate string
}

// ApplyOverride applies an override to the effective configuration
func (e *EffectivePushoverConfig) ApplyOverride(override *PushoverOverride) {
    if override.Enabled != nil {
        e.Enabled = *override.Enabled
    }
    if override.UserKey != "" {
        e.UserKey = override.UserKey
    }
    if override.Priority != nil {
        e.Priority = *override.Priority
    }
    if override.Sound != "" {
        e.Sound = override.Sound
    }
    if override.QuietHours != nil {
        e.QuietHours = override.QuietHours
    }
    if override.RealertInterval > 0 {
        e.RealertInterval = override.RealertInterval
    }
    if override.MaxRealerts != nil {
        e.MaxRealerts = *override.MaxRealerts
    }
    if override.SendRecovery != nil {
        e.SendRecovery = *override.SendRecovery
    }
    if override.Title != "" {
        e.Title = override.Title
    }
    if override.MessageTemplate != "" {
        e.MessageTemplate = override.MessageTemplate
    }
}

// IsQuietTime checks if the current time falls within quiet hours
func (e *EffectivePushoverConfig) IsQuietTime() bool {
    if e.QuietHours == nil || !e.QuietHours.Enabled {
        return false
    }

    loc, err := time.LoadLocation(e.QuietHours.Timezone)
    if err != nil {
        loc = time.UTC
    }

    now := time.Now().In(loc)
    hour := now.Hour()

    start := e.QuietHours.StartHour
    end := e.QuietHours.EndHour

    // Handle cases where quiet hours span midnight
    if start <= end {
        return hour >= start && hour < end
    } else {
        return hour >= start || hour < end
    }
}
