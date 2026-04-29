/**
 * Contacts States - Loading and error states
 */
Object.assign(ContactsManager, {
showLoadingState() {
    const container = document.getElementById('contactsList');
    if (!container) return;

    container.innerHTML = `
        <div class="loading-state">
            <div class="loading-spinner"></div>
            <div class="loading-text">Loading contacts...</div>
        </div>
    `;
},

showErrorState(message) {
    const container = document.getElementById('contactsList');
    if (!container) return;

    container.innerHTML = `
        <div class="error-state">
            <div class="error-icon">⚠️</div>
            <div class="error-message">${this.escapeHtml(message)}</div>
            <button class="btn btn-secondary" data-action="contacts-retry">Retry</button>
        </div>
    `;
},

showDetailError(message) {
    const container = document.getElementById('contactDetail');
    if (!container) return;

    container.innerHTML = `
        <div class="empty-state">
            <div class="empty-icon">⚠️</div>
            <div class="empty-title">Error</div>
            <div class="empty-message">${this.escapeHtml(message)}</div>
        </div>
    `;
},
});
