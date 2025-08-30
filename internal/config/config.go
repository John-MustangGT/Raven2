// internal/config/config.go
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
    Server     ServerConfig     `yaml:"server"`
    Web        WebConfig        `yaml:"web"`
    Database   DatabaseConfig   `yaml:"database"`
    Prometheus PrometheusConfig `yaml:"prometheus"`
    Monitoring MonitoringConfig `yaml:"monitoring"`
    Logging    LoggingConfig    `yaml:"logging"`
    Hosts      []HostConfig     `yaml:"hosts"`
    Checks     []CheckConfig    `yaml:"checks"`
    Include    IncludeConfig    `yaml:"include"` // New include configuration
}

type IncludeConfig struct {
    Directory string   `yaml:"directory"` // Directory containing config files to include
    Pattern   string   `yaml:"pattern"`   // File pattern to match (default: "*.yaml")
    Enabled   bool     `yaml:"enabled"`   // Whether includes are enabled
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
    Root         string   `yaml:"root"`         // Root file to serve at "/"
    Files        []string `yaml:"files"`       // List of files to serve
    HeaderLink   string   `yaml:"header_link"` // Link for the sidebar header/logo
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
    DefaultInterval time.Duration `yaml:"default_interval"`
    MaxRetries      int           `yaml:"max_retries"`
    Timeout         time.Duration `yaml:"timeout"`
    BatchSize       int           `yaml:"batch_size"`
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
    ID        string                   `yaml:"id"`
    Name      string                   `yaml:"name"`
    Type      string                   `yaml:"type"`
    Hosts     []string                 `yaml:"hosts"`
    Interval  map[string]time.Duration `yaml:"interval"`
    Threshold int                      `yaml:"threshold"`
    Timeout   time.Duration            `yaml:"timeout"`
    Enabled   bool                     `yaml:"enabled"`
    Options   map[string]interface{}   `yaml:"options"`
}

// PartialConfig represents a partial configuration that can be merged
type PartialConfig struct {
    Server     *ServerConfig     `yaml:"server,omitempty"`
    Web        *WebConfig        `yaml:"web,omitempty"`
    Database   *DatabaseConfig   `yaml:"database,omitempty"`
    Prometheus *PrometheusConfig `yaml:"prometheus,omitempty"`
    Monitoring *MonitoringConfig `yaml:"monitoring,omitempty"`
    Logging    *LoggingConfig    `yaml:"logging,omitempty"`
    Hosts      []HostConfig      `yaml:"hosts,omitempty"`
    Checks     []CheckConfig     `yaml:"checks,omitempty"`
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
    // This allows include files to override main config settings

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
}

// mergeChecks handles smart merging of check configurations
// If a check with the same ID exists and the new check only has ID and hosts,
// append the hosts to the existing check. Otherwise, add as a new check.
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

// isPartialCheckDefinition determines if a check config is a partial definition
// A partial definition only has ID and hosts specified, all other fields are zero values
func isPartialCheckDefinition(check CheckConfig) bool {
    // Check if only ID and hosts are specified (all other fields are zero values)
    return check.ID != "" &&
           len(check.Hosts) > 0 &&
           check.Name == "" &&
           check.Type == "" &&
           len(check.Interval) == 0 &&
           check.Threshold == 0 &&
           check.Timeout == 0 &&
           !check.Enabled && // false is zero value for bool
           len(check.Options) == 0
}

// appendHostsToCheck appends new hosts to an existing check, avoiding duplicates
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
    // ServeStatic is a boolean, so we need to check if it was explicitly set
    // For simplicity, we'll always take the partial value
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
    main.Enabled = partial.Enabled // Always take the partial value for boolean
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
    if cfg.Server.Port == "" {
        cfg.Server.Port = ":8000"
    }
    if cfg.Server.Workers == 0 {
        cfg.Server.Workers = 3
    }
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
    
    if cfg.Prometheus.MetricsPath == "" {
        cfg.Prometheus.MetricsPath = "/metrics"
    }
    if cfg.Logging.Level == "" {
        cfg.Logging.Level = "info"
    }
    if cfg.Logging.Format == "" {
        cfg.Logging.Format = "text"
    }
}

func validate(cfg *Config) error {
    if cfg.Server.Workers < 1 {
        return fmt.Errorf("server.workers must be at least 1")
    }
    if cfg.Database.Type != "boltdb" {
        return fmt.Errorf("only boltdb is supported currently")
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
    
    // Note: We don't validate for duplicate check IDs anymore since 
    // the same check ID can appear multiple times for host list merging
    
    return nil
}

// isValidURL checks if a string is a valid URL
func isValidURL(str string) bool {
    // Simple URL validation - starts with http:// or https://
    return len(str) > 7 && (str[:7] == "http://" || (len(str) > 8 && str[:8] == "https://"))
}

// containsPathTraversal checks if a filename contains path traversal sequences
func containsPathTraversal(filename string) bool {
    // Simple check for common path traversal patterns
    dangerous := []string{"../", "..\\", "/", "\\"}
    for _, pattern := range dangerous {
        if len(pattern) > 0 && (pattern == "/" || pattern == "\\") {
            // Only check for leading slashes
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
    // Basic validation - reject patterns with path separators
    if strings.Contains(pattern, "/") || strings.Contains(pattern, "\\") {
        return false
    }
    // Try to use the pattern with filepath.Match to see if it's valid
    _, err := filepath.Match(pattern, "test.yaml")
    return err == nil
}
