// js/components/dashboard-view.js
window.DashboardView = {
    props: {
        stats: Object,
        recentActivity: Array
    },
    methods: {
        formatTime(timestamp) {
            return window.RavenUtils.formatTime(timestamp);
        }
    },
    template: `
        <div>
            <div class="metrics-grid">
                <div class="metric-card">
                    <div class="metric-header">
                        <span class="metric-title">Healthy</span>
                        <div class="metric-icon ok">
                            <i class="fas fa-check"></i>
                        </div>
                    </div>
                    <div class="metric-value">{{ stats.ok }}</div>
                    <div class="metric-change">Hosts online</div>
                </div>
                <div class="metric-card">
                    <div class="metric-header">
                        <span class="metric-title">Warning</span>
                        <div class="metric-icon warning">
                            <i class="fas fa-exclamation-triangle"></i>
                        </div>
                    </div>
                    <div class="metric-value">{{ stats.warning }}</div>
                    <div class="metric-change">Need attention</div>
                </div>
                <div class="metric-card">
                    <div class="metric-header">
                        <span class="metric-title">Critical</span>
                        <div class="metric-icon critical">
                            <i class="fas fa-times-circle"></i>
                        </div>
                    </div>
                    <div class="metric-value">{{ stats.critical }}</div>
                    <div class="metric-change">Require action</div>
                </div>
                <div class="metric-card">
                    <div class="metric-header">
                        <span class="metric-title">Unknown</span>
                        <div class="metric-icon unknown">
                            <i class="fas fa-question-circle"></i>
                        </div>
                    </div>
                    <div class="metric-value">{{ stats.unknown }}</div>
                    <div class="metric-change">Status unclear</div>
                </div>
            </div>

            <!-- Recent Activity -->
            <div class="data-table">
                <div class="table-header">
                    <h3 class="table-title">Recent Activity</h3>
                </div>
                <div class="table-content">
                    <table>
                        <thead>
                            <tr>
                                <th>Time</th>
                                <th>Host</th>
                                <th>Check</th>
                                <th>Status</th>
                                <th>Message</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr v-for="activity in recentActivity" :key="activity.id">
                                <td>{{ formatTime(activity.timestamp) }}</td>
                                <td>{{ activity.host }}</td>
                                <td>{{ activity.check }}</td>
                                <td>
                                    <span class="status-badge" :class="'status-' + activity.status">
                                        <div class="status-indicator" :class="'status-' + activity.status"></div>
                                        {{ activity.status }}
                                    </span>
                                </td>
                                <td>{{ activity.message }}</td>
                            </tr>
                        </tbody>
                    </table>
                </div>
            </div>
        </div>
    `
};
