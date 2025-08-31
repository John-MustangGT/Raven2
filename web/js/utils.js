// js/utils.js - Utility functions
window.RavenUtils = {
    formatTime(timestamp) {
        if (!timestamp) return 'Never';
        return new Date(timestamp).toLocaleString();
    },

    formatDuration(duration) {
        if (!duration) return 'Just now';
        
        const seconds = Math.floor(duration / 1000);
        const minutes = Math.floor(seconds / 60);
        const hours = Math.floor(minutes / 60);
        const days = Math.floor(hours / 24);
        
        if (days > 0) return `${days}d ${hours % 24}h`;
        if (hours > 0) return `${hours}h ${minutes % 60}m`;
        if (minutes > 0) return `${minutes}m`;
        return `${seconds}s`;
    },

    formatBuildTime(buildTime) {
        if (!buildTime || buildTime === 'unknown') return 'Unknown';
        try {
            return new Date(buildTime).toLocaleString();
        } catch {
            return buildTime;
        }
    },

    getStatusName(exitCode) {
        switch (exitCode) {
            case 0: return 'ok';
            case 1: return 'warning';
            case 2: return 'critical';
            default: return 'unknown';
        }
    },

    calculateDuration(timestamp) {
        if (!timestamp) return 0;
        return Date.now() - new Date(timestamp).getTime();
    },

    showNotification(app, type, message) {
        app.notification = { show: true, type, message };
        setTimeout(() => {
            app.notification.show = false;
        }, 5000);
    },

    getEmptyHostForm() {
        return {
            name: '',
            display_name: '',
            ipv4: '',
            hostname: '',
            group: 'default',
            enabled: true
        };
    },

    getEmptyCheckForm() {
        return {
            name: '',
            type: 'ping',
            hosts: [],
            interval: {
                ok: '5m',
                warning: '2m',
                critical: '1m',
                unknown: '1m'
            },
            threshold: 3,
            timeout: '30s',
            enabled: true,
            options: {}
        };
    }
};
