// internal/monitoring/engine.go
package monitoring

import (
    "context"
    "sync"
    "time"

    "github.com/sirupsen/logrus"
    "internal/config"
    "internal/database"
    "internal/metrics"
)

type Engine struct {
    config    *config.Config
    store     database.Store
    metrics   *metrics.Collector
    scheduler *Scheduler
    plugins   map[string]Plugin
    mu        sync.RWMutex
    running   bool
}

type Plugin interface {
    Name() string
    Init(options map[string]interface{}) error
    Execute(ctx context.Context, host *database.Host) (*CheckResult, error)
}

type CheckResult struct {
    ExitCode   int
    Output     string
    PerfData   string
    LongOutput string
    Duration   time.Duration
}

func NewEngine(cfg *config.Config, store database.Store, metricsCollector *metrics.Collector) (*Engine, error) {
    engine := &Engine{
        config:  cfg,
        store:   store,
        metrics: metricsCollector,
        plugins: make(map[string]Plugin),
    }

    // Initialize plugins
    if err := engine.loadPlugins(); err != nil {
        return nil, err
    }

    // Initialize scheduler
    scheduler := NewScheduler(engine)
    engine.scheduler = scheduler

    return engine, nil
}

func (e *Engine) Start(ctx context.Context) error {
    e.mu.Lock()
    if e.running {
        e.mu.Unlock()
        return nil
    }
    e.running = true
    e.mu.Unlock()

    logrus.Info("Starting monitoring engine")

    // Load configuration into database
    if err := e.syncConfig(); err != nil {
        logrus.WithError(err).Error("Failed to sync config")
        return err
    }

    // Start scheduler
    return e.scheduler.Start(ctx)
}

func (e *Engine) Stop() {
    e.mu.Lock()
    defer e.mu.Unlock()
    
    if !e.running {
        return
    }
    
    logrus.Info("Stopping monitoring engine")
    e.scheduler.Stop()
    e.running = false
}

func (e *Engine) RefreshConfig() error {
    logrus.Info("Refreshing configuration")
    return e.syncConfig()
}

func (e *Engine) syncConfig() error {
    // Sync hosts
    for _, hostCfg := range e.config.Hosts {
        host := &database.Host{
            ID:          hostCfg.ID,
            Name:        hostCfg.Name,
            DisplayName: hostCfg.DisplayName,
            IPv4:        hostCfg.IPv4,
            Hostname:    hostCfg.Hostname,
            Group:       hostCfg.Group,
            Enabled:     hostCfg.Enabled,
            Tags:        hostCfg.Tags,
        }

        // Try to get existing host
        existing, err := e.store.GetHost(context.Background(), host.ID)
        if err != nil {
            // Host doesn't exist, create it
            host.CreatedAt = time.Now()
            host.UpdatedAt = time.Now()
            if err := e.store.CreateHost(context.Background(), host); err != nil {
                logrus.WithError(err).WithField("host", host.Name).Error("Failed to create host")
                continue
            }
            logrus.WithField("host", host.Name).Info("Created host")
        } else {
            // Update existing host
            existing.Name = host.Name
            existing.DisplayName = host.DisplayName
            existing.IPv4 = host.IPv4
            existing.Hostname = host.Hostname
            existing.Group = host.Group
            existing.Enabled = host.Enabled
            existing.Tags = host.Tags
            existing.UpdatedAt = time.Now()
            
            if err := e.store.UpdateHost(context.Background(), existing); err != nil {
                logrus.WithError(err).WithField("host", host.Name).Error("Failed to update host")
                continue
            }
        }
    }

    // Sync checks
    for _, checkCfg := range e.config.Checks {
        check := &database.Check{
            ID:        checkCfg.ID,
            Name:      checkCfg.Name,
            Type:      checkCfg.Type,
            Hosts:     checkCfg.Hosts,
            Interval:  checkCfg.Interval,
            Threshold: checkCfg.Threshold,
            Timeout:   checkCfg.Timeout,
            Enabled:   checkCfg.Enabled,
            Options:   checkCfg.Options,
        }

        // Try to get existing check
        existing, err := e.store.GetCheck(context.Background(), check.ID)
        if err != nil {
            // Check doesn't exist, create it
            check.CreatedAt = time.Now()
            check.UpdatedAt = time.Now()
            if err := e.store.CreateCheck(context.Background(), check); err != nil {
                logrus.WithError(err).WithField("check", check.Name).Error("Failed to create check")
                continue
            }
            logrus.WithField("check", check.Name).Info("Created check")
        } else {
            // Update existing check
            existing.Name = check.Name
            existing.Type = check.Type
            existing.Hosts = check.Hosts
            existing.Interval = check.Interval
            existing.Threshold = check.Threshold
            existing.Timeout = check.Timeout
            existing.Enabled = check.Enabled
            existing.Options = check.Options
            existing.UpdatedAt = time.Now()
            
            if err := e.store.UpdateCheck(context.Background(), existing); err != nil {
                logrus.WithError(err).WithField("check", check.Name).Error("Failed to update check")
                continue
            }
        }
    }

    return nil
}

func (e *Engine) loadPlugins() error {
    // Register built-in plugins
    e.plugins["ping"] = &PingPlugin{}
    e.plugins["nagios"] = &NagiosPlugin{}
    
    logrus.WithField("plugins", len(e.plugins)).Info("Loaded plugins")
    return nil
}
