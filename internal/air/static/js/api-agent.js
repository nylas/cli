/**
 * API Agent - Rules and policy endpoints for Nylas-managed accounts
 */
Object.assign(AirAPI, {
    async getPolicies() {
        return this.request('/policies');
    },

    async getRules() {
        return this.request('/rules');
    },

    async getWorkspace() {
        return this.request('/workspace');
    },

    async getAgentLists() {
        return this.request('/lists');
    }
});
