// js/components/check-modal.js
window.CheckModal = {
    props: {
        show: Boolean,
        editing: Object,
        form: Object,
        hosts: Array,
        saving: Boolean
    },
    emits: ['close', 'save'],
    template: `
        <div class="modal-overlay" :class="{ active: show }" @click.self="$emit('close')">
            <div class="modal">
                <div class="modal-header">
                    <h3 class="modal-title">{{ editing ? 'Edit Check' : 'Add Check' }}</h3>
                    <button class="close-btn" @click="$emit('close')">
                        <i class="fas fa-times"></i>
                    </button>
                </div>
                <form @submit.prevent="$emit('save')">
                    <div class="form-group">
                        <label class="form-label">Name *</label>
                        <input 
                            v-model="form.name" 
                            class="form-input" 
                            type="text" 
                            required
                            placeholder="e.g., Ping Check"
                        >
                    </div>
                    <div class="form-group">
                        <label class="form-label">Type *</label>
                        <select v-model="form.type" class="form-input" required>
                            <option value="ping">Ping</option>
                            <option value="nagios">Nagios Plugin</option>
                            <option value="http">HTTP</option>
                            <option value="https">HTTPS</option>
                        </select>
                    </div>
                    <div class="form-group">
                        <label class="form-label">Hosts *</label>
                        <select v-model="form.hosts" class="form-input" multiple size="4" required>
                            <option v-for="host in hosts" :key="host.id" :value="host.id">
                                {{ host.display_name || host.name }} ({{ host.ipv4 || host.hostname }})
                            </option>
                        </select>
                        <small style="color: var(--text-muted);">Hold Ctrl/Cmd to select multiple hosts</small>
                    </div>
                    <div class="form-group">
                        <label class="form-label">Check Intervals</label>
                        <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 1rem;">
                            <div>
                                <label class="form-label" style="font-size: 0.875rem;">OK</label>
                                <input v-model="form.interval.ok" class="form-input" placeholder="5m">
                            </div>
                            <div>
                                <label class="form-label" style="font-size: 0.875rem;">Warning</label>
                                <input v-model="form.interval.warning" class="form-input" placeholder="2m">
                            </div>
                            <div>
                                <label class="form-label" style="font-size: 0.875rem;">Critical</label>
                                <input v-model="form.interval.critical" class="form-input" placeholder="1m">
                            </div>
                            <div>
                                <label class="form-label" style="font-size: 0.875rem;">Unknown</label>
                                <input v-model="form.interval.unknown" class="form-input" placeholder="1m">
                            </div>
                        </div>
                    </div>
                    <div class="form-group">
                        <label class="form-label">Threshold</label>
                        <input 
                            v-model="form.threshold" 
                            class="form-input" 
                            type="number"
                            min="1"
                            max="10"
                            placeholder="3"
                        >
                        <small style="color: var(--text-muted);">Number of consecutive failures before status change</small>
                    </div>
                    <div class="form-group">
                        <label class="form-label">Timeout</label>
                        <input 
                            v-model="form.timeout" 
                            class="form-input" 
                            type="text"
                            placeholder="30s"
                        >
                    </div>
                    <div class="form-group">
                        <label class="form-checkbox">
                            <input v-model="form.enabled" type="checkbox">
                            <span>Enabled</span>
                        </label>
                    </div>
                    <div class="form-actions">
                        <button type="button" class="btn btn-secondary" @click="$emit('close')">Cancel</button>
                        <button type="submit" class="btn btn-primary" :disabled="saving">
                            {{ saving ? 'Saving...' : 'Save' }}
                        </button>
                    </div>
                </form>
            </div>
        </div>
    `
};
