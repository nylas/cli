/* Undo Send Manager - Email undo functionality */

const UndoSendManager = {
    config: { enabled: true, grace_period_sec: 10 },
    pendingSends: new Map(),
    currentPendingId: null,

    async init() {
        try {
            const response = await AirAPI.getUndoSendConfig();
            this.config = response.config || this.config;
        } catch (error) {
            console.error('Failed to load undo send config:', error);
        }
        console.log('%c↩️ Undo Send module loaded', 'color: #22c55e;');
    },

    // Called from HTML template button - send with undo from compose
    async sendWithUndoFromCompose() {
        // Close the dropdown first
        if (typeof ScheduledSendManager !== 'undefined') {
            ScheduledSendManager.closeDropdown();
        }

        if (typeof ComposeManager === 'undefined') {
            if (typeof showToast === 'function') {
                showToast('error', 'Error', 'Compose not available');
            }
            return;
        }

        const data = ComposeManager.getFormData();
        if (!data.to || data.to.length === 0) {
            if (typeof showToast === 'function') {
                showToast('error', 'Error', 'Please add at least one recipient');
            }
            return;
        }

        try {
            await this.sendWithUndo(data);
            ComposeManager.close();
        } catch (error) {
            if (typeof showToast === 'function') {
                showToast('error', 'Error', 'Failed to send: ' + error.message);
            }
        }
    },

    // Undo the current pending send (called from undo toast button)
    async undo() {
        if (this.currentPendingId) {
            await this.cancelSend(this.currentPendingId);
        }
    },

    async sendWithUndo(messageData) {
        if (!this.config.enabled) {
            // Send immediately without undo
            return AirAPI.sendMessage(messageData);
        }

        try {
            const result = await AirAPI.sendWithUndo(messageData);

            if (result.pending_id) {
                this.showUndoToast(result.pending_id, result.send_at);
            }

            return result;
        } catch (error) {
            throw error;
        }
    },

    showUndoToast(pendingId, sendAt) {
        const gracePeriod = this.config.grace_period_sec || 10;
        let remaining = gracePeriod;

        // Store current pending ID for the undo() method
        this.currentPendingId = pendingId;

        // Use the existing toast from the HTML template if available
        let toast = document.getElementById('undoToast');
        if (!toast) {
            // Create undo toast
            toast = document.createElement('div');
            toast.className = 'undo-toast';
            toast.id = 'undoToast';
            toast.innerHTML = `
                <div class="undo-content">
                    <span class="undo-message">Message sent</span>
                    <span class="undo-timer">${remaining}s</span>
                </div>
                <button class="undo-btn" data-action="undo-send">Undo</button>
            `;
            document.body.appendChild(toast);
        }

        // Update timer display
        const timerEl = toast.querySelector('.undo-timer') || toast.querySelector('#undoTimer');
        if (timerEl) timerEl.textContent = `${remaining}s`;

        setTimeout(() => toast.classList.add('active'), 10);

        // Store pending send
        this.pendingSends.set(pendingId, { toast, sendAt });

        // Countdown timer
        const interval = setInterval(() => {
            remaining--;
            if (timerEl) timerEl.textContent = `${remaining}s`;

            if (remaining <= 0) {
                clearInterval(interval);
                this.removePendingSend(pendingId, false);
            }
        }, 1000);

        // Store interval for cleanup
        this.pendingSends.get(pendingId).interval = interval;
    },

    async cancelSend(pendingId) {
        try {
            await AirAPI.cancelPendingSend(pendingId);
            this.removePendingSend(pendingId, true);

            if (typeof showToast === 'function') {
                showToast('info', 'Cancelled', 'Message cancelled');
            }

            // Reopen compose with the message content
            // (would need to store message data for this)
        } catch (error) {
            console.error('Failed to cancel send:', error);
            if (typeof showToast === 'function') {
                showToast('error', 'Error', 'Could not cancel: ' + error.message);
            }
        }
    },

    removePendingSend(pendingId, cancelled) {
        const pending = this.pendingSends.get(pendingId);
        if (pending) {
            if (pending.interval) {
                clearInterval(pending.interval);
            }
            if (pending.toast) {
                pending.toast.classList.remove('active');
                // Don't remove the toast from DOM if it's the template one
                if (pending.toast.id !== 'undoToast') {
                    setTimeout(() => pending.toast.remove(), 300);
                }
            }
        }
        this.pendingSends.delete(pendingId);
        this.currentPendingId = null;
    }
};

// =============================================================================
// TEMPLATES MANAGER
// =============================================================================

