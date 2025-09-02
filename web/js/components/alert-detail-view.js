// js/components/alert-detail-view.js - Detailed alert view showing all hosts with this alert
window.AlertDetailView = {
    props: {
        alert: Object,
        affectedHosts: Array,
        alertStatuses: Array,
        loading: Boolean,
        loadingHosts: Boolean
    },
    emits: ['back-to-alerts', 'refresh-alert-data', 'view-host-detail'],
    methods: {
        formatTime(timestamp) {
            return window.RavenUtils.formatTime(timestamp);
        },
        formatDuration(duration) {
            return window.RavenUtils.formatDuration(duration);
        },
        getStatusName(exitCode) {
            return window.RavenUtils.getStatusName(exitCode);
        },
        calculateDuration(timestamp) {
            return window.RavenUtils.calculateDuration(timestamp);
        },
        getSeverityIcon(severity) {
            switch (severity) {
                case 'critical': return 'fas fa-times-circle';
                case 'warning': return 'fas fa-exclamation-triangle';
                case 'unknown': return 'fas fa-question-circle';
                default: return 'fas fa-info-circle';
            }
        },
        getSeverityColor(severity) {
            switch (severity) {
                case 'critical': return 'var(--danger-color)';
                case 'warning': return 'var(--warning-color)';
                case 'unknown': return 'var(--text-muted)';
                default: return 'var(--primary-color)';
            }
        },
        getAlertAge() {
            if (!this.alert || !this.alert.timestamp) return 'Unknown';
            const duration = this.calculateDuration(this.alert.timestamp);
            return this.formatDuration(duration);
        },
        getAffectedHostsCount() {
            return this.affectedHosts ? this.affectedHosts.length : 0;
        },
        getRecentAlertHistory() {
            if (!this.alertStatuses) return [];
            
            return this.alertStatuses
                .filter(status => {
                    // Only show statuses for this specific check/alert type
                    return status.check_id === this.alert.check || 
                           status.check_name === this.alert.check_name ||
                           status.check === this.alert.check;
                })
                .slice(0, 20)
                .map(status => ({
                    ...status,
                    host_name: status.host_name || status.host_id || status.host,
                    check_name: status.check_name || this.alert.check_name || this.alert.check,
                    status: this.getStatusName(status.exit_code),
                    duration_since: this.calculateDuration(status.timestamp)
                }));
        },
        getHostsAffectedByThisAlert() {
            if (!this.affectedHosts) return [];
            
            return this.affectedHosts.map(host => {
                // Find the specific alert/status for this check on this host
                let relevantStatus = null;
                let softFailInfo = null;
                
                // Look for soft fail info for this specific check
                if (host.soft_fail_info) {
                    for (const [checkId, failInfo] of Object.entries(host.soft_fail_info)) {
                        if (checkId === this.alert.check || 
                            failInfo.check_name === this.alert.check_name ||
                            failInfo.check_name === this.alert.check) {
                            softFailInfo = failInfo;
                            break;
                        }
                    }
                }
                
                // Find most recent status for this check
                if (this.alertStatuses) {
                    relevantStatus = this.alertStatuses.find(status => 
                        status.host_id === host.id && 
                        (status.check_id === this.alert.check ||
                         status.check_name === this.alert.check_name ||
                         status.check === this.alert.check)
                    );
                }
                
                return {
                    ...host,
                    relevantStatus,
                    softFailInfo,
                    alertSeverity: this.alert.severity,
                    lastAlertTime: relevantStatus ? relevantStatus.timestamp : null
                };
            }).sort((a, b) => {
                // Sort by severity, then by most recent alert time
                const severityOrder = { critical: 3, warning: 2, unknown: 1 };
                const aSev = severityOrder[a.alertSeverity] || 0;
                const bSev = severityOrder[b.alertSeverity] || 0;
                
                if (aSev !== bSev) return bSev - aSev;
                
                const aTime = a.lastAlertTime ? new Date(a.lastAlertTime) : new Date(0);
                const bTime = b.lastAlertTime ? new Date(b.lastAlertTime) : new Date(0);
                return bTime - aTime;
            });
        },
        getAlertTrend() {
            const recentHistory = this.getRecentAlertHistory();
            if (recentHistory.length < 3) return 'insufficient-data';
            
            const recent = recentHistory.slice(0, 5);
            const failCount = recent.filter(h => h.exit_code !== 0).length;
            
            if (failCount >= 4) return 'worsening';
            if (failCount <= 1) return 'improving';
            return 'stable';
        },
        getTrendIcon(trend) {
            switch (trend) {
                case 'worsening': return 'fas fa-arrow-down';
                case 'improving': return 'fas fa-arrow-up';
                case 'stable': return 'fas fa-equals';
                default: return 'fas fa-question';
            }
        },
        getTrendColor(trend) {
            switch (trend) {
                case 'worsening': return 'var(--danger-color)';
                case 'improving': return 'var(--success-color)';
                case 'stable': return 'var(--warning-color)';
                default: return 'var(--text-muted)';
            }
        }
    },
    template: `
        <div class="alert-detail-view">
            <!-- Header with Back Button -->
            <div class="detail-header">
                <button class="btn btn-secondary" @click="$emit('back-to-alerts')">
                    <i class="fas fa-arrow-left"></i>
                    <span>Back to Alerts</span>
                </button>
                <div class="detail-header-info">
                    <h1 class="detail-title" v-if="alert">
                        <i :class="getSeverityIcon(alert.severity)" :style="{ color: getSeverityColor(alert.severity) }"></i>
                        {{ alert.check_name || alert.check }} Alert
                        <span class="detail-subtitle">{{ alert.severity.toUpperCase() }}</span>
                    </h1>
                    <div class="detail-actions">
                        <button class="btn btn-secondary" @click="$emit('refresh-alert-data')" :disabled="loading">
                            <i class="fas fa-sync" :class="{ 'fa-spin': loading }"></i>
                            <span>Refresh</span>
                        </button>
                    </div>
                </div>
            </div>

            <div v-if="loading" class="loading">
                <div class="spinner"></div>
                Loading alert details...
            </div>

            <div v-else-if="!alert" class="loading">
                <i class="fas fa-exclamation-triangle" style="color: var(--warning-color); font-size: 2rem; margin-right: 1rem;"></i>
                Alert not found
            </div>

            <div v-else class="alert-detail-content">
                <!-- Alert Overview Cards -->
                <div class="metrics-grid" style="margin-bottom: 2rem;">
                    <div class="metric-card">
                        <div class="metric-header">
                            <span class="metric-title">Alert Severity</span>
                            <div class="metric-icon" :class="alert.severity">
                                <i :class="getSeverityIcon(alert.severity)"></i>
                            </div>
                        </div>
                        <div class="metric-value">{{ alert.severity.toUpperCase() }}</div>
                        <div class="metric-change">{{ alert.check_name || alert.check }}</div>
                    </div>
                    
                    <div class="metric-card">
                        <div class="metric-header">
                            <span class="metric-title">Alert Age</span>
                            <div class="metric-icon unknown">
                                <i class="fas fa-clock"></i>
                            </div>
                        </div>
                        <div class="metric-value" style="font-size: 1.5rem;">{{ getAlertAge() }}</div>
                        <div class="metric-change">First detected: {{ formatTime(alert.timestamp) }}</div>
                    </div>
                    
                    <div class="metric-card">
                        <div class="metric-header">
                            <span class="metric-title">Affected Hosts</span>
                            <div class="metric-icon critical">
                                <i class="fas fa-server"></i>
                            </div>
                        </div>
                        <div class="metric-value">{{ getAffectedHostsCount() }}</div>
                        <div class="metric-change">{{ getAffectedHostsCount() === 1 ? 'host' : 'hosts' }} experiencing this alert</div>
                    </div>
                    
                    <div class="metric-card">
                        <div class="metric-header">
                            <span class="metric-title">Alert Trend</span>
                            <div class="metric-icon" :style="{ color: getTrendColor(getAlertTrend()) }">
                                <i :class="getTrendIcon(getAlertTrend())"></i>
                            </div>
                        </div>
                        <div class="metric-value" :style="{ color: getTrendColor(getAlertTrend()), fontSize: '1.5rem' }">
                            {{ getAlertTrend().replace('-', ' ').toUpperCase() }}
                        </div>
                        <div class="metric-change">Based on recent history</div>
                    </div>
                </div>

                <!-- Alert Information -->
                <div class="data-table" style="margin-bottom: 2rem;">
                    <div class="table-header">
                        <h3 class="table-title">Alert Details</h3>
                    </div>
                    <div class="table-content">
                        <table>
                            <tbody>
                                <tr>
                                    <td style="font-weight: 600; width: 200px;">Alert ID</td>
                                    <td style="font-family: monospace;">{{ alert.id }}</td>
                                </tr>
                                <tr>
                                    <td style="font-weight: 600;">Check Name</td>
                                    <td>{{ alert.check_name || alert.check }}</td>
                                </tr>
                                <tr>
                                    <td style="font-weight: 600;">Severity</td>
                                    <td>
                                        <span class="status-badge" :class="'status-' + alert.severity">
                                            <div class="status-indicator" :class="'status-' + alert.severity"></div>
                                            {{ alert.severity.toUpperCase() }}
                                        </span>
                                    </td>
                                </tr>
                                <tr>
                                    <td style="font-weight: 600;">First Detected</td>
                                    <td>{{ formatTime(alert.timestamp) }}</td>
                                </tr>
                                <tr>
                                    <td style="font-weight: 600;">Duration</td>
                                    <td>{{ getAlertAge() }}</td>
                                </tr>
                                <tr>
                                    <td style="font-weight: 600;">Primary Host</td>
                                    <td>{{ alert.host_name || alert.host }}</td>
                                </tr>
                                <tr>
                                    <td style="font-weight: 600;">Message</td>
                                    <td style="word-break: break-word;">{{ alert.message || 'No message available' }}</td>
                                </tr>
                            </tbody>
                        </table>
                    </div>
                </div>

                <!-- Affected Hosts -->
                <div class="data-table" style="margin-bottom: 2rem;">
                    <div class="table-header">
                        <h3 class="table-title">
                            <i class="fas fa-server" style="color: var(--danger-color);"></i>
                            Hosts Affected by This Alert ({{ getAffectedHostsCount() }})
                        </h3>
                    </div>
                    <div class="table-content">
                        <div v-if="loadingHosts" class="loading">
                            <div class="spinner"></div>
                            Loading affected hosts...
                        </div>
                        
                        <div v-else-if="getAffectedHostsCount() === 0" class="loading">
                            <i class="fas fa-info-circle" style="color: var(--primary-color); font-size: 2rem; margin-right: 1rem;"></i>
                            No hosts currently affected by this alert
                        </div>
                        
                        <table v-else>
                            <thead>
                                <tr>
                                    <th>Host</th>
                                    <th>IP Address</th>
                                    <th>Alert Status</th>
                                    <th>Soft Fails</th>
                                    <th>Last Alert</th>
                                    <th>Actions</th>
                                </tr>
                            </thead>
                            <tbody>
                                <tr v-for="host in getHostsAffectedByThisAlert()" :key="host.id">
                                    <td>
                                        <div>
                                            <div style="font-weight: 500;">{{ host.display_name || host.name }}</div>
                                            <div style="font-size: 0.875rem; color: var(--text-muted);">{{ host.name }}</div>
                                        </div>
                                    </td>
                                    <td>
                                        <span style="font-family: monospace;">{{ host.ipv4 || host.hostname || 'N/A' }}</span>
                                    </td>
                                    <td>
                                        <span class="status-badge" :class="'status-' + host.alertSeverity">
                                            <div class="status-indicator" :class="'status-' + host.alertSeverity"></div>
                                            {{ host.alertSeverity.toUpperCase() }}
                                        </span>
                                    </td>
                                    <td>
                                        <div v-if="host.softFailInfo">
                                            <span class="soft-fail-indicator">
                                                {{ host.softFailInfo.current_fails }}/{{ host.softFailInfo.threshold_max }}
                                            </span>
                                            <div style="font-size: 0.8rem; color: var(--text-muted);">
                                                Since {{ formatTime(host.softFailInfo.first_fail_time) }}
                                            </div>
                                        </div>
                                        <span v-else style="color: var(--text-muted);">N/A</span>
                                    </td>
                                    <td>
                                        <div v-if="host.lastAlertTime">
                                            {{ formatTime(host.lastAlertTime) }}
                                            <div style="font-size: 0.8rem; color: var(--text-muted);">
                                                {{ formatDuration(calculateDuration(host.lastAlertTime)) }} ago
                                            </div>
                                        </div>
                                        <span v-else style="color: var(--text-muted);">Never</span>
                                    </td>
                                    <td>
                                        <button class="btn btn-secondary btn-small" 
                                                @click="$emit('view-host-detail', host.id)"
                                                title="View detailed host information">
                                            <i class="fas fa-eye"></i>
                                            <span class="btn-text">View Host</span>
                                        </button>
                                    </td>
                                </tr>
                            </tbody>
                        </table>
                    </div>
                </div>

                <!-- Recent Alert History -->
                <div class="data-table">
                    <div class="table-header">
                        <h3 class="table-title">Recent Alert History</h3>
                        <div class="search-box">
                            <span style="color: var(--text-muted); font-size: 0.875rem;">
                                Last 20 occurrences of this alert
                            </span>
                        </div>
                    </div>
                    <div class="table-content">
                        <div v-if="loading" class="loading">
                            <div class="spinner"></div>
                            Loading alert history...
                        </div>
                        
                        <div v-else-if="getRecentAlertHistory().length === 0" class="loading">
                            <i class="fas fa-info-circle" style="color: var(--primary-color); font-size: 2rem; margin-right: 1rem;"></i>
                            No alert history available
                        </div>
                        
                        <table v-else>
                            <thead>
                                <tr>
                                    <th>Timestamp</th>
                                    <th>Host</th>
                                    <th>Status</th>
                                    <th>Message</th>
                                    <th>Duration</th>
                                </tr>
                            </thead>
                            <tbody>
                                <tr v-for="status in getRecentAlertHistory()" :key="status.id">
                                    <td>{{ formatTime(status.timestamp) }}</td>
                                    <td>
                                        <strong>{{ status.host_name }}</strong>
                                        <div style="font-size: 0.875rem; color: var(--text-muted);">
                                            {{ status.host_id }}
                                        </div>
                                    </td>
                                    <td>
                                        <span class="status-badge" :class="'status-' + status.status">
                                            <div class="status-indicator" :class="'status-' + status.status"></div>
                                            {{ status.status.toUpperCase() }}
                                        </span>
                                    </td>
                                    <td style="max-width: 400px;">
                                        <div style="word-break: break-word;">{{ status.output || 'No output' }}</div>
                                    </td>
                                    <td>{{ status.duration || formatDuration(status.duration_since) }}</td>
                                </tr>
                            </tbody>
                        </table>
                    </div>
                </div>
            </div>
        </div>
    `
};
