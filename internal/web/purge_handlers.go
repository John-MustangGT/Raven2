// Add these handlers to your existing internal/web/handlers.go file
// Or create a new file internal/web/purge_handlers.go

package web

import (
    "context"
    "net/http"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/sirupsen/logrus"
)

// Add these methods to your existing Server struct

// setupPurgeRoutes adds purge endpoints to your existing router
// Call this from your existing setupRoutes method:
func (s *Server) setupPurgeRoutes() {
    api := s.router.Group("/api")
    
    // Alert management endpoints
    alerts := api.Group("/alerts")
    {
        alerts.DELETE("/purge", s.purgeStaleAlerts)
        alerts.DELETE("/purge/hosts", s.purgeOrphanedHosts)
        alerts.DELETE("/purge/checks", s.purgeOrphanedChecks)
        alerts.DELETE("/purge/all", s.purgeAllStaleData)
    }
    
    // Enhanced configuration endpoints
    config := api.Group("/config")
    {
        config.POST("/refresh", s.refreshConfigWithPurge)
    }
}

// DELETE /api/alerts/purge - Purge stale alerts
func (s *Server) purgeStaleAlerts(c *gin.Context) {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    // Get the alert manager from engine
    alertManager := s.engine.GetAlertManager()
    
    if err := alertManager.PurgeStaleAlerts(ctx); err != nil {
        logrus.WithError(err).Error("Failed to purge stale alerts")
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to purge stale alerts"})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{
        "message": "Stale alerts purged successfully",
        "timestamp": time.Now(),
    })
}

// DELETE /api/alerts/purge/hosts - Purge orphaned hosts
func (s *Server) purgeOrphanedHosts(c *gin.Context) {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    alertManager := s.engine.GetAlertManager()
    
    if err := alertManager.PurgeOrphanedHosts(ctx); err != nil {
        logrus.WithError(err).Error("Failed to purge orphaned hosts")
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to purge orphaned hosts"})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{
        "message": "Orphaned hosts purged successfully",
        "timestamp": time.Now(),
    })
}

// DELETE /api/alerts/purge/checks - Purge orphaned checks
func (s *Server) purgeOrphanedChecks(c *gin.Context) {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    alertManager := s.engine.GetAlertManager()
    
    if err := alertManager.PurgeOrphanedChecks(ctx); err != nil {
        logrus.WithError(err).Error("Failed to purge orphaned checks")
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to purge orphaned checks"})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{
        "message": "Orphaned checks purged successfully",
        "timestamp": time.Now(),
    })
}

// DELETE /api/alerts/purge/all - Purge all stale data
func (s *Server) purgeAllStaleData(c *gin.Context) {
    ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
    defer cancel()
    
    alertManager := s.engine.GetAlertManager()
    
    if err := alertManager.PurgeAll(ctx); err != nil {
        logrus.WithError(err).Error("Failed to purge all stale data")
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to purge all stale data"})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{
        "message": "All stale data purged successfully",
        "timestamp": time.Now(),
    })
}

// POST /api/config/refresh - Refresh configuration with purge
func (s *Server) refreshConfigWithPurge(c *gin.Context) {
    logrus.Info("Configuration refresh with purge requested")
    
    if err := s.engine.RefreshConfigWithPurge(); err != nil {
        logrus.WithError(err).Error("Configuration refresh with purge failed")
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Configuration refresh failed"})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{
        "message": "Configuration refreshed and stale data purged successfully",
        "timestamp": time.Now(),
    })
}
