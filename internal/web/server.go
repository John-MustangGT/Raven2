// internal/web/server.go
package web

import (
    "context"
    "net/http"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/prometheus/client_golang/prometheus/promhttp"
    "github.com/sirupsen/logrus"
    "raven2/internal/config"
    "raven2/internal/database"
    "raven2/internal/metrics"
    "raven2/internal/monitoring"
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
    //s.router.Static("/static", "./web/static")

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
