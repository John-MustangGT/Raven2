// js/app.js - Main application
const { createApp } = Vue;

createApp({
    components: {
        'sidebar-component': window.SidebarComponent,
        'header-component': window.HeaderComponent,
        'dashboard-view': window.DashboardView,
        'hosts-view': window.HostsView,
        'checks-view': window.ChecksView,
        'alerts-view': window.AlertsView,
        'about-view': window.AboutView,
        'settings-view': window.SettingsView,
        'host-modal': window.HostModal,
        'check-modal': window.CheckModal,
        'notification-component': window.NotificationComponent
    },
    data() {
        return {
            currentView: 'dashboard',
            theme: localStorage.getItem('theme') || 'light',
            sidebarOpen: false,
            sidebarCollapsed: localStorage.getItem('sidebarCollapsed') === 'true',
            connected: false,
            loading: false,
            loadingBuildInfo: false,
            saving: false,
            searchQuery: '',
            filterGroup: '',
            searchChecksQuery: '',
            filterCheckType: '',
            alertFilter: 'all',
            showHostModal: false,
            showCheckModal: false,
            editingHost: null,
            editingCheck: null,
            hosts: [],
            checks: [],
            alerts: [],
            stats: {
                ok: 0,
                warning: 0,
                critical: 0,
                unknown: 0
            },
            recentActivity: [],
            buildInfo: {
                version: 'Loading...',
                git_commit: '',
                git_branch: '',
                build_time: '',
                go_version: '',
                go_os: '',
                go_arch: '',
                cgo_enabled: '',
                build_flags: '',
                modules: []
            },
            webConfig: {
                header_link: 'https://github.com/John-MustangGT/raven2',
                serve_static: false,
                root: 'index.html'
            },
            hostForm: window.RavenUtils.getEmptyHostForm(),
            checkForm: window.RavenUtils.getEmptyCheckForm(),
            settings: {
                darkMode: localStorage.getItem('theme') === 'dark',
                refreshInterval: 30
            },
            notification: {
                show: false,
                type: '',
                message: ''
            },
            websocket: null
        }
    },
    computed: {
        pageTitle() {
            const titles = {
                dashboard: 'Dashboard',
                hosts: 'Hosts',
                checks: 'Checks',
                alerts: 'Alerts',
                about: 'About',
                settings: 'Settings'
            };
            return titles[this.currentView] || 'Dashboard';
        },
        filteredHosts() {
            return this.hosts.filter(host => {
                const matchesSearch = !this.searchQuery || 
                    host.name.toLowerCase().includes(this.searchQuery.toLowerCase()) ||
                    (host.display_name && host.display_name.toLowerCase().includes(this.searchQuery.toLowerCase())) ||
                    (host.ipv4 && host.ipv4.includes(this.searchQuery)) ||
                    (host.hostname && host.hostname.toLowerCase().includes(this.searchQuery.toLowerCase()));
                
                const matchesGroup = !this.filterGroup || host.group === this.filterGroup;
                
                return matchesSearch && matchesGroup;
            });
        },
        groups() {
            const groups = [...new Set(this.hosts.map(host => host.group))];
            return groups.filter(Boolean).sort();
        },
        filteredChecks() {
            return this.checks.filter(check => {
                const matchesSearch = !this.searchChecksQuery || 
                    check.name.toLowerCase().includes(this.searchChecksQuery.toLowerCase()) ||
                    (check.id && check.id.toLowerCase().includes(this.searchChecksQuery.toLowerCase())) ||
                    check.type.toLowerCase().includes(this.searchChecksQuery.toLowerCase());
                
                const matchesType = !this.filterCheckType || check.type === this.filterCheckType;
                
                return matchesSearch && matchesType;
            });
        },
        checkTypes() {
            const types = [...new Set(this.checks.map(check => check.type))];
            return types.filter(Boolean).sort();
        },
        activeAlerts() {
            return this.alerts.filter(alert => alert.severity !== 'ok');
        },
        alertMetrics() {
            return {
                active: this.activeAlerts.length,
                critical: this.alerts.filter(alert => alert.severity === 'critical').length,
                warning: this.alerts.filter(alert => alert.severity === 'warning').length
            };
        },
        filteredAlerts() {
            if (this.alertFilter === 'all') {
                return this.activeAlerts;
            }
            return this.alerts.filter(alert => alert.severity === this.alertFilter);
        }
    },
    async mounted() {
        document.documentElement.setAttribute('data-theme', this.theme);
        await this.loadWebConfig();
        await this.loadHosts();
        this.connectWebSocket();
        await this.loadStats();
        
        // Auto-refresh data
        setInterval(() => {
            this.loadStats();
            if (this.currentView === 'hosts') {
                this.loadHosts();
            } else if (this.currentView === 'checks') {
                this.loadChecks();
            } else if (this.currentView === 'alerts') {
                this.loadAlerts();
            }
        }, this.settings.refreshInterval * 1000);

        // Close sidebar when clicking outside on mobile
        document.addEventListener('click', (e) => {
            if (window.innerWidth <= 1024 && this.sidebarOpen) {
                const sidebar = document.querySelector('.sidebar');
                const menuToggle = document.querySelector('.menu-toggle');
                if (!sidebar.contains(e.target) && !menuToggle.contains(e.target)) {
                    this.sidebarOpen = false;
                }
            }
        });
    },
    methods: {
        // View management
        setView(view) {
            this.currentView = view;
            this.closeSidebar();
            
            if (view === 'hosts') {
                this.loadHosts();
            } else if (view === 'checks') {
                this.loadChecks();
            } else if (view === 'alerts') {
                this.loadAlerts();
            } else if (view === 'about') {
                this.loadBuildInfo();
            }
        },

        // Data loading methods
        async loadWebConfig() {
            this.webConfig = await window.RavenAPI.loadWebConfig();
        },

        async loadHosts() {
            this.loading = true;
            try {
                this.hosts = await window.RavenAPI.loadHosts();
            } catch (error) {
                console.error('Failed to load hosts:', error);
                window.RavenUtils.showNotification(this, 'error', 'Failed to load hosts');
            } finally {
                this.loading = false;
            }
        },

        async loadChecks() {
            this.loading = true;
            try {
                this.checks = await window.RavenAPI.loadChecks();
            } catch (error) {
                console.error('Failed to load checks:', error);
                window.RavenUtils.showNotification(this, 'error', 'Failed to load checks');
            } finally {
                this.loading = false;
            }
        },

        async loadStats() {
            try {
                const statuses = await window.RavenAPI.loadStatus(1000);
                
                this.stats = {
                    ok: statuses.filter(s => s.exit_code === 0).length,
                    warning: statuses.filter(s => s.exit_code === 1).length,
                    critical: statuses.filter(s => s.exit_code === 2).length,
                    unknown: statuses.filter(s => s.exit_code === 3).length
                };

                this.recentActivity = statuses
                    .sort((a, b) => new Date(b.timestamp) - new Date(a.timestamp))
                    .slice(0, 10)
                    .map(status => ({
                        id: status.id,
                        timestamp: status.timestamp,
                        host: status.host_id,
                        check: status.check_id,
                        status: window.RavenUtils.getStatusName(status.exit_code),
                        message: status.output
                    }));
            } catch (error) {
                console.error('Failed to load stats:', error);
            }
        },

        async loadAlerts() {
            this.loading = true;
            try {
                const statuses = await window.RavenAPI.loadStatus(100);
                
                this.alerts = statuses
                    .filter(status => status.exit_code > 0)
                    .map(status => ({
                        id: status.id || `${status.host_id}-${status.check_id}`,
                        timestamp: status.timestamp,
                        severity: window.RavenUtils.getStatusName(status.exit_code),
                        host: status.host_id,
                        check: status.check_id,
                        message: status.output || 'No message',
                        duration: window.RavenUtils.calculateDuration(status.timestamp)
                    }))
                    .sort((a, b) => {
                        const severityOrder = { critical: 3, warning: 2, unknown: 1 };
                        const aSev = severityOrder[a.severity] || 0;
                        const bSev = severityOrder[b.severity] || 0;
                        
                        if (aSev !== bSev) return bSev - aSev;
                        return new Date(b.timestamp) - new Date(a.timestamp);
                    });
                    
            } catch (error) {
                console.error('Failed to load alerts:', error);
                window.RavenUtils.showNotification(this, 'error', 'Failed to load alerts');
            } finally {
                this.loading = false;
            }
        },

        async loadBuildInfo() {
            this.loadingBuildInfo = true;
            try {
                this.buildInfo = await window.RavenAPI.loadBuildInfo();
            } catch (error) {
                console.error('Failed to load build info:', error);
                window.RavenUtils.showNotification(this, 'error', 'Failed to load build information');
            } finally {
                this.loadingBuildInfo = false;
            }
        },

        // UI control methods
        toggleSidebar() {
            this.sidebarOpen = !this.sidebarOpen;
        },

        closeSidebar() {
            this.sidebarOpen = false;
        },

        toggleSidebarCollapse() {
            this.sidebarCollapsed = !this.sidebarCollapsed;
            localStorage.setItem('sidebarCollapsed', this.sidebarCollapsed.toString());
        },

        toggleTheme() {
            this.theme = this.theme === 'light' ? 'dark' : 'light';
            localStorage.setItem('theme', this.theme);
            document.documentElement.setAttribute('data-theme', this.theme);
            this.settings.darkMode = this.theme === 'dark';
        },

        // Host management
        openAddHostModal() {
            this.editingHost = null;
            this.hostForm = window.RavenUtils.getEmptyHostForm();
            this.showHostModal = true;
        },

        editHost(host) {
            this.editingHost = host;
            this.hostForm = {
                name: host.name,
                display_name: host.display_name || '',
                ipv4: host.ipv4 || '',
                hostname: host.hostname || '',
                group: host.group || '',
                enabled: host.enabled
            };
            this.showHostModal = true;
        },

        closeHostModal() {
            this.showHostModal = false;
            this.editingHost = null;
            this.hostForm = window.RavenUtils.getEmptyHostForm();
        },

        async saveHost() {
            this.saving = true;
            try {
                if (this.editingHost) {
                    await window.RavenAPI.updateHost(this.editingHost.id, this.hostForm);
                    window.RavenUtils.showNotification(this, 'success', 'Host updated successfully');
                } else {
                    await window.RavenAPI.createHost(this.hostForm);
                    window.RavenUtils.showNotification(this, 'success', 'Host created successfully');
                }
                
                this.closeHostModal();
                this.loadHosts();
            } catch (error) {
                console.error('Failed to save host:', error);
                window.RavenUtils.showNotification(this, 'error', 'Failed to save host');
            } finally {
                this.saving = false;
            }
        },

        async deleteHost(host) {
            if (!confirm(`Are you sure you want to delete ${host.name}?`)) {
                return;
            }
            
            try {
                await window.RavenAPI.deleteHost(host.id);
                window.RavenUtils.showNotification(this, 'success', 'Host deleted successfully');
                this.loadHosts();
            } catch (error) {
                console.error('Failed to delete host:', error);
                window.RavenUtils.showNotification(this, 'error', 'Failed to delete host');
            }
        },

        // Check management
        openAddCheckModal() {
            this.editingCheck = null;
            this.checkForm = window.RavenUtils.getEmptyCheckForm();
            this.showCheckModal = true;
        },

        editCheck(check) {
            this.editingCheck = check;
            this.checkForm = {
                name: check.name,
                type: check.type,
                hosts: check.hosts || [],
                interval: check.interval || {},
                threshold: check.threshold || 3,
                timeout: check.timeout || '30s',
                enabled: check.enabled,
                options: check.options || {}
            };
            this.showCheckModal = true;
        },

        closeCheckModal() {
            this.showCheckModal = false;
            this.editingCheck = null;
            this.checkForm = window.RavenUtils.getEmptyCheckForm();
        },

        async saveCheck() {
            this.saving = true;
            try {
                if (this.editingCheck) {
                    await window.RavenAPI.updateCheck(this.editingCheck.id, this.checkForm);
                    window.RavenUtils.showNotification(this, 'success', 'Check updated successfully');
                } else {
                    await window.RavenAPI.createCheck(this.checkForm);
                    window.RavenUtils.showNotification(this, 'success', 'Check created successfully');
                }
                
                this.closeCheckModal();
                this.loadChecks();
            } catch (error) {
                console.error('Failed to save check:', error);
                window.RavenUtils.showNotification(this, 'error', 'Failed to save check');
            } finally {
                this.saving = false;
            }
        },

        async deleteCheck(check) {
            if (!confirm(`Are you sure you want to delete the check "${check.name}"?`)) {
                return;
            }
            
            try {
                await window.RavenAPI.deleteCheck(check.id);
                window.RavenUtils.showNotification(this, 'success', 'Check deleted successfully');
                this.loadChecks();
            } catch (error) {
                console.error('Failed to delete check:', error);
                window.RavenUtils.showNotification(this, 'error', 'Failed to delete check');
            }
        },

        // Settings actions
        exportConfig() {
            window.RavenUtils.showNotification(this, 'info', 'Configuration export not implemented yet');
        },

        refreshData() {
            this.loadStats();
            this.loadWebConfig();
            if (this.currentView === 'hosts') {
                this.loadHosts();
            } else if (this.currentView === 'checks') {
                this.loadChecks();
            } else if (this.currentView === 'alerts') {
                this.loadAlerts();
            }
            window.RavenUtils.showNotification(this, 'success', 'Data refreshed');
        },

        // Utility methods
        formatTime(timestamp) {
            return window.RavenUtils.formatTime(timestamp);
        },

        formatDuration(duration) {
            return window.RavenUtils.formatDuration(duration);
        },

        formatBuildTime(buildTime) {
            return window.RavenUtils.formatBuildTime(buildTime);
        },

        // WebSocket connection
        connectWebSocket() {
            const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
            const wsUrl = `${protocol}//${window.location.host}/ws`;
            
            this.websocket = new WebSocket(wsUrl);
            
            this.websocket.onopen = () => {
                this.connected = true;
                console.log('WebSocket connected');
            };
            
            this.websocket.onmessage = (event) => {
                const message = JSON.parse(event.data);
                if (message.type === 'status_update') {
                    this.loadStats();
                    if (this.currentView === 'hosts') {
                        this.loadHosts();
                    }
                }
            };
            
            this.websocket.onclose = () => {
                this.connected = false;
                console.log('WebSocket disconnected');
                setTimeout(() => this.connectWebSocket(), 5000);
            };
            
            this.websocket.onerror = (error) => {
                console.error('WebSocket error:', error);
                this.connected = false;
            };
        }
    }
}).mount('#app');
