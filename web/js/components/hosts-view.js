// js/components/hosts-view.js - Enhanced with check names display (CORRECTED VERSION)
window.HostsView = {
    props: {
        hosts: Array,
        loading: Boolean,
        searchQuery: String,
        filterGroup: String,
        groups: Array,
        filteredHosts: Array
    },
    emits: ['update:search-query', 'update:filter-group', 'edit-host', 'delete-host'],
    methods: {
        formatTime(timestamp) {
            return window.RavenUtils.formatTime(timestamp);
        },
        formatIPStatus(ipOK, lastChecked) {
            return window.RavenUtils.formatIPStatus(ipOK, lastChecked);
        },
        formatSoftFailStatus(softFailInfo) {
            return window.RavenUtils.formatSoftFailStatus(softFailInfo);
        },
        formatOKDuration(okInfo) {
            return window.RavenUtils.formatOKDuration(okInfo);
        },
        getIPCheckClass(ipOK) {
            return window.RavenUtils.getIPCheckClass(ipOK);
        },
        // NEW: Format check names for display
        getCheckNameWithFallback(checkId, host) {
            // Try to get the check name from the host's check names mapping
            if (host.check_names && host.check_names[checkId]) {
                return host.check_names[checkId];
            }
            // Fallback to check ID if name not available
            return checkId;
        },
        // NEW: Format soft fail display with check names
        formatSoftFailsForDisplay(softFailInfo, host) {
            if (!softFailInfo || Object.keys(softFailInfo).length === 0) {
                return null;
            }

            const results = [];
            for (const [checkId, failInfo] of Object.entries(softFailInfo)) {
                const checkName = failInfo.check_name || this.getCheckNameWithFallback(checkId, host);
                results.push({
                    checkId: checkId,
                    checkName: checkName,
                    currentFails: failInfo.current_fails,
                    thresholdMax: failInfo.threshold_max,
                    firstFailTime: failInfo.first_fail_time,
                    lastFailTime: failInfo.last_fail_time
                });
            }
            
            return results;
        },
        // NEW: Format OK duration display with check names
        formatOKDurationForDisplay(okDuration, host) {
            if (!okDuration || Object.keys(okDuration).length === 0) {
                return null;
            }

            const results = [];
            for (const [checkId, okInfo] of Object.entries(okDuration)) {
                const checkName = okInfo.check_name || this.getCheckNameWithFallback(checkId, host);
                results.push({
                    checkId: checkId,
                    checkName: checkName,
                    okSince: okInfo.ok_since,
                    duration: okInfo.duration,
                    checkCount: okInfo.check_count
                });
            }
            
            return results;
        }
    },
    template: `
        <div class="data-table">
            <div class="table-header">
                <h3 class="table-title">Hosts ({{ hosts.length }})</h3>
                <div class="search-box">
                    <input 
                        :value="searchQuery"
                        @input="$emit('update:search-query', $event.target.value)"
                        class="search-input" 
                        placeholder="Search hosts..."
                        type="text"
                    >
                    <select :value="filterGroup" 
                            @change="$emit('update:filter-group', $event.target.value)"
                            class="form-input" style="width: auto;">
                        <option value="">All Groups</option>
                        <option v-for="group in groups" :key="group" :value="group">{{ group }}</option>
                    </select>
                </div>
            </div>
            <div class="table-content">
                <div v-if="loading" class="loading">
                    <div class="spinner"></div>
                    Loading hosts...
                </div>
                <table v-else>
                    <thead>
                        <tr>
                            <th>Name</th>
                            <th>Address</th>
                            <th>Group</th>
                            <th>Status & Recent Monitoring</th>
                            <th>Last Check</th>
                            <th>Actions</th>
                        </tr>
                    </thead>
                    <tbody>
                        <tr v-for="(host, index) in filteredHosts" :key="host.id + '-' + index">
                            <td>
                                <div class="host-details">
                                    <div style="font-weight: 500;">{{ host.display_name || host.name }}</div>
                                    <div style="font-size: 0.875rem; color: var(--text-muted);">{{ host.name }}</div>
                                </div>
                            </td>
                            <td>
                                <div class="host-address">
                                    <div class="host-address-main">{{ host.ipv4 || host.hostname || 'N/A' }}</div>
                                    <div v-if="host.ipv4" 
                                         class="ip-check-indicator" 
                                         :class="getIPCheckClass(host.ip_address_ok)" 
                                         :title="formatIPStatus(host.ip_address_ok, host.ip_last_checked)">
                                        <i :class="host.ip_address_ok ? 'fas fa-check' : 'fas fa-times'"></i>
                                        {{ host.ip_address_ok ? 'IP OK' : 'IP Fail' }}
                                    </div>
                                </div>
                                <div v-if="host.ipv4 && host.hostname" class="host-address-alt">
                                    {{ host.hostname }}
                                </div>
                            </td>
                            <td>{{ host.group }}</td>
                            <td>
                                <div class="status-details">
                                    <div class="status-main">
                                        <span class="status-badge" :class="'status-' + host.status">
                                            <div class="status-indicator" :class="'status-' + host.status"></div>
                                            {{ host.status.toUpperCase() }}
                                        </span>
                                    </div>
                                    
                                    <!-- Enhanced: Show soft fail indicators with CHECK NAMES -->
                                    <div v-if="formatSoftFailsForDisplay(host.soft_fail_info, host)" class="monitoring-results">
                                        <div class="monitoring-section">
                                            <div class="monitoring-header">
                                                <i class="fas fa-exclamation-triangle" style="color: var(--warning-color);"></i>
                                                <strong>Failing Tests:</strong>
                                            </div>
                                            <div class="monitoring-items">
                                                <div v-for="failInfo in formatSoftFailsForDisplay(host.soft_fail_info, host)" 
                                                     :key="failInfo.checkId" 
                                                     class="monitoring-item soft-fail-item">
                                                    <div class="monitoring-test-name">
                                                        <i class="fas fa-vial" style="color: var(--warning-color);"></i>
                                                        <strong>{{ failInfo.checkName }}</strong>
                                                    </div>
                                                    <div class="monitoring-details">
                                                        <span class="soft-fail-indicator">
                                                            {{ failInfo.currentFails }}/{{ failInfo.thresholdMax }} failures
                                                        </span>
                                                        <span class="monitoring-time" :title="'First failure: ' + formatTime(failInfo.firstFailTime)">
                                                            since {{ formatTime(failInfo.firstFailTime) }}
                                                        </span>
                                                    </div>
                                                </div>
                                            </div>
                                        </div>
                                    </div>
                                    
                                    <!-- Enhanced: Show OK duration with CHECK NAMES -->
                                    <div v-if="formatOKDurationForDisplay(host.ok_duration, host)" class="monitoring-results">
                                        <div class="monitoring-section">
                                            <div class="monitoring-header">
                                                <i class="fas fa-check-circle" style="color: var(--success-color);"></i>
                                                <strong>Healthy Tests:</strong>
                                            </div>
                                            <div class="monitoring-items">
                                                <div v-for="okInfo in formatOKDurationForDisplay(host.ok_duration, host)" 
                                                     :key="okInfo.checkId" 
                                                     class="monitoring-item ok-item">
                                                    <div class="monitoring-test-name">
                                                        <i class="fas fa-vial" style="color: var(--success-color);"></i>
                                                        <strong>{{ okInfo.checkName }}</strong>
                                                    </div>
                                                    <div class="monitoring-details">
                                                        <span class="ok-duration">
                                                            OK for {{ okInfo.duration }}
                                                        </span>
                                                        <span class="monitoring-time" :title="'OK since: ' + formatTime(okInfo.okSince)">
                                                            ({{ okInfo.checkCount }} checks)
                                                        </span>
                                                    </div>
                                                </div>
                                            </div>
                                        </div>
                                    </div>
                                </div>
                            </td>
                            <td>{{ formatTime(host.last_check) }}</td>
                            <td>
                                <div class="actions">
                                    <button class="btn btn-secondary btn-small" @click="$emit('edit-host', host)">
                                        <i class="fas fa-edit"></i>
                                    </button>
                                    <button class="btn btn-danger btn-small" @click="$emit('delete-host', host)">
                                        <i class="fas fa-trash"></i>
                                    </button>
                                </div>
                            </td>
                        </tr>
                    </tbody>
                </table>
            </div>
        </div>
    `
};
