// internal/monitoring/scheduler.go
package monitoring

import (
    "context"
    "math/rand"
    "sync"
    "time"
    "fmt"

    "github.com/sirupsen/logrus"
    "raven2/internal/database"
)

type Scheduler struct {
    engine      *Engine
    jobQueue    chan *Job
    resultQueue chan *JobResult
    workers     []*Worker
    running     bool
    mu          sync.RWMutex
}

type Job struct {
    ID       string
    HostID   string
    CheckID  string
    Host     *database.Host
    Check    *database.Check
    NextRun  time.Time
    Retries  int
    State    int // Current state (0=OK, 1=Warning, 2=Critical, 3=Unknown)
    StateAge int // How many checks have returned this state
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

func NewScheduler(engine *Engine) *Scheduler {
    return &Scheduler{
        engine:      engine,
        jobQueue:    make(chan *Job, 1000),
        resultQueue: make(chan *JobResult, 1000),
    }
}

func (s *Scheduler) Start(ctx context.Context) error {
    s.mu.Lock()
    defer s.mu.Unlock()

    if s.running {
        return nil
    }

    s.running = true
    logrus.Info("Starting scheduler")

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

            // Get current status to determine next run interval
            statuses, err := s.engine.store.GetStatus(context.Background(), database.StatusFilters{
                HostID:  host.ID,
                CheckID: check.ID,
                Limit:   1,
            })

            var currentState int = 3 // Unknown by default
            var lastRun time.Time
            
            if len(statuses) > 0 {
                currentState = statuses[0].ExitCode
                lastRun = statuses[0].Timestamp
            }

            // Determine interval based on current state
            var interval time.Duration
            switch currentState {
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

            nextRun := lastRun.Add(interval)
            
            // Add some jitter to prevent thundering herd
            jitter := time.Duration(rand.Intn(int(interval.Seconds()*0.1))) * time.Second
            nextRun = nextRun.Add(jitter)

            if nextRun.Before(now) {
                job := &Job{
                    ID:      host.ID + ":" + check.ID,
                    HostID:  host.ID,
                    CheckID: check.ID,
                    Host:    host,
                    Check:   &check,
                    NextRun: now,
                    State:   currentState,
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

    // Store result
    status := &database.Status{
        HostID:     result.Job.HostID,
        CheckID:    result.Job.CheckID,
        ExitCode:   result.Result.ExitCode,
        Output:     result.Result.Output,
        PerfData:   result.Result.PerfData,
        LongOutput: result.Result.LongOutput,
        Duration:   result.Result.Duration.Seconds() * 1000, // Convert to milliseconds
        Timestamp:  time.Now(),
    }

    if err := s.engine.store.UpdateStatus(ctx, status); err != nil {
        logrus.WithError(err).Error("Failed to store status")
        return
    }

    // Record metrics
    s.engine.metrics.RecordCheckResult(
        result.Job.Host.Name,
        result.Job.Check.Type,
        result.Result.ExitCode,
        result.Result.Duration,
    )

    s.engine.metrics.UpdateHostStatus(
        result.Job.Host.Name,
        result.Job.Host.Group,
        result.Job.Check.Type,
        result.Result.ExitCode,
    )

    logrus.WithFields(logrus.Fields{
        "host":     result.Job.Host.Name,
        "check":    result.Job.Check.Name,
        "exit":     result.Result.ExitCode,
        "duration": result.Result.Duration,
    }).Debug("Check completed")
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
