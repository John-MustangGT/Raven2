// js/api.js - Enhanced API service layer with detail view support
window.RavenAPI = {
    // Web configuration
    async loadWebConfig() {
        try {
            const response = await axios.get('/api/web-config');
            return response.data.data;
        } catch (error) {
            console.error('Failed to load web config:', error);
            return {
                header_link: 'https://github.com/John-MustangGT/raven2',
                serve_static: false,
                root: 'index.html'
            };
        }
    },

    // Host management
    async loadHosts() {
        const response = await axios.get('/api/hosts');
        return response.data.data || [];
    },

    async loadHost(hostId) {
        const response = await axios.get(`/api/hosts/${hostId}`);
        return response.data.data;
    },

    async createHost(hostData) {
        await axios.post('/api/hosts', hostData);
    },

    async updateHost(id, hostData) {
        await axios.put(`/api/hosts/${id}`, hostData);
    },

    async deleteHost(id) {
        await axios.delete(`/api/hosts/${id}`);
    },

    // Check management
    async loadChecks() {
        const response = await axios.get('/api/checks');
        return response.data.data || [];
    },

    async loadCheck(checkId) {
        const response = await axios.get(`/api/checks/${checkId}`);
        return response.data.data;
    },

    async createCheck(checkData) {
        await axios.post('/api/checks', checkData);
    },

    async updateCheck(id, checkData) {
        await axios.put(`/api/checks/${id}`, checkData);
    },

    async deleteCheck(id) {
        await axios.delete(`/api/checks/${id}`);
    },

    // Status and monitoring
    async loadStatus(limit = 100, filters = {}) {
        const params = new URLSearchParams();
        if (limit) params.append('limit', limit.toString());
        if (filters.hostId) params.append('host_id', filters.hostId);
        if (filters.checkId) params.append('check_id', filters.checkId);
        if (filters.exitCode !== undefined) params.append('exit_code', filters.exitCode.toString());
        
        const response = await axios.get(`/api/status?${params.toString()}`);
        return response.data.data || [];
    },

    async loadStatusHistory(hostId, checkId = null, since = null) {
        const params = new URLSearchParams();
        if (since) params.append('since', since);
        
        let url = `/api/status/history/${hostId}`;
        if (checkId) url += `/${checkId}`;
        if (params.toString()) url += `?${params.toString()}`;
        
        const response = await axios.get(url);
        return response.data.data || [];
    },

    // Enhanced status queries for detail views
    async loadHostStatuses(hostId, limit = 50) {
        return this.loadStatus(limit, { hostId });
    },

    async loadCheckStatuses(checkId, limit = 100) {
        return this.loadStatus(limit, { checkId });
    },

    async loadFailingStatuses(limit = 100) {
        const statuses = await this.loadStatus(limit * 2); // Get more to filter
        return statuses.filter(status => status.exit_code > 0).slice(0, limit);
    },

    // Alert management
    async loadAlerts(limit = 100) {
        const params = new URLSearchParams();
        if (limit) params.append('limit', limit.toString());
        
        const response = await axios.get(`/api/alerts?${params.toString()}`);
        return response.data.data || [];
    },

    async loadAlertsSummary() {
        const response = await axios.get('/api/alerts/summary');
        return response.data.data || {};
    },

    // System information
    async loadBuildInfo() {
        const response = await axios.get('/api/build-info');
        return response.data.data;
    },

    async loadStats() {
        const response = await axios.get('/api/stats');
        return response.data.data || {};
    },

    async healthCheck() {
        const response = await axios.get('/api/health');
        return response.data;
    },

    // Enhanced queries for detail views
    async findHostsWithAlert(checkId, checkName = null) {
        try {
            // Get all hosts
            const hosts = await this.loadHosts();
            
            // Filter hosts that have issues with this specific check
            const affectedHosts = hosts.filter(host => {
                // Check soft fail info
                if (host.soft_fail_info) {
                    for (const [hostCheckId, failInfo] of Object.entries(host.soft_fail_info)) {
                        if (hostCheckId === checkId || 
                            failInfo.check_name === checkName) {
                            return true;
                        }
                    }
                }
                return false;
            });
            
            return affectedHosts;
        } catch (error) {
            console.error('Failed to find hosts with alert:', error);
            return [];
        }
    },

    async loadHostsInGroup(groupName) {
        const hosts = await this.loadHosts();
        return hosts.filter(host => host.group === groupName);
    },

    async loadHostsByStatus(status) {
        const hosts = await this.loadHosts();
        return hosts.filter(host => host.status === status);
    },

    // Batch operations
    async loadMultipleHosts(hostIds) {
        const promises = hostIds.map(id => this.loadHost(id).catch(err => null));
        const results = await Promise.allSettled(promises);
        return results
            .filter(result => result.status === 'fulfilled' && result.value)
            .map(result => result.value);
    },

    // Search and filtering
    async searchHosts(query, limit = 50) {
        const hosts = await this.loadHosts();
        const searchTerm = query.toLowerCase();
        
        return hosts.filter(host => 
            host.name.toLowerCase().includes(searchTerm) ||
            (host.display_name && host.display_name.toLowerCase().includes(searchTerm)) ||
            (host.ipv4 && host.ipv4.includes(searchTerm)) ||
            (host.hostname && host.hostname.toLowerCase().includes(searchTerm))
        ).slice(0, limit);
    },

    async searchChecks(query, limit = 50) {
        const checks = await this.loadChecks();
        const searchTerm = query.toLowerCase();
        
        return checks.filter(check =>
            check.name.toLowerCase().includes(searchTerm) ||
            check.type.toLowerCase().includes(searchTerm) ||
            (check.id && check.id.toLowerCase().includes(searchTerm))
        ).slice(0, limit);
    },

    // Real-time data polling
    async pollStatus(callback, interval = 30000) {
        const poll = async () => {
            try {
                const statuses = await this.loadStatus(1000);
                callback(null, statuses);
            } catch (error) {
                callback(error, null);
            }
        };

        // Initial poll
        await poll();

        // Set up interval
        return setInterval(poll, interval);
    },

    // Data aggregation helpers
    async getHostSummary(hostId) {
        try {
            const [host, statuses] = await Promise.all([
                this.loadHost(hostId),
                this.loadHostStatuses(hostId, 100)
            ]);

            if (!host) return null;

            // Aggregate status information
            const summary = {
                host,
                totalChecks: 0,
                recentStatuses: statuses.slice(0, 10),
                statusCounts: {
                    ok: statuses.filter(s => s.exit_code === 0).length,
                    warning: statuses.filter(s => s.exit_code === 1).length,
                    critical: statuses.filter(s => s.exit_code === 2).length,
                    unknown: statuses.filter(s => s.exit_code === 3).length
                },
                lastCheckTime: statuses.length > 0 ? statuses[0].timestamp : null,
                healthScore: 0
            };

            // Calculate health score (0-100)
            const total = statuses.length;
            if (total > 0) {
                const okCount = summary.statusCounts.ok;
                summary.healthScore = Math.round((okCount / total) * 100);
            }

            // Count unique checks
            const uniqueChecks = new Set(statuses.map(s => s.check_id));
            summary.totalChecks = uniqueChecks.size;

            return summary;
        } catch (error) {
            console.error('Failed to get host summary:', error);
            return null;
        }
    },

    async getAlertSummary(alert) {
        try {
            const [statuses, affectedHosts] = await Promise.all([
                this.loadCheckStatuses(alert.check, 50),
                this.findHostsWithAlert(alert.check, alert.check_name)
            ]);

            return {
                alert,
                occurrences: statuses.filter(s => s.exit_code > 0).length,
                affectedHostCount: affectedHosts.length,
                firstOccurrence: statuses.length > 0 ? statuses[statuses.length - 1].timestamp : null,
                lastOccurrence: statuses.length > 0 ? statuses[0].timestamp : null,
                severity: alert.severity,
                trend: this.calculateAlertTrend(statuses)
            };
        } catch (error) {
            console.error('Failed to get alert summary:', error);
            return null;
        }
    },

    // Trend analysis
    calculateAlertTrend(statuses) {
        if (!statuses || statuses.length < 5) return 'insufficient-data';
        
        const recent = statuses.slice(0, 5);
        const older = statuses.slice(5, 10);
        
        const recentFails = recent.filter(s => s.exit_code > 0).length;
        const olderFails = older.length > 0 ? older.filter(s => s.exit_code > 0).length : 0;
        
        const recentFailRate = recentFails / recent.length;
        const olderFailRate = older.length > 0 ? olderFails / older.length : 0;
        
        if (recentFailRate > olderFailRate * 1.2) return 'worsening';
        if (recentFailRate < olderFailRate * 0.8) return 'improving';
        return 'stable';
    },

    // Export functionality
    async exportData(type, format = 'json') {
        let data;
        let filename;
        
        switch (type) {
            case 'hosts':
                data = await this.loadHosts();
                filename = `raven-hosts-${new Date().toISOString().split('T')[0]}.${format}`;
                break;
            case 'checks':
                data = await this.loadChecks();
                filename = `raven-checks-${new Date().toISOString().split('T')[0]}.${format}`;
                break;
            case 'alerts':
                data = await this.loadAlerts();
                filename = `raven-alerts-${new Date().toISOString().split('T')[0]}.${format}`;
                break;
            case 'status':
                data = await this.loadStatus(1000);
                filename = `raven-status-${new Date().toISOString().split('T')[0]}.${format}`;
                break;
            default:
                throw new Error(`Unknown export type: ${type}`);
        }

        return { data, filename };
    },

    // Utility methods
    async testConnection() {
        try {
            await this.healthCheck();
            return true;
        } catch (error) {
            return false;
        }
    },

    formatError(error) {
        if (error.response && error.response.data && error.response.data.error) {
            return error.response.data.error;
        }
        if (error.message) {
            return error.message;
        }
        return 'An unknown error occurred';
    },

    // Cache management (simple in-memory cache)
    _cache: new Map(),
    _cacheTimeout: 30000, // 30 seconds

    async getCached(key, fetchFn) {
        const cached = this._cache.get(key);
        if (cached && Date.now() - cached.timestamp < this._cacheTimeout) {
            return cached.data;
        }

        const data = await fetchFn();
        this._cache.set(key, {
            data,
            timestamp: Date.now()
        });

        return data;
    },

    clearCache() {
        this._cache.clear();
    },

    // WebSocket helper (if available)
    createWebSocket(path = '/ws') {
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const wsUrl = `${protocol}//${window.location.host}${path}`;
        return new WebSocket(wsUrl);
    }
};
