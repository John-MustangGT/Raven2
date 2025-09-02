// internal/config/config.go - Enhanced with Pushover notification support
package config

import (
    "fmt"
    "os"
    "path/filepath"
    "strings"
    "time"

    "gopkg.in/yaml.v3"
)

type Config struct {
    Server       ServerConfig       `yaml:"server"`
    Web          WebConfig          `yaml:"web"`
    Database     DatabaseConfig     `yaml:"database"`
    Prometheus   PrometheusConfig   `yaml:"prometheus"`
    Monitoring   MonitoringConfig   `yaml:"monitoring"`
    Logging      LoggingConfig      `yaml:"logging"`
    Notifications NotificationConfig `yaml:"notifications"` // NEW: Notification configuration
    Hosts        []HostConfig       `yaml:"hosts"`
    Checks       []CheckConfig      `yaml:"checks"`
    Include      IncludeConfig      `yaml:"include"`
}

// NEW: Notification configuration
type NotificationConfig struct {
    Enabled  bool           `yaml:"enabled"`
    Pushover PushoverConfig `yaml:"pushover"`
    // Future: Email, Slack, Discord, etc.
}

// NEW: Pushover-specific configuration
type PushoverConfig struct {
    Enabled     bool              `yaml:"enabled"`
    APIToken    string            `yaml:"api_token"`
    UserKey     string            `yaml:"user_key"`
    Priority    int               `yaml:"priority"`      // -2 (silent), -1 (quiet), 0 (normal), 1 (high), 2 (emergency)
    Retry       int               `yaml:"retry"`         // For emergency priority (seconds)
    Expire      int               `yaml:"expire"`        // For emergency priority (seconds)
    Sound       string            `yaml:"sound"`         // notification sound
    Device      string            `yaml:"device"`        // specific device name (optional)
    Title       string            `yaml:"title"`         // notification title template
    Template    string            `yaml:"template"`      // message template
    OnlyOnState []string          `yaml:"only_on_state"` // Only notify for specific states: critical, warning, recovery
    Throttle    ThrottleConfig    `yaml:"throttle"`      // Rate limiting
}

// NEW: Notification throttling configuration
type ThrottleConfig struct {
    Enabled     bool          `yaml:"enabled"`
    Window      time.Duration `yaml:"window"`       // Time window for throttling
    MaxPerHost  int           `yaml:"max_per_host"` // Max notifications per host in window
    MaxTotal    int           `yaml:"max_total"`    // Max total notifications in window
}

type IncludeConfig struct {
    Directory string   `yaml:"directory"`
    Pattern   string   `yaml:"pattern"`
    Enabled   bool     `yaml:"enabled"`
}

type ServerConfig struct {
    Port         string        `yaml:"port"`
    Workers      int           `yaml:"workers"`
    PluginDir    string        `yaml:"plugin_dir"`
    ReadTimeout  time.Duration `yaml:"read_timeout"`
    WriteTimeout time.Duration `yaml:"write_timeout"`
}

type WebConfig struct {
    AssetsDir    string   `yaml:"assets_dir"`
    StaticDir    string   `yaml:"static_dir"`
    ServeStatic  bool     `yaml:"serve_static"`
    Root         string   `yaml:"root"`
    Files        []string `yaml:"files"`
    HeaderLink   string   `yaml:"header_link"`
}

type DatabaseConfig struct {
    Type              string        `yaml:"type"`
    Path              string        `yaml:"path"`
    BackupInterval    time.Duration `yaml:"backup_interval"`
    CleanupInterval   time.Duration `yaml:"cleanup_interval"`
    HistoryRetention  time.Duration `yaml:"history_retention"`
    CompactInterval   time.Duration `yaml:"compact_interval"`
}

type PrometheusConfig struct {
    Enabled     bool   `yaml:"enabled"`
    MetricsPath string `yaml:"metrics_path"`
    PushGateway string `yaml:"push_gateway"`
}

type MonitoringConfig struct {
    DefaultInterval   time.Duration `yaml:"default_interval"`
    MaxRetries        int           `yaml:"max_retries"`
    Timeout           time.Duration `yaml:"timeout"`
    BatchSize         int           `yaml:"batch_size"`
    DefaultThreshold  int           `yaml:"default_threshold"`
    SoftFailEnabled   bool          `yaml:"soft_fail_enabled"`
}

type LoggingConfig struct {
    Level  string `yaml:"level"`
    Format string `yaml:"format"`
}

type HostConfig struct {
    ID          string            `yaml:"id"`
    Name        string            `yaml:"name"`
    DisplayName string            `yaml:"display_name"`
    IPv4        string            `yaml:"ipv4"`
    Hostname    string            `yaml:"hostname"`
    Group       string            `yaml:"group"`
    Enabled     bool              `yaml:"enabled"`
    Tags        map[string]string `yaml:"tags"`
}

type CheckConfig struct {
    ID              string                   `yaml:"id"`
    Name            string                   `yaml:"name"`
    Type            string                   `yaml:"type"`
    Hosts           []string                 `yaml:"hosts"`
    Interval        map[string]time.Duration `yaml:"interval"`
    Threshold       int                      `yaml:"threshold"`
    SoftFailEnabled *bool                    `yaml:"soft_fail_enabled"`
    Timeout         time.Duration            `yaml:"timeout"`
    Enabled         bool                     `yaml:"enabled"`
    Options         map[string]interface{}   `yaml:"options"`
}

// PartialConfig represents a partial configuration that can be merged
type PartialConfig struct {
    Server        *ServerConfig        `yaml:"server,omitempty"`
    Web           *WebConfig           `yaml:"web,omitempty"`
    Database      *DatabaseConfig      `yaml:"database,omitempty"`
    Prometheus    *PrometheusConfig    `yaml:"prometheus,omitempty"`
    Monitoring    *MonitoringConfig    `yaml:"monitoring,omitempty"`
    Logging       *LoggingConfig       `yaml:"logging,omitempty"`
    Notifications *NotificationConfig  `yaml:"notifications,omitempty"` // NEW
    Hosts         []HostConfig         `yaml:"hosts,omitempty"`
    Checks        []CheckConfig        `yaml:"checks,omitempty"`
}

func Load(filename string) (*Config, error) {
    // Load the main config file
    config, err := loadConfigFile(filename)
    if err != nil {
        return nil, fmt.Errorf("failed to load main config file: %w", err)
    }

    // Process includes if enabled
    if config.Include.Enabled && config.Include.Directory != "" {
        if err := loadIncludes(config, filepath.Dir(filename)); err != nil {
            return nil, fmt.Errorf("failed to load includes: %w", err)
        }
    }

    // Set defaults
    setDefaults(config)

    // Validate
    if err := validate(config); err != nil {
        return nil, fmt.Errorf("invalid configuration: %w", err)
    }

    return config, nil
}

func loadConfigFile(filename string) (*Config, error) {
    data, err := os.ReadFile(filename)
    if err != nil {
        return nil, fmt.Errorf("failed to read config file: %w", err)
    }

    var config Config
    if err := yaml.Unmarshal(data, &config); err != nil {
        return nil, fmt.Errorf("failed to parse YAML: %w", err)
    }

    return &config, nil
}

func loadIncludes(config *Config, baseDir string) error {
    includeDir := config.Include.Directory
    
    // Make include directory relative to main config file if not absolute
    if !filepath.IsAbs(includeDir) {
        includeDir = filepath.Join(baseDir, includeDir)
    }

    // Check if include directory exists
    if _, err := os.Stat(includeDir); os.IsNotExist(err) {
        return fmt.Errorf("include directory does not exist: %s", includeDir)
    }

    // Set default pattern if not specified
    pattern := config.Include.Pattern
    if pattern == "" {
        pattern = "*.yaml"
    }

    // Find matching files
    matches, err := filepath.Glob(filepath.Join(includeDir, pattern))
    if err != nil {
        return fmt.Errorf("failed to glob include pattern: %w", err)
    }

    // Also check for .yml files if pattern is default
    if pattern == "*.yaml" {
        ymlMatches, err := filepath.Glob(filepath.Join(includeDir, "*.yml"))
        if err != nil {
            return fmt.Errorf("failed to glob .yml files: %w", err)
        }
        matches = append(matches, ymlMatches...)
    }

    // Sort files for consistent ordering
    if len(matches) > 1 {
        // Simple sort by filename
        for i := 0; i < len(matches)-1; i++ {
            for j := i + 1; j < len(matches); j++ {
                if filepath.Base(matches[i]) > filepath.Base(matches[j]) {
                    matches[i], matches[j] = matches[j], matches[i]
                }
            }
        }
    }

    // Load and merge each include file
    for _, match := range matches {
        if err := loadAndMergeInclude(config, match); err != nil {
            return fmt.Errorf("failed to load include file %s: %w", match, err)
        }
    }

    return nil
}

func loadAndMergeInclude(config *Config, filename string) error {
    data, err := os.ReadFile(filename)
    if err != nil {
        return fmt.Errorf("failed to read include file: %w", err)
    }

    var partial PartialConfig
    if err := yaml.Unmarshal(data, &partial); err != nil {
        return fmt.Errorf("failed to parse include file YAML: %w", err)
    }

    // Merge the partial config into the main config
    mergePartialConfig(config, &partial)

    return nil
}

func mergePartialConfig(config *Config, partial *PartialConfig) {
    // Merge hosts (append to existing)
    if len(partial.Hosts) > 0 {
        config.Hosts = append(config.Hosts, partial.Hosts...)
    }

    // Merge checks with smart host appending
    if len(partial.Checks) > 0 {
        mergeChecks(config, partial.Checks)
    }

    // For other sections, only override if they exist in the partial config
    if partial.Server != nil {
        mergeServerConfig(&config.Server, partial.Server)
    }

    if partial.Web != nil {
        mergeWebConfig(&config.Web, partial.Web)
    }

    if partial.Database != nil {
        mergeDatabaseConfig(&config.Database, partial.Database)
    }

    if partial.Prometheus != nil {
        mergePrometheusConfig(&config.Prometheus, partial.Prometheus)
    }

    if partial.Monitoring != nil {
        mergeMonitoringConfig(&config.Monitoring, partial.Monitoring)
    }

    if partial.Logging != nil {
        mergeLoggingConfig(&config.Logging, partial.Logging)
    }

    // NEW: Merge notification config
    if partial.Notifications != nil {
        mergeNotificationConfig(&config.Notifications, partial.Notifications)
    }
}

// NEW: Merge notification configuration
func mergeNotificationConfig(main *NotificationConfig, partial *NotificationConfig) {
    main.Enabled = partial.Enabled
    
    // Merge Pushover config
    if partial.Pushover.APIToken != "" {
        main.Pushover.APIToken = partial.Pushover.APIToken
    }
    if partial.Pushover.UserKey != "" {
        main.Pushover.UserKey = partial.Pushover.UserKey
    }
    if partial.Pushover.Priority != 0 || !main.Pushover.Enabled {
        main.Pushover.Priority = partial.Pushover.Priority
    }
    if partial.Pushover.Retry != 0 {
        main.Pushover.Retry = partial.Pushover.Retry
    }
    if partial.Pushover.Expire != 0 {
        main.Pushover.Expire = partial.Pushover.Expire
    }
    if partial.Pushover.Sound != "" {
        main.Pushover.Sound = partial.Pushover.Sound
    }
    if partial.Pushover.Device != "" {
        main.Pushover.Device = partial.Pushover.Device
    }
    if partial.Pushover.Title != "" {
        main.Pushover.Title = partial.Pushover.Title
    }
    if partial.Pushover.Template != "" {
        main.Pushover.Template = partial.Pushover.Template
    }
    if len(partial.Pushover.OnlyOnState) > 0 {
        main.Pushover.OnlyOnState = partial.Pushover.OnlyOnState
    }
    
    main.Pushover.Enabled = partial.Pushover.Enabled
    
    // Merge throttle config
    if partial.Pushover.Throttle.Enabled {
        main.Pushover.Throttle = partial.Pushover.Throttle
    }
}

func mergeChecks(config *Config, newChecks []CheckConfig) {
    // Create a map of existing checks by ID for quick lookup
    existingChecks := make(map[string]*CheckConfig)
    for i := range config.Checks {
        existingChecks[config.Checks[i].ID] = &config.Checks[i]
    }

    for _, newCheck := range newChecks {
        if existingCheck, exists := existingChecks[newCheck.ID]; exists {
            // Check if this is a partial definition (only ID and hosts specified)
            if isPartialCheckDefinition(newCheck) {
                // Append hosts to existing check
                appendHostsToCheck(existingCheck, newCheck.Hosts)
            } else {
                // This is a full check definition, replace the existing one
                *existingCheck = newCheck
            }
        } else {
            // New check, add it to the config
            config.Checks = append(config.Checks, newCheck)
            existingChecks[newCheck.ID] = &config.Checks[len(config.Checks)-1]
        }
    }
}

func isPartialCheckDefinition(check CheckConfig) bool {
    // Check if only ID and hosts are specified (all other fields are zero values)
    return check.ID != "" &&
           len(check.Hosts) > 0 &&
           check.Name == "" &&
           check.Type == "" &&
           len(check.Interval) == 0 &&
           check.Threshold == 0 &&
           check.Timeout == 0 &&
           !check.Enabled &&
           len(check.Options) == 0 &&
           check.SoftFailEnabled == nil
}

func appendHostsToCheck(existingCheck *CheckConfig, newHosts []string) {
    // Create a set of existing hosts for quick lookup
    existingHosts := make(map[string]bool)
    for _, host := range existingCheck.Hosts {
        existingHosts[host] = true
    }

    // Append new hosts that don't already exist
    for _, host := range newHosts {
        if !existingHosts[host] {
            existingCheck.Hosts = append(existingCheck.Hosts, host)
        }
    }
}

func mergeServerConfig(main *ServerConfig, partial *ServerConfig) {
    if partial.Port != "" {
        main.Port = partial.Port
    }
    if partial.Workers != 0 {
        main.Workers = partial.Workers
    }
    if partial.PluginDir != "" {
        main.PluginDir = partial.PluginDir
    }
    if partial.ReadTimeout != 0 {
        main.ReadTimeout = partial.ReadTimeout
    }
    if partial.WriteTimeout != 0 {
        main.WriteTimeout = partial.WriteTimeout
    }
}

func mergeWebConfig(main *WebConfig, partial *WebConfig) {
    if partial.AssetsDir != "" {
        main.AssetsDir = partial.AssetsDir
    }
    if partial.StaticDir != "" {
        main.StaticDir = partial.StaticDir
    }
    if partial.Root != "" {
        main.Root = partial.Root
    }
    if partial.HeaderLink != "" {
        main.HeaderLink = partial.HeaderLink
    }
    main.ServeStatic = partial.ServeStatic
    
    if len(partial.Files) > 0 {
        main.Files = append(main.Files, partial.Files...)
    }
}

func mergeDatabaseConfig(main *DatabaseConfig, partial *DatabaseConfig) {
    if partial.Type != "" {
        main.Type = partial.Type
    }
    if partial.Path != "" {
        main.Path = partial.Path
    }
    if partial.BackupInterval != 0 {
        main.BackupInterval = partial.BackupInterval
    }
    if partial.CleanupInterval != 0 {
        main.CleanupInterval = partial.CleanupInterval
    }
    if partial.HistoryRetention != 0 {
        main.HistoryRetention = partial.HistoryRetention
    }
    if partial.CompactInterval != 0 {
        main.CompactInterval = partial.CompactInterval
    }
}

func mergePrometheusConfig(main *PrometheusConfig, partial *PrometheusConfig) {
    main.Enabled = partial.Enabled
    if partial.MetricsPath != "" {
        main.MetricsPath = partial.MetricsPath
    }
    if partial.PushGateway != "" {
        main.PushGateway = partial.PushGateway
    }
}

func mergeMonitoringConfig(main *MonitoringConfig, partial *MonitoringConfig) {
    if partial.DefaultInterval != 0 {
        main.DefaultInterval = partial.DefaultInterval
    }
    if partial.MaxRetries != 0 {
        main.MaxRetries = partial.MaxRetries
    }
    if partial.Timeout != 0 {
        main.Timeout = partial.Timeout
    }
    if partial.BatchSize != 0 {
        main.BatchSize = partial.BatchSize
    }
    if partial.DefaultThreshold != 0 {
        main.DefaultThreshold = partial.DefaultThreshold
    }
    main.SoftFailEnabled = partial.SoftFailEnabled
}

func mergeLoggingConfig(main *LoggingConfig, partial *LoggingConfig) {
    if partial.Level != "" {
        main.Level = partial.Level
    }
    if partial.Format != "" {
        main.Format = partial.Format
    }
}

func setDefaults(cfg *Config) {
    // Server defaults
    if cfg.Server.Port == "" {
        cfg.Server.Port = ":8000"
    }
    if cfg.Server.Workers == 0 {
        cfg.Server.Workers = 3
    }
    
    // Database defaults
    if cfg.Database.Type == "" {
        cfg.Database.Type = "boltdb"
    }
    if cfg.Database.Path == "" {
        cfg.Database.Path = "./data/raven.db"
    }
    
    // Web defaults
    if cfg.Web.StaticDir == "" {
        cfg.Web.StaticDir = "static"
    }
    if cfg.Web.Root == "" {
        cfg.Web.Root = "index.html"
    }
    if cfg.Web.HeaderLink == "" {
        cfg.Web.HeaderLink = "https://github.com/John-MustangGT/raven2"
    }
    
    // Include defaults
    if cfg.Include.Pattern == "" {
        cfg.Include.Pattern = "*.yaml"
    }
    
    // Monitoring defaults
    if cfg.Monitoring.DefaultInterval == 0 {
        cfg.Monitoring.DefaultInterval = 5 * time.Minute
    }
    if cfg.Monitoring.DefaultThreshold == 0 {
        cfg.Monitoring.DefaultThreshold = 3
    }
    if cfg.Monitoring.Timeout == 0 {
        cfg.Monitoring.Timeout = 30 * time.Second
    }
    
    // Prometheus defaults
    if cfg.Prometheus.MetricsPath == "" {
        cfg.Prometheus.MetricsPath = "/metrics"
    }
    
    // Logging defaults
    if cfg.Logging.Level == "" {
        cfg.Logging.Level = "info"
    }
    if cfg.Logging.Format == "" {
        cfg.Logging.Format = "text"
    }
    
    // NEW: Notification defaults
    if cfg.Notifications.Pushover.Title == "" {
        cfg.Notifications.Pushover.Title = "Raven Alert: {{.Host}}"
    }
    if cfg.Notifications.Pushover.Template == "" {
        cfg.Notifications.Pushover.Template = "{{.Check}} on {{.Host}} is {{.Status}}: {{.Output}}"
    }
    if len(cfg.Notifications.Pushover.OnlyOnState) == 0 {
        cfg.Notifications.Pushover.OnlyOnState = []string{"critical", "warning", "recovery"}
    }
    if cfg.Notifications.Pushover.Sound == "" {
        cfg.Notifications.Pushover.Sound = "pushover"
    }
    // Throttle defaults
    if cfg.Notifications.Pushover.Throttle.Window == 0 {
        cfg.Notifications.Pushover.Throttle.Window = 15 * time.Minute
    }
    if cfg.Notifications.Pushover.Throttle.MaxPerHost == 0 {
        cfg.Notifications.Pushover.Throttle.MaxPerHost = 5
    }
    if cfg.Notifications.Pushover.Throttle.MaxTotal == 0 {
        cfg.Notifications.Pushover.Throttle.MaxTotal = 20
    }
}

func validate(cfg *Config) error {
    if cfg.Server.Workers < 1 {
        return fmt.Errorf("server.workers must be at least 1")
    }
    if cfg.Database.Type != "boltdb" {
        return fmt.Errorf("only boltdb is supported currently")
    }
    
    // Validate monitoring configuration
    if cfg.Monitoring.DefaultThreshold < 1 {
        return fmt.Errorf("monitoring.default_threshold must be at least 1")
    }
    if cfg.Monitoring.DefaultInterval <= 0 {
        return fmt.Errorf("monitoring.default_interval must be positive")
    }
    
    // Validate web configuration
    if cfg.Web.Root == "" {
        return fmt.Errorf("web.root cannot be empty")
    }
    
    // Validate header link if provided
    if cfg.Web.HeaderLink != "" {
        if !isValidURL(cfg.Web.HeaderLink) {
            return fmt.Errorf("web.header_link must be a valid URL")
        }
    }
    
    // NEW: Validate notification configuration
    if cfg.Notifications.Enabled && cfg.Notifications.Pushover.Enabled {
        if cfg.Notifications.Pushover.APIToken == "" {
            return fmt.Errorf("notifications.pushover.api_token is required when Pushover is enabled")
        }
        if cfg.Notifications.Pushover.UserKey == "" {
            return fmt.Errorf("notifications.pushover.user_key is required when Pushover is enabled")
        }
        if cfg.Notifications.Pushover.Priority < -2 || cfg.Notifications.Pushover.Priority > 2 {
            return fmt.Errorf("notifications.pushover.priority must be between -2 and 2")
        }
        // Emergency priority requires retry and expire
        if cfg.Notifications.Pushover.Priority == 2 {
            if cfg.Notifications.Pushover.Retry < 30 {
                return fmt.Errorf("notifications.pushover.retry must be at least 30 seconds for emergency priority")
            }
            if cfg.Notifications.Pushover.Expire < 60 {
                return fmt.Errorf("notifications.pushover.expire must be at least 60 seconds for emergency priority")
            }
            if cfg.Notifications.Pushover.Expire > 10800 {
                return fmt.Errorf("notifications.pushover.expire cannot exceed 10800 seconds (3 hours)")
            }
        }
    }
    
    // If assets_dir is specified, validate it exists
    if cfg.Web.AssetsDir != "" {
        if _, err := os.Stat(cfg.Web.AssetsDir); err != nil {
            return fmt.Errorf("web.assets_dir '%s' does not exist or is not accessible: %w", cfg.Web.AssetsDir, err)
        }
    }
    
    // Validate that files in the files list are reasonable
    for _, filename := range cfg.Web.Files {
        if filename == "" {
            return fmt.Errorf("web.files contains empty filename")
        }
        // Check for path traversal attempts
        if containsPathTraversal(filename) {
            return fmt.Errorf("web.files contains invalid filename with path traversal: %s", filename)
        }
    }
    
    // Validate include configuration
    if cfg.Include.Enabled {
        if cfg.Include.Directory == "" {
            return fmt.Errorf("include.directory must be specified when include.enabled is true")
        }
        if cfg.Include.Pattern != "" && !isValidGlobPattern(cfg.Include.Pattern) {
            return fmt.Errorf("include.pattern contains invalid glob pattern: %s", cfg.Include.Pattern)
        }
    }
    
    // Validate for duplicate host IDs
    hostIDs := make(map[string]bool)
    for _, host := range cfg.Hosts {
        if hostIDs[host.ID] {
            return fmt.Errorf("duplicate host ID: %s", host.ID)
        }
        hostIDs[host.ID] = true
    }
    
    // Validate check configurations
    for _, check := range cfg.Checks {
        if check.Threshold < 0 {
            return fmt.Errorf("check '%s' has invalid threshold: %d (must be >= 0)", check.ID, check.Threshold)
        }
        if check.Timeout <= 0 {
            check.Timeout = cfg.Monitoring.Timeout // Use default if not specified
        }
        
        // Validate that hosts exist
        for _, hostID := range check.Hosts {
            hostExists := false
            for _, host := range cfg.Hosts {
                if host.ID == hostID {
                    hostExists = true
                    break
                }
            }
            if !hostExists {
                return fmt.Errorf("check '%s' references non-existent host: %s", check.ID, hostID)
            }
        }
        
        // Validate intervals
        if len(check.Interval) == 0 {
            // Set default intervals if not specified
            check.Interval = map[string]time.Duration{
                "ok":       cfg.Monitoring.DefaultInterval,
                "warning":  cfg.Monitoring.DefaultInterval / 2,
                "critical": cfg.Monitoring.DefaultInterval / 4,
                "unknown":  cfg.Monitoring.DefaultInterval,
            }
        }
        
        // Ensure all required intervals are present
        requiredStates := []string{"ok", "warning", "critical", "unknown"}
        for _, state := range requiredStates {
            if _, exists := check.Interval[state]; !exists {
                check.Interval[state] = cfg.Monitoring.DefaultInterval
            }
        }
    }
    
    return nil
}

// GetEffectiveThreshold returns the effective threshold for a check
func (c *CheckConfig) GetEffectiveThreshold(globalDefault int) int {
    if c.Threshold > 0 {
        return c.Threshold
    }
    return globalDefault
}

// IsSoftFailEnabled returns whether soft fail is enabled for this check
func (c *CheckConfig) IsSoftFailEnabled(globalEnabled bool) bool {
    if c.SoftFailEnabled != nil {
        return *c.SoftFailEnabled
    }
    return globalEnabled
}

// isValidURL checks if a string is a valid URL
func isValidURL(str string) bool {
    return len(str) > 7 && (str[:7] == "http://" || (len(str) > 8 && str[:8] == "https://"))
}

// containsPathTraversal checks if a filename contains path traversal sequences
func containsPathTraversal(filename string) bool {
    dangerous := []string{"../", "..\\", "/", "\\"}
    for _, pattern := range dangerous {
        if len(pattern) > 0 && (pattern == "/" || pattern == "\\") {
            if len(filename) > 0 && (filename[0] == '/' || filename[0] == '\\') {
                return true
            }
        } else if len(filename) >= len(pattern) {
            for i := 0; i <= len(filename)-len(pattern); i++ {
                if filename[i:i+len(pattern)] == pattern {
                    return true
                }
            }
        }
    }
    return false
}

// isValidGlobPattern checks if a string is a valid glob pattern
func isValidGlobPattern(pattern string) bool {
    if strings.Contains(pattern, "/") || strings.Contains(pattern, "\\") {
        return false
    }
    _, err := filepath.Match(pattern, "test.yaml")
    return err == nil
}
