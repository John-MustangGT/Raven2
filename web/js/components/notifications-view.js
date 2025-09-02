// web/js/components/notifications-view.js - Pushover notification settings management
window.NotificationsView = {
    props: {
        loading: Boolean,
        saving: Boolean
    },
    emits: ['save-settings', 'test-notification'],
    data() {
        return {
            settings: {
                enabled: false,
                pushover: {
                    enabled: false,
                    api_token: '',
                    user_key: '',
                    priority: 0,
                    retry: 60,
                    expire: 3600,
                    sound: 'pushover',
                    device: '',
                    title: 'Raven Alert: {{.Host}}',
                    template: '{{.StatusEmoji}} {{.Check}} on {{.Host}} is {{.Status}}: {{.Output}}',
                    only_on_state: ['critical', 'warning', 'recovery'],
                    throttle: {
                        enabled: true,
                        window_minutes: 15,
                        max_per_host: 5,
                        max_total: 20
                    }
                }
            },
            stats: {
                enabled: false,
                pushover_enabled: false,
                throttle_enabled: false,
                stats: {}
            },
            availableSounds: [],
            templateVariables: {},
            testMessage: 'Test notification from Raven monitoring system!',
            showTemplateHelp: false,
            validationErrors: {},
            hasUnsavedChanges: false
        }
    },
    computed: {
        priorityOptions() {
            return [
                { value: -2, label: 'Silent (-2)', description: 'Generate no notification/alert' },
                { value: -1, label: 'Quiet (-1)', description: 'Always send as quiet notification' },
                { value: 0, label: 'Normal (0)', description: 'Normal priority' },
                { value: 1, label: 'High (1)', description: 'Display as high-priority' },
                { value: 2, label: 'Emergency (2)', description: 'Require confirmation from user' }
            ];
        },
        stateOptions() {
            return [
                { value: 'critical', label: 'Critical', description: 'Service failures' },
                { value: 'warning', label: 'Warning', description: 'Service degradation' },
                { value: 'recovery', label: 'Recovery', description: 'Return to OK status' },
                { value: 'unknown', label: 'Unknown', description: 'Status unclear' }
            ];
        },
        emergencyPrioritySelected() {
            return this.settings.pushover.priority === 2;
        },
        isConfigValid() {
            return Object.keys(this.validationErrors).length === 0;
        }
    },
    watch: {
        settings: {
            handler() {
                this.hasUnsavedChanges = true;
                this.validateSettings();
            },
            deep: true
        }
    },
    async mounted() {
        await this.loadSettings();
        await this.loadSounds();
        await this.loadTemplateVariables();
        await this.loadStats();
    },
    methods: {
        async loadSettings() {
            try {
                const response = await axios.get('/api/notifications/settings');
                this.settings = { ...this.settings, ...response.data.data };
                this.hasUnsavedChanges = false;
            } catch (error) {
                console.error('Failed to load notification settings:', error);
                window.RavenUtils.showNotification(this.$parent, 'error', 'Failed to load notification settings');
            }
        },

        async loadSounds() {
            try {
                const response = await axios.get('/api/notifications/pushover/sounds');
                this.availableSounds = response.data.data;
            } catch (error) {
                console.error('Failed to load Pushover sounds:', error);
                // Fallback to basic sounds
                this.availableSounds = [
                    { value: 'pushover', label: 'Pushover (default)' },
                    { value: 'bike', label: 'Bike' },
                    { value: 'cashregister', label: 'Cash Register' },
                    { value: 'classical', label: 'Classical' },
                    { value: 'siren', label: 'Siren' },
                    { value: 'none', label: 'Silent' }
                ];
            }
        },

        async loadTemplateVariables() {
            try {
                const response = await axios.get('/api/notifications/template/variables');
                this.templateVariables = response.data.data;
            } catch (error) {
                console.error('Failed to load template variables:', error);
            }
        },

        async loadStats() {
            try {
                const response = await axios.get('/api/notifications/stats');
                this.stats = response.data.data;
            } catch (error) {
                console.error('Failed to load notification stats:', error);
            }
        },

        async saveSettings() {
            if (!this.isConfigValid) {
                window.RavenUtils.showNotification(this.$parent, 'error', 'Please fix validation errors before saving');
                return;
            }

            this.$emit('save-settings');
            
            try {
                await axios.put('/api/notifications/settings', this.settings);
                window.RavenUtils.showNotification(this.$parent, 'success', 'Notification settings saved successfully');
                this.hasUnsavedChanges = false;
                await this.loadStats();
            } catch (error) {
                console.error('Failed to save notification settings:', error);
                const message = error.response?.data?.error || 'Failed to save settings';
                window.RavenUtils.showNotification(this.$parent, 'error', message);
            }
        },

        async testNotification() {
            if (!this.settings.enabled || !this.settings.pushover.enabled) {
                window.RavenUtils.showNotification(this.$parent, 'error', 'Enable notifications first');
                return;
            }

            this.$emit('test-notification');
            
            try {
                await axios.post('/api/notifications/test', {
                    message: this.testMessage
                });
                window.RavenUtils.showNotification(this.$parent, 'success', 'Test notification sent! Check your device.');
            } catch (error) {
                console.error('Failed to send test notification:', error);
                const message = error.response?.data?.error || 'Failed to send test notification';
                window.RavenUtils.showNotification(this.$parent, 'error', message);
            }
        },

        async validateSettings() {
            this.validationErrors = {};

            if (this.settings.enabled && this.settings.pushover.enabled) {
                // Validate API token
                if (!this.settings.pushover.api_token || this.settings.pushover.api_token.trim() === '') {
                    this.validationErrors.api_token = 'API token is required';
                } else if (!this.isMaskedToken(this.settings.pushover.api_token) && this.settings.pushover.api_token.length !== 30) {
                    this.validationErrors.api_token = 'API token should be 30 characters long';
                }

                // Validate user key
                if (!this.settings.pushover.user_key || this.settings.pushover.user_key.trim() === '') {
                    this.validationErrors.user_key = 'User key is required';
                } else if (!this.isMaskedToken(this.settings.pushover.user_key) && this.settings.pushover.user_key.length !== 30) {
                    this.validationErrors.user_key = 'User key should be 30 characters long';
                }

                // Validate emergency priority settings
                if (this.settings.pushover.priority === 2) {
                    if (this.settings.pushover.retry < 30) {
                        this.validationErrors.retry = 'Retry interval must be at least 30 seconds for emergency priority';
                    }
                    if (this.settings.pushover.expire < 60 || this.settings.pushover.expire > 10800) {
                        this.validationErrors.expire = 'Expire time must be between 60 and 10800 seconds for emergency priority';
                    }
                }

                // Validate templates
                if (this.settings.pushover.title.trim() === '') {
                    this.validationErrors.title = 'Title template cannot be empty';
                }
                if (this.settings.pushover.template.trim() === '') {
                    this.validationErrors.template = 'Message template cannot be empty';
                }

                // Validate throttle settings
                if (this.settings.pushover.throttle.enabled) {
                    if (this.settings.pushover.throttle.window_minutes < 1) {
                        this.validationErrors.throttle_window = 'Throttle window must be at least 1 minute';
                    }
                    if (this.settings.pushover.throttle.max_per_host < 1) {
                        this.validationErrors.throttle_max_per_host = 'Max per host must be at least 1';
                    }
                    if (this.settings.pushover.throttle.max_total < 1) {
                        this.validationErrors.throttle_max_total = 'Max total must be at least 1';
                    }
                }
            }
        },

        isMaskedToken(token) {
            return token && token.includes('*');
        },

        insertTemplate(templateText, field) {
            const textarea = this.$refs[field];
            if (textarea) {
                const start = textarea.selectionStart;
                const end = textarea.selectionEnd;
                const value = this.settings.pushover[field];
                
                this.settings.pushover[field] = value.substring(0, start) + templateText + value.substring(end);
                
                this.$nextTick(() => {
                    textarea.focus();
                    textarea.selectionStart = textarea.selectionEnd = start + templateText.length;
                });
            }
        },

        useExampleTemplate(example) {
            this.settings.pushover.title = example.title;
            this.settings.pushover.template = example.template;
        },

        resetToDefaults() {
            if (confirm('Reset to default settings? This will lose all your customizations.')) {
                this.settings.pushover.title = 'Raven Alert: {{.Host}}';
                this.settings.pushover.template = '{{.StatusEmoji}} {{.Check}} on {{.Host}} is {{.Status}}: {{.Output}}';
                this.settings.pushover.priority = 0;
                this.settings.pushover.sound = 'pushover';
                this.settings.pushover.only_on_state = ['critical', 'warning', 'recovery'];
                this.settings.pushover.throttle = {
                    enabled: true,
                    window_minutes: 15,
                    max_per_host: 5,
                    max_total: 20
                };
            }
        },

        async refreshStats() {
            await this.loadStats();
        },

        formatUptime(stats) {
            // This would format uptime statistics if available
            return 'N/A';
        }
    },
    template: `
        <div>
            <!-- Settings Header -->
            <div class="data-table">
                <div class="table-header">
                    <h3 class="table-title">
                        <i class="fas fa-bell"></i> Push Notification Settings
                    </h3>
                    <div class="search-box">
                        <button class="btn btn-secondary" @click="refreshStats">
                            <i class="fas fa-sync"></i>
                            <span class="btn-text">Refresh Stats</span>
                        </button>
                        <button class="btn btn-secondary" 
                                :disabled="!settings.enabled || !settings.pushover.enabled || saving"
                                @click="testNotification">
                            <i class="fas fa-paper-plane"></i>
                            <span class="btn-text">Send Test</span>
                        </button>
                        <button class="btn btn-primary" 
                                :disabled="!hasUnsavedChanges || !isConfigValid || saving"
                                @click="saveSettings">
                            <i class="fas fa-save"></i>
                            <span class="btn-text">{{ saving ? 'Saving...' : 'Save Settings' }}</span>
                        </button>
                    </div>
                </div>
            </div>

            <!-- Status Cards -->
            <div class="metrics-grid" style="margin-bottom: 2rem;">
                <div class="metric-card">
                    <div class="metric-header">
                        <span class="metric-title">Notifications</span>
                        <div class="metric-icon" :class="{ 'ok': stats.enabled, 'unknown': !stats.enabled }">
                            <i class="fas fa-bell"></i>
                        </div>
                    </div>
                    <div class="metric-value">{{ stats.enabled ? 'Enabled' : 'Disabled' }}</div>
                    <div class="metric-change">System status</div>
                </div>

                <div class="metric-card">
                    <div class="metric-header">
                        <span class="metric-title">Pushover</span>
                        <div class="metric-icon" :class="{ 'ok': stats.pushover_enabled, 'unknown': !stats.pushover_enabled }">
                            <i class="fab fa-telegram"></i>
                        </div>
                    </div>
                    <div class="metric-value">{{ stats.pushover_enabled ? 'Active' : 'Inactive' }}</div>
                    <div class="metric-change">Service status</div>
                </div>

                <div class="metric-card">
                    <div class="metric-header">
                        <span class="metric-title">Throttling</span>
                        <div class="metric-icon" :class="{ 'warning': stats.throttle_enabled, 'unknown': !stats.throttle_enabled }">
                            <i class="fas fa-tachometer-alt"></i>
                        </div>
                    </div>
                    <div class="metric-value">{{ stats.throttle_enabled ? 'Enabled' : 'Disabled' }}</div>
                    <div class="metric-change">Rate limiting</div>
                </div>

                <div class="metric-card">
                    <div class="metric-header">
                        <span class="metric-title">Recent Notifications</span>
                        <div class="metric-icon ok">
                            <i class="fas fa-chart-line"></i>
                        </div>
                    </div>
                    <div class="metric-value">{{ stats.stats.throttle_total_recent || 0 }}</div>
                    <div class="metric-change">Last {{ settings.pushover.throttle.window_minutes || 15 }} minutes</div>
                </div>
            </div>

            <!-- Main Settings -->
            <div class="data-table" style="margin-bottom: 2rem;">
                <div class="table-header">
                    <h4 class="table-title">General Settings</h4>
                </div>
                <div style="padding: 2rem;">
                    <div class="form-group">
                        <label class="form-checkbox">
                            <input v-model="settings.enabled" type="checkbox">
                            <span>Enable Notifications</span>
                        </label>
                        <small style="color: var(--text-muted); display: block; margin-top: 0.5rem;">
                            Master switch for all notification services
                        </small>
                    </div>
                </div>
            </div>

            <!-- Pushover Settings -->
            <div class="data-table" style="margin-bottom: 2rem;">
                <div class="table-header">
                    <h4 class="table-title">
                        <i class="fab fa-telegram"></i> Pushover Configuration
                    </h4>
                    <div>
                        <a href="https://pushover.net/" target="_blank" rel="noopener noreferrer" 
                           class="btn btn-secondary btn-small">
                            <i class="fas fa-external-link-alt"></i>
                            <span class="btn-text">Get Pushover Account</span>
                        </a>
                    </div>
                </div>
                <div style="padding: 2rem;">
                    <div class="form-group">
                        <label class="form-checkbox">
                            <input v-model="settings.pushover.enabled" 
                                   type="checkbox" 
                                   :disabled="!settings.enabled">
                            <span>Enable Pushover Notifications</span>
                        </label>
                    </div>

                    <div v-if="settings.pushover.enabled">
                        <!-- API Credentials -->
                        <div class="form-group">
                            <label class="form-label">
                                API Token *
                                <a href="https://pushover.net/apps" target="_blank" rel="noopener noreferrer" 
                                   style="margin-left: 0.5rem; font-size: 0.875rem;">
                                    <i class="fas fa-external-link-alt"></i> Get Token
                                </a>
                            </label>
                            <input v-model="settings.pushover.api_token" 
                                   class="form-input" 
                                   :class="{ error: validationErrors.api_token }"
                                   type="password" 
                                   placeholder="Enter your Pushover API token">
                            <small v-if="validationErrors.api_token" class="error-text">{{ validationErrors.api_token }}</small>
                        </div>

                        <div class="form-group">
                            <label class="form-label">
                                User Key *
                                <a href="https://pushover.net/" target="_blank" rel="noopener noreferrer"
                                   style="margin-left: 0.5rem; font-size: 0.875rem;">
                                    <i class="fas fa-external-link-alt"></i> Get Key
                                </a>
                            </label>
                            <input v-model="settings.pushover.user_key" 
                                   class="form-input" 
                                   :class="{ error: validationErrors.user_key }"
                                   type="password" 
                                   placeholder="Enter your Pushover user key">
                            <small v-if="validationErrors.user_key" class="error-text">{{ validationErrors.user_key }}</small>
                        </div>

                        <!-- Priority & Sound Settings -->
                        <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 2rem;">
                            <div class="form-group">
                                <label class="form-label">Priority Level</label>
                                <select v-model.number="settings.pushover.priority" class="form-input">
                                    <option v-for="priority in priorityOptions" 
                                            :key="priority.value" 
                                            :value="priority.value">
                                        {{ priority.label }} - {{ priority.description }}
                                    </option>
                                </select>
                            </div>

                            <div class="form-group">
                                <label class="form-label">Notification Sound</label>
                                <select v-model="settings.pushover.sound" class="form-input">
                                    <option v-for="sound in availableSounds" 
                                            :key="sound.value" 
                                            :value="sound.value">
                                        {{ sound.label }}
                                    </option>
                                </select>
                            </div>
                        </div>

                        <!-- Emergency Priority Settings -->
                        <div v-if="emergencyPrioritySelected" 
                             style="background: var(--light-bg); padding: 1rem; border-radius: 0.5rem; margin: 1rem 0;">
                            <h5 style="margin-bottom: 1rem; color: var(--danger-color);">
                                <i class="fas fa-exclamation-triangle"></i> Emergency Priority Settings
                            </h5>
                            <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 1rem;">
                                <div class="form-group">
                                    <label class="form-label">Retry Interval (seconds) *</label>
                                    <input v-model.number="settings.pushover.retry" 
                                           class="form-input" 
                                           :class="{ error: validationErrors.retry }"
                                           type="number" 
                                           min="30" 
                                           max="10800">
                                    <small v-if="validationErrors.retry" class="error-text">{{ validationErrors.retry }}</small>
                                    <small v-else style="color: var(--text-muted);">How often to retry (minimum 30 seconds)</small>
                                </div>

                                <div class="form-group">
                                    <label class="form-label">Expire After (seconds) *</label>
                                    <input v-model.number="settings.pushover.expire" 
                                           class="form-input" 
                                           :class="{ error: validationErrors.expire }"
                                           type="number" 
                                           min="60" 
                                           max="10800">
                                    <small v-if="validationErrors.expire" class="error-text">{{ validationErrors.expire }}</small>
                                    <small v-else style="color: var(--text-muted);">Stop retrying after this time (max 3 hours)</small>
                                </div>
                            </div>
                        </div>

                        <!-- Device Settings -->
                        <div class="form-group">
                            <label class="form-label">Target Device (Optional)</label>
                            <input v-model="settings.pushover.device" 
                                   class="form-input" 
                                   type="text" 
                                   placeholder="Leave empty to send to all devices">
                            <small style="color: var(--text-muted);">Specific device name to send notifications to</small>
                        </div>
                    </div>
                </div>
            </div>

            <!-- Message Templates -->
            <div v-if="settings.pushover.enabled" class="data-table" style="margin-bottom: 2rem;">
                <div class="table-header">
                    <h4 class="table-title">
                        <i class="fas fa-edit"></i> Message Templates
                    </h4>
                    <div>
                        <button class="btn btn-secondary btn-small" 
                                @click="showTemplateHelp = !showTemplateHelp">
                            <i class="fas fa-question-circle"></i>
                            <span class="btn-text">Template Help</span>
                        </button>
                        <button class="btn btn-secondary btn-small" @click="resetToDefaults">
                            <i class="fas fa-undo"></i>
                            <span class="btn-text">Reset Defaults</span>
                        </button>
                    </div>
                </div>
                <div style="padding: 2rem;">
                    <!-- Template Help Panel -->
                    <div v-if="showTemplateHelp" 
                         style="background: var(--light-bg); padding: 1rem; border-radius: 0.5rem; margin-bottom: 2rem; border: 1px solid var(--border-color);">
                        <h5 style="margin-bottom: 1rem;">
                            <i class="fas fa-lightbulb"></i> Template Variables & Examples
                        </h5>
                        
                        <!-- Quick Insert Buttons -->
                        <div style="margin-bottom: 1rem;">
                            <strong>Quick Insert:</strong>
                            <div style="display: flex; gap: 0.5rem; flex-wrap: wrap; margin-top: 0.5rem;">
                                <button v-for="vars in templateVariables.host_variables" 
                                        :key="vars.variable"
                                        class="btn btn-secondary btn-small" 
                                        @click="insertTemplate(vars.variable, 'template')"
                                        style="font-size: 0.75rem;">
                                    {{ vars.variable }}
                                </button>
                                <button v-for="vars in templateVariables.status_variables" 
                                        :key="vars.variable"
                                        class="btn btn-secondary btn-small" 
                                        @click="insertTemplate(vars.variable, 'template')"
                                        style="font-size: 0.75rem;">
                                    {{ vars.variable }}
                                </button>
                            </div>
                        </div>

                        <!-- Examples -->
                        <div v-if="templateVariables.examples">
                            <strong>Examples:</strong>
                            <div style="display: grid; gap: 1rem; margin-top: 0.5rem;">
                                <div v-for="example in templateVariables.examples" 
                                     :key="example.name"
                                     style="background: white; padding: 1rem; border-radius: 0.25rem; border: 1px solid var(--border-color);">
                                    <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 0.5rem;">
                                        <strong>{{ example.name }}</strong>
                                        <button class="btn btn-primary btn-small" 
                                                @click="useExampleTemplate(example)">
                                            Use This
                                        </button>
                                    </div>
                                    <div style="font-family: monospace; font-size: 0.875rem; background: var(--light-bg); padding: 0.5rem; border-radius: 0.25rem;">
                                        <div><strong>Title:</strong> {{ example.title }}</div>
                                        <div style="margin-top: 0.25rem;"><strong>Template:</strong> {{ example.template }}</div>
                                    </div>
                                </div>
                            </div>
                        </div>
                    </div>

                    <!-- Title Template -->
                    <div class="form-group">
                        <label class="form-label">Notification Title Template *</label>
                        <input v-model="settings.pushover.title" 
                               ref="title"
                               class="form-input" 
                               :class="{ error: validationErrors.title }"
                               type="text" 
                               placeholder="e.g., Raven Alert: {{.Host}}">
                        <small v-if="validationErrors.title" class="error-text">{{ validationErrors.title }}</small>
                    </div>

                    <!-- Message Template -->
                    <div class="form-group">
                        <label class="form-label">Message Template *</label>
                        <textarea v-model="settings.pushover.template" 
                                  ref="template"
                                  class="form-input" 
                                  :class="{ error: validationErrors.template }"
                                  rows="4" 
                                  placeholder="e.g., {{.StatusEmoji}} {{.Check}} on {{.Host}} is {{.Status}}: {{.Output}}"></textarea>
                        <small v-if="validationErrors.template" class="error-text">{{ validationErrors.template }}</small>
                        <small v-else style="color: var(--text-muted);">Use Go template syntax with available variables</small>
                    </div>

                    <!-- Notification States -->
                    <div class="form-group">
                        <label class="form-label">Send Notifications For</label>
                        <div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 1rem; margin-top: 0.5rem;">
                            <label v-for="state in stateOptions" 
                                   :key="state.value"
                                   class="form-checkbox">
                                <input v-model="settings.pushover.only_on_state" 
                                       :value="state.value" 
                                       type="checkbox">
                                <span>{{ state.label }}</span>
                                <small style="display: block; color: var(--text-muted); margin-top: 0.25rem;">
                                    {{ state.description }}
                                </small>
                            </label>
                        </div>
                    </div>
                </div>
            </div>

            <!-- Throttling Settings -->
            <div v-if="settings.pushover.enabled" class="data-table" style="margin-bottom: 2rem;">
                <div class="table-header">
                    <h4 class="table-title">
                        <i class="fas fa-tachometer-alt"></i> Rate Limiting & Throttling
                    </h4>
                </div>
                <div style="padding: 2rem;">
                    <div class="form-group">
                        <label class="form-checkbox">
                            <input v-model="settings.pushover.throttle.enabled" type="checkbox">
                            <span>Enable Notification Throttling</span>
                        </label>
                        <small style="color: var(--text-muted); display: block; margin-top: 0.5rem;">
                            Prevents notification spam by limiting the rate of notifications
                        </small>
                    </div>

                    <div v-if="settings.pushover.throttle.enabled">
                        <div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 1rem;">
                            <div class="form-group">
                                <label class="form-label">Time Window (minutes)</label>
                                <input v-model.number="settings.pushover.throttle.window_minutes" 
                                       class="form-input" 
                                       :class="{ error: validationErrors.throttle_window }"
                                       type="number" 
                                       min="1" 
                                       max="1440">
                                <small v-if="validationErrors.throttle_window" class="error-text">{{ validationErrors.throttle_window }}</small>
                            </div>

                            <div class="form-group">
                                <label class="form-label">Max Per Host</label>
                                <input v-model.number="settings.pushover.throttle.max_per_host" 
                                       class="form-input" 
                                       :class="{ error: validationErrors.throttle_max_per_host }"
                                       type="number" 
                                       min="1" 
                                       max="100">
                                <small v-if="validationErrors.throttle_max_per_host" class="error-text">{{ validationErrors.throttle_max_per_host }}</small>
                            </div>

                            <div class="form-group">
                                <label class="form-label">Max Total</label>
                                <input v-model.number="settings.pushover.throttle.max_total" 
                                       class="form-input" 
                                       :class="{ error: validationErrors.throttle_max_total }"
                                       type="number" 
                                       min="1" 
                                       max="500">
                                <small v-if="validationErrors.throttle_max_total" class="error-text">{{ validationErrors.throttle_max_total }}</small>
                            </div>
                        </div>
                    </div>
                </div>
            </div>

            <!-- Test Notification -->
            <div v-if="settings.enabled && settings.pushover.enabled" class="data-table">
                <div class="table-header">
                    <h4 class="table-title">
                        <i class="fas fa-paper-plane"></i> Test Notification
                    </h4>
                </div>
                <div style="padding: 2rem;">
                    <div class="form-group">
                        <label class="form-label">Test Message</label>
                        <input v-model="testMessage" 
                               class="form-input" 
                               type="text" 
                               placeholder="Enter a test message">
                    </div>
                    <div style="display: flex; gap: 1rem; align-items: center;">
                        <button class="btn btn-primary" @click="testNotification">
                            <i class="fas fa-paper-plane"></i>
                            Send Test Notification
                        </button>
                        <small style="color: var(--text-muted);">
                            This will send a test notification to your Pushover device(s)
                        </small>
                    </div>
                </div>
            </div>
        </div>
    `
};
