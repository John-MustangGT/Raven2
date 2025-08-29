// internal/web/handlers.go
package web

import (
    "net/http"
    "strconv"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
    "github.com/sirupsen/logrus"
    "github.com/your-org/raven/internal/database"
)

type HostRequest struct {
    Name        string            `json:"name" binding:"required"`
    DisplayName string            `json:"display_name"`
    IPv4        string            `json:"ipv4"`
    Hostname    string            `json:"hostname"`
    Group       string            `json:"group"`
    Enabled     bool              `json:"enabled"`
    Tags        map[string]string `json:"tags"`
}

type HostResponse struct {
    *database.Host
    Status     string    `json:"status"`
    LastCheck  time.Time `json:"last_check"`
    NextCheck  time.Time `json:"next_check"`
    CheckCount int       `json:"check_count"`
}

// GET /api/hosts
func (s *Server) getHosts(c *gin.Context) {
    group := c.Query("group")
    enabledStr := c.Query("enabled")
    
    filters := database.HostFilters{
        Group: group,
    }
    
    if enabledStr != "" {
        enabled := enabledStr == "true"
        filters.Enabled = &enabled
    }

    hosts, err := s.store.GetHosts(c.Request.Context(), filters)
    if err != nil {
        logrus.WithError(err).Error("Failed to get hosts")
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get hosts"})
        return
    }

    // Enhance with status information
    response := make([]HostResponse, 0, len(hosts))
    for _, host := range hosts {
        hostResp := HostResponse{
            Host:   &host,
            Status: s.getHostStatus(c.Request.Context(), host.ID),
        }
        response = append(response, hostResp)
    }

    c.JSON(http.StatusOK, gin.H{
        "data":  response,
        "count": len(response),
    })
}

// GET /api/hosts/:id
func (s *Server) getHost(c *gin.Context) {
    id := c.Param("id")
    
    host, err := s.store.GetHost(c.Request.Context(), id)
    if err != nil {
        if err.Error() == "host not found" {
            c.JSON(http.StatusNotFound, gin.H{"error": "Host not found"})
            return
        }
        logrus.WithError(err).Error("Failed to get host")
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get host"})
        return
    }

    c.JSON(http.StatusOK, gin.H{"data": host})
}

// POST /api/hosts
func (s *Server) createHost(c *gin.Context) {
    var req HostRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    host := &database.Host{
        ID:          uuid.New().String(),
        Name:        req.Name,
        DisplayName: req.DisplayName,
        IPv4:        req.IPv4,
        Hostname:    req.Hostname,
        Group:       req.Group,
        Enabled:     req.Enabled,
        Tags:        req.Tags,
        CreatedAt:   time.Now(),
        UpdatedAt:   time.Now(),
    }

    if host.Group == "" {
        host.Group = "default"
    }
    if host.Tags == nil {
        host.Tags = make(map[string]string)
    }

    if err := s.store.CreateHost(c.Request.Context(), host); err != nil {
        logrus.WithError(err).Error("Failed to create host")
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create host"})
        return
    }

    // Notify monitoring engine of new host
    s.engine.RefreshConfig()

    c.JSON(http.StatusCreated, gin.H{"data": host})
}

// PUT /api/hosts/:id
func (s *Server) updateHost(c *gin.Context) {
    id := c.Param("id")
    
    var req HostRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    host, err := s.store.GetHost(c.Request.Context(), id)
    if err != nil {
        if err.Error() == "host not found" {
            c.JSON(http.StatusNotFound, gin.H{"error": "Host not found"})
            return
        }
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get host"})
        return
    }

    // Update fields
    host.Name = req.Name
    host.DisplayName = req.DisplayName
    host.IPv4 = req.IPv4
    host.Hostname = req.Hostname
    host.Group = req.Group
    host.Enabled = req.Enabled
    host.Tags = req.Tags
    host.UpdatedAt = time.Now()

    if err := s.store.UpdateHost(c.Request.Context(), host); err != nil {
        logrus.WithError(err).Error("Failed to update host")
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update host"})
        return
    }

    // Notify monitoring engine of host change
    s.engine.RefreshConfig()

    c.JSON(http.StatusOK, gin.H{"data": host})
}

// DELETE /api/hosts/:id
func (s *Server) deleteHost(c *gin.Context) {
    id := c.Param("id")
    
    if err := s.store.DeleteHost(c.Request.Context(), id); err != nil {
        logrus.WithError(err).Error("Failed to delete host")
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete host"})
        return
    }

    // Notify monitoring engine
    s.engine.RefreshConfig()

    c.JSON(http.StatusOK, gin.H{"message": "Host deleted successfully"})
}

// GET /api/status
func (s *Server) getStatus(c *gin.Context) {
    limitStr := c.DefaultQuery("limit", "100")
    limit, _ := strconv.Atoi(limitStr)

    filters := database.StatusFilters{
        HostID:  c.Query("host_id"),
        CheckID: c.Query("check_id"),
        Limit:   limit,
    }

    if exitCodeStr := c.Query("exit_code"); exitCodeStr != "" {
        if exitCode, err := strconv.Atoi(exitCodeStr); err == nil {
            filters.ExitCode = &exitCode
        }
    }

    statuses, err := s.store.GetStatus(c.Request.Context(), filters)
    if err != nil {
        logrus.WithError(err).Error("Failed to get status")
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get status"})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "data":  statuses,
        "count": len(statuses),
    })
}

func (s *Server) getHostStatus(ctx context.Context, hostID string) string {
    // Get latest status for host
    statuses, err := s.store.GetStatus(ctx, database.StatusFilters{
        HostID: hostID,
        Limit:  1,
    })
    
    if err != nil || len(statuses) == 0 {
        return "unknown"
    }

    switch statuses[0].ExitCode {
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
