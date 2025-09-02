// js/components/host-detail-view.js - Detailed host view with all monitoring data
window.HostDetailView = {
    props: {
        host: Object,
        hostStatuses: Array,
        loading: Boolean,
        loadingStatuses: Boolean
    },
    emits: ['back-to-hosts', 'refresh-host-data'],
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
        getIPCheckClass(ipOK) {
            return window.RavenUtils.getIPCheckClass(ipOK);
        },
        formatIPStatus(ipOK, lastChecked) {
            if (!lastChecked) return 'Never checked';
            const status = ipOK ? 'OK' : 'Failed';
            const time = this.formatTime(lastChecked);
            return `${status} (${time})`;
        },
        getCheckNameWithFallback(checkId) {
            if (this.host && this.host.check_names && this.host.check_names[checkId]) {
                return this.host.check_names[checkId];
            }
            return checkId || 'Unknown Check';
        },
        formatSoftFailsForDisplay(softFailInfo) {
            if (!softFailInfo || Object.keys(softFailInfo).length === 0) return null;
            
            const results = [];
            for (const [checkId, failInfo] of Object.entries(softFailInfo)) {
                if (!failInfo) continue;
                
                const checkName = failInfo.check_name || this.getCheckNameWithFallback(checkId);
                results.push({
                    checkId,
                    checkName,
                    currentFails: failInfo.current_fails || 0,
                    thresholdMax: failInfo.threshold_max || 3,
                    firstFailTime: failInfo.first_fail_time,
                    lastFailTime: failInfo.last_fail_time
                });
            }
            
            return results.length > 0 ? results : null;
        },
        formatOKDurationForDisplay(okDuration) {
            if (!okDuration || Object.keys(okDuration).length === 0) return null;
            
            const results = [];
            for (const [checkId, okInfo] of Object.entries(okDuration)) {
                if (!okInfo) continue;
                
                const checkName = okInfo.check_name || this.getCheckNameWithFallback(checkId);
                results.push({
                    checkId,
                    checkName,
                    okSince: okInfo.ok_since,
                    duration: okInfo.duration || 'Unknown',
                    checkCount: okInfo.check_count || 0
                });
            }
            
            return results.length > 0 ? results : null;
        },
        getStatusHistory() {
            if (!this.hostStatuses || this.hostStatuses.length === 0) return [];
            
            return this.hostStatuses.slice(0, 20).map(status => ({
                id: status.id,
                timestamp: status.timestamp,
                checkName: status.check_name || this.getCheckNameWithFallback(status.check_id),
                exitCode: status.exit_code,
                status: this.getStatusName(status.exit_code),
                output: status.output || 'No output',
                duration: status.duration
            }));
        },
        getRecentChecks() {
            if (!this.hostStatuses) return [];
            
            // Group by check and get the most recent status for each
            const checkMap = new Map();
            
            this.hostStatuses.forEach(status => {
                const checkId = status.check_id;
                if (!checkMap.has(checkId) || 
                    new Date(status.timestamp) > new Date(checkMap.get(checkId).timestamp)) {
                    checkMap.set(checkId, {
                        ...status,
                        checkName: status.check_name || this.getCheckNameWithFallback(checkId)
                    });
                }
            });
            
            return Array.from(checkMap.values())
                .sort((a, b) => new Date(b.timestamp) - new Date(a.timestamp));
        }
    },
    template: `
        <div class="host-detail-view">
            <!-- Header with Back Button -->
            <div class="detail-header">
                <button class="btn btn-secondary" @click="$emit('back-to-hosts')">
                    <i class="fas fa-arrow-left"></i>
                    <span>Back to Hosts</span>
                </button>
                <div class="detail-header-info">
                    <h1 class="detail-title" v-if="host">
                        {{ host.display_name || host.name }}
                        <span v-if="host.display_name" class="detail-subtitle">{{ host.name }}</span>
                    </h1>
                    <div class="detail-actions">
                        <button class="btn btn-secondary" @click="$emit('refresh-host-data')" :disabled="loading">
                            <i class="fas fa-sync" :class="{ 'fa-spin': loading }"></i>
                            <span>Refresh</span>
                        </button>
                    </div>
                </div>
            </div>

            <div v-if="loading" class="loading">
                <div class="spinner"></div>
                Loading host details...
            </div>

            <div v-else-if="!host" class="loading">
                <i class="fas fa-exclamation-triangle" style="color: var(--warning-color); font-size: 2rem; margin-right: 1rem;"></i>
                Host not found
            </div>

            <div v-else class="host-detail-content">
                <!-- Host Overview Cards -->
                <div class="metrics-grid" style="margin-bottom: 2rem;">
                    <div class="metric-card">
                        <div class="metric-header">
                            <span class="metric-title">Current Status</span>
                            <div class="metric-icon" :class="host.status">
                                <i class="fas" :class="{
                                    'fa-check': host.status === 'ok',
                                    'fa-exclamation-triangle': host.status === 'warning',
                                    'fa-times-circle': host.status === 'critical',
                                    'fa-question-circle': host.status === 'unknown'
                                }"></i>
                            </div>
                        </div>
                        <div class="metric-value">{{ host.status.toUpperCase() }}</div>
                        <div class="metric-change">Last checked: {{ formatTime(host.last_check) }}</div>
                    </div>
                    
                    <div class="metric-card">
                        <div class="metric-header">
                            <span class="metric-title">IP Connectivity</span>
                            <div class="metric-icon" :class="host.ip_address_ok ? 'ok' : 'critical'">
                                <i class="fas" :class="host.ip_address_ok ? 'fa-wifi' : 'fa-wifi-off'"></i>
                            </div>
                        </div>
                        <div class="metric-value">{{ host.ip_address_ok ? 'OK' : 'FAILED' }}</div>
                        <div class="metric-change">{{ formatIPStatus(host.ip_address_ok, host.ip_last_checked) }}</div>
                    </div>
                    
                    <div class="metric-card" v-if="formatSoftFailsForDisplay(host.soft_fail_info)">
                        <div class="metric-header">
                            <span class="metric-title">Failing Tests</span>
                            <div class="metric-icon warning">
                                <i class="fas fa-exclamation-triangle"></i>
                            </div>
                        </div>
                        <div class="metric-value">{{ formatSoftFailsForDisplay(host.soft_fail_info).length }}</div>
                        <div class="metric-change">Tests approaching failure threshold</div>
                    </div>
                    
                    <div class="metric-card" v-if="formatOKDurationForDisplay(host.ok_duration)">
                        <div class="metric-header">
                            <span class="metric-title">Healthy Tests</span>
                            <div class="metric-icon ok">
                                <i class="fas fa-check-circle"></i>
                            </div>
                        </div>
                        <div class="metric-value">{{ formatOKDurationForDisplay(host.ok_duration).length }}</div>
                        <div class="metric-change">Tests running successfully</div>
                    </div>
                </div>

                <!-- Host Information -->
                <div class="data-table" style="margin-bottom: 2rem;">
                    <div class="table-header">
                        <h3 class="table-title">Host Information</h3>
                    </div>
                    <div class="table-content">
                        <table>
                            <tbody>
                                <tr>
                                    <td style="font-weight: 600; width: 200px;">Name</td>
                                    <td>{{ host.name }}</td>
                                </tr>
                                <tr v-if="host.display_name">
                                    <td style="font-weight: 600;">Display Name</td>
                                    <td>{{ host.display_name }}</td>
                                </tr>
                                <tr>
                                    <td style="font-weight: 600;">IP Address</td>
                                    <td>
                                        <span style="font-family: monospace;">{{ host.ipv4 || 'N/A' }}</span>
                                        <span v-if="host.ipv4" 
                                              class="ip-check-indicator" 
                                              :class="getIPCheckClass(host.ip_address_ok)"
                                              style="margin-left: 0.5rem;">
                                            <i :class="host.ip_address_ok ? 'fas fa-check' : 'fas fa-times'"></i>
                                            {{ host.ip_address_ok ? 'Reachable' : 'Unreachable' }}
                                        </span>
                                    </td>
                                </tr>
                                <tr v-if="host.hostname">
                                    <td style="font-weight: 600;">Hostname</td>
                                    <td style="font-family: monospace;">{{ host.hostname }}</td>
                                </tr>
                                <tr>
                                    <td style="font-weight: 600;">Group</td>
                                    <td>{{ host.group || 'default' }}</td>
                                </tr>
                                <tr>
                                    <td style="font-weight: 600;">Status</td>
                                    <td>
                                        <span v-if="host.enabled" class="status-badge status-ok">
                                            <div class="status-indicator status-ok"></div>
                                            Enabled
                                        </span>
                                        <span v-else class="status-badge status-unknown">
                                            <div class="status-indicator status-unknown"></div>
                                            Disabled
                                        </span>
                                    </td>
                                </tr>
                                <tr>
                                    <td style="font-weight: 600;">Created</td>
                                    <td>{{ formatTime(host.created_at) }}</td>
                                </tr>
                                <tr>
                                    <td style="font-weight: 600;">Last Updated</td>
                                    <td>{{ formatTime(host.updated_at) }}</td>
                                </tr>
                            </tbody>
                        </table>
                    </div>
                </div>

                <!-- Failing Tests Details -->
                <div v-if="formatSoftFailsForDisplay(host.soft_fail_info)" class="data-table" style="margin-bottom: 2rem;">
                    <div class="table-header">
                        <h3 class="table-title">
                            <i class="fas fa-exclamation-triangle" style="color: var(--warning-color);"></i>
                            Failing Tests ({{ formatSoftFailsForDisplay(host.soft_fail_info).length }})
                        </h3>
                    </div>
                    <div class="table-content">
                        <table>
                            <thead>
                                <tr>
                                    <th>Check Name</th>
                                    <th>Failures</th>
                                    <th>Threshold</th>
                                    <th>First Failure</th>
                                    <th>Last Failure</th>
                                    <th>Status</th>
                                </tr>
                            </thead>
                            <tbody>
                                <tr v-for="failInfo in formatSoftFailsForDisplay(host.soft_fail_info)" :key="failInfo.checkId">
                                    <td>
                                        <div style="display: flex; align-items: center; gap: 0.5rem;">
                                            <i class="fas fa-vial" style="color: var(--warning-color);"></i>
                                            <strong>{{ failInfo.checkName }}</strong>
                                        </div>
                                    </td>
                                    <td>
                                        <span class="soft-fail-indicator">
                                            {{ failInfo.currentFails }}/{{ failInfo.thresholdMax }}
                                        </span>
                                    </td>
                                    <td>{{ failInfo.thresholdMax }} failures</td>
                                    <td>{{ formatTime(failInfo.firstFailTime) }}</td>
                                    <td>{{ formatTime(failInfo.lastFailTime) }}</td>
                                    <td>
                                        <span v-if="failInfo.currentFails >= failInfo.thresholdMax" class="status-badge status-critical">
                                            <div class="status-indicator status-critical"></div>
                                            HARD FAIL
                                        </span>
                                        <span v-else class="status-badge status-warning">
                                            <div class="status-indicator status-warning"></div>
                                            SOFT FAIL
                                        </span>
                                    </td>
                                </tr>
                            </tbody>
                        </table>
                    </div>
                </div>

                <!-- Healthy Tests Details -->
                <div v-if="formatOKDurationForDisplay(host.ok_duration)" class="data-table" style="margin-bottom: 2rem;">
                    <div class="table-header">
                        <h3 class="table-title">
                            <i class="fas fa-check-circle" style="color: var(--success-color);"></i>
                            Healthy Tests ({{ formatOKDurationForDisplay(host.ok_duration).length }})
                        </h3>
                    </div>
                    <div class="table-content">
                        <table>
                            <thead>
                                <tr>
                                    <th>Check Name</th>
                                    <th>OK Duration</th>
                                    <th>Check Count</th>
                                    <th>OK Since</th>
                                    <th>Status</th>
                                </tr>
                            </thead>
                            <tbody>
                                <tr v-for="okInfo in formatOKDurationForDisplay(host.ok_duration)" :key="okInfo.checkId">
                                    <td>
                                        <div style="display: flex; align-items: center; gap: 0.5rem;">
                                            <i class="fas fa-vial" style="color: var(--success-color);"></i>
                                            <strong>{{ okInfo.checkName }}</strong>
                                        </div>
                                    </td>
                                    <td>
                                        <span class="ok-duration">
                                            {{ okInfo.duration }}
                                        </span>
                                    </td>
                                    <td>{{ okInfo.checkCount }} checks</td>
                                    <td>{{ formatTime(okInfo.okSince) }}</td>
                                    <td>
                                        <span class="status-badge status-ok">
                                            <div class="status-indicator status-ok"></div>
                                            HEALTHY
                                        </span>
                                    </td>
                                </tr>
                            </tbody>
                        </table>
                    </div>
                </div>

                <!-- Recent Check Results -->
                <div class="data-table" style="margin-bottom: 2rem;">
                    <div class="table-header">
                        <h3 class="table-title">Recent Check Results</h3>
                        <div class="search-box">
                            <span style="color: var(--text-muted); font-size: 0.875rem;">
                                Last {{ getRecentChecks().length }} checks
                            </span>
                        </div>
                    </div>
                    <div class="table-content">
                        <div v-if="loadingStatuses" class="loading">
                            <div class="spinner"></div>
                            Loading check history...
                        </div>
                        
                        <div v-else-if="getRecentChecks().length === 0" class="loading">
                            <i class="fas fa-info-circle" style="color: var(--primary-color); font-size: 2rem; margin-right: 1rem;"></i>
                            No check results available
                        </div>
                        
                        <table v-else>
                            <thead>
                                <tr>
                                    <th>Check</th>
                                    <th>Status</th>
                                    <th>Last Run</th>
                                    <th>Output</th>
                                    <th>Duration</th>
                                </tr>
                            </thead>
                            <tbody>
                                <tr v-for="status in getRecentChecks()" :key="status.id">
                                    <td>
                                        <strong>{{ status.checkName }}</strong>
                                        <div style="font-size: 0.875rem; color: var(--text-muted);">{{ status.check_id }}</div>
                                    </td>
                                    <td>
                                        <span class="status-badge" :class="'status-' + getStatusName(status.exit_code)">
                                            <div class="status-indicator" :class="'status-' + getStatusName(status.exit_code)"></div>
                                            {{ getStatusName(status.exit_code).toUpperCase() }}
                                        </span>
                                    </td>
                                    <td>{{ formatTime(status.timestamp) }}</td>
                                    <td style="max-width: 300px;">
                                        <div style="word-break: break-word;">{{ status.output || 'No output' }}</div>
                                    </td>
                                    <td>{{ status.duration || 'N/A' }}</td>
                                </tr>
                            </tbody>
                        </table>
                    </div>
                </div>

                <!-- Full Status History -->
                <div class="data-table">
                    <div class="table-header">
                        <h3 class="table-title">Status History</h3>
                        <div class="search-box">
                            <span style="color: var(--text-muted); font-size: 0.875rem;">
                                Last 20 status entries
                            </span>
                        </div>
                    </div>
                    <div class="table-content">
                        <div v-if="loadingStatuses" class="loading">
                            <div class="spinner"></div>
                            Loading status history...
                        </div>
                        
                        <div v-else-if="getStatusHistory().length === 0" class="loading">
                            <i class="fas fa-info-circle" style="color: var(--primary-color); font-size: 2rem; margin-right: 1rem;"></i>
                            No status history available
                        </div>
                        
                        <table v-else>
                            <thead>
                                <tr>
                                    <th>Timestamp</th>
                                    <th>Check</th>
                                    <th>Status</th>
                                    <th>Output</th>
                                </tr>
                            </thead>
                            <tbody>
                                <tr v-for="status in getStatusHistory()" :key="status.id">
                                    <td>{{ formatTime(status.timestamp) }}</td>
                                    <td>
                                        <strong>{{ status.checkName }}</strong>
                                    </td>
                                    <td>
                                        <span class="status-badge" :class="'status-' + status.status">
                                            <div class="status-indicator" :class="'status-' + status.status"></div>
                                            {{ status.status.toUpperCase() }}
                                        </span>
                                    </td>
                                    <td style="max-width: 400px;">
                                        <div style="word-break: break-word;">{{ status.output }}</div>
                                    </td>
                                </tr>
                            </tbody>
                        </table>
                    </div>
                </div>
            </div>
        </div>
    `
};
