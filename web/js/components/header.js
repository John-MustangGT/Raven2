// js/components/header.js
window.HeaderComponent = {
    props: {
        pageTitle: String,
        connected: Boolean,
        theme: String,
        currentView: String
    },
    emits: ['toggle-sidebar', 'toggle-theme', 'open-add-host-modal'],
    template: `
        <div class="header">
            <div style="display: flex; align-items: center; gap: 1rem;">
                <button class="menu-toggle" @click="$emit('toggle-sidebar')" title="Toggle menu">
                    <i class="fas fa-bars"></i>
                </button>
                <h1 class="page-title">{{ pageTitle }}</h1>
            </div>
            <div class="header-actions">
                <div class="connection-status">
                    <div class="connection-indicator" :class="{ disconnected: !connected }"></div>
                    {{ connected ? 'Connected' : 'Disconnected' }}
                </div>
                <button class="theme-toggle" @click="$emit('toggle-theme')" 
                        :title="theme === 'dark' ? 'Light mode' : 'Dark mode'">
                    <i :class="theme === 'dark' ? 'fas fa-sun' : 'fas fa-moon'"></i>
                </button>
                <button class="btn btn-primary" v-if="currentView === 'hosts'" 
                        @click="$emit('open-add-host-modal')">
                    <i class="fas fa-plus"></i>
                    <span class="btn-text">Add Host</span>
                </button>
            </div>
        </div>
    `
};
