// js/components/dashboard-view.js - Enhanced with soft fail and OK duration info
window.DashboardView = {
    props: {
        stats: Object,
        recentActivity: Array
    },
    methods: {
        formatTime(timestamp) {
            return window.RavenUtils.formatTime(timestamp);
        },
        formatSoftFailStatus(softFailInfo) {
            return window.RavenUtils.formatSoftFailStatus(softFailInfo);
        },
        formatOKDuration(okInfo) {
            return window.RavenUtils.formatOKDuration(okInfo);
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

            <!-- Recent Activity with Enhanced Status Info -->
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
                                <td>{{ activity.host_name || activity.host }}</td>
                                <td>{{ activity.check_name || activity.check }}</td>
                                <td>
                                    <div class="enhanced-status-badge">
                                        <div class="status-main">
                                            <span class="status-badge" :class="'status-' + activity.status">
                                                <div class="status-indicator" :class="'status-' + activity.status"></div>
                                                {{ activity.status.toUpperCase() }}
                                                <span v-if="activity.soft_fails_info && activity.status !== 'ok'" class="soft-fail-indicator">
                                                    ({{ activity.soft_fails_info.current_fails }}/{{ activity.soft_fails_info.threshold_max }})
                                                </span>
                                            </span>
                                        </div>
                                        <div v-if="activity.ok_info && activity.status === 'ok'" class="ok-duration">
                                            <i class="fas fa-clock"></i>
                                            OK since {{ formatTime(activity.ok_info.ok_since) }}
                                        </div>
                                    </div>
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
