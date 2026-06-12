/**
 * Studio API client. Every mutation resolves to the fresh board state the
 * server returns, so callers re-render from server truth.
 */
window.StudioAPI = {
    async request(method, path, body) {
        const options = { method, headers: {} };
        if (body !== undefined) {
            options.headers['Content-Type'] = 'application/json';
            options.body = JSON.stringify(body);
        }
        const response = await fetch(path, options);
        const payload = await response.json().catch(() => ({}));
        if (!response.ok) {
            const error = new Error(payload.message || payload.error || ('request failed: ' + response.status));
            error.code = payload.error || '';
            error.status = response.status;
            throw error;
        }
        return payload;
    },

    getBoard() {
        return this.request('GET', '/api/board');
    },

    deleteAccount(id) {
        return this.request('DELETE', '/api/accounts/' + encodeURIComponent(id));
    },

    deletePolicy(id) {
        return this.request('DELETE', '/api/policies/' + encodeURIComponent(id));
    },

    deleteRule(id) {
        return this.request('DELETE', '/api/rules/' + encodeURIComponent(id));
    },

    deleteList(id) {
        return this.request('DELETE', '/api/lists/' + encodeURIComponent(id));
    },

    deleteWorkspace(id) {
        return this.request('DELETE', '/api/workspaces/' + encodeURIComponent(id));
    },

    patchWorkspace(id, body) {
        return this.request('PATCH', '/api/workspaces/' + encodeURIComponent(id), body);
    },

    createAccount(body) {
        return this.request('POST', '/api/accounts', body);
    },

    createWorkspace(body) {
        return this.request('POST', '/api/workspaces', body);
    },

    createPolicy(body) {
        return this.request('POST', '/api/policies', body);
    },

    updatePolicy(id, body) {
        return this.request('PATCH', '/api/policies/' + encodeURIComponent(id), body);
    },

    getPolicy(id) {
        return this.request('GET', '/api/policies/' + encodeURIComponent(id));
    },

    createRule(body) {
        return this.request('POST', '/api/rules', body);
    },

    updateRule(id, body) {
        return this.request('PATCH', '/api/rules/' + encodeURIComponent(id), body);
    },

    createList(body) {
        return this.request('POST', '/api/lists', body);
    },

    getListItems(id) {
        return this.request('GET', '/api/lists/' + encodeURIComponent(id) + '/items');
    },

    addListItems(id, items) {
        return this.request('POST', '/api/lists/' + encodeURIComponent(id) + '/items', { items });
    },

    removeListItems(id, items) {
        return this.request('DELETE', '/api/lists/' + encodeURIComponent(id) + '/items', { items });
    },

    moveAccount(id, workspaceID) {
        return this.request('POST', '/api/accounts/' + encodeURIComponent(id) + '/move', { workspace_id: workspaceID });
    },

    rotatePassword(id, appPassword) {
        return this.request('PATCH', '/api/accounts/' + encodeURIComponent(id), { app_password: appPassword });
    },

    sendTestEmail(grantID) {
        return this.request('POST', '/api/actions/test-email', { grant_id: grantID });
    }
};
