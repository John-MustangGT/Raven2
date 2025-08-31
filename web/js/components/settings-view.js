// js/components/settings-view.js
window.SettingsView = {
    props: {
        settings: Object,
        connected: Boolean,
        buildInfo: Object,
        webConfig: Object
    },
    emits: ['toggle-theme', 'export-config', 'refresh-data'],
    template: `
        <div class="data-table">
            <div class="table-header">
                <h3 class="table-title">System Settings</h3>
            </div>
            <div style="padding: 2rem;">
                <div class="form-group">
                    <h4>Web Interface</h4>
                    <label class="form-checkbox">
                        <input v-model="settings.darkMode" type="checkbox" @change="$emit('toggle-theme')">
                        <span>Dark Mode</span>
                    </label>
                </div>
                
                <div class="form-group">
                    <h4>Refresh Settings</h4>
                    <label class="form-label">Auto-refresh interval (seconds)</label>
                    <input v-model="settings.refreshInterval" class="form-input" type="number" min="10" max="300" style="width: 200px;">
                </div>

                <div class="form-group">
                    <h4>System Information</h4>
                    <div style="background: var(--light-bg); padding: 1rem; border-radius: 0.5rem;">
                        <div><strong>Version:</strong> {{ buildInfo.version || 'Raven v2.0.0' }}</div>
                        <div><strong>API Status:</strong> <span style="color: var(--success-color);">✓ Connected</span></div>
                        <div><strong>WebSocket:</strong> 
                            <span :style="{ color: connected ? 'var(--success-color)' : 'var(--danger-color)' }">
                                {{ connected ? '✓ Connected' : '✗ Disconnected' }}
                            </span>
                        </div>
                        <div><strong>Header Link:</strong> 
                            <a :href="webConfig.header_link" target="_blank" rel="noopener noreferrer" style="color: var(--primary-color);">
                                {{ webConfig.header_link }}
                            </a>
                        </div>
                    </div>
                </div>

                <div class="form-group">
                    <h4>Actions</h4>
                    <div style="display: flex; gap: 1rem;">
                        <button class="btn btn-secondary" @click="$emit('export-config')">
                            <i class="fas fa-download"></i>
                            Export Configuration
                        </button>
                        <button class="btn btn-secondary" @click="$emit('refresh-data')">
                            <i class="fas fa-sync"></i>
                            Refresh All Data
                        </button>
                    </div>
                </div>
            </div>
        </div>
    `
};
