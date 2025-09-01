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
        ipOK, ipLastChecked := s.checkIPAddress(host.IPv4, host.Hostname)

        // CHANGE: Use NEW functions with names
        softFailInfo := s.getSoftFailInfoWithNames(c.Request.Context(), host.ID)
        okDuration := s.getOKDurationInfoWithNames(c.Request.Context(), host.ID)
        checkNames := s.getCheckNamesForHost(c.Request.Context(), host.ID)

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
            CheckNames:    checkNames,    // NEW: Add this line
        }
        response = append(response, hostResp)
    }

    c.JSON(http.StatusOK, gin.H{
        "data":  response,
        "count": len(response),
    })
}
