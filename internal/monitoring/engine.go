// internal/monitoring/engine.go - Enhanced with Pushover notifications
package monitoring

import (
    "context"
    "sync"
    "time"

    "github.com/sirupsen/logrus"
    "raven2/internal/config"
    "raven2/internal/database"
    "raven2/internal/metrics"
    "raven2/internal/notifications"
)

type Engine struct {
    config         *config.Config
    store          database.Store
    metrics        *metrics.Collector
    alertManager   *SimpleAlertManager
    notifications  *notifications.NotificationService
    scheduler      *Scheduler
    plugins        map[string]Plugin
    stateTracker   *StateTracker // Track previous states for notifications
    mu             sync.RWMutex
    running        bool
}

// StateTracker tracks previous states to detect changes for notifications
type StateTracker struct {
    previousStates map[string]int // host:check -> exit_code
    mu             sync.RWMutex
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
        config:       cfg,
        store:        store,
        metrics:      metricsCollector,
        plugins:      make(map[string]Plugin),
        alertManager: NewSimpleAlertManager(store, cfg),
        stateTracker: NewStateTracker(),
    }

    // Initialize notification service
    if cfg.Notifications.Enabled {
        notificationService, err := notifications.NewNotificationService(&cfg.Notifications)
        if err != nil {
            logrus.WithError(err).Error("Failed to initialize notification service")
            // Don't fail the entire engine, just log the error
        } else {
            engine.notifications = notificationService
            logrus.Info("Notification service initialized successfully")
        }
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

func NewStateTracker() *StateTracker {
    return &StateTracker{
        previousStates: make(map[string]int),
    }
}

func (s *StateTracker) GetPreviousState(hostID, checkID string) int {
    s.mu.RLock()
    defer s.mu.RUnlock()
    
    key := fmt.Sprintf("%s:%s", hostID, checkID)
    if state, exists := s.previousStates[key]; exists {
        return state
    }
    return 3 // Unknown if no previous state
}

func (s *StateTracker) UpdateState(hostID, checkID string, exitCode int) {
    s.mu.Lock()
    defer s.mu.Unlock()
    
    key := fmt.Sprintf("%s:%s", hostID, checkID)
    s.previousStates[key] = exitCode
}

func (e *Engine) Start(ctx context.Context) error {
    e.mu.Lock()
    if e.running {
        e.mu.Unlock()
        return nil
    }
    e.running = true
    e.mu.Unlock()

    logrus.Info("Starting monitoring engine with notifications")

    // Load configuration into database
    if err := e.syncConfig(); err != nil {
        logrus.WithError(err).Error("Failed to sync config")
        return err
    }

    // Initialize state tracker from existing database
    if err := e.initializeStateTracker(); err != nil {
        logrus.WithError(err).Warn("Failed to initialize state tracker")
    }

    purgeInterval := 6 * time.Hour
    if e.config.Database.CleanupInterval > 0 {
        purgeInterval = e.config.Database.CleanupInterval
    }
    e.alertManager.SchedulePeriodicPurge(ctx, purgeInterval)

    // Start scheduler
    return e.scheduler.Start(ctx)
}

func (e *Engine) initializeStateTracker() error {
    // Load recent statuses to initialize state tracker
    statuses, err := e.store.GetStatus(context.Background(), database.StatusFilters{
        Limit: 1000, // Get recent statuses
    })
    if err != nil {
        return err
    }

    // Build a map of latest statuses by host:check
    latestStatuses := make(map[string]*database.Status)
    for i := range statuses {
        status := &statuses[i]
        key := fmt.Sprintf("%s:%s", status.HostID, status.CheckID)
        
        // Keep the most recent status for each host:check combination
        if existing, exists := latestStatuses[key]; !exists || status.Timestamp.After(existing.Timestamp) {
            latestStatuses[key] = status
        }
    }

    // Initialize state tracker with these statuses
    for key, status := range latestStatuses {
        parts := strings.Split(key, ":")
        if len(parts) == 2 {
            e.stateTracker.UpdateState(parts[0], parts[1], status.ExitCode)
        }
    }

    logrus.WithField("initialized_states", len(latestStatuses)).Info("State tracker initialized")
    return nil
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

func (e *Engine) GetAlertManager() *SimpleAlertManager {
    return e.alertManager
}

func (e *Engine) RefreshConfigWithPurge() error {
    logrus.Info("Refreshing configuration with alert purging")
    
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    if err := e.RefreshConfig(); err != nil {
        return err
    }

    e.alertManager.config = e.config

    if err := e.alertManager.PurgeAll(ctx); err != nil {
        logrus.WithError(err).Warn("Alert purge completed with errors")
    }

    return nil
}

// ProcessStatusChange handles status changes and triggers notifications
func (e *Engine) ProcessStatusChange(ctx context.Context, result *JobResult) error {
    if result == nil || result.Job == nil || result.Result == nil {
        return fmt.Errorf("invalid job result")
    }

    hostID := result.Job.HostID
    checkID := result.Job.CheckID
    currentExit := result.Result.ExitCode
    
    // Get previous state
    previousExit := e.stateTracker.GetPreviousState(hostID, checkID)
    
    // Determine if this is a recovery (going from non-OK to OK)
    isRecovery := previousExit != 0 && currentExit == 0
    
    // Determine if this is a state change that warrants notification
    shouldNotify := e.shouldNotifyForStateChange(previousExit, currentExit, isRecovery)
    
    if shouldNotify && e.notifications != nil {
        // Create notification event
        event := &notifications.NotificationEvent{
            Host:         result.Job.Host,
            Check:        result.Job.Check,
            Status: &database.Status{
                HostID:     hostID,
                CheckID:    checkID,
                ExitCode:   currentExit,
                Output:     result.Result.Output,
                PerfData:   result.Result.PerfData,
                LongOutput: result.Result.LongOutput,
                Duration:   result.Result.Duration.Seconds() * 1000,
                Timestamp:  time.Now(),
            },
            PreviousExit: previousExit,
            Timestamp:    time.Now(),
            IsRecovery:   isRecovery,
        }

        // Send notification asynchronously to avoid blocking the monitoring loop
        go func() {
            if err := e.notifications.SendNotification(ctx, event); err != nil {
                logrus.WithError(err).WithFields(logrus.Fields{
                    "host":  result.Job.Host.Name,
                    "check": result.Job.Check.Name,
                    "status": getStatusName(currentExit),
                }).Error("Failed to send notification")
            }
        }()
        
        logrus.WithFields(logrus.Fields{
            "host":         result.Job.Host.Name,
            "check":        result.Job.Check.Name,
            "previous":     getStatusName(previousExit),
            "current":      getStatusName(currentExit),
            "is_recovery":  isRecovery,
        }).Info("Notification triggered for status change")
    }
    
    // Update state tracker
    e.stateTracker.UpdateState(hostID, checkID, currentExit)
    
    return nil
}

// shouldNotifyForStateChange determines if a notification should be sent
func (e *Engine) shouldNotifyForStateChange(previousExit, currentExit int, isRecovery bool) bool {
    // Always notify on recovery
    if isRecovery {
        return true
    }
    
    // Notify when going from OK to non-OK
    if previousExit == 0 && currentExit != 0 {
        return true
    }
    
    // Notify when changing between non-OK states (e.g., warning to critical)
    if previousExit != 0 && currentExit != 0 && previousExit != currentExit {
        return true
    }
    
    // Don't notify for repeated states or OK to OK transitions
    return false
}

// TestNotifications sends a test notification
func (e *Engine) TestNotifications(ctx context.Context) error {
    if e.notifications == nil {
        return fmt.Errorf("notifications are not configured")
    }
    
    message := "Test notification from Raven monitoring system. If you receive this, notifications are working correctly!"
    
    return e.notifications.TestNotification(ctx, message)
}

// GetNotificationStats returns notification statistics
func (e *Engine) GetNotificationStats() map[string]interface{} {
    if e.notifications == nil {
        return map[string]interface{}{
            "enabled": false,
        }
    }
    
    return e.notifications.GetStats()
}

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
