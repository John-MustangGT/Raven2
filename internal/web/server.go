// internal/web/server.go
package web

import (
    "context"
    "net/http"
    "path/filepath"
    "time"
    "os"
    "strings"

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

// Update the setupRoutes function to include new routes
func (s *Server) setupRoutes() {
    // Configure static file serving based on config
    if s.config.Web.ServeStatic {
        var staticDir string
        
        // Determine static directory
        if s.config.Web.AssetsDir != "" {
            // Use configured assets directory
            if filepath.IsAbs(s.config.Web.StaticDir) {
                staticDir = s.config.Web.StaticDir
            } else {
                staticDir = filepath.Join(s.config.Web.AssetsDir, s.config.Web.StaticDir)
            }
        } else {
            // Auto-detect static directory
            possibleStaticDirs := []string{
                "./web/static",
                "/usr/lib/raven/web/static",
                "/opt/raven/web/static",
            }
            
            for _, dir := range possibleStaticDirs {
                if _, err := os.Stat(dir); err == nil {
                    staticDir = dir
                    break
                }
            }
        }
        
        // Enable static serving if directory exists
        if staticDir != "" {
            if _, err := os.Stat(staticDir); err == nil {
                s.router.Static("/static", staticDir)
                logrus.WithField("static_dir", staticDir).Debug("Enabled static file serving")
            } else {
                logrus.WithField("static_dir", staticDir).Warn("Configured static directory not found")
            }
        }
    }

    // Main page
    s.router.GET("/", s.serveSPA)
    
    // Favicon routes
    s.router.GET("/favicon.ico", s.serveFaviconICO)
    s.router.GET("/favicon.svg", s.serveFavicon)

    // API routes
    api := s.router.Group("/api")
    {
        // Host endpoints
        api.GET("/hosts", s.getHosts)
        api.GET("/hosts/:id", s.getHost)
        api.POST("/hosts", s.createHost)
        api.PUT("/hosts/:id", s.updateHost)
        api.DELETE("/hosts/:id", s.deleteHost)

        // Check endpoints
        api.GET("/checks", s.getChecks)
        api.GET("/checks/:id", s.getCheck)
        api.POST("/checks", s.createCheck)
        api.PUT("/checks/:id", s.updateCheck)
        api.DELETE("/checks/:id", s.deleteCheck)

        // Status endpoints
        api.GET("/status", s.getStatus)
        api.GET("/status/history/:host/:check", s.getStatusHistory)

        // Alert endpoints
        api.GET("/alerts", s.getAlerts)
        api.GET("/alerts/summary", s.getAlertsSummary)

        // System endpoints
        api.GET("/stats", s.getStats)
        api.GET("/health", s.healthCheck)
        api.GET("/diagnostics/web", s.webDiagnostics)
        api.GET("/build-info", s.getBuildInfo) // New build info endpoint
    }

    // WebSocket endpoint
    s.router.GET("/ws", s.handleWebSocket)

    // Prometheus metrics
    if s.config.Prometheus.Enabled {
        s.router.GET(s.config.Prometheus.MetricsPath, gin.WrapH(promhttp.Handler()))
    }
}

// ... (rest of the existing server.go methods remain the same)
// serveSPA, serveErrorPage, healthCheck, webDiagnostics, etc.

// Update the serveSPA function to use config
func (s *Server) serveSPA(c *gin.Context) {
    var indexPath string
    var err error
    
    // If assets directory is configured, try that first
    if s.config.Web.AssetsDir != "" {
        configuredPath := filepath.Join(s.config.Web.AssetsDir, "index.html")
        if _, err = os.Stat(configuredPath); err == nil {
            indexPath = configuredPath
        } else {
            logrus.WithField("configured_path", configuredPath).Warn("Configured web assets directory not found, falling back to auto-detection")
        }
    }
    
    // If no configured path worked, try default locations
    if indexPath == "" {
        possiblePaths := []string{
            "web/index.html",                    // Development path
            "./web/index.html",                  // Alternative development path
            "/usr/lib/raven/web/index.html",     // Production package path
            "/opt/raven/web/index.html",         // Alternative production path
        }
        
        // Find the first existing path
        for _, path := range possiblePaths {
            if _, err = os.Stat(path); err == nil {
                indexPath = path
                break
            }
        }
    }
    
    if indexPath == "" {
        logrus.WithError(err).Error("Web interface files not found")
        s.serveErrorPage(c)
        return
    }
    
    // Log which path we're using (debug level)
    logrus.WithField("path", indexPath).Debug("Serving web interface from")
    
    c.Header("Content-Type", "text/html")
    c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
    c.Header("Pragma", "no-cache")
    c.Header("Expires", "0")
    
    // Serve the file
    c.File(indexPath)
}

// Extract error page to separate method
func (s *Server) serveErrorPage(c *gin.Context) {
    c.Header("Content-Type", "text/html")
    c.String(http.StatusInternalServerError, `
<!DOCTYPE html>
<html>
<head>
    <title>Raven - Web Interface Not Found</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; background-color: #f5f5f5; }
        .error-container { 
            background: white; 
            padding: 30px; 
            border-radius: 8px; 
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
            max-width: 700px;
        }
        h1 { color: #e74c3c; }
        .config-info { background: #e3f2fd; padding: 15px; border-left: 4px solid #2196f3; margin: 20px 0; }
        .paths { background: #f8f9fa; padding: 15px; border-left: 4px solid #e74c3c; margin: 20px 0; }
        .solution { background: #e8f5e9; padding: 15px; border-left: 4px solid #4caf50; margin: 20px 0; }
        code { background: #f4f4f4; padding: 2px 6px; border-radius: 3px; font-family: monospace; }
        .config-example { background: #f8f9fa; padding: 10px; border-radius: 4px; margin: 10px 0; }
    </style>
</head>
<body>
    <div class="error-container">
        <h1>🐦 Raven Web Interface Not Found</h1>
        <p>The Raven web interface files could not be located.</p>
        
        <div class="config-info">
            <strong>Current Configuration:</strong><br>
            Configured assets directory: <code>%s</code><br>
            (Empty means auto-detect)
        </div>
        
        <div class="paths">
            <strong>Auto-detection searched paths:</strong><br>
            • <code>web/index.html</code> (development)<br>
            • <code>./web/index.html</code> (alternative development)<br>
            • <code>/usr/lib/raven/web/index.html</code> (Debian package)<br>
            • <code>/opt/raven/web/index.html</code> (alternative package)
        </div>
        
        <div class="solution">
            <strong>Solutions:</strong><br><br>
            
            <strong>1. Configure explicit path in config.yaml:</strong><br>
            <div class="config-example">
web:<br>
&nbsp;&nbsp;assets_dir: "/usr/lib/raven/web"&nbsp;&nbsp;# For package install<br>
&nbsp;&nbsp;assets_dir: "./web"&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;# For development<br>
&nbsp;&nbsp;serve_static: true
            </div>
            
            <strong>2. If running from source:</strong> Make sure <code>web/index.html</code> exists<br><br>
            
            <strong>3. If installed from package:</strong> Try reinstalling:<br>
            <code>sudo dpkg -i --force-confask raven_*.deb</code><br><br>
            
            <strong>4. Check installation:</strong> <code>ls -la /usr/lib/raven/web/</code>
        </div>
        
        <p><strong>API Status:</strong> ✅ The REST API is working</p>
        <p><strong>Alternative access:</strong> You can use the API directly at <code>/api/</code> endpoints</p>
        <p><strong>Diagnostics:</strong> Visit <code>/api/diagnostics/web</code> for detailed information</p>
        
        <hr>
        <p><em>Raven v2.0 Network Monitoring</em></p>
    </div>
</body>
</html>`, s.config.Web.AssetsDir)
}

// ... (include all other existing methods from the original server.go)

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

// Update the healthCheck function
func (s *Server) healthCheck(c *gin.Context) {
    health := gin.H{
        "status":    "healthy",
        "timestamp": time.Now(),
        "version":   Version, // Use build version
        "services":  gin.H{},
    }
    
    services := health["services"].(gin.H)
    
    // Check database connectivity
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    if _, err := s.store.GetHosts(ctx, database.HostFilters{}); err != nil {
        services["database"] = gin.H{
            "status": "unhealthy",
            "error":  err.Error(),
        }
        health["status"] = "degraded"
    } else {
        services["database"] = gin.H{"status": "healthy"}
    }
    
    // Check web assets
    webPaths := []string{
        "web/index.html",
        "./web/index.html", 
        "/usr/lib/raven/web/index.html",
        "/opt/raven/web/index.html",
    }
    
    webAssetFound := false
    var webPath string
    for _, path := range webPaths {
        if _, err := os.Stat(path); err == nil {
            webAssetFound = true
            webPath = path
            break
        }
    }
    
    if webAssetFound {
        services["web_interface"] = gin.H{
            "status": "healthy",
            "path":   webPath,
        }
    } else {
        services["web_interface"] = gin.H{
            "status": "unhealthy",
            "error":  "Web interface files not found",
            "searched_paths": webPaths,
        }
        health["status"] = "degraded"
    }
    
    // Check WebSocket clients
    services["websocket"] = gin.H{
        "status":           "healthy", 
        "active_clients":   len(s.wsClients),
    }
    
    // Check monitoring engine
    services["monitoring"] = gin.H{"status": "healthy"}
    
    // Set HTTP status based on overall health
    httpStatus := http.StatusOK
    if health["status"] == "degraded" {
        httpStatus = http.StatusServiceUnavailable
    }
    
    c.JSON(httpStatus, health)
}

// Update the webDiagnostics function to include config info
func (s *Server) webDiagnostics(c *gin.Context) {
    diagnostics := gin.H{
        "timestamp": time.Now(),
        "configuration": gin.H{
            "assets_dir":    s.config.Web.AssetsDir,
            "static_dir":    s.config.Web.StaticDir,
            "serve_static":  s.config.Web.ServeStatic,
        },
        "web_asset_search": gin.H{},
    }
    
    // Determine search paths based on configuration
    var searchPaths []string
    
    if s.config.Web.AssetsDir != "" {
        // If assets directory is configured, check that first
        configuredPath := filepath.Join(s.config.Web.AssetsDir, "index.html")
        searchPaths = append(searchPaths, configuredPath)
    }
    
    // Add default search paths
    defaultPaths := []string{
        "web/index.html",
        "./web/index.html",
        "/usr/lib/raven/web/index.html", 
        "/opt/raven/web/index.html",
    }
    searchPaths = append(searchPaths, defaultPaths...)
    
    pathResults := make([]gin.H, 0, len(searchPaths))
    
    for i, path := range searchPaths {
        result := gin.H{
            "path": path,
            "priority": i + 1, // Show search order
        }
        
        if i == 0 && s.config.Web.AssetsDir != "" {
            result["source"] = "configured"
        } else {
            result["source"] = "default"
        }
        
        if stat, err := os.Stat(path); err == nil {
            result["exists"] = true
            result["size"] = stat.Size()
            result["modified"] = stat.ModTime()
            result["readable"] = true
            
            // Try to read first few bytes to verify it's HTML
            if file, err := os.Open(path); err == nil {
                buffer := make([]byte, 200)
                if n, err := file.Read(buffer); err == nil {
                    content := string(buffer[:n])
                    result["looks_like_html"] = strings.Contains(strings.ToLower(content), "<!doctype html") || 
                                               strings.Contains(strings.ToLower(content), "<html")
                    result["preview"] = content
                }
                file.Close()
            }
        } else {
            result["exists"] = false
            result["error"] = err.Error()
        }
        
        pathResults = append(pathResults, result)
    }
    
    diagnostics["web_asset_search"] = pathResults
    
    // Check current working directory
    if cwd, err := os.Getwd(); err == nil {
        diagnostics["working_directory"] = cwd
    }
    
    // Check static directory if configured
    if s.config.Web.ServeStatic {
        staticDiagnostics := gin.H{}
        
        var staticDir string
        if s.config.Web.AssetsDir != "" {
            if filepath.IsAbs(s.config.Web.StaticDir) {
                staticDir = s.config.Web.StaticDir
            } else {
                staticDir = filepath.Join(s.config.Web.AssetsDir, s.config.Web.StaticDir)
            }
        }
        
        if staticDir != "" {
            staticDiagnostics["configured_path"] = staticDir
            if stat, err := os.Stat(staticDir); err == nil {
                staticDiagnostics["exists"] = true
                staticDiagnostics["is_directory"] = stat.IsDir()
                staticDiagnostics["modified"] = stat.ModTime()
                
                // List contents if it's a directory
                if stat.IsDir() {
                    if files, err := os.ReadDir(staticDir); err == nil {
                        fileNames := make([]string, 0, len(files))
                        for _, file := range files {
                            fileNames = append(fileNames, file.Name())
                        }
                        staticDiagnostics["contents"] = fileNames
                    }
                }
            } else {
                staticDiagnostics["exists"] = false
                staticDiagnostics["error"] = err.Error()
            }
        }
        
        diagnostics["static_directory"] = staticDiagnostics
    }
    
    c.JSON(http.StatusOK, diagnostics)
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
