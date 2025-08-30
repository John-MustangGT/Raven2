// internal/config/config.go
package config

import (
    "fmt"
    "os"
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

func Load(filename string) (*Config, error) {
    data, err := os.ReadFile(filename)
    if err != nil {
        return nil, fmt.Errorf("failed to read config file: %w", err)
    }

    var config Config
    if err := yaml.Unmarshal(data, &config); err != nil {
        return nil, fmt.Errorf("failed to parse YAML: %w", err)
    }

    // Set defaults
    setDefaults(&config)

    // Validate
    if err := validate(&config); err != nil {
        return nil, fmt.Errorf("invalid configuration: %w", err)
    }

    return &config, nil
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
    // Note: AssetsDir defaults to empty (auto-detect)
    // ServeStatic defaults to false (zero value)
    // Files defaults to empty slice (will use common defaults)
    
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
    
    return nil
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
