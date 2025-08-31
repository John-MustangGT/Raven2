// js/components/about-view.js
window.AboutView = {
    props: {
        buildInfo: Object,
        loadingBuildInfo: Boolean,
        webConfig: Object
    },
    methods: {
        formatBuildTime(buildTime) {
            return window.RavenUtils.formatBuildTime(buildTime);
        }
    },
    template: `
        <div class="data-table">
            <div class="table-header">
                <h3 class="table-title">About Raven</h3>
            </div>
            <div style="padding: 2rem;">
                <div v-if="loadingBuildInfo" class="loading">
                    <div class="spinner"></div>
                    Loading build information...
                </div>
                
                <div v-else>
                    <div style="text-align: center; margin-bottom: 2rem;">
                        <div style="font-size: 3rem; color: var(--primary-color); margin-bottom: 1rem;">
                            <i class="fas fa-crow"></i>
                        </div>
                        <h2 style="margin-bottom: 0.5rem;">Raven Network Monitoring</h2>
                        <p style="color: var(--text-muted); font-size: 1.125rem;">
                            Advanced network monitoring and alerting system
                        </p>
                        <p style="color: var(--text-muted); font-size: 0.875rem; margin-top: 1rem;">
                            Visit: <a :href="webConfig.header_link" target="_blank" rel="noopener noreferrer" style="color: var(--primary-color);">{{ webConfig.header_link }}</a>
                        </p>
                    </div>

                    <div class="metrics-grid" style="margin-bottom: 2rem;">
                        <div class="metric-card">
                            <div class="metric-header">
                                <span class="metric-title">Version</span>
                                <div class="metric-icon" style="background: #e0f2fe; color: #0277bd;">
                                    <i class="fas fa-tag"></i>
                                </div>
                            </div>
                            <div class="metric-value" style="font-size: 1.5rem;">{{ buildInfo.version }}</div>
                            <div class="metric-change">Current release</div>
                        </div>
                        
                        <div class="metric-card">
                            <div class="metric-header">
                                <span class="metric-title">Git Commit</span>
                                <div class="metric-icon" style="background: #fce4ec; color: #c2185b;">
                                    <i class="fab fa-git-alt"></i>
                                </div>
                            </div>
                            <div class="metric-value" style="font-size: 1rem; font-family: monospace;">
                                {{ buildInfo.git_commit ? buildInfo.git_commit.substring(0, 8) : 'unknown' }}
                            </div>
                            <div class="metric-change">{{ buildInfo.git_branch || 'unknown' }} branch</div>
                        </div>
                        
                        <div class="metric-card">
                            <div class="metric-header">
                                <span class="metric-title">Build Time</span>
                                <div class="metric-icon" style="background: #e8f5e9; color: #2e7d32;">
                                    <i class="fas fa-clock"></i>
                                </div>
                            </div>
                            <div class="metric-value" style="font-size: 0.875rem;">{{ formatBuildTime(buildInfo.build_time) }}</div>
                            <div class="metric-change">Compilation date</div>
                        </div>
                        
                        <div class="metric-card">
                            <div class="metric-header">
                                <span class="metric-title">Go Version</span>
                                <div class="metric-icon" style="background: #e3f2fd; color: #1976d2;">
                                    <i class="fab fa-golang"></i>
                                </div>
                            </div>
                            <div class="metric-value" style="font-size: 1.25rem;">{{ buildInfo.go_version || 'unknown' }}</div>
                            <div class="metric-change">{{ buildInfo.go_os }}/{{ buildInfo.go_arch }}</div>
                        </div>
                    </div>

                    <!-- Build Details -->
                    <div class="data-table" style="margin-bottom: 2rem;">
                        <div class="table-header">
                            <h4 class="table-title">Build Details</h4>
                        </div>
                        <div class="table-content">
                            <table>
                                <tbody>
                                    <tr>
                                        <td style="font-weight: 600; width: 200px;">Operating System</td>
                                        <td>{{ buildInfo.go_os || 'unknown' }}</td>
                                    </tr>
                                    <tr>
                                        <td style="font-weight: 600;">Architecture</td>
                                        <td>{{ buildInfo.go_arch || 'unknown' }}</td>
                                    </tr>
                                    <tr>
                                        <td style="font-weight: 600;">CGO Enabled</td>
                                        <td>
                                            <span :class="buildInfo.cgo_enabled === 'true' ? 'status-badge status-ok' : 'status-badge status-unknown'">
                                                {{ buildInfo.cgo_enabled === 'true' ? 'Yes' : 'No' }}
                                            </span>
                                        </td>
                                    </tr>
                                    <tr>
                                        <td style="font-weight: 600;">Build Flags</td>
                                        <td style="font-family: monospace; font-size: 0.875rem;">
                                            {{ buildInfo.build_flags || 'none' }}
                                        </td>
                                    </tr>
                                    <tr>
                                        <td style="font-weight: 600;">Git Branch</td>
                                        <td>
                                            <span class="status-badge" style="background: var(--light-bg); color: var(--text-primary);">
                                                {{ buildInfo.git_branch || 'unknown' }}
                                            </span>
                                        </td>
                                    </tr>
                                    <tr>
                                        <td style="font-weight: 600;">Full Git Commit</td>
                                        <td style="font-family: monospace; font-size: 0.875rem;">
                                            {{ buildInfo.git_commit || 'unknown' }}
                                        </td>
                                    </tr>
                                </tbody>
                            </table>
                        </div>
                    </div>

                    <!-- Dependencies -->
                    <div class="data-table">
                        <div class="table-header">
                            <h4 class="table-title">Go Modules & Dependencies</h4>
                        </div>
                        <div class="table-content">
                            <table>
                                <thead>
                                    <tr>
                                        <th>Module</th>
                                        <th>Version</th>
                                        <th>Status</th>
                                    </tr>
                                </thead>
                                <tbody>
                                    <tr v-for="module in buildInfo.modules" :key="module.path">
                                        <td style="font-family: monospace; font-size: 0.875rem;">
                                            {{ module.path }}
                                        </td>
                                        <td>
                                            <span class="status-badge" style="background: var(--light-bg); color: var(--text-primary);">
                                                {{ module.version }}
                                            </span>
                                        </td>
                                        <td>
                                            <span v-if="module.replace" class="status-badge status-warning">
                                                <i class="fas fa-exchange-alt"></i> Replaced
                                            </span>
                                            <span v-else class="status-badge status-ok">
                                                <i class="fas fa-check"></i> Direct
                                            </span>
                                        </td>
                                    </tr>
                                </tbody>
                            </table>
                        </div>
                    </div>

                    <!-- Footer -->
                    <div style="text-align: center; margin-top: 2rem; padding-top: 2rem; border-top: 1px solid var(--border-color); color: var(--text-muted);">
                        <p>
                            <i class="fas fa-heart" style="color: var(--danger-color);"></i>
                            Built with Go and Vue.js
                        </p>
                        <p style="font-size: 0.875rem; margin-top: 0.5rem;">
                            © 2024 Raven Network Monitoring System
                        </p>
                    </div>
                </div>
            </div>
        </div>
    `
};
