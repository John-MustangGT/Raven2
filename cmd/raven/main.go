package main

import (
    "context"
    "flag"
    "fmt"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/sirupsen/logrus"
    "internal/config"
    "internal/database"
    "internal/metrics"
    "internal/monitoring"
    "internal/web"
)

func main() {
    configFile := flag.String("config", "config.yaml", "Configuration file path")
    version := flag.Bool("version", false, "Show version information")
    flag.Parse()

    if *version {
        fmt.Printf("Raven Network Monitoring v2.0.0\nBuild: %s\n", getBuildInfo())
        os.Exit(0)
    }

    // Load configuration
    cfg, err := config.Load(*configFile)
    if err != nil {
        logrus.Fatalf("Failed to load config: %v", err)
    }

    // Setup logging
    setupLogging(cfg.Logging)

    logrus.WithFields(logrus.Fields{
        "config_file": *configFile,
        "port":        cfg.Server.Port,
        "workers":     cfg.Server.Workers,
    }).Info("Starting Raven monitoring system")

    // Initialize database
    store, err := database.NewBoltStore(cfg.Database.Path)
    if err != nil {
        logrus.Fatalf("Failed to initialize database: %v", err)
    }
    defer store.Close()

    // Initialize metrics
    metricsCollector := metrics.NewCollector(store)

    // Initialize monitoring engine
    engine, err := monitoring.NewEngine(cfg, store, metricsCollector)
    if err != nil {
        logrus.Fatalf("Failed to initialize monitoring engine: %v", err)
    }

    // Initialize web server
    webServer := web.NewServer(cfg, store, engine, metricsCollector)

    // Start services
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // Start monitoring engine
    go engine.Start(ctx)

    // Start web server
    go webServer.Start(ctx)

    // Wait for shutdown signal
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
    
    sig := <-sigChan
    logrus.WithField("signal", sig).Info("Received shutdown signal")

    // Graceful shutdown
    cancel()
    
    // Give services time to shutdown
    time.Sleep(2 * time.Second)
    logrus.Info("Shutdown complete")
}

func setupLogging(cfg config.LoggingConfig) {
    level, err := logrus.ParseLevel(cfg.Level)
    if err != nil {
        level = logrus.InfoLevel
    }
    logrus.SetLevel(level)

    if cfg.Format == "json" {
        logrus.SetFormatter(&logrus.JSONFormatter{})
    } else {
        logrus.SetFormatter(&logrus.TextFormatter{
            FullTimestamp: true,
        })
    }
}

func getBuildInfo() string {
    return "dev-build" // This would be replaced by build system
}

