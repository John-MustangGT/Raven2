// internal/monitoring/scheduler.go - Enhanced with notification integration
package monitoring

import (
    "context"
    "math/rand"
    "sync"
    "time"
    "fmt"
    "strings"

    "github.com/sirupsen/logrus"
    "raven2/internal/database"
)

type Scheduler struct {
    engine       *Engine
    jobQueue     chan *Job
    resultQueue  chan *JobResult
    workers      []*Worker
    running      bool
    mu           sync.RWMutex
    stateTracker *StateTracker // Track state changes for soft fails
}

type Job struct {
    ID       string
    HostID   string
    CheckID  string
    Host     *database.Host
    Check    *database.Check
    NextRun  time.Time
    Retries  int
    State    int // Current reported state (0=OK, 1=Warning, 2=Critical, 3=Unknown)
    StateAge int // How many consecutive checks have returned this state
}

type JobResult struct {
    Job    *Job
    Result *CheckResult
    Error  error
}

type Worker struct {
    id      int
    engine  *Engine
    jobs    chan *Job
    results chan *JobResult
    quit    chan bool
}

// StateTracker manages soft fail logic for host/check combinations
type StateTracker struct {
    states map[string]*StateInfo
    mu     sync.RWMutex
}

type StateInfo struct {
    CurrentState     int       // The state we're reporting (what's stored in DB)
    PendingState     int       // The state we're seeing in checks
    ConsecutiveCount int       // How many consecutive times we've seen the pending state
    LastStateChange  time.Time // When we last changed the current state
    LastCheckTime    time.Time // When we last ran this check
    SoftFailEnabled  bool      // Whether soft fail is enabled for this check
    Threshold        int       // How many consecutive failures needed to change state
}

func NewScheduler(engine *Engine) *Scheduler {
    return &Scheduler{
        engine:       engine,
        jobQueue:     make(chan *Job, 1000),
        resultQueue:  make(chan *JobResult, 1000),
        stateTracker: NewStateTracker(),
    }
}

func NewStateTracker() *StateTracker {
    return &StateTracker{
        states: make(map[string]*StateInfo),
    }
}

func (s *Scheduler) Start(ctx context.Context) error {
    s.mu.Lock()
    defer s.mu.Unlock()

    if s.running {
        return nil
    }

    s.running = true
    logrus.Info("Starting scheduler with soft fail support and notifications")

    // Initialize state tracker from existing database states
    if err := s.initializeStateTracker(); err != nil {
        logrus.WithError(err).Warn("Failed to initialize state tracker from database")
    }

    // Start workers
    workerCount := s.engine.config.Server.Workers
    s.workers = make([]*Worker, workerCount)
    
    for i := 0; i < workerCount; i++ {
        worker := &Worker{
            id:      i,
            engine:  s.engine,
            jobs:    s.jobQueue,
            results: s.resultQueue,
            quit:    make(chan bool),
        }
        s.workers[i] = worker
        go worker.start()
        logrus.WithField("worker", i).Info("Started worker")
    }

    // Start result processor
    go s.processResults()

    // Start job scheduler
    go s.scheduleJobs(ctx)

    return nil
}

func (s *Scheduler) Stop() {
    s.mu.Lock()
    defer s.mu.Unlock()

    if !s.running {
        return
    }

    logrus.Info("Stopping scheduler")
    s.running = false

    // Stop workers
    for _, worker := range s.workers {
        worker.stop()
    }
}

func (s *Scheduler) initializeStateTracker() error {
    checks, err := s.engine.store.GetChecks(context.Background())
    if err != nil {
        return fmt.Errorf("failed to get checks: %w", err)
    }

    for _, check := range checks {
        for _, hostID := range check.Hosts {
            key := fmt.Sprintf("%s:%s", hostID, check.ID)
            
            // Get current status from database
            statuses, _ := s.engine.store.GetStatus(context.Background(), database.StatusFilters{
                HostID:  hostID,
                CheckID: check.ID,
                Limit:   1,
            })

            threshold := s.getThreshold(&check)
            
            stateInfo := &StateInfo{
                CurrentState:     3, // Unknown by default
                PendingState:     3,
                ConsecutiveCount: 0,
                LastStateChange:  time.Now(),
                LastCheckTime:    time.Now(),
                SoftFailEnabled:  s.isSoftFailEnabled(&check),
                Threshold:        threshold,
            }

            if len(statuses) > 0 {
                stateInfo.CurrentState = statuses[0].ExitCode
                stateInfo.PendingState = statuses[0].ExitCode
                stateInfo.LastCheckTime = statuses[0].Timestamp
            }

            s.stateTracker.states[key] = stateInfo
        }
    }

    logrus.WithField("tracked_states", len(s.stateTracker.states)).Info("Initialized state tracker")
    return nil
}

func (s *Scheduler) getThreshold(check *database.Check) int {
    // Check if threshold is specified in check configuration
    if check.Threshold > 0 {
        return check.Threshold
    }
    
    // Fall back to default from monitoring config
    return s.engine.config.Monitoring.DefaultThreshold
}

func (s *Scheduler) isSoftFailEnabled(check *database.Check) bool {
    threshold := s.getThreshold(check)
    return s.engine.config.Monitoring.SoftFailEnabled && threshold > 1
}

func (s *Scheduler) scheduleJobs(ctx context.Context) {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            s.processSchedule()
        }
    }
}

func (s *Scheduler) processSchedule() {
    checks, err := s.engine.store.GetChecks(context.Background())
    if err != nil {
        logrus.WithError(err).Error("Failed to get checks")
        return
    }

    now := time.Now()
    scheduled := 0

    for _, check := range checks {
        if !check.Enabled {
            continue
        }

        for _, hostID := range check.Hosts {
            host, err := s.engine.store.GetHost(context.Background(), hostID)
            if err != nil || !host.Enabled {
                continue
            }

            key := fmt.Sprintf("%s:%s", hostID, check.ID)
            
            s.stateTracker.mu.RLock()
            stateInfo, exists := s.stateTracker.states[key]
            s.stateTracker.mu.RUnlock()
            
            if !exists {
                // Initialize state info for this host/check combination
                threshold := s.getThreshold(&check)
                stateInfo = &StateInfo{
                    CurrentState:     3, // Unknown
                    PendingState:     3,
                    ConsecutiveCount: 0,
                    LastStateChange:  now,
                    LastCheckTime:    now,
                    SoftFailEnabled:  s.isSoftFailEnabled(&check),
                    Threshold:        threshold,
                }
                
                s.stateTracker.mu.Lock()
                s.stateTracker.states[key] = stateInfo
                s.stateTracker.mu.Unlock()
            }

            // Determine interval based on current reported state (not pending state)
            var interval time.Duration
            switch stateInfo.CurrentState {
            case 0:
                interval = check.Interval["ok"]
            case 1:
                interval = check.Interval["warning"]
            case 2:
                interval = check.Interval["critical"]
            default:
                interval = check.Interval["unknown"]
            }

            if interval == 0 {
                interval = s.engine.config.Monitoring.DefaultInterval
            }

            // If we're in a pending state change, check more frequently
            if stateInfo.SoftFailEnabled && stateInfo.PendingState != stateInfo.CurrentState {
                // Use a shorter interval for pending state verification
                interval = interval / 3
                if interval < 30*time.Second {
                    interval = 30 * time.Second
                }
            }

            nextRun := stateInfo.LastCheckTime.Add(interval)
            
            // Add some jitter to prevent thundering herd
            jitter := time.Duration(rand.Intn(int(interval.Seconds()*0.1))) * time.Second
            nextRun = nextRun.Add(jitter)

            if nextRun.Before(now) {
                job := &Job{
                    ID:      key,
                    HostID:  hostID,
                    CheckID: check.ID,
                    Host:    host,
                    Check:   &check,
                    NextRun: now,
                    State:   stateInfo.CurrentState,
                }

                select {
                case s.jobQueue <- job:
                    scheduled++
                default:
                    logrus.Warn("Job queue full, dropping job")
                }
            }
        }
    }

    if scheduled > 0 {
        logrus.WithField("count", scheduled).Debug("Scheduled jobs")
    }
}

func (s *Scheduler) processResults() {
    for result := range s.resultQueue {
        s.handleResult(result)
    }
}

func (s *Scheduler) handleResult(result *JobResult) {
    ctx := context.Background()
    key := fmt.Sprintf("%s:%s", result.Job.HostID, result.Job.CheckID)
    
    if result.Error != nil {
        logrus.WithError(result.Error).
            WithFields(logrus.Fields{
                "host":  result.Job.Host.Name,
                "check": result.Job.Check.Name,
            }).Error("Check execution failed")
        
        // Create failure status
        result.Result = &CheckResult{
            ExitCode:   3,
            Output:     "Check execution failed: " + result.Error.Error(),
            PerfData:   "",
            LongOutput: result.Error.Error(),
            Duration:   0,
        }
    }

    // Update state tracker with new result
    reportedState := s.updateStateTracker(key, result.Result.ExitCode)
    
    // Get state info for logging
    s.stateTracker.mu.RLock()
    stateInfo := s.stateTracker.states[key]
    s.stateTracker.mu.RUnlock()

    // Store result with the reported state (may be different from actual result due to soft fail)
    status := &database.Status{
        HostID:     result.Job.HostID,
        CheckID:    result.Job.CheckID,
        ExitCode:   reportedState,
        Output:     result.Result.Output,
        PerfData:   result.Result.PerfData,
        LongOutput: result.Result.LongOutput,
        Duration:   result.Result.Duration.Seconds() * 1000, // Convert to milliseconds
        Timestamp:  time.Now(),
    }

    // If we're in soft fail mode and states don't match, add soft fail info to output
    if stateInfo.SoftFailEnabled && result.Result.ExitCode != reportedState {
        status.Output = fmt.Sprintf("SOFT FAIL (%d/%d) - %s", 
            stateInfo.ConsecutiveCount, stateInfo.Threshold, result.Result.Output)
        
        status.LongOutput = fmt.Sprintf("Soft fail protection active. Consecutive non-OK results: %d/%d required.\nOriginal output: %s\nOriginal long output: %s",
            stateInfo.ConsecutiveCount, stateInfo.Threshold, result.Result.Output, result.Result.LongOutput)
    }

    if err := s.engine.store.UpdateStatus(ctx, status); err != nil {
        logrus.WithError(err).Error("Failed to store status")
        return
    }

    // IMPORTANT: Process status change for notifications
    // This must be called AFTER storing the status but before metrics
    if err := s.engine.ProcessStatusChange(ctx, result); err != nil {
        logrus.WithError(err).WithFields(logrus.Fields{
            "host":  result.Job.Host.Name,
            "check": result.Job.Check.Name,
        }).Error("Failed to process status change for notifications")
    }

    // Record metrics using the reported state
    s.engine.metrics.RecordCheckResult(
        result.Job.Host.Name,
        result.Job.Check.Type,
        reportedState,
        result.Result.Duration,
    )

    s.engine.metrics.UpdateHostStatus(
        result.Job.Host.Name,
        result.Job.Host.Group,
        result.Job.Check.Type,
        reportedState,
    )

    logFields := logrus.Fields{
        "host":     result.Job.Host.Name,
        "check":    result.Job.Check.Name,
        "exit":     result.Result.ExitCode,
        "reported": reportedState,
        "duration": result.Result.Duration,
    }

    if stateInfo.SoftFailEnabled && result.Result.ExitCode != reportedState {
        logFields["soft_fail"] = true
        logFields["consecutive"] = stateInfo.ConsecutiveCount
        logFields["threshold"] = stateInfo.Threshold
    }

    logrus.WithFields(logFields).Debug("Check completed")
}

func (s *Scheduler) updateStateTracker(key string, newExitCode int) int {
    s.stateTracker.mu.Lock()
    defer s.stateTracker.mu.Unlock()
    
    stateInfo, exists := s.stateTracker.states[key]
    if !exists {
        // This shouldn't happen, but handle it gracefully
        stateInfo = &StateInfo{
            CurrentState:     newExitCode,
            PendingState:     newExitCode,
            ConsecutiveCount: 1,
            LastStateChange:  time.Now(),
            LastCheckTime:    time.Now(),
            SoftFailEnabled:  false,
            Threshold:        1,
        }
        s.stateTracker.states[key] = stateInfo
        return newExitCode
    }

    stateInfo.LastCheckTime = time.Now()
    
    // If soft fail is not enabled, just update and return the new state
    if !stateInfo.SoftFailEnabled {
        if stateInfo.CurrentState != newExitCode {
            stateInfo.LastStateChange = time.Now()
        }
        stateInfo.CurrentState = newExitCode
        stateInfo.PendingState = newExitCode
        stateInfo.ConsecutiveCount = 1
        return newExitCode
    }

    // Soft fail logic
    if newExitCode == stateInfo.PendingState {
        // Same state as before, increment counter
        stateInfo.ConsecutiveCount++
    } else {
        // Different state, reset counter
        stateInfo.PendingState = newExitCode
        stateInfo.ConsecutiveCount = 1
    }

    // Check if we should change the reported state
    shouldChangeState := false
    
    if newExitCode == 0 {
        // Recovery to OK state - immediate transition
        shouldChangeState = true
    } else if stateInfo.CurrentState == 0 && newExitCode != 0 {
        // Transitioning from OK to non-OK - apply soft fail logic
        shouldChangeState = stateInfo.ConsecutiveCount >= stateInfo.Threshold
    } else if stateInfo.CurrentState != 0 && newExitCode != 0 {
        // Already in non-OK state, transitioning to different non-OK state
        // Apply soft fail logic for state changes between non-OK states
        shouldChangeState = stateInfo.ConsecutiveCount >= stateInfo.Threshold
    } else {
        // Other transitions (shouldn't happen with the above logic, but safety)
        shouldChangeState = stateInfo.ConsecutiveCount >= stateInfo.Threshold
    }

    if shouldChangeState {
        if stateInfo.CurrentState != newExitCode {
            stateInfo.LastStateChange = time.Now()
            logrus.WithFields(logrus.Fields{
                "key":              key,
                "old_state":        stateInfo.CurrentState,
                "new_state":        newExitCode,
                "consecutive_count": stateInfo.ConsecutiveCount,
                "threshold":        stateInfo.Threshold,
            }).Info("State change confirmed after soft fail period")
        }
        stateInfo.CurrentState = newExitCode
        stateInfo.ConsecutiveCount = 1 // Reset counter after state change
    }

    return stateInfo.CurrentState
}

func (w *Worker) start() {
    for {
        select {
        case job := <-w.jobs:
            w.executeJob(job)
        case <-w.quit:
            return
        }
    }
}

func (w *Worker) stop() {
    w.quit <- true
}

func (w *Worker) executeJob(job *Job) {
    start := time.Now()
    
    plugin, exists := w.engine.plugins[job.Check.Type]
    if !exists {
        w.results <- &JobResult{
            Job:    job,
            Result: nil,
            Error:  fmt.Errorf("unknown check type: %s", job.Check.Type),
        }
        return
    }

    ctx, cancel := context.WithTimeout(context.Background(), job.Check.Timeout)
    defer cancel()

    result, err := plugin.Execute(ctx, job.Host)
    if result != nil {
        result.Duration = time.Since(start)
    }

    w.results <- &JobResult{
        Job:    job,
        Result: result,
        Error:  err,
    }
}
