// js/components/alerts-view.js
window.AlertsView = {
    props: {
        alerts: Array,
        loading: Boolean,
        alertMetrics: Object,
        alertFilter: String,
        filteredAlerts: Array
    },
    emits: ['update:alert-filter'],
    methods: {
        formatTime(timestamp) {
            return window.RavenUtils.formatTime(timestamp);
        },
        formatDuration(duration) {
            return window.RavenUtils.formatDuration(duration);
        }
    },
    template: `
        <div>
            <div class="metrics-grid" style="margin-bottom: 1rem;">
                <div class="metric-card">
                    <div class="metric-header">
                        <span class="metric-title">Active Alerts</span>
                        <div class="metric-icon critical">
                            <i class="fas fa-exclamation-triangle"></i>
                        </div>
                    </div>
                    <div class="metric-value">{{ alertMetrics.active }}</div>
                    <div class="metric-change">Requiring attention</div>
                </div>
                <div class="metric-card">
                    <div class="metric-header">
                        <span class="metric-title">Critical</span>
                        <div class="metric-icon critical">
                            <i class="fas fa-times-circle"></i>
                        </div>
                    </div>
                    <div class="metric-value">{{ alertMetrics.critical }}</div>
                    <div class="metric-change">High priority</div>
                </div>
                <div class="metric-card">
                    <div class="metric-header">
                        <span class="metric-title">Warnings</span>
                        <div class="metric-icon warning">
                            <i class="fas fa-exclamation-triangle"></i>
                        </div>
                    </div>
                    <div class="metric-value">{{ alertMetrics.warning }}</div>
                    <div class="metric-change">Medium priority</div>
                </div>
            </div>

            <div class="data-table">
                <div class="table-header">
                    <h3 class="table-title">Active Alerts</h3>
                    <div class="search-box">
                        <select :value="alertFilter" 
                                @change="$emit('update:alert-filter', $event.target.value)"
                                class="form-input" style="width: auto;">
                            <option value="all">All Alerts</option>
                            <option value="critical">Critical Only</option>
                            <option value="warning">Warning Only</option>
                            <option value="unknown">Unknown Only</option>
                        </select>
                    </div>
                </div>
                <div class="table-content">
                    <div v-if="loading" class="loading">
                        <div class="spinner"></div>
                        Loading alerts...
                    </div>
                    
                    <div v-else-if="alerts.length === 0" class="loading">
                        <i class="fas fa-check-circle" style="color: var(--success-color); font-size: 2rem; margin-right: 1rem;"></i>
                        No active alerts - all systems healthy!
                    </div>
                    
                    <table v-else>
                        <thead>
                            <tr>
                                <th>Time</th>
                                <th>Severity</th>
                                <th>Host</th>
                                <th>Check</th>
                                <th>Message</th>
                                <th>Duration</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr v-for="alert in filteredAlerts" :key="alert.id">
                                <td>{{ formatTime(alert.timestamp) }}</td>
                                <td>
                                    <span class="status-badge" :class="'status-' + alert.severity">
                                        <div class="status-indicator" :class="'status-' + alert.severity"></div>
                                        {{ alert.severity.toUpperCase() }}
                                    </span>
                                </td>
                                <td>{{ alert.host }}</td>
                                <td>{{ alert.check }}</td>
                                <td>{{ alert.message }}</td>
                                <td>{{ formatDuration(alert.duration) }}</td>
                            </tr>
                        </tbody>
                    </table>
                </div>
            </div>
        </div>
    `
};
