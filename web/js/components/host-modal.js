// js/components/host-modal.js
window.HostModal = {
    props: {
        show: Boolean,
        editing: Object,
        form: Object,
        saving: Boolean
    },
    emits: ['close', 'save'],
    template: `
        <div class="modal-overlay" :class="{ active: show }" @click.self="$emit('close')">
            <div class="modal">
                <div class="modal-header">
                    <h3 class="modal-title">{{ editing ? 'Edit Host' : 'Add Host' }}</h3>
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
                            placeholder="e.g., web-server-01"
                        >
                    </div>
                    <div class="form-group">
                        <label class="form-label">Display Name</label>
                        <input 
                            v-model="form.display_name" 
                            class="form-input" 
                            type="text"
                            placeholder="e.g., Web Server 1"
                        >
                    </div>
                    <div class="form-group">
                        <label class="form-label">IP Address</label>
                        <input 
                            v-model="form.ipv4" 
                            class="form-input" 
                            type="text"
                        >
                    </div>
                    <div class="form-group">
                        <label class="form-label">Hostname</label>
                        <input 
                            v-model="form.hostname" 
                            class="form-input" 
                            type="text"
                            placeholder="e.g., web01.example.com"
                        >
                    </div>
                    <div class="form-group">
                        <label class="form-label">Group</label>
                        <input 
                            v-model="form.group" 
                            class="form-input" 
                            type="text"
                            placeholder="e.g., servers"
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
