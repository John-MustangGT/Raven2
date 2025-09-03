// js/components/notifications-view.js - New frontend component for notification settings
window.NotificationsView = {
    data() {
        return {
            loading: false,
            saving: false,
            testing: false,
            notificationStatus: {
                enabled: false,
                pushover_enabled: false,
                pushover_configured: false,
                services: []
            },
            pushoverConfig: {
                enabled: false,
                user_key: '',
                api_token: '',
                device: '',
                priority: 0,
                sound: 'pushover',
                quiet_hours: {
                    enabled: false,
                    start_hour: 22,
                    end_hour: 7,
                    timezone: 'UTC'
                },
                realert_interval: '1h',
                max_realerts: 5,
                send_recovery: true,
                title: '',
                url_title: 'View in Raven',
                url: '',
                test_on_save: false
            },
            testResults: {
                success: false,
                message: '',
                timestamp: null
            },
            showAdvanced: false,
            availableSounds: [
                'pushover', 'bike', 'bugle', 'cashregister', 'classical', 'cosmic',
                'falling', 'gamelan', 'incoming', 'intermission', 'magic', 'mechanical',
                'pianobar', 'siren', 'spacealarm', 'tugboat', 'alien', 'climb',
                'persistent', 'echo', 'updown', 'none'
            ],
            timezones: [
                'UTC', 'America/New_York', 'America/Chicago', 'America/Denver',
                'America/Los_Angeles', 'Europe/London', 'Europe/Paris', 'Europe/Berlin',
                'Asia/Tokyo', 'Asia/Shanghai', 'Australia/Sydney'
            ]
        };
    },
    async mounted() {
        await this.loadNotificationStatus();
        if (this.notificationStatus.pushover_enabled) {
            await this.loadPushoverConfig();
        }
    },
    methods: {
        async loadNotificationStatus() {
            this.loading = true;
            try {
                const response = await axios.get('/api/notifications/status');
                this.notificationStatus = response.data.data;
            } catch (error) {
                console.error('Failed to load notification status:', error);
                window.RavenUtils.showNotification(this.$parent, 'error', 'Failed to load notification status');
            } finally {
                this.loading = false;
            }
        },

        async loadPushoverConfig() {
            try {
                const response = await axios.get('/api/notifications/pushover/config');
                const config = response.data.data;
                
                this.pushoverConfig = {
                    enabled: config.enabled,
                    user_key: config.user_key_set ? '••••••••••••••••' : '',
                    api_token: config.api_token_set ? '••••••••••••••••' : '',
                    device: config.device || '',
                    priority: config.priority,
                    sound: config.sound || 'pushover',
                    quiet_hours: config.quiet_hours || {
                        enabled: false,
                        start_hour: 22,
                        end_hour: 7,
                        timezone: 'UTC'
                    },
                    realert_interval: config.realert_interval || '1h',
                    max_realerts: config.max_realerts,
                    send_recovery: config.send_recovery,
                    title: config.title || '',
                    url_title: config.url_title || 'View in Raven',
                    url: config.url || '',
                    test_on_save: false
                };
            } catch (error) {
                console.error('Failed to load Pushover config:', error);
            }
        },

        async savePushoverConfig() {
            this.saving = true;
            this.testResults = { success: false, message: '', timestamp: null };

            try {
                // Only send actual values if they've been changed (not masked)
                const configToSave = { ...this.pushoverConfig };
                if (configToSave.user_key === '••••••••••••••••') {
                    delete configToSave.user_key;
                }
                if (configToSave.api_token === '••••••••••••••••') {
                    delete configToSave.api_token;
                }

                const response = await axios.put('/api/notifications/pushover/config', configToSave);
                
                if (response.data.test_result) {
                    this.testResults = {
                        success: response.data.test_result === 'success',
                        message: response.data.test_message || response.data.test_error || '',
                        timestamp: new Date()
                    };
                }

                window.RavenUtils.showNotification(this.$parent, 'success', 'Pushover configuration saved successfully');
                await this.loadNotificationStatus();
            } catch (error) {
                console.error('Failed to save Pushover config:', error);
                const errorMessage = error.response?.data?.error || 'Failed to save configuration';
                window.RavenUtils.showNotification(this.$parent, 'error', errorMessage);
            } finally {
                this.saving = false;
            }
        },

        async testPushoverConfig() {
            this.testing = true;
            this.testResults = { success: false, message: '', timestamp: null };

            try {
                const response = await axios.post('/api/notifications/pushover/test');
                this.testResults = {
                    success: true,
                    message: response.data.message,
                    timestamp: new Date()
                };
                window.RavenUtils.showNotification(this.$parent, 'success', 'Test notification sent!');
            } catch (error) {
                console.error('Pushover test failed:', error);
                const errorMessage = error.response?.data?.details || error.response?.data?.error || 'Test failed';
                this.testResults = {
                    success: false,
                    message: errorMessage,
                    timestamp: new Date()
                };
                window.RavenUtils.showNotification(this.$parent, 'error', 'Test notification failed');
            } finally {
                this.testing = false;
            }
        },

        formatTime(hour) {
            const period = hour >= 12 ? 'PM' : 'AM';
            const displayHour = hour % 12 || 12;
            return `${displayHour}:00 ${period}`;
        },

        getPriorityDescription(priority) {
            switch (priority) {
                case -2: return 'Lowest (Silent)';
                case -1: return 'Low (No Sound)';
                case 0: return 'Normal';
                case 1: return 'High';
                case 2: return 'Emergency (Bypass Quiet Hours)';
                default: return 'Normal';
            }
        },

        isValidDuration(duration) {
            return /^\d+[smhd]$/.test(duration.trim());
        }
    },
    template: `
        <div class="notifications-view">
            <div class="data-table">
                <div class="table-header">
                    <h3 class="table-title">
                        <i class="fas fa-bell"></i>
                        Notification Settings
                    </h3>
                </div>
                
                <div v-if="loading" class="loading">
                    <div class="spinner"></div>
                    Loading notification settings...
                </div>
                
                <div v-else style="padding: 2rem;">
                    <!-- Notification Status Overview -->
                    <div class="metrics-grid" style="margin-bottom: 2rem;">
                        <div class="metric-card">
                            <div class="metric-header">
                                <span class="metric-title">Pushover Status</span>
                                <div class="metric-icon" :class="notificationStatus.pushover_enabled ? 'ok' : 'unknown'">
                                    <i class="fas fa-mobile-alt"></i>
                                </div>
                            </div>
                            <div class="metric-value">{{ notificationStatus.pushover_enabled ? 'Enabled' : 'Disabled' }}</div>
                            <div class="metric-change">{{ notificationStatus.pushover_configured ? 'Configured' : 'Not configured' }}</div>
                        </div>
                        
                        <div class="metric-card" v-if="testResults.timestamp">
                            <div class="metric-header">
                                <span class="metric-title">Last Test</span>
                                <div class="metric-icon" :class="testResults.success ? 'ok' : 'critical'">
                                    <i :class="testResults.success ? 'fas fa-check' : 'fas fa-times'"></i>
                                </div>
                            </div>
                            <div class="metric-value">{{ testResults.success ? 'Success' : 'Failed' }}</div>
                            <div class="metric-change">{{ testResults.timestamp.toLocaleTimeString() }}</div>
                        </div>
                    </div>

                    <!-- Pushover Configuration -->
                    <div class="data-table" style="margin-bottom: 2rem;">
                        <div class="table-header">
                            <h4 class="table-title">
                                <i class="fab fa-pushed"></i>
                                Pushover Configuration
                            </h4>
                            <div class="search-box">
                                <button class="btn btn-secondary" @click="testPushoverConfig" 
                                        :disabled="!pushoverConfig.enabled || testing">
                                    <i class="fas fa-paper-plane" :class="{ 'fa-spin': testing }"></i>
                                    <span>{{ testing ? 'Testing...' : 'Test' }}</span>
                                </button>
                                <button class="btn btn-primary" @click="savePushoverConfig" :disabled="saving">
                                    <i class="fas fa-save"></i>
                                    <span>{{ saving ? 'Saving...' : 'Save' }}</span>
                                </button>
                            </div>
                        </div>
                        
                        <div class="table-content" style="padding: 1.5rem;">
                            <!-- Basic Settings -->
                            <div class="form-group">
                                <label class="form-checkbox">
                                    <input v-model="pushoverConfig.enabled" type="checkbox">
                                    <span>Enable Pushover Notifications</span>
                                </label>
                                <small style="color: var(--text-muted); display: block; margin-top: 0.5rem;">
                                    Send alert notifications via Pushover mobile app
                                </small>
                            </div>

                            <div v-if="pushoverConfig.enabled">
                                <div class="form-group">
                                    <label class="form-label">User Key *</label>
                                    <input v-model="pushoverConfig.user_key" 
                                           class="form-input" 
                                           type="password"
                                           placeholder="Your 30-character user key">
                                    <small style="color: var(--text-muted);">
                                        Found in your Pushover account settings
                                    </small>
                                </div>

                                <div class="form-group">
                                    <label class="form-label">API Token *</label>
                                    <input v-model="pushoverConfig.api_token" 
                                           class="form-input" 
                                           type="password"
                                           placeholder="Your application token">
                                    <small style="color: var(--text-muted);">
                                        Create an application at pushover.net to get this token
                                    </small>
                                </div>

                                <div class="form-group">
                                    <label class="form-label">Device (Optional)</label>
                                    <input v-model="pushoverConfig.device" 
                                           class="form-input" 
                                           type="text"
                                           placeholder="Leave empty to send to all devices">
                                </div>

                                <div class="form-group">
                                    <label class="form-label">Priority</label>
                                    <select v-model.number="pushoverConfig.priority" class="form-input">
                                        <option value="-2">Lowest (Silent)</option>
                                        <option value="-1">Low (No Sound)</option>
                                        <option value="0">Normal</option>
                                        <option value="1">High</option>
                                        <option value="2">Emergency (Bypass Quiet Hours)</option>
                                    </select>
                                    <small style="color: var(--text-muted);">
                                        {{ getPriorityDescription(pushoverConfig.priority) }}
                                    </small>
                                </div>

                                <div class="form-group">
                                    <label class="form-label">Notification Sound</label>
                                    <select v-model="pushoverConfig.sound" class="form-input">
                                        <option v-for="sound in availableSounds" :key="sound" :value="sound">
                                            {{ sound.charAt(0).toUpperCase() + sound.slice(1) }}
                                        </option>
                                    </select>
                                </div>

                                <!-- Advanced Settings -->
                                <div class="form-group">
                                    <button type="button" 
                                            class="btn btn-secondary" 
                                            @click="showAdvanced = !showAdvanced">
                                        <i :class="showAdvanced ? 'fas fa-chevron-up' : 'fas fa-chevron-down'"></i>
                                        <span>{{ showAdvanced ? 'Hide' : 'Show' }} Advanced Settings</span>
                                    </button>
                                </div>

                                <div v-if="showAdvanced" style="border-top: 1px solid var(--border-color); padding-top: 1.5rem; margin-top: 1rem;">
                                    <!-- Quiet Hours -->
                                    <div class="form-group">
                                        <label class="form-checkbox">
                                            <input v-model="pushoverConfig.quiet_hours.enabled" type="checkbox">
                                            <span>Enable Quiet Hours</span>
                                        </label>
                                        <small style="color: var(--text-muted); display: block; margin-top: 0.5rem;">
                                            Suppress notifications during specified hours
                                        </small>
                                    </div>

                                    <div v-if="pushoverConfig.quiet_hours.enabled" 
                                         style="background: var(--light-bg); padding: 1rem; border-radius: 0.5rem; margin-top: 1rem;">
                                        <div style="display: grid; grid-template-columns: 1fr 1fr 1fr; gap: 1rem;">
                                            <div>
                                                <label class="form-label">Start Hour</label>
                                                <select v-model.number="pushoverConfig.quiet_hours.start_hour" class="form-input">
                                                    <option v-for="hour in 24" :key="hour-1" :value="hour-1">
                                                        {{ formatTime(hour-1) }}
                                                    </option>
                                                </select>
                                            </div>
                                            <div>
                                                <label class="form-label">End Hour</label>
                                                <select v-model.number="pushoverConfig.quiet_hours.end_hour" class="form-input">
                                                    <option v-for="hour in 24" :key="hour-1" :value="hour-1">
                                                        {{ formatTime(hour-1) }}
                                                    </option>
                                                </select>
                                            </div>
                                            <div>
                                                <label class="form-label">Timezone</label>
                                                <select v-model="pushoverConfig.quiet_hours.timezone" class="form-input">
                                                    <option v-for="tz in timezones" :key="tz" :value="tz">
                                                        {{ tz }}
                                                    </option>
                                                </select>
                                            </div>
                                        </div>
                                    </div>

                                    <!-- Realert Settings -->
                                    <div class="form-group">
                                        <label class="form-label">Realert Interval</label>
                                        <input v-model="pushoverConfig.realert_interval" 
                                               class="form-input" 
                                               type="text"
                                               placeholder="1h"
                                               :class="{ 'error': pushoverConfig.realert_interval && !isValidDuration(pushoverConfig.realert_interval) }">
                                        <small style="color: var(--text-muted);">
                                            How often to resend notifications for ongoing issues (e.g., 30m, 1h, 2h)
                                        </small>
                                    </div>

                                    <div class="form-group">
                                        <label class="form-label">Maximum Realerts</label>
                                        <input v-model.number="pushoverConfig.max_realerts" 
                                               class="form-input" 
                                               type="number"
                                               min="0"
                                               max="50"
                                               placeholder="5">
                                        <small style="color: var(--text-muted);">
                                            Maximum number of realerts to send per issue (0 = no limit)
                                        </small>
                                    </div>

                                    <div class="form-group">
                                        <label class="form-checkbox">
                                            <input v-model="pushoverConfig.send_recovery" type="checkbox">
                                            <span>Send Recovery Notifications</span>
                                        </label>
                                        <small style="color: var(--text-muted); display: block; margin-top: 0.5rem;">
                                            Send notification when issues are resolved
                                        </small>
                                    </div>

                                    <!-- Message Customization -->
                                    <div class="form-group">
                                        <label class="form-label">Custom Title</label>
                                        <input v-model="pushoverConfig.title" 
                                               class="form-input" 
                                               type="text"
                                               placeholder="Raven Monitor">
                                        <small style="color: var(--text-muted);">
                                            Prefix for notification titles (leave empty for default)
                                        </small>
                                    </div>

                                    <div class="form-group">
                                        <label class="form-label">URL</label>
                                        <input v-model="pushoverConfig.url" 
                                               class="form-input" 
                                               type="url"
                                               placeholder="http://your-server.com/hosts/{HOST_ID}">
                                        <small style="color: var(--text-muted);">
                                            URL to include in notifications. Use {HOST_ID}, {CHECK_ID}, etc. for placeholders
                                        </small>
                                    </div>

                                    <div class="form-group">
                                        <label class="form-label">URL Title</label>
                                        <input v-model="pushoverConfig.url_title" 
                                               class="form-input" 
                                               type="text"
                                               placeholder="View in Raven">
                                    </div>

                                    <div class="form-group">
                                        <label class="form-checkbox">
                                            <input v-model="pushoverConfig.test_on_save" type="checkbox">
                                            <span>Test Configuration When Saving</span>
                                        </label>
                                    </div>
                                </div>
                            </div>
                        </div>
                    </div>

                    <!-- Test Results -->
                    <div v-if="testResults.timestamp" class="data-table">
                        <div class="table-header">
                            <h4 class="table-title">
                                <i :class="testResults.success ? 'fas fa-check-circle' : 'fas fa-times-circle'"
                                   :style="{ color: testResults.success ? 'var(--success-color)' : 'var(--danger-color)' }"></i>
                                Test Results
                            </h4>
                        </div>
                        <div class="table-content" style="padding: 1rem;">
                            <div :class="testResults.success ? 'alert alert-success' : 'alert alert-error'"
                                 style="padding: 1rem; border-radius: 0.5rem; margin-bottom: 1rem;">
                                <strong>{{ testResults.success ? 'Success!' : 'Failed!' }}</strong>
                                <div style="margin-top: 0.5rem;">{{ testResults.message }}</div>
                                <div style="font-size: 0.875rem; color: var(--text-muted); margin-top: 0.5rem;">
                                    Tested at {{ testResults.timestamp.toLocaleString() }}
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    `
};

// Add CSS for alerts
const style = document.createElement('style');
style.textContent = `
.alert {
    padding: 1rem;
    border-radius: 0.5rem;
    margin-bottom: 1rem;
    border: 1px solid;
}

.alert-success {
    background: rgba(16, 185, 129, 0.1);
    border-color: rgba(16, 185, 129, 0.3);
    color: var(--success-color);
}

.alert-error {
    background: rgba(239, 68, 68, 0.1);
    border-color: rgba(239, 68, 68, 0.3);
    color: var(--danger-color);
}

[data-theme="dark"] .alert-success {
    background: rgba(16, 185, 129, 0.15);
    border-color: rgba(16, 185, 129, 0.4);
}

[data-theme="dark"] .alert-error {
    background: rgba(239, 68, 68, 0.15);
    border-color: rgba(239, 68, 68, 0.4);
}
`;
document.head.appendChild(style);
