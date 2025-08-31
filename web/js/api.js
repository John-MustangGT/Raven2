// js/api.js - API service layer
window.RavenAPI = {
    async loadWebConfig() {
        try {
            const response = await axios.get('/api/web-config');
            return response.data.data;
        } catch (error) {
            console.error('Failed to load web config:', error);
            return {
                header_link: 'https://github.com/John-MustangGT/raven2',
                serve_static: false,
                root: 'index.html'
            };
        }
    },

    async loadHosts() {
        const response = await axios.get('/api/hosts');
        return response.data.data || [];
    },

    async loadChecks() {
        const response = await axios.get('/api/checks');
        return response.data.data || [];
    },

    async loadStatus(limit = 100) {
        const response = await axios.get(`/api/status?limit=${limit}`);
        return response.data.data || [];
    },

    async loadBuildInfo() {
        const response = await axios.get('/api/build-info');
        return response.data.data;
    },

    async createHost(hostData) {
        await axios.post('/api/hosts', hostData);
    },

    async updateHost(id, hostData) {
        await axios.put(`/api/hosts/${id}`, hostData);
    },

    async deleteHost(id) {
        await axios.delete(`/api/hosts/${id}`);
    },

    async createCheck(checkData) {
        await axios.post('/api/checks', checkData);
    },

    async updateCheck(id, checkData) {
        await axios.put(`/api/checks/${id}`, checkData);
    },

    async deleteCheck(id) {
        await axios.delete(`/api/checks/${id}`);
    }
};
