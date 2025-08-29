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
    return nil
}
