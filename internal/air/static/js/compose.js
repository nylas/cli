// ====================================
// COMPOSE MODULE
// Handles email composition and sending
// ====================================

const ComposeManager = {
    isOpen: false,
    mode: 'new', // 'new', 'reply', 'forward'
    currentDraftId: null,
    replyToEmail: null,
    autoSaveTimer: null,

    // Get compose modal elements
    getElements() {
        return {
            modal: document.getElementById('composeModal'),
            form: document.getElementById('composeForm'),
            to: document.getElementById('composeTo'),
            cc: document.getElementById('composeCc'),
            bcc: document.getElementById('composeBcc'),
            subject: document.getElementById('composeSubject'),
            body: document.getElementById('composeBody'),
            sendBtn: document.getElementById('composeSend'),
            saveBtn: document.getElementById('composeSave'),
            discardBtn: document.getElementById('composeDiscard'),
            closeBtn: document.querySelector('#composeModal .modal-close')
        };
    },

    init() {
        const els = this.getElements();
        if (!els.modal) return;

        // Send button
        if (els.sendBtn) {
            els.sendBtn.addEventListener('click', () => this.send());
        }

        // Save as draft button
        if (els.saveBtn) {
            els.saveBtn.addEventListener('click', () => this.saveDraft());
        }

        // Discard button
        if (els.discardBtn) {
            els.discardBtn.addEventListener('click', () => this.discard());
        }

        // Close button
        if (els.closeBtn) {
            els.closeBtn.addEventListener('click', () => this.close());
        }

        // Auto-save on input change (debounced)
        ['to', 'cc', 'bcc', 'subject', 'body'].forEach(field => {
            if (els[field]) {
                els[field].addEventListener('input', () => this.scheduleAutoSave());
            }
        });

        // Escape key to close
        document.addEventListener('keydown', (e) => {
            if (e.key === 'Escape' && this.isOpen) {
                this.close();
            }
        });

        // Keyboard shortcut: Ctrl/Cmd + Enter to send
        document.addEventListener('keydown', (e) => {
            if ((e.ctrlKey || e.metaKey) && e.key === 'Enter' && this.isOpen) {
                e.preventDefault();
                this.send();
            }
        });

        console.log('%c📝 Compose module loaded', 'color: #22c55e;');
    },

    open(mode = 'new') {
        const els = this.getElements();
        if (!els.modal) return;

        this.mode = mode;
        this.isOpen = true;
        this.currentDraftId = null;

        // Remove hidden state (class and inline style)
        els.modal.classList.remove('hidden');
        els.modal.style.display = '';
        els.modal.classList.add('active');
        els.modal.setAttribute('aria-hidden', 'false');

        // Focus the primary recipient field immediately so keyboard input
        // doesn't race against a delayed autofocus timer.
        if (els.to) {
            els.to.focus();
        }
    },

    openReply(email) {
        this.open('reply');
        this.replyToEmail = email;

        const els = this.getElements();
        const sender = email.from && email.from[0];
        const senderEmail = sender ? sender.email : '';

        // Pre-fill fields
        if (els.to) els.to.value = senderEmail;
        if (els.subject) els.subject.value = email.subject?.startsWith('Re:') ? email.subject : `Re: ${email.subject || ''}`;

        // Quote original message
        const quoteHeader = `\n\n-------- Original Message --------\nFrom: ${sender?.name || senderEmail}\nDate: ${new Date(email.date * 1000).toLocaleString()}\nSubject: ${email.subject || ''}\n\n`;
        if (els.body) els.body.value = quoteHeader + (email.snippet || '');
    },

    openForward(email) {
        this.open('forward');
        this.replyToEmail = email;

        const els = this.getElements();
        const sender = email.from && email.from[0];

        // Pre-fill subject
        if (els.subject) els.subject.value = email.subject?.startsWith('Fwd:') ? email.subject : `Fwd: ${email.subject || ''}`;

        // Forward with original content
        const fwdHeader = `\n\n-------- Forwarded Message --------\nFrom: ${sender?.name || sender?.email || 'Unknown'}\nDate: ${new Date(email.date * 1000).toLocaleString()}\nSubject: ${email.subject || ''}\nTo: ${email.to?.map(t => t.email).join(', ') || ''}\n\n`;
        if (els.body) els.body.value = fwdHeader + (email.snippet || '');
    },

    close() {
        const els = this.getElements();
        if (!els.modal) return;

        // Clear auto-save timer
        if (this.autoSaveTimer) {
            clearTimeout(this.autoSaveTimer);
            this.autoSaveTimer = null;
        }

        this.isOpen = false;
        els.modal.classList.remove('active');
        els.modal.classList.add('hidden');
        els.modal.setAttribute('aria-hidden', 'true');

        // Reset form
        this.reset();
    },

    reset() {
        const els = this.getElements();
        if (els.to) els.to.value = '';
        if (els.cc) els.cc.value = '';
        if (els.bcc) els.bcc.value = '';
        if (els.subject) els.subject.value = '';
        if (els.body) els.body.value = '';
        this.currentDraftId = null;
        this.replyToEmail = null;
        this.mode = 'new';
    },

    getFormData() {
        const els = this.getElements();
        const data = {
            to: this.parseRecipients(els.to?.value || ''),
            cc: this.parseRecipients(els.cc?.value || ''),
            bcc: this.parseRecipients(els.bcc?.value || ''),
            subject: els.subject?.value || '',
            body: els.body?.value || ''
        };

        // Pin the send to the grant the page was rendered for so the user's
        // visible "from" account matches what the backend actually uses,
        // even if the persisted default has drifted.
        const grantId = document.body && document.body.dataset && document.body.dataset.defaultGrantId;
        if (grantId) {
            data.grant_id = grantId;
        }

        // For replies, include reply_to_message_id and thread_id for proper threading
        if (this.mode === 'reply' && this.replyToEmail) {
            data.reply_to_message_id = this.replyToEmail.id;
            // Include thread_id if available to keep the conversation threaded
            if (this.replyToEmail.thread_id) {
                data.thread_id = this.replyToEmail.thread_id;
            }
        }

        return data;
    },

    parseRecipients(str) {
        if (!str.trim()) return [];
        return str.split(/[,;]/)
            .map(s => s.trim())
            .filter(s => s.length > 0)
            .map(email => ({ email }));
    },

    async send() {
        const els = this.getElements();
        const data = this.getFormData();

        // Validate recipients
        if (data.to.length === 0) {
            if (typeof showToast === 'function') {
                showToast('error', 'Error', 'Please add at least one recipient');
            }
            if (els.to) els.to.focus();
            return;
        }

        // Disable send button
        if (els.sendBtn) {
            els.sendBtn.disabled = true;
            els.sendBtn.textContent = 'Sending...';
        }

        try {
            // If we have a draft, send the draft
            if (this.currentDraftId) {
                await AirAPI.sendDraft(this.currentDraftId);
            } else {
                // Send directly
                await AirAPI.sendMessage(data);
            }

            if (typeof showToast === 'function') {
                showToast('success', 'Sent', 'Message sent successfully');
            }

            this.close();

            // Refresh email list if in sent folder
            if (typeof EmailListManager !== 'undefined' && EmailListManager.currentFolder === 'sent') {
                await EmailListManager.loadEmails('sent');
            }
        } catch (error) {
            console.error('Failed to send message:', error);
            if (typeof showToast === 'function') {
                showToast('error', 'Error', 'Failed to send message: ' + error.message);
            }
        } finally {
            if (els.sendBtn) {
                els.sendBtn.disabled = false;
                els.sendBtn.textContent = 'Send';
            }
        }
    },

    scheduleAutoSave() {
        // Debounce auto-save
        if (this.autoSaveTimer) {
            clearTimeout(this.autoSaveTimer);
        }
        this.autoSaveTimer = setTimeout(() => this.saveDraft(true), 3000);
    },

    async saveDraft(isAutoSave = false) {
        const data = this.getFormData();

        // Don't save empty drafts
        if (!data.to.length && !data.subject && !data.body) {
            return;
        }

        try {
            if (this.currentDraftId) {
                // Update existing draft
                await AirAPI.updateDraft(this.currentDraftId, data);
            } else {
                // Create new draft
                const result = await AirAPI.createDraft(data);
                if (result.draft && result.draft.id) {
                    this.currentDraftId = result.draft.id;
                }
            }

            if (!isAutoSave && typeof showToast === 'function') {
                showToast('info', 'Saved', 'Draft saved');
            }
        } catch (error) {
            console.error('Failed to save draft:', error);
            if (!isAutoSave && typeof showToast === 'function') {
                showToast('error', 'Error', 'Failed to save draft');
            }
        }
    },

    async discard() {
        // If we have a draft, delete it
        if (this.currentDraftId) {
            try {
                await AirAPI.deleteDraft(this.currentDraftId);
            } catch (error) {
                console.error('Failed to delete draft:', error);
            }
        }

        if (typeof showToast === 'function') {
            showToast('warning', 'Discarded', 'Draft discarded');
        }

        this.close();
    }
};

// Initialize when DOM is ready
document.addEventListener('DOMContentLoaded', () => {
    if (document.getElementById('composeModal')) {
        ComposeManager.init();
    }
});
