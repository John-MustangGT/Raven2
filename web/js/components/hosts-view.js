// FIXED web/js/components/hosts-view.js

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
            if (!lastChecked) return 'Never checked';
            const status = ipOK ? 'OK' : 'Failed';
            const time = this.formatTime(lastChecked);
            return `${status} (${time})`;
        },
        getIPCheckClass(ipOK) {
            return ipOK ? 'ip-check-ok' : 'ip-check-fail';
        },
        
        // FIXED: Get check name with better fallback logic
        getCheckNameWithFallback(checkId, host) {
            // Try multiple sources for check name
            if (host.check_names && host.check_names[checkId]) {
                return host.check_names[checkId];
            }
            
            // Try from soft fail info
            if (host.soft_fail_info && host.soft_fail_info[checkId] && host.soft_fail_info[checkId].check_name) {
                return host.soft_fail_info[checkId].check_name;
            }
            
            // Try from OK duration info  
            if (host.ok_duration && host.ok_duration[checkId] && host.ok_duration[checkId].check_name) {
                return host.ok_duration[checkId].check_name;
            }
            
            // Fallback to formatted check ID
            return checkId || 'Unknown Check';
        },
        
        // FIXED: Format soft fails with better error handling and debugging
        formatSoftFailsForDisplay(softFailInfo, host) {
            console.log('Formatting soft fails for host:', host.name, 'softFailInfo:', softFailInfo);
            
            if (!softFailInfo || typeof softFailInfo !== 'object') {
                console.log('No soft fail info or invalid format');
                return null;
            }
            
            const keys = Object.keys(softFailInfo);
            if (keys.length === 0) {
                console.log('Soft fail info is empty');
                return null;
            }

            const results = [];
            for (const [checkId, failInfo] of Object.entries(softFailInfo)) {
                if (!failInfo || typeof failInfo !== 'object') {
                    console.log('Invalid fail info for check:', checkId);
                    continue;
                }
                
                // Get check name with multiple fallback options
                let checkName = 'Unknown Check';
                if (failInfo.check_name) {
                    checkName = failInfo.check_name;
                } else {
                    checkName = this.getCheckNameWithFallback(checkId, host);
                }
                
                const result = {
                    checkId: checkId,
                    checkName: checkName,
                    currentFails: failInfo.current_fails || 0,
                    thresholdMax: failInfo.threshold_max || 3,
                    firstFailTime: failInfo.first_fail_time,
                    lastFailTime: failInfo.last_fail_time
                };
                
                console.log('Adding soft fail result:', result);
                results.push(result);
            }
            
            console.log('Final soft fail results:', results);
            return results.length > 0 ? results : null;
        },
        
        // FIXED: Format OK duration with better error handling
        formatOKDurationForDisplay(okDuration, host) {
            console.log('Formatting OK duration for host:', host.name, 'okDuration:', okDuration);
            
            if (!okDuration || typeof okDuration !== 'object') {
                console.log('No OK duration info or invalid format');
                return null;
            }
            
            const keys = Object.keys(okDuration);
            if (keys.length === 0) {
                console.log('OK duration info is empty');
                return null;
            }

            const results = [];
            for (const [checkId, okInfo] of Object.entries(okDuration)) {
                if (!okInfo || typeof okInfo !== 'object') {
                    console.log('Invalid OK info for check:', checkId);
                    continue;
                }
                
                // Get check name with multiple fallback options
                let checkName = 'Unknown Check';
                if (okInfo.check_name) {
                    checkName = okInfo.check_name;
                } else {
                    checkName = this.getCheckNameWithFallback(checkId, host);
                }
                
                const result = {
                    checkId: checkId,
                    checkName: checkName,
                    okSince: okInfo.ok_since,
                    duration: okInfo.duration || 'Unknown',
                    checkCount: okInfo.check_count || 0
                };
                
                console.log('Adding OK duration result:', result);
                results.push(result);
            }
            
            console.log('Final OK duration results:', results);
            return results.length > 0 ? results : null;
        },
        
        // HELPER: Check if host has any monitoring data
        hasMonitoringData(host) {
            const hasSoftFails = host.soft_fail_info && Object.keys(host.soft_fail_info).length > 0;
            const hasOKDuration = host.ok_duration && Object.keys(host.ok_duration).length > 0;
            return hasSoftFails || hasOKDuration;
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
                            <th>Status & Monitoring Details</th>
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
                                    <!-- Main Status Badge -->
                                    <div class="status-main">
                                        <span class="status-badge" :class="'status-' + host.status">
                                            <div class="status-indicator" :class="'status-' + host.status"></div>
                                            {{ host.status.toUpperCase() }}
                                        </span>
                                    </div>
                                    
                                    <!-- FIXED: Enhanced Monitoring Results Display -->
                                    <div v-if="hasMonitoringData(host)" class="monitoring-results">
                                        
                                        <!-- Failing Tests Section -->
                                        <div v-if="formatSoftFailsForDisplay(host.soft_fail_info, host)" class="monitoring-section">
                                            <div class="monitoring-header">
                                                <i class="fas fa-exclamation-triangle" style="color: var(--warning-color);"></i>
                                                <strong>Failing Tests ({{ formatSoftFailsForDisplay(host.soft_fail_info, host).length }}):</strong>
                                            </div>
                                            <div class="monitoring-items">
                                                <div v-for="failInfo in formatSoftFailsForDisplay(host.soft_fail_info, host)" 
                                                     :key="'fail-' + failInfo.checkId" 
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
                                        
                                        <!-- Healthy Tests Section -->
                                        <div v-if="formatOKDurationForDisplay(host.ok_duration, host)" class="monitoring-section">
                                            <div class="monitoring-header">
                                                <i class="fas fa-check-circle" style="color: var(--success-color);"></i>
                                                <strong>Healthy Tests ({{ formatOKDurationForDisplay(host.ok_duration, host).length }}):</strong>
                                            </div>
                                            <div class="monitoring-items">
                                                <div v-for="okInfo in formatOKDurationForDisplay(host.ok_duration, host)" 
                                                     :key="'ok-' + okInfo.checkId" 
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
                                    
                                    <!-- DEBUG: Show raw data in development -->
                                    <!-- 
                                    <details style="margin-top: 0.5rem; font-size: 0.8rem;">
                                        <summary>Debug Data</summary>
                                        <pre>{{ JSON.stringify({
                                            soft_fail_info: host.soft_fail_info,
                                            ok_duration: host.ok_duration,
                                            check_names: host.check_names
                                        }, null, 2) }}</pre>
                                    </details>
                                    -->
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
