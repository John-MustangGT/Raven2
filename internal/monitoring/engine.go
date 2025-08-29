// internal/monitoring/engine.go
package monitoring

import (
    "context"
    "sync"
    "time"

    "github.com/sirupsen/logrus"
    "github.com/your-org/raven/internal/config"
    "github.com/your-org/raven/internal/database"
    "github.com/your-org/raven/internal/metrics"
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

// internal/monitoring/scheduler.go
package monitoring

import (
    "context"
    "math/rand"
    "sync"
    "time"

    "github.com/sirupsen/logrus"
    "github.com/your-org/raven/internal/database"
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
                    Check:   check,
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

// internal/monitoring/plugins.go
package monitoring

import (
    "context"
    "fmt"
    "os/exec"
    "regexp"
    "strconv"
    "strings"
    "time"

    "github.com/your-org/raven/internal/database"
)

// PingPlugin implements basic ping checks
type PingPlugin struct{}

func (p *PingPlugin) Name() string {
    return "ping"
}

func (p *PingPlugin) Init(options map[string]interface{}) error {
    return nil
}

func (p *PingPlugin) Execute(ctx context.Context, host *database.Host) (*CheckResult, error) {
    target := host.IPv4
    if target == "" {
        target = host.Hostname
    }
    if target == "" {
        return &CheckResult{
            ExitCode:   3,
            Output:     "No IP address or hostname configured",
            PerfData:   "",
            LongOutput: "",
        }, nil
    }

    cmd := exec.CommandContext(ctx, "ping", "-c", "3", target)
    output, err := cmd.Output()

    if err != nil {
        return &CheckResult{
            ExitCode:   2,
            Output:     "Ping failed",
            PerfData:   "",
            LongOutput: string(output),
        }, nil
    }

    // Parse ping output
    outputStr := string(output)
    
    // Extract packet loss
    lossRegex := regexp.MustCompile(`(\d+)% packet loss`)
    lossMatches := lossRegex.FindStringSubmatch(outputStr)
    
    // Extract average RTT
    rttRegex := regexp.MustCompile(`avg = ([\d.]+)`)
    rttMatches := rttRegex.FindStringSubmatch(outputStr)

    var loss int
    var rtt float64

    if len(lossMatches) > 1 {
        loss, _ = strconv.Atoi(lossMatches[1])
    }
    
    if len(rttMatches) > 1 {
        rtt, _ = strconv.ParseFloat(rttMatches[1], 64)
    }

    // Determine status based on thresholds
    exitCode := 0
    status := "OK"
    
    if loss > 25 || rtt > 100 {
        exitCode = 2
        status = "CRITICAL"
    } else if loss > 10 || rtt > 50 {
        exitCode = 1
        status = "WARNING"
    }

    return &CheckResult{
        ExitCode:   exitCode,
        Output:     fmt.Sprintf("PING %s - %s", status, target),
        PerfData:   fmt.Sprintf("rtt=%.2fms;50;100;0 loss=%d%%;10;25;0", rtt, loss),
        LongOutput: fmt.Sprintf("RTT: %.2fms, Loss: %d%%", rtt, loss),
    }, nil
}

// NagiosPlugin executes Nagios-compatible check plugins
type NagiosPlugin struct{}

func (p *NagiosPlugin) Name() string {
    return "nagios"
}

func (p *NagiosPlugin) Init(options map[string]interface{}) error {
    return nil
}

func (p *NagiosPlugin) Execute(ctx context.Context, host *database.Host) (*CheckResult, error) {
    // This would be implemented based on your existing nagios plugin logic
    // For now, return a placeholder
    return &CheckResult{
        ExitCode:   0,
        Output:     "Nagios check OK",
        PerfData:   "",
        LongOutput: "Nagios plugin executed successfully",
    }, nil
}

// internal/metrics/prometheus.go
package metrics

import (
    "context"
    "time"

    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
    "github.com/your-org/raven/internal/database"
)

// Prometheus metrics
var (
    CheckDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "raven_check_duration_seconds",
            Help:    "Time spent executing checks",
            Buckets: prometheus.DefBuckets,
        },
        []string{"host", "check_type", "status"},
    )

    CheckTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "raven_checks_total",
            Help: "Total number of checks executed",
        },
        []string{"host", "check_type", "status"},
    )

    HostStatus = promauto.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "raven_host_status",
            Help: "Current status of hosts (0=OK, 1=Warning, 2=Critical, 3=Unknown)",
        },
        []string{"host", "group", "check_type"},
    )

    ActiveHosts = promauto.NewGauge(
        prometheus.GaugeOpts{
            Name: "raven_active_hosts_total",
            Help: "Number of active hosts being monitored",
        },
    )

    ActiveChecks = promauto.NewGauge(
        prometheus.GaugeOpts{
            Name: "raven_active_checks_total",
            Help: "Number of active checks configured",
        },
    )

    DatabaseOperations = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "raven_database_operations_total",
            Help: "Total database operations performed",
        },
        []string{"operation", "status"},
    )

    WebSocketConnections = promauto.NewGauge(
        prometheus.GaugeOpts{
            Name: "raven_websocket_connections_active",
            Help: "Number of active WebSocket connections",
        },
    )
)

type Collector struct {
    store database.Store
}

func NewCollector(store database.Store) *Collector {
    return &Collector{store: store}
}

func (c *Collector) RecordCheckResult(host, checkType string, exitCode int, duration time.Duration) {
    status := getStatusLabel(exitCode)
    CheckDuration.WithLabelValues(host, checkType, status).Observe(duration.Seconds())
    CheckTotal.WithLabelValues(host, checkType, status).Inc()
}

func (c *Collector) UpdateHostStatus(host, group, checkType string, exitCode int) {
    HostStatus.WithLabelValues(host, group, checkType).Set(float64(exitCode))
}

func (c *Collector) UpdateSystemMetrics(ctx context.Context) error {
    hosts, err := c.store.GetHosts(ctx, database.HostFilters{})
    if err != nil {
        DatabaseOperations.WithLabelValues("get_hosts", "error").Inc()
        return err
    }
    DatabaseOperations.WithLabelValues("get_hosts", "success").Inc()

    enabledHosts := 0
    for _, host := range hosts {
        if host.Enabled {
            enabledHosts++
        }
    }
    ActiveHosts.Set(float64(enabledHosts))

    checks, err := c.store.GetChecks(ctx)
    if err != nil {
        DatabaseOperations.WithLabelValues("get_checks", "error").Inc()
        return err
    }
    DatabaseOperations.WithLabelValues("get_checks", "success").Inc()

    enabledChecks := 0
    for _, check := range checks {
        if check.Enabled {
            enabledChecks++
        }
    }
    ActiveChecks.Set(float64(enabledChecks))

    return nil
}

func (c *Collector) RecordWebSocketConnection(delta int) {
    WebSocketConnections.Add(float64(delta))
}

func getStatusLabel(exitCode int) string {
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

// internal/web/server.go
package web

import (
    "context"
    "net/http"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/prometheus/client_golang/prometheus/promhttp"
    "github.com/sirupsen/logrus"
    "github.com/your-org/raven/internal/config"
    "github.com/your-org/raven/internal/database"
    "github.com/your-org/raven/internal/metrics"
    "github.com/your-org/raven/internal/monitoring"
)

type Server struct {
    config    *config.Config
    store     database.Store
    engine    *monitoring.Engine
    metrics   *metrics.Collector
    router    *gin.Engine
    wsClients map[*WSClient]bool
    server    *http.Server
}

func NewServer(cfg *config.Config, store database.Store, engine *monitoring.Engine, metricsCollector *metrics.Collector) *Server {
    if cfg.Logging.Level != "debug" {
        gin.SetMode(gin.ReleaseMode)
    }

    router := gin.New()
    router.Use(gin.Logger())
    router.Use(gin.Recovery())
    router.Use(corsMiddleware())

    server := &Server{
        config:    cfg,
        store:     store,
        engine:    engine,
        metrics:   metricsCollector,
        router:    router,
        wsClients: make(map[*WSClient]bool),
    }

    server.setupRoutes()
    return server
}

func (s *Server) Start(ctx context.Context) error {
    s.server = &http.Server{
        Addr:         s.config.Server.Port,
        Handler:      s.router,
        ReadTimeout:  s.config.Server.ReadTimeout,
        WriteTimeout: s.config.Server.WriteTimeout,
    }

    logrus.WithField("port", s.config.Server.Port).Info("Starting web server")

    // Start metrics update routine
    go s.updateMetricsRoutine(ctx)

    // Start server in goroutine
    go func() {
        if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            logrus.WithError(err).Fatal("Failed to start server")
        }
    }()

    return nil
}

func (s *Server) Stop(ctx context.Context) error {
    if s.server != nil {
        return s.server.Shutdown(ctx)
    }
    return nil
}

func (s *Server) setupRoutes() {
    // Static files
    s.router.Static("/static", "./web/static")
    s.router.LoadHTMLGlob("web/templates/*")

    // Main page
    s.router.GET("/", s.serveSPA)

    // API routes
    api := s.router.Group("/api")
    {
        api.GET("/hosts", s.getHosts)
        api.GET("/hosts/:id", s.getHost)
        api.POST("/hosts", s.createHost)
        api.PUT("/hosts/:id", s.updateHost)
        api.DELETE("/hosts/:id", s.deleteHost)

        api.GET("/checks", s.getChecks)
        api.GET("/checks/:id", s.getCheck)
        api.POST("/checks", s.createCheck)
        api.PUT("/checks/:id", s.updateCheck)
        api.DELETE("/checks/:id", s.deleteCheck)

        api.GET("/status", s.getStatus)
        api.GET("/status/history/:host/:check", s.getStatusHistory)

        api.GET("/stats", s.getStats)
        api.GET("/health", s.healthCheck)
    }

    // WebSocket endpoint
    s.router.GET("/ws", s.handleWebSocket)

    // Prometheus metrics
    if s.config.Prometheus.Enabled {
        s.router.GET(s.config.Prometheus.MetricsPath, gin.WrapH(promhttp.Handler()))
    }
}

func (s *Server) serveSPA(c *gin.Context) {
    c.Header("Content-Type", "text/html")
    c.Status(http.StatusOK)
    // Serve the modern UI we created
    c.File("web/index.html")
}

func (s *Server) healthCheck(c *gin.Context) {
    c.JSON(http.StatusOK, gin.H{
        "status":    "healthy",
        "timestamp": time.Now(),
        "version":   "2.0.0",
    })
}

func (s *Server) getStats(c *gin.Context) {
    statuses, err := s.store.GetStatus(c.Request.Context(), database.StatusFilters{
        Limit: 1000, // Get recent statuses for stats
    })
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get status"})
        return
    }

    stats := map[string]int{
        "ok":       0,
        "warning":  0,
        "critical": 0,
        "unknown":  0,
    }

    for _, status := range statuses {
        switch status.ExitCode {
        case 0:
            stats["ok"]++
        case 1:
            stats["warning"]++
        case 2:
            stats["critical"]++
        default:
            stats["unknown"]++
        }
    }

    c.JSON(http.StatusOK, gin.H{"data": stats})
}

func (s *Server) getChecks(c *gin.Context) {
    checks, err := s.store.GetChecks(c.Request.Context())
    if err != nil {
        logrus.WithError(err).Error("Failed to get checks")
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get checks"})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "data":  checks,
        "count": len(checks),
    })
}

func (s *Server) getCheck(c *gin.Context) {
    id := c.Param("id")
    
    check, err := s.store.GetCheck(c.Request.Context(), id)
    if err != nil {
        if err.Error() == "check not found" {
            c.JSON(http.StatusNotFound, gin.H{"error": "Check not found"})
            return
        }
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get check"})
        return
    }

    c.JSON(http.StatusOK, gin.H{"data": check})
}

func (s *Server) createCheck(c *gin.Context) {
    var req database.Check
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    if err := s.store.CreateCheck(c.Request.Context(), &req); err != nil {
        logrus.WithError(err).Error("Failed to create check")
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create check"})
        return
    }

    s.engine.RefreshConfig()
    c.JSON(http.StatusCreated, gin.H{"data": req})
}

func (s *Server) updateCheck(c *gin.Context) {
    // Implementation similar to updateHost
    c.JSON(http.StatusNotImplemented, gin.H{"error": "Not implemented yet"})
}

func (s *Server) deleteCheck(c *gin.Context) {
    // Implementation similar to deleteHost
    c.JSON(http.StatusNotImplemented, gin.H{"error": "Not implemented yet"})
}

func (s *Server) getStatusHistory(c *gin.Context) {
    hostID := c.Param("host")
    checkID := c.Param("check")
    
    since := time.Now().Add(-24 * time.Hour) // Last 24 hours by default
    if sinceStr := c.Query("since"); sinceStr != "" {
        if parsedSince, err := time.Parse(time.RFC3339, sinceStr); err == nil {
            since = parsedSince
        }
    }

    history, err := s.store.GetStatusHistory(c.Request.Context(), hostID, checkID, since)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get status history"})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "data":  history,
        "count": len(history),
    })
}

func (s *Server) updateMetricsRoutine(ctx context.Context) {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            if err := s.metrics.UpdateSystemMetrics(ctx); err != nil {
                logrus.WithError(err).Error("Failed to update system metrics")
            }
        }
    }
}

func corsMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Header("Access-Control-Allow-Origin", "*")
        c.Header("Access-Control-Allow-Credentials", "true")
        c.Header("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
        c.Header("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

        if c.Request.Method == "OPTIONS" {
            c.AbortWithStatus(204)
            return
        }

        c.Next()
    }
}

// internal/database/boltstore.go - Complete BoltDB implementation
package database

import (
    "context"
    "encoding/json"
    "fmt"
    "os"
    "strings"
    "time"

    "github.com/google/uuid"
    "go.etcd.io/bbolt"
)

var (
    HostsBucket      = []byte("hosts")
    ChecksBucket     = []byte("checks")
    StatusBucket     = []byte("status")
    StatusHistBucket = []byte("status_history")
    MetaBucket       = []byte("meta")
)

type BoltStore struct {
    db   *bbolt.DB
    path string
}

func NewBoltStore(path string) (Store, error) {
    // Create directory if it doesn't exist
    if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
        return nil, fmt.Errorf("failed to create data directory: %w", err)
    }

    db, err := bbolt.Open(path, 0600, &bbolt.Options{
        Timeout: 1 * time.Second,
    })
    if err != nil {
        return nil, fmt.Errorf("failed to open BoltDB: %w", err)
    }

    store := &BoltStore{db: db, path: path}

    if err := store.initBuckets(); err != nil {
        db.Close()
        return nil, fmt.Errorf("failed to initialize buckets: %w", err)
    }

    return store, nil
}

func (s *BoltStore) initBuckets() error {
    return s.db.Update(func(tx *bbolt.Tx) error {
        buckets := [][]byte{HostsBucket, ChecksBucket, StatusBucket, StatusHistBucket, MetaBucket}
        for _, bucket := range buckets {
            if _, err := tx.CreateBucketIfNotExists(bucket); err != nil {
                return fmt.Errorf("failed to create bucket %s: %w", bucket, err)
            }
        }
        return nil
    })
}

func (s *BoltStore) GetHosts(ctx context.Context, filters HostFilters) ([]Host, error) {
    var hosts []Host

    err := s.db.View(func(tx *bbolt.Tx) error {
        b := tx.Bucket(HostsBucket)
        return b.ForEach(func(k, v []byte) error {
            var host Host
            if err := json.Unmarshal(v, &host); err != nil {
                return fmt.Errorf("failed to unmarshal host %s: %w", k, err)
            }

            // Apply filters
            if filters.Group != "" && host.Group != filters.Group {
                return nil
            }
            if filters.Enabled != nil && host.Enabled != *filters.Enabled {
                return nil
            }

            hosts = append(hosts, host)
            return nil
        })
    })

    return hosts, err
}

func (s *BoltStore) GetHost(ctx context.Context, id string) (*Host, error) {
    var host Host

    err := s.db.View(func(tx *bbolt.Tx) error {
        b := tx.Bucket(HostsBucket)
        v := b.Get([]byte(id))
        if v == nil {
            return fmt.Errorf("host not found")
        }
        return json.Unmarshal(v, &host)
    })

    if err != nil {
        return nil, err
    }
    return &host, nil
}

func (s *BoltStore) CreateHost(ctx context.Context, host *Host) error {
    if host.ID == "" {
        host.ID = uuid.New().String()
    }
    host.CreatedAt = time.Now()
    host.UpdatedAt = time.Now()

    return s.db.Update(func(tx *bbolt.Tx) error {
        b := tx.Bucket(HostsBucket)
        
        data, err := json.Marshal(host)
        if err != nil {
            return fmt.Errorf("failed to marshal host: %w", err)
        }

        return b.Put([]byte(host.ID), data)
    })
}

func (s *BoltStore) UpdateHost(ctx context.Context, host *Host) error {
    host.UpdatedAt = time.Now()

    return s.db.Update(func(tx *bbolt.Tx) error {
        b := tx.Bucket(HostsBucket)
        
        data, err := json.Marshal(host)
        if err != nil {
            return fmt.Errorf("failed to marshal host: %w", err)
        }

        return b.Put([]byte(host.ID), data)
    })
}

func (s *BoltStore) DeleteHost(ctx context.Context, id string) error {
    return s.db.Update(func(tx *bbolt.Tx) error {
        b := tx.Bucket(HostsBucket)
        return b.Delete([]byte(id))
    })
}

func (s *BoltStore) GetChecks(ctx context.Context) ([]Check, error) {
    var checks []Check

    err := s.db.View(func(tx *bbolt.Tx) error {
        b := tx.Bucket(ChecksBucket)
        return b.ForEach(func(k, v []byte) error {
            var check Check
            if err := json.Unmarshal(v, &check); err != nil {
                return fmt.Errorf("failed to unmarshal check %s: %w", k, err)
            }
            checks = append(checks, check)
            return nil
        })
    })

    return checks, err
}

func (s *BoltStore) GetCheck(ctx context.Context, id string) (*Check, error) {
    var check Check

    err := s.db.View(func(tx *bbolt.Tx) error {
        b := tx.Bucket(ChecksBucket)
        v := b.Get([]byte(id))
        if v == nil {
            return fmt.Errorf("check not found")
        }
        return json.Unmarshal(v, &check)
    })

    if err != nil {
        return nil, err
    }
    return &check, nil
}

func (s *BoltStore) CreateCheck(ctx context.Context, check *Check) error {
    if check.ID == "" {
        check.ID = uuid.New().String()
    }
    check.CreatedAt = time.Now()
    check.UpdatedAt = time.Now()

    return s.db.Update(func(tx *bbolt.Tx) error {
        b := tx.Bucket(ChecksBucket)
        
        data, err := json.Marshal(check)
        if err != nil {
            return fmt.Errorf("failed to marshal check: %w", err)
        }

        return b.Put([]byte(check.ID), data)
    })
}

func (s *BoltStore) GetStatus(ctx context.Context, filters StatusFilters) ([]Status, error) {
    var statuses []Status

    err := s.db.View(func(tx *bbolt.Tx) error {
        b := tx.Bucket(StatusBucket)
        return b.ForEach(func(k, v []byte) error {
            var status Status
            if err := json.Unmarshal(v, &status); err != nil {
                return nil // Skip malformed entries
            }

            // Apply filters
            if filters.HostID != "" && status.HostID != filters.HostID {
                return nil
            }
            if filters.CheckID != "" && status.CheckID != filters.CheckID {
                return nil
            }
            if filters.ExitCode != nil && status.ExitCode != *filters.ExitCode {
                return nil
            }

            statuses = append(statuses, status)
            
            if filters.Limit > 0 && len(statuses) >= filters.Limit {
                return fmt.Errorf("limit_reached")
            }

            return nil
        })
    })

    if err != nil && err.Error() == "limit_reached" {
        err = nil
    }

    return statuses, err
}

func (s *BoltStore) UpdateStatus(ctx context.Context, status *Status) error {
    if status.ID == "" {
        status.ID = uuid.New().String()
    }

    return s.db.Update(func(tx *bbolt.Tx) error {
        b := tx.Bucket(StatusBucket)
        
        // Store current status
        key := fmt.Sprintf("%s:%s", status.HostID, status.CheckID)
        data, err := json.Marshal(status)
        if err != nil {
            return fmt.Errorf("failed to marshal status: %w", err)
        }

        if err := b.Put([]byte(key), data); err != nil {
            return err
        }

        // Also store in history
        hb := tx.Bucket(StatusHistBucket)
        histKey := fmt.Sprintf("%s:%s:%d", status.HostID, status.CheckID, status.Timestamp.Unix())
        return hb.Put([]byte(histKey), data)
    })
}

func (s *BoltStore) GetStatusHistory(ctx context.Context, hostID, checkID string, since time.Time) ([]Status, error) {
    var statuses []Status

    err := s.db.View(func(tx *bbolt.Tx) error {
        b := tx.Bucket(StatusHistBucket)
        c := b.Cursor()

        prefix := fmt.Sprintf("%s:%s:", hostID, checkID)
        
        for k, v := c.Seek([]byte(prefix)); k != nil && strings.HasPrefix(string(k), prefix); k, v = c.Next() {
            var status Status
            if err := json.Unmarshal(v, &status); err != nil {
                continue
            }

            if status.Timestamp.After(since) {
                statuses = append(statuses, status)
            }
        }

        return nil
    })

    return statuses, err
}

func (s *BoltStore) Close() error {
    return s.db.Close()
}
