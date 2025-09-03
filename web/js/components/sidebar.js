// js/components/sidebar.js
window.SidebarComponent = {
    props: {
        currentView: String,
        sidebarOpen: Boolean,
        sidebarCollapsed: Boolean,
        webConfig: Object
    },
    emits: ['set-view', 'toggle-sidebar-collapse'],
    template: `
        <div class="sidebar" :class="{ 'mobile-open': sidebarOpen, collapsed: sidebarCollapsed }">
            <button class="sidebar-toggle" @click="$emit('toggle-sidebar-collapse')" 
                    :title="sidebarCollapsed ? 'Expand sidebar' : 'Collapse sidebar'">
                <i :class="sidebarCollapsed ? 'fas fa-chevron-right' : 'fas fa-chevron-left'"></i>
            </button>
            <div class="logo">
                <h1>
                    <a :href="webConfig.header_link" target="_blank" rel="noopener noreferrer" 
                       class="logo-link" :title="'Visit: ' + webConfig.header_link">
                        <i class="fas fa-crow"></i>
                        <span v-if="!sidebarCollapsed">Raven</span>
                    </a>
                </h1>
            </div>
            <ul class="nav-menu">
                <li class="nav-item">
                    <a class="nav-link" :class="{ active: currentView === 'dashboard' }" 
                       @click="$emit('set-view', 'dashboard')">
                        <i class="fas fa-tachometer-alt"></i>
                        <span>Dashboard</span>
                    </a>
                </li>
                <li class="nav-item">
                    <a class="nav-link" :class="{ active: currentView === 'hosts' }" 
                       @click="$emit('set-view', 'hosts')">
                        <i class="fas fa-server"></i>
                        <span>Hosts</span>
                    </a>
                </li>
                <li class="nav-item">
                    <a class="nav-link" :class="{ active: currentView === 'checks' }" 
                       @click="$emit('set-view', 'checks')">
                        <i class="fas fa-check-circle"></i>
                        <span>Checks</span>
                    </a>
                </li>
                <li class="nav-item">
                    <a class="nav-link" :class="{ active: currentView === 'alerts' }" 
                       @click="$emit('set-view', 'alerts')">
                        <i class="fas fa-exclamation-triangle"></i>
                        <span>Alerts</span>
                    </a>
                </li>
                <li class="nav-item">
                    <a class="nav-link" :class="{ active: currentView === 'notifications' }" 
                       @click="$emit('set-view', 'notifications')">
                        <i class="fas fa-bell"></i>
                        <span>Notifications</span>
                    </a>
                </li>
                <li class="nav-item">
                    <a class="nav-link" :class="{ active: currentView === 'about' }" 
                       @click="$emit('set-view', 'about')">
                        <i class="fas fa-info-circle"></i>
                        <span>About</span>
                    </a>
                </li>
                <li class="nav-item">
                    <a class="nav-link" :class="{ active: currentView === 'settings' }" 
                       @click="$emit('set-view', 'settings')">
                        <i class="fas fa-cog"></i>
                        <span>Settings</span>
                    </a>
                </li>
            </ul>
        </div>
    `
};
