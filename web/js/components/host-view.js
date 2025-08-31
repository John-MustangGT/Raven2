// js/components/hosts-view.js - Enhanced with IP checks and soft fail info
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
                            <th>Status</th>
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
                                        
                                        <!-- Show soft fail indicators for failing checks -->
                                        <div v-if="host.soft_fail_info && Object.keys(host.soft_fail_info).length > 0" 
                                             class="soft-fail-indicator">
                                            <i class="fas fa-exclamation-triangle"></i>
                                            <span v-for="(failInfo, checkId) in host.soft_fail_info" :key="checkId" 
                                                  :title="'Check failing: ' + failInfo.current_fails + '/' + failInfo.threshold_max + ' failures since ' + formatTime(failInfo.first_fail_time)">
                                                {{ failInfo.current_fails }}/{{ failInfo.threshold_max }}
                                            </span>
                                        </div>
                                    </div>
                                    
                                    <!-- Show OK duration for healthy hosts -->
                                    <div v-if="host.ok_duration && Object.keys(host.ok_duration).length > 0" class="status-meta">
                                        <div v-for="(okInfo, checkId) in host.ok_duration" :key="checkId" class="ok-duration">
                                            <i class="fas fa-clock"></i>
                                            OK since {{ formatTime(okInfo.ok_since) }} ({{ okInfo.duration }})
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
