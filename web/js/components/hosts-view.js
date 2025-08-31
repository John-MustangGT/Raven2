// js/components/hosts-view.js
window.HostsView = {
    props: {
        hosts: Array,
        loading: Boolean,
        searchQuery: String,
        filterGroup: String,
        groups: Array,
        filteredHosts: Array
    },
    emits: ['update:search-query', 'update:filter-group', 'edit-host', 'delete-host'],
    methods: {
        formatTime(timestamp) {
            return window.RavenUtils.formatTime(timestamp);
        }
    },
    template: `
        <div class="data-table">
            <div class="table-header">
                <h3 class="table-title">Hosts ({{ hosts.length }})</h3>
                <div class="search-box">
                    <input 
                        :value="searchQuery"
                        @input="$emit('update:search-query', $event.target.value)"
                        class="search-input" 
                        placeholder="Search hosts..."
                        type="text"
                    >
                    <select :value="filterGroup" 
                            @change="$emit('update:filter-group', $event.target.value)"
                            class="form-input" style="width: auto;">
                        <option value="">All Groups</option>
                        <option v-for="group in groups" :key="group" :value="group">{{ group }}</option>
                    </select>
                </div>
            </div>
            <div class="table-content">
                <div v-if="loading" class="loading">
                    <div class="spinner"></div>
                    Loading hosts...
                </div>
                <table v-else>
                    <thead>
                        <tr>
                            <th>Name</th>
                            <th>Address</th>
                            <th>Group</th>
                            <th>Status</th>
                            <th>Last Check</th>
                            <th>Actions</th>
                        </tr>
                    </thead>
                    <tbody>
                        <tr v-for="(host, index) in filteredHosts" :key="host.id + '-' + index">
                            <td>
                                <div>
                                    <div style="font-weight: 500;">{{ host.display_name || host.name }}</div>
                                    <div style="font-size: 0.875rem; color: var(--text-muted);">{{ host.name }}</div>
                                </div>
                            </td>
                            <td>
                                <div>
                                    <div>{{ host.ipv4 || host.hostname || 'N/A' }}</div>
                                    <div v-if="host.ipv4 && host.hostname" style="font-size: 0.875rem; color: var(--text-muted);">
                                        {{ host.hostname }}
                                    </div>
                                </div>
                            </td>
                            <td>{{ host.group }}</td>
                            <td>
                                <span class="status-badge" :class="'status-' + host.status">
                                    <div class="status-indicator" :class="'status-' + host.status"></div>
                                    {{ host.status }}
                                </span>
                            </td>
                            <td>{{ formatTime(host.last_check) }}</td>
                            <td>
                                <div class="actions">
                                    <button class="btn btn-secondary btn-small" @click="$emit('edit-host', host)">
                                        <i class="fas fa-edit"></i>
                                    </button>
                                    <button class="btn btn-danger btn-small" @click="$emit('delete-host', host)">
                                        <i class="fas fa-trash"></i>
                                    </button>
                                </div>
                            </td>
                        </tr>
                    </tbody>
                </table>
            </div>
        </div>
    `
};
