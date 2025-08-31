// js/components/checks-view.js
window.ChecksView = {
    props: {
        checks: Array,
        loading: Boolean,
        searchChecksQuery: String,
        filterCheckType: String,
        checkTypes: Array,
        filteredChecks: Array
    },
    emits: ['update:search-checks-query', 'update:filter-check-type', 'open-add-check-modal', 'edit-check', 'delete-check'],
    template: `
        <div class="data-table">
            <div class="table-header">
                <h3 class="table-title">Checks ({{ checks.length }})</h3>
                <div class="search-box">
                    <input 
                        :value="searchChecksQuery"
                        @input="$emit('update:search-checks-query', $event.target.value)"
                        class="search-input" 
                        placeholder="Search checks..."
                        type="text"
                    >
                    <select :value="filterCheckType" 
                            @change="$emit('update:filter-check-type', $event.target.value)"
                            class="form-input" style="width: auto;">
                        <option value="">All Types</option>
                        <option v-for="type in checkTypes" :key="type" :value="type">{{ type }}</option>
                    </select>
                    <button class="btn btn-primary" @click="$emit('open-add-check-modal')">
                        <i class="fas fa-plus"></i>
                        <span class="btn-text">Add Check</span>
                    </button>
                </div>
            </div>
            <div class="table-content">
                <div v-if="loading" class="loading">
                    <div class="spinner"></div>
                    Loading checks...
                </div>
                
                <div v-else-if="checks.length === 0" style="padding: 1rem; background: var(--light-bg); margin: 1rem; border-radius: 0.5rem;">
                    <h4>No checks configured</h4>
                    <p>Click "Add Check" to create your first monitoring check.</p>
                </div>
                
                <table v-else>
                    <thead>
                        <tr>
                            <th>Name</th>
                            <th>Type</th>
                            <th>Hosts</th>
                            <th>Interval</th>
                            <th>Status</th>
                            <th>Actions</th>
                        </tr>
                    </thead>
                    <tbody>
                        <tr v-for="(check, index) in filteredChecks" :key="(check.id || index) + '-' + index">
                            <td>
                                <div>
                                    <div style="font-weight: 500;">{{ check.name || 'Unnamed Check' }}</div>
                                    <div style="font-size: 0.875rem; color: var(--text-muted);">{{ check.id || 'No ID' }}</div>
                                </div>
                            </td>
                            <td>
                                <span class="status-badge" style="background: var(--light-bg); color: var(--text-primary);">
                                    {{ check.type || 'Unknown' }}
                                </span>
                            </td>
                            <td>{{ (check.hosts && check.hosts.length) || 0 }} hosts</td>
                            <td>
                                <div style="font-size: 0.875rem;">
                                    <div>OK: {{ (check.interval && check.interval.ok) || 'N/A' }}</div>
                                    <div>Critical: {{ (check.interval && check.interval.critical) || 'N/A' }}</div>
                                </div>
                            </td>
                            <td>
                                <span v-if="check.enabled" class="status-badge status-ok">
                                    <div class="status-indicator status-ok"></div>
                                    Enabled
                                </span>
                                <span v-else class="status-badge status-unknown">
                                    <div class="status-indicator status-unknown"></div>
                                    Disabled
                                </span>
                            </td>
                            <td>
                                <div class="actions">
                                    <button class="btn btn-secondary btn-small" @click="$emit('edit-check', check)">
                                        <i class="fas fa-edit"></i>
                                    </button>
                                    <button class="btn btn-danger btn-small" @click="$emit('delete-check', check)">
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
