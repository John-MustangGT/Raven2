// internal/database/models.go
package database

import (
    "time"
)

type Host struct {
    ID          string            `json:"id"`
    Name        string            `json:"name"`
    DisplayName string            `json:"display_name"`
    IPv4        string            `json:"ipv4"`
    Hostname    string            `json:"hostname"`
    Group       string            `json:"group"`
    Enabled     bool              `json:"enabled"`
    Tags        map[string]string `json:"tags"`
    CreatedAt   time.Time         `json:"created_at"`
    UpdatedAt   time.Time         `json:"updated_at"`
}

type Check struct {
    ID        string                   `json:"id"`
    Name      string                   `json:"name"`
    Type      string                   `json:"type"`
    Hosts     []string                 `json:"hosts"`
    Interval  map[string]time.Duration `json:"interval"`
    Threshold int                      `json:"threshold"`
    Timeout   time.Duration            `json:"timeout"`
    Enabled   bool                     `json:"enabled"`
    Options   map[string]interface{}   `json:"options"`
    CreatedAt time.Time                `json:"created_at"`
    UpdatedAt time.Time                `json:"updated_at"`
}

type Status struct {
    ID         string    `json:"id"`
    HostID     string    `json:"host_id"`
    CheckID    string    `json:"check_id"`
    ExitCode   int       `json:"exit_code"`
    Output     string    `json:"output"`
    PerfData   string    `json:"perf_data"`
    LongOutput string    `json:"long_output"`
    Duration   float64   `json:"duration_ms"`
    Timestamp  time.Time `json:"timestamp"`
}

type HostFilters struct {
    Group   string
    Enabled *bool
    Tags    map[string]string
}

type StatusFilters struct {
    HostID   string
    CheckID  string
    ExitCode *int
    Since    *time.Time
    Limit    int
}
