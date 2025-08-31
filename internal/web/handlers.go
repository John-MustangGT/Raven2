// internal/web/handlers.go
package web

import (
    "context"
    "fmt"
    "net"
    "net/http"
    "strconv"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
    "github.com/sirupsen/logrus"
    "raven2/internal/database"
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

// Enhanced HostResponse with IP check status and additional fields
type HostResponse struct {
    *database.Host
    Status        string            `json:"status"`
    LastCheck     time.Time         `json:"last_check"`
    NextCheck     time.Time         `json:"next_check"`
    CheckCount    int               `json:"check_count"`
    IPAddressOK   bool              `json:"ip_address_ok"`
    IPLastChecked time.Time         `json:"ip_last_checked"`
    SoftFailInfo  map[string]*SoftFailStatus `json:"soft_fail_info,omitempty"`
    OKDuration    map[string]*OKDurationInfo `json:"ok_duration,omitempty"`
}

// SoftFailStatus tracks consecutive failures for a check
type SoftFailStatus struct {
    CurrentFails  int       `json:"current_fails"`
    ThresholdMax  int       `json:"threshold_max"`
    FirstFailTime time.Time `json:"first_fail_time"`
    LastFailTime  time.Time `json:"last_fail_time"`
}

// OKDurationInfo tracks how long a check has been OK
type OKDurationInfo struct {
    OKSince    time.Time `json:"ok_since"`
    Duration   string    `json:"duration"`
    CheckCount int       `json:"check_count"`
}

// Enhanced status response with additional context
type StatusResponse struct {
    *database.Status
    SoftFailsInfo *SoftFailStatus `json:"soft_fails_info,omitempty"`
    OKInfo        *OKDurationInfo `json:"ok_info,omitempty"`
    CheckName     string          `json:"check_name"`
    HostName      string          `json:"host_name"`
}

// CheckRequest represents the request body for creating/updating checks
type CheckRequest struct {
    Name      string                   `json:"name" binding:"required"`
    Type      string                   `json:"type" binding:"required"`
    Hosts     []string                 `json:"hosts" binding:"required"`
    Interval  map[string]string        `json:"interval"`
    Threshold int                      `json:"threshold"`
    Timeout   string                   `json:"timeout"`
    Enabled   bool                     `json:"enabled"`
    Options   map[string]interface{}   `json:"options"`
}

// Alert represents an alert derived from status data
type Alert struct {
    ID        string    `json:"id"`
    Timestamp time.Time `json:"timestamp"`
    Severity  string    `json:"severity"`
    Host      string    `json:"host"`
    Check     string    `json:"check"`
    Message   string    `json:"message"`
    Duration  int64     `json:"duration"` // milliseconds
}

// GET /api/hosts - Enhanced to include IP checks and soft fail info
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

    // Enhance with comprehensive status information
    response := make([]HostResponse, 0, len(hosts))
    for i := range hosts {
        host := hosts[i]
        
        // Get overall status for this specific host
        status := s.getHostStatus(c.Request.Context(), host.ID)
        
        // Get latest status timestamp for this host
        statuses, err := s.store.GetStatus(c.Request.Context(), database.StatusFilters{
            HostID: host.ID,
            Limit:  1,
        })
        
        var lastCheck time.Time
        if err == nil && len(statuses) > 0 {
            lastCheck = statuses[0].Timestamp
        }

        // Check IP address connectivity
//        ipOK, ipLastChecked := s.checkIPAddress(host.IPv4, host.Hostname)
        var ipOK = true
        var ipLastChecked = time.Now()

        // Get soft fail information for all checks on this host
        softFailInfo := s.getSoftFailInfo(c.Request.Context(), host.ID)

        // Get OK duration information for all checks on this host
        okDuration := s.getOKDurationInfo(c.Request.Context(), host.ID)

        hostResp := HostResponse{
            Host:          &host,
            Status:        status,
            LastCheck:     lastCheck,
            NextCheck:     time.Time{}, // TODO: Calculate from scheduler
            CheckCount:    0,           // TODO: Count active checks for this host
            IPAddressOK:   ipOK,
            IPLastChecked: ipLastChecked,
            SoftFailInfo:  softFailInfo,
            OKDuration:    okDuration,
        }
        response = append(response, hostResp)
    }

    c.JSON(http.StatusOK, gin.H{
        "data":  response,
        "count": len(response),
    })
}

// checkIPAddress performs a basic connectivity test to the host's IP or hostname
func (s *Server) checkIPAddress(ipv4, hostname string) (bool, time.Time) {
    checkTime := time.Now()
    
    target := ipv4
    if target == "" {
        target = hostname
    }
    if target == "" {
        return false, checkTime
    }

    // Simple TCP dial test to common ports (ping alternative that works through firewalls)
    timeout := 3 * time.Second
    ports := []string{"80", "443", "22", "21"} // Common ports to test
    
    for _, port := range ports {
        conn, err := net.DialTimeout("tcp", net.JoinHostPort(target, port), timeout)
        if err == nil {
            conn.Close()
            return true, checkTime
        }
    }

    // Try ICMP ping as fallback (may not work in all environments)
    // This is a simplified check - in production you might use a proper ping library
    return false, checkTime
}

// getSoftFailInfo retrieves soft failure information for all checks on a host
func (s *Server) getSoftFailInfo(ctx context.Context, hostID string) map[string]*SoftFailStatus {
    softFailInfo := make(map[string]*SoftFailStatus)

    // Get recent statuses for this host to analyze failure patterns
    statuses, err := s.store.GetStatus(ctx, database.StatusFilters{
        HostID: hostID,
        Limit:  100, // Get enough history to analyze patterns
    })
    
    if err != nil {
        logrus.WithError(err).Error("Failed to get status for soft fail analysis")
        return softFailInfo
    }

    // Group statuses by check_id and analyze failure patterns
    checkStatuses := make(map[string][]*database.Status)
    for i := range statuses {
        checkID := statuses[i].CheckID
        if checkStatuses[checkID] == nil {
            checkStatuses[checkID] = make([]*database.Status, 0)
        }
        checkStatuses[checkID] = append(checkStatuses[checkID], &statuses[i])
    }

    // Analyze each check's failure pattern
    for checkID, statusList := range checkStatuses {
        if len(statusList) == 0 {
            continue
        }

        // Sort by timestamp (most recent first)
        // statusList is already sorted from the database query

        // Look for consecutive failures at the start of the list (most recent)
        consecutiveFails := 0
        var firstFailTime, lastFailTime time.Time
        
        for _, status := range statusList {
            if status.ExitCode != 0 { // Non-OK status
                consecutiveFails++
                lastFailTime = status.Timestamp
                if firstFailTime.IsZero() {
                    firstFailTime = status.Timestamp
                }
            } else {
                break // Stop at first OK status
            }
        }

        // Only include if there are current failures
        if consecutiveFails > 0 {
            // Get the check threshold (default to 3 if not available)
            check, err := s.store.GetCheck(ctx, checkID)
            threshold := 3
            if err == nil && check.Threshold > 0 {
                threshold = check.Threshold
            }

            softFailInfo[checkID] = &SoftFailStatus{
                CurrentFails:  consecutiveFails,
                ThresholdMax:  threshold,
                FirstFailTime: firstFailTime,
                LastFailTime:  lastFailTime,
            }
        }
    }

    return softFailInfo
}

// getOKDurationInfo retrieves information about how long checks have been OK
func (s *Server) getOKDurationInfo(ctx context.Context, hostID string) map[string]*OKDurationInfo {
    okDurationInfo := make(map[string]*OKDurationInfo)

    // Get recent statuses for this host
    statuses, err := s.store.GetStatus(ctx, database.StatusFilters{
        HostID: hostID,
        Limit:  1000, // Get more history for OK duration analysis
    })
    
    if err != nil {
        logrus.WithError(err).Error("Failed to get status for OK duration analysis")
        return okDurationInfo
    }

    // Group statuses by check_id
    checkStatuses := make(map[string][]*database.Status)
    for i := range statuses {
        checkID := statuses[i].CheckID
        if checkStatuses[checkID] == nil {
            checkStatuses[checkID] = make([]*database.Status, 0)
        }
        checkStatuses[checkID] = append(checkStatuses[checkID], &statuses[i])
    }

    // Analyze OK duration for each check
    for checkID, statusList := range checkStatuses {
        if len(statusList) == 0 {
            continue
        }

        // Check if the most recent status is OK
        if statusList[0].ExitCode == 0 {
            okSince := statusList[0].Timestamp
            okCount := 1

            // Count consecutive OK statuses
            for i := 1; i < len(statusList); i++ {
                if statusList[i].ExitCode == 0 {
                    okSince = statusList[i].Timestamp
                    okCount++
                } else {
                    break // Stop at first non-OK status
                }
            }

            duration := time.Since(okSince)
            durationStr := formatDuration(duration)

            okDurationInfo[checkID] = &OKDurationInfo{
                OKSince:    okSince,
                Duration:   durationStr,
                CheckCount: okCount,
            }
        }
    }

    return okDurationInfo
}

// GET /api/status - Enhanced to include soft fail and OK duration info
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

    // Enhance statuses with additional context
    enhancedStatuses := make([]StatusResponse, 0, len(statuses))
    
    for i := range statuses {
        status := statuses[i]
        
        // Get check name
        checkName := status.CheckID
        if check, err := s.store.GetCheck(c.Request.Context(), status.CheckID); err == nil {
            checkName = check.Name
        }

        // Get host name
        hostName := status.HostID
        if host, err := s.store.GetHost(c.Request.Context(), status.HostID); err == nil {
            if host.DisplayName != "" {
                hostName = host.DisplayName
            } else {
                hostName = host.Name
            }
        }

        enhancedStatus := StatusResponse{
            Status:    &status,
            CheckName: checkName,
            HostName:  hostName,
        }

        // Add soft fail info for non-OK statuses
        if status.ExitCode != 0 {
            softFailInfo := s.getSoftFailInfo(c.Request.Context(), status.HostID)
            if info, exists := softFailInfo[status.CheckID]; exists {
                enhancedStatus.SoftFailsInfo = info
            }
        }

        // Add OK duration info for OK statuses
        if status.ExitCode == 0 {
            okDurationInfo := s.getOKDurationInfo(c.Request.Context(), status.HostID)
            if info, exists := okDurationInfo[status.CheckID]; exists {
                enhancedStatus.OKInfo = info
            }
        }

        enhancedStatuses = append(enhancedStatuses, enhancedStatus)
    }

    c.JSON(http.StatusOK, gin.H{
        "data":  enhancedStatuses,
        "count": len(enhancedStatuses),
    })
}

// Helper function to format duration in a human-readable way
func formatDuration(d time.Duration) string {
    if d < time.Minute {
        return fmt.Sprintf("%.0fs", d.Seconds())
    } else if d < time.Hour {
        return fmt.Sprintf("%.0fm", d.Minutes())
    } else if d < 24*time.Hour {
        hours := int(d.Hours())
        minutes := int(d.Minutes()) % 60
        if minutes > 0 {
            return fmt.Sprintf("%dh %dm", hours, minutes)
        }
        return fmt.Sprintf("%dh", hours)
    } else {
        days := int(d.Hours()) / 24
        hours := int(d.Hours()) % 24
        if hours > 0 {
            return fmt.Sprintf("%dd %dh", days, hours)
        }
        return fmt.Sprintf("%dd", days)
    }
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

// PUT /api/checks/:id - Update existing check
func (s *Server) updateCheck(c *gin.Context) {
    id := c.Param("id")
    
    var req CheckRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    // Get existing check
    check, err := s.store.GetCheck(c.Request.Context(), id)
    if err != nil {
        if err.Error() == "check not found" {
            c.JSON(http.StatusNotFound, gin.H{"error": "Check not found"})
            return
        }
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get check"})
        return
    }

    // Parse interval durations
    intervalDurations := make(map[string]time.Duration)
    for state, intervalStr := range req.Interval {
        if duration, err := time.ParseDuration(intervalStr); err == nil {
            intervalDurations[state] = duration
        } else {
            c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid interval format for " + state + ": " + intervalStr})
            return
        }
    }

    // Parse timeout
    var timeout time.Duration
    if req.Timeout != "" {
        if t, err := time.ParseDuration(req.Timeout); err == nil {
            timeout = t
        } else {
            c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid timeout format: " + req.Timeout})
            return
        }
    }

    // Update check fields
    check.Name = req.Name
    check.Type = req.Type
    check.Hosts = req.Hosts
    check.Interval = intervalDurations
    check.Threshold = req.Threshold
    check.Timeout = timeout
    check.Enabled = req.Enabled
    check.Options = req.Options
    check.UpdatedAt = time.Now()

    if err := s.store.UpdateCheck(c.Request.Context(), check); err != nil {
        logrus.WithError(err).Error("Failed to update check")
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update check"})
        return
    }

    // Notify monitoring engine of check change
    s.engine.RefreshConfig()

    c.JSON(http.StatusOK, gin.H{"data": check})
}

// DELETE /api/checks/:id - Delete existing check
func (s *Server) deleteCheck(c *gin.Context) {
    id := c.Param("id")
    
    // Verify check exists
    _, err := s.store.GetCheck(c.Request.Context(), id)
    if err != nil {
        if err.Error() == "check not found" {
            c.JSON(http.StatusNotFound, gin.H{"error": "Check not found"})
            return
        }
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get check"})
        return
    }

    if err := s.store.DeleteCheck(c.Request.Context(), id); err != nil {
        logrus.WithError(err).Error("Failed to delete check")
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete check"})
        return
    }

    // Notify monitoring engine
    s.engine.RefreshConfig()

    c.JSON(http.StatusOK, gin.H{"message": "Check deleted successfully"})
}

// GET /api/alerts - Get current alerts
func (s *Server) getAlerts(c *gin.Context) {
    limitStr := c.DefaultQuery("limit", "100")
    limit, _ := strconv.Atoi(limitStr)
    
    severityFilter := c.Query("severity") // optional: critical, warning, unknown

    // Get recent status entries that indicate problems
    statuses, err := s.store.GetStatus(c.Request.Context(), database.StatusFilters{
        Limit: limit,
    })
    if err != nil {
        logrus.WithError(err).Error("Failed to get status for alerts")
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get alerts"})
        return
    }

    // Convert problematic statuses to alerts
    var alerts []Alert
    now := time.Now()
    
    for _, status := range statuses {
        if status.ExitCode == 0 {
            continue // Skip OK statuses
        }

        severity := getStatusName(status.ExitCode)
        
        // Apply severity filter if specified
        if severityFilter != "" && severity != severityFilter {
            continue
        }

        alert := Alert{
            ID:        status.ID,
            Timestamp: status.Timestamp,
            Severity:  severity,
            Host:      status.HostID,
            Check:     status.CheckID,
            Message:   status.Output,
            Duration:  now.Sub(status.Timestamp).Milliseconds(),
        }
        
        alerts = append(alerts, alert)
    }

    c.JSON(http.StatusOK, gin.H{
        "data":  alerts,
        "count": len(alerts),
    })
}

// GET /api/alerts/summary - Get alert summary statistics
func (s *Server) getAlertsSummary(c *gin.Context) {
    statuses, err := s.store.GetStatus(c.Request.Context(), database.StatusFilters{
        Limit: 1000, // Get more data for accurate summary
    })
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get alert summary"})
        return
    }

    summary := map[string]int{
        "active":   0,
        "critical": 0,
        "warning":  0,
        "unknown":  0,
    }

    for _, status := range statuses {
        if status.ExitCode > 0 {
            summary["active"]++
            
            switch status.ExitCode {
            case 1:
                summary["warning"]++
            case 2:
                summary["critical"]++
            case 3:
                summary["unknown"]++
            }
        }
    }

    c.JSON(http.StatusOK, gin.H{"data": summary})
}

// POST /api/checks - Update the existing createCheck to handle intervals properly
func (s *Server) createCheck(c *gin.Context) {
    var req CheckRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    // Parse interval durations
    intervalDurations := make(map[string]time.Duration)
    for state, intervalStr := range req.Interval {
        if duration, err := time.ParseDuration(intervalStr); err == nil {
            intervalDurations[state] = duration
        } else {
            c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid interval format for " + state + ": " + intervalStr})
            return
        }
    }

    // Parse timeout
    var timeout time.Duration
    if req.Timeout != "" {
        if t, err := time.ParseDuration(req.Timeout); err == nil {
            timeout = t
        } else {
            c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid timeout format: " + req.Timeout})
            return
        }
    }

    check := &database.Check{
        ID:        uuid.New().String(),
        Name:      req.Name,
        Type:      req.Type,
        Hosts:     req.Hosts,
        Interval:  intervalDurations,
        Threshold: req.Threshold,
        Timeout:   timeout,
        Enabled:   req.Enabled,
        Options:   req.Options,
        CreatedAt: time.Now(),
        UpdatedAt: time.Now(),
    }

    if err := s.store.CreateCheck(c.Request.Context(), check); err != nil {
        logrus.WithError(err).Error("Failed to create check")
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create check"})
        return
    }

    s.engine.RefreshConfig()
    c.JSON(http.StatusCreated, gin.H{"data": check})
}

// Helper function to convert exit codes to status names
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
