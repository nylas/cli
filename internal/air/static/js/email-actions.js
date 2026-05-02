/* Email Actions - User actions on emails */

// Track archive/delete operations in flight so a rapid double-click
// doesn't issue two API calls (each one sends a real Nylas request and,
// for archive, a toast with its own Undo timer pointing at a stale
// closure). The set is keyed by emailId; entries are removed when the
// API call resolves.
const inFlightDestructive = new Set();

// Build an "empty state" panel via DOM construction so we never
// re-introduce innerHTML to wipe the detail pane. textContent is
// XSS-safe by construction; the helper keeps archive/delete consistent.
function buildEmailDetailEmptyState(messageText) {
    const empty = document.createElement('div');
    empty.className = 'empty-state';
    const msg = document.createElement('div');
    msg.className = 'empty-message';
    msg.textContent = messageText || 'Select an email to view';
    empty.appendChild(msg);
    return empty;
}

Object.assign(EmailListManager, {
async optimisticUpdate(emailId, updateFn, apiCall, successMsg, rollbackFn) {
    // crypto.randomUUID guarantees no collision on rapid double-fire
    // (Date.now() resolution can repeat within a single millisecond).
    const operationId = (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function')
        ? `${emailId}-${crypto.randomUUID()}`
        : `${emailId}-${Date.now()}-${Math.random().toString(36).slice(2)}`;
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
            span.textContent = '★'; // ★ — DOM-safe equivalent of &#9733;
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
    // Guard against double-click. Each archive sends a real API call AND
    // a toast with its own Undo timer; without this, a frustrated user
    // tap-tapping the button would issue redundant requests and stack
    // multiple Undo toasts whose closures point at stale state.
    if (inFlightDestructive.has(emailId)) return false;

    const email = this.emails.find(e => e.id === emailId);
    if (!email) return false;

    // Resolve the archive payload differently per provider:
    //   - Microsoft / IMAP / EWS: a real Archive folder exists; move there.
    //   - Gmail: archive == remove the INBOX label, since labels are flat
    //     and there is no destination folder.
    // computeArchiveFolders returns null when neither path works (e.g. an
    // exotic provider where the email isn't in INBOX and we have no
    // Archive folder), in which case we surface a clear error.
    const newFolders = this.computeArchiveFolders(email);
    if (newFolders === null) {
        if (typeof showToast === 'function') {
            showToast('error', 'Archive unavailable', 'No archive target found for this account');
        }
        return false;
    }

    // Snapshot enough state to roll back the optimistic UI on failure.
    // Capture the index for splicing back, but rollback re-validates by
    // emailId since applyFilter may reorder between remove and failure.
    const originalIndex = this.emails.indexOf(email);
    const originalEmail = { ...email };
    const wasSelected = this.selectedEmailId === emailId;

    inFlightDestructive.add(emailId);

    // Optimistic removal — instant UI feedback. We restore the email
    // below if the API call fails, so the UI never lies for long.
    this.emails = this.emails.filter(e => e.id !== emailId);
    this.applyFilter();
    this.renderEmails();

    // Clear detail pane if this was selected. We rebuild the empty
    // state via DOM construction (textContent) instead of innerHTML so
    // there's no template-injection surface even though the strings
    // are currently literals.
    if (wasSelected) {
        this.selectedEmailId = null;
        const detailPane = document.querySelector('.email-detail');
        if (detailPane) {
            detailPane.replaceChildren(buildEmailDetailEmptyState());
        }
    }

    // restoreEmail re-inserts originalEmail and replays selection state
    // so the list and detail pane stay consistent on rollback. Looking
    // up by ID rather than splicing at the captured index protects
    // against applyFilter() reordering between optimistic remove and
    // failure — pin the row by identity, not position.
    //
    // Awaits selectEmail when restoring selection so that callers'
    // promises don't resolve while the detail pane is still showing
    // the empty state from the optimistic clear. selectEmail fetches
    // the full body, and "rolled back" should mean "fully visible
    // again", not "row is back but pane is empty".
    const restoreEmail = async () => {
        if (!this.emails.some(e => e.id === emailId)) {
            const insertAt = Math.min(originalIndex, this.emails.length);
            this.emails.splice(insertAt, 0, originalEmail);
        }
        this.applyFilter();
        this.renderEmails();
        if (wasSelected && typeof this.selectEmail === 'function') {
            await this.selectEmail(emailId);
        }
    };

    try {
        await AirAPI.updateEmail(emailId, { folders: newFolders });

        if (typeof showToast === 'function') {
            showToast('success', 'Archived', 'Email moved to archive', {
                action: 'Undo',
                onAction: async () => {
                    // Restore must succeed BEFORE we lie to the user that the
                    // email is back. The previous version spliced + toasted
                    // "Restored" unconditionally, so a failed restore PUT
                    // showed a green-path toast while the email stayed
                    // archived on the server — visible reality drifted from
                    // server reality on the next refresh.
                    //
                    // Refuse to send an empty folders array on Undo: a
                    // Gmail account that started without INBOX (rare but
                    // possible — drafts/sent paths) would otherwise land
                    // unfiled instead of restored. Surface a clear error
                    // and let the user investigate manually.
                    const restoreFolders = Array.isArray(originalEmail.folders) ? originalEmail.folders : [];
                    if (restoreFolders.length === 0) {
                        showToast('error', 'Restore unavailable', 'Original folder unknown — restore manually from archive');
                        return;
                    }
                    try {
                        await AirAPI.updateEmail(emailId, { folders: restoreFolders });
                    } catch (err) {
                        console.warn('[archive] undo failed for', emailId, err);
                        showToast('error', 'Restore failed', 'Email is still archived — please try again');
                        return;
                    }
                    await restoreEmail();
                    showToast('info', 'Restored', 'Email restored');
                }
            });
        }
        return true;
    } catch (err) {
        console.warn('[archive] failed for', emailId, err);
        // Roll back the optimistic removal — including the selection
        // state — so list and detail pane stay consistent.
        await restoreEmail();
        if (typeof showToast === 'function') {
            showToast('error', 'Archive failed', 'Could not archive — please try again');
        }
        return false;
    } finally {
        inFlightDestructive.delete(emailId);
    }
},

// computeArchiveFolders works out the new folders array for an archive
// action. Returns null when neither strategy applies, in which case the
// caller surfaces a "no archive target" error to the user.
computeArchiveFolders(email) {
    const current = Array.isArray(email.folders) ? email.folders.slice() : [];

    // Strategy 1 — provider exposes a typed Archive folder
    // (Microsoft/IMAP/EWS). Match by system_folder only, NOT name —
    // a user-created Gmail label literally named "Archive" must not be
    // mistaken for the system destination. Gmail accounts surface
    // "All Mail" as system_folder='all', not 'archive', so this branch
    // does not fire for Gmail and Strategy 2 still applies.
    //
    // Doing this BEFORE the INBOX-removal filter is the fix for an IMAP/
    // EWS account where folders=['INBOX'] (literal name): the old order
    // stripped INBOX first and sent folders:[] upstream, moving the
    // message out of every folder instead of into Archive.
    const archiveByType = (this.folders || []).find(
        f => (f.system_folder || '').toLowerCase() === 'archive'
    );
    if (archiveByType) {
        return [archiveByType.id];
    }

    // Strategy 2 — Gmail: archive means "remove the INBOX label".
    // Reached only when the account has no typed Archive folder, so
    // user-created labels remain safe.
    const filtered = current.filter(f => String(f).toUpperCase() !== 'INBOX');
    if (filtered.length !== current.length) {
        return filtered;
    }

    return null;
},

// findSystemFolder resolves a folder by its system_folder type. Falls back
// to a name-based lookup so demo mode (which sometimes ships folders
// without a populated system_folder) still works.
findSystemFolder(type) {
    if (!this.folders || !this.folders.length) return null;
    const byType = this.folders.find(f => (f.system_folder || '').toLowerCase() === type);
    if (byType) return byType;
    return this.folders.find(f => (f.name || '').toLowerCase() === type) || null;
},

async deleteEmail(emailId) {
    // Same in-flight guard as archive — each delete sends a real API
    // call, and the user has no recourse if we fire two by accident.
    if (inFlightDestructive.has(emailId)) return false;

    const email = this.emails.find(e => e.id === emailId);
    if (!email) return false;

    // Store original for undo
    const originalIndex = this.emails.indexOf(email);
    const originalEmail = { ...email };

    inFlightDestructive.add(emailId);

    // Optimistic removal - instant UI feedback
    this.emails = this.emails.filter(e => e.id !== emailId);
    this.applyFilter();
    this.renderEmails();

    // Clear detail pane if this was selected. Use DOM construction so
    // archive and delete share the same hardened empty-state path —
    // bypassing innerHTML even though the message is currently a literal.
    if (this.selectedEmailId === emailId) {
        this.selectedEmailId = null;
        const detailPane = document.querySelector('.email-detail');
        if (detailPane) {
            detailPane.replaceChildren(buildEmailDetailEmptyState());
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
        return true;
    } catch (error) {
        console.error('Failed to delete email:', error);

        // Rollback - restore email to list
        this.emails.splice(originalIndex, 0, originalEmail);
        this.applyFilter();
        this.renderEmails();

        if (typeof showToast === 'function') {
            showToast('error', 'Error', 'Failed to delete email');
        }
        return false;
    } finally {
        inFlightDestructive.delete(emailId);
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
