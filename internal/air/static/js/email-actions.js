/* Email Actions - User actions on emails */

Object.assign(EmailListManager, {
async optimisticUpdate(emailId, updateFn, apiCall, successMsg, rollbackFn) {
    const operationId = `${emailId}-${Date.now()}`;
    const email = this.emails.find(e => e.id === emailId);
    if (!email) return;

    // Store original state for rollback
    const originalState = JSON.parse(JSON.stringify(email));
    this.pendingOperations.set(operationId, { emailId, originalState });

    // Apply optimistic update immediately
    updateFn(email);
    this.updateEmailInUI(emailId);

    try {
        // Make API call
        await apiCall();

        // Success - clear pending operation FIRST, then re-render so the
        // 'pending-update' spinner class is removed. Previously the
        // delete happened but updateEmailInUI was never called again,
        // leaving the spinner ::after pseudo-element rendering forever.
        this.pendingOperations.delete(operationId);
        this.updateEmailInUI(emailId);
        if (successMsg && typeof showToast === 'function') {
            showToast('success', successMsg.title, successMsg.message);
        }
    } catch (error) {
        console.error(`Optimistic update failed for ${emailId}:`, error);

        // Rollback to original state
        if (rollbackFn) {
            rollbackFn(email, originalState);
        } else {
            Object.assign(email, originalState);
        }
        // Clear pending operation BEFORE re-rendering so the spinner
        // class is removed. Doing it the other way leaves the spinner
        // visible until the next unrelated state update.
        this.pendingOperations.delete(operationId);
        this.updateEmailInUI(emailId);

        if (typeof showToast === 'function') {
            showToast('error', 'Error', 'Failed to update. Changes reverted.');
        }
    }
},

// Update a single email item in the UI without full re-render
updateEmailInUI(emailId) {
    const email = this.emails.find(e => e.id === emailId);
    const item = document.querySelector(`.email-item[data-email-id="${emailId}"]`);
    if (!item || !email) return;

    // Update unread class
    item.classList.toggle('unread', email.unread);

    // Update starred indicator
    let starredEl = item.querySelector('.starred');
    if (email.starred && !starredEl) {
        const actionsEl = item.querySelector('.email-actions-mini');
        if (actionsEl) {
            const span = document.createElement('span');
            span.className = 'starred';
            span.title = 'Starred';
            span.innerHTML = '&#9733;';
            actionsEl.prepend(span);
        }
    } else if (!email.starred && starredEl) {
        starredEl.remove();
    }

    // Add visual feedback for pending operation
    if (this.hasPendingOperation(emailId)) {
        item.classList.add('pending-update');
    } else {
        item.classList.remove('pending-update');
    }
},

// Check if email has pending operation
hasPendingOperation(emailId) {
    for (const [, op] of this.pendingOperations) {
        if (op.emailId === emailId) return true;
    }
    return false;
},

async markAsRead(emailId) {
    // Optimistic update - instant UI feedback
    await this.optimisticUpdate(
        emailId,
        (email) => { email.unread = false; },
        () => AirAPI.updateEmail(emailId, { unread: false }),
        null, // Silent - no toast for mark as read
        (email, original) => { email.unread = original.unread; }
    );

    // Also update filtered emails
    this.applyFilter();
},

async toggleStar(emailId) {
    const email = this.emails.find(e => e.id === emailId);
    if (!email) return;

    const newStarred = !email.starred;

    // Optimistic update with immediate feedback
    await this.optimisticUpdate(
        emailId,
        (e) => { e.starred = newStarred; },
        () => AirAPI.updateEmail(emailId, { starred: newStarred }),
        { title: newStarred ? 'Starred' : 'Unstarred', message: newStarred ? 'Email starred' : 'Star removed' },
        (e, original) => { e.starred = original.starred; }
    );

    // Re-render detail view if viewing this email
    if (this.selectedEmailId === emailId && this.selectedEmailFull) {
        this.selectedEmailFull.starred = email.starred;
        this.renderEmailDetail(this.selectedEmailFull);
    }
},

async archiveEmail(emailId) {
    const email = this.emails.find(e => e.id === emailId);
    if (!email) return;

    // Store original position for undo
    const originalIndex = this.emails.indexOf(email);
    const originalEmail = { ...email };

    // Optimistic removal - instant UI feedback
    this.emails = this.emails.filter(e => e.id !== emailId);
    this.applyFilter();
    this.renderEmails();

    // Clear detail pane if this was selected
    if (this.selectedEmailId === emailId) {
        this.selectedEmailId = null;
        const detailPane = document.querySelector('.email-detail');
        if (detailPane) {
            detailPane.innerHTML = '<div class="empty-state"><div class="empty-message">Select an email to view</div></div>';
        }
    }

    // Show toast with undo option
    if (typeof showToast === 'function') {
        showToast('success', 'Archived', 'Email moved to archive', {
            action: 'Undo',
            onAction: () => {
                // Restore email
                this.emails.splice(originalIndex, 0, originalEmail);
                this.applyFilter();
                this.renderEmails();
                showToast('info', 'Restored', 'Email restored');
            }
        });
    }

    // Note: Archive API call would go here if available
},

async deleteEmail(emailId) {
    const email = this.emails.find(e => e.id === emailId);
    if (!email) return;

    // Store original for undo
    const originalIndex = this.emails.indexOf(email);
    const originalEmail = { ...email };

    // Optimistic removal - instant UI feedback
    this.emails = this.emails.filter(e => e.id !== emailId);
    this.applyFilter();
    this.renderEmails();

    // Clear detail pane if this was selected
    if (this.selectedEmailId === emailId) {
        this.selectedEmailId = null;
        const detailPane = document.querySelector('.email-detail');
        if (detailPane) {
            detailPane.innerHTML = '<div class="empty-state"><div class="empty-message">Select an email to view</div></div>';
        }
    }

    try {
        await AirAPI.deleteEmail(emailId);

        if (typeof showToast === 'function') {
            showToast('warning', 'Deleted', 'Email moved to trash', {
                action: 'Undo',
                onAction: async () => {
                    // Note: Undo delete would require undelete API
                    showToast('info', 'Note', 'Check trash to restore');
                }
            });
        }
    } catch (error) {
        console.error('Failed to delete email:', error);

        // Rollback - restore email to list
        this.emails.splice(originalIndex, 0, originalEmail);
        this.applyFilter();
        this.renderEmails();

        if (typeof showToast === 'function') {
            showToast('error', 'Error', 'Failed to delete email');
        }
    }
},

replyToEmail(emailId) {
    // Use full email data (includes thread_id) if available
    const email = this.selectedEmailFull && this.selectedEmailFull.id === emailId
        ? this.selectedEmailFull
        : this.emails.find(e => e.id === emailId);
    if (email && typeof ComposeManager !== 'undefined') {
        ComposeManager.openReply(email);
    }
},

forwardEmail(emailId) {
    // Use full email data if available
    const email = this.selectedEmailFull && this.selectedEmailFull.id === emailId
        ? this.selectedEmailFull
        : this.emails.find(e => e.id === emailId);
    if (email && typeof ComposeManager !== 'undefined') {
        ComposeManager.openForward(email);
    }
},

});
