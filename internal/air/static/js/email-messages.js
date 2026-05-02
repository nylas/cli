/* Email Messages - Loading and pagination */

// debug() gates the noisy lifecycle traces that were left behind during
// the loadEmails refactor. Set window.AIR_DEBUG=true in devtools to
// re-enable them; production output stays quiet so the dev console isn't
// drowning every keystroke.
function debug(...args) {
    if (typeof window !== 'undefined' && window.AIR_DEBUG) {
        console.debug(...args);
    }
}

Object.assign(EmailListManager, {
// loadEmails returns one of three string outcomes:
//   - 'loaded'      → fetch succeeded and the list re-rendered
//   - 'in-progress' → another loadEmails was already running; the caller
//                     should treat this as a benign no-op, NOT a failure
//   - 'failed'      → fetch raised; an error toast was shown internally
//
// Returning a string outcome — rather than rethrowing — keeps long-standing
// callers (compose.js, app-init.js, settings.js) working unchanged. The
// boolean predecessor conflated "skipped because already loading" with
// "fetch failed", so the mobile pull-to-refresh handler showed
// "Refresh failed" when the user pulled twice in quick succession even
// though no fetch had failed.
//
// For backwards compatibility callers can still treat the return value
// as truthy/falsy: 'loaded' and 'in-progress' are truthy, 'failed' is a
// non-empty string but distinguishable; new callers should compare
// explicitly via `result === 'loaded'`.
async loadEmails(folderOrOptions = null) {
    if (this.isLoading) return 'in-progress';
    this.isLoading = true;

    // Support both string (folder) and object (options) parameter
    let folder = null;
    let search = null;
    let from = null;
    if (typeof folderOrOptions === 'string') {
        folder = folderOrOptions;
    } else if (folderOrOptions && typeof folderOrOptions === 'object') {
        folder = folderOrOptions.folder || null;
        search = folderOrOptions.search || null;
        from = folderOrOptions.from || null;
    }

    debug('[loadEmails] Starting...', { folder, search, from, limit: 50 });

    let outcome = 'failed';
    try {
        const options = { limit: 50 }; // Increased from 10 to 50 to fill viewport
        if (folder) {
            this.currentFolder = folder;
            options.folder = folder;
        }
        if (search) {
            this.currentSearch = search;
            options.search = search;
        }
        if (from) {
            options.from = from;
        }

        const data = await AirAPI.getEmails(options);
        debug('[loadEmails] API response:', {
            emailCount: data.emails?.length || 0,
            hasMore: data.has_more,
            nextCursor: data.next_cursor
        });

        this.emails = data.emails || [];
        this.nextCursor = data.next_cursor;
        this.hasMore = data.has_more;

        // Apply current filter
        this.applyFilter();
        this.renderEmails();
        outcome = 'loaded';
    } catch (error) {
        console.error('[loadEmails] Failed to load emails:', error);
        if (typeof showToast === 'function') {
            showToast('error', 'Error', 'Failed to load emails');
        }
    } finally {
        this.isLoading = false;
        debug('[loadEmails] Complete. Triggering ensureScrollable...');
        // Auto-load more AFTER isLoading is set to false
        // Use setTimeout to ensure DOM has updated
        setTimeout(() => this.ensureScrollable(), 100);
    }
    return outcome;
},

async loadMore() {
    if (!this.hasMore || !this.nextCursor || this.isLoading) return;
    this.isLoading = true;

    try {
        const data = await AirAPI.getEmails({
            folder: this.currentFolder,
            cursor: this.nextCursor
        });

        this.emails = [...this.emails, ...(data.emails || [])];
        this.nextCursor = data.next_cursor;
        this.hasMore = data.has_more;

        // Apply current filter
        this.applyFilter();
        this.renderEmails(true); // Append mode
    } catch (error) {
        console.error('Failed to load more emails:', error);
    } finally {
        this.isLoading = false;
        // Auto-load more AFTER isLoading is set to false
        // Use setTimeout to ensure DOM has updated
        setTimeout(() => this.ensureScrollable(), 100);
    }
},

// Ensure the email list has enough content to be scrollable
// Auto-loads more emails if content doesn't fill viewport
ensureScrollable() {
    const emailList = document.querySelector('.email-list');

    debug('[ensureScrollable] Starting check...', {
        hasEmailList: !!emailList,
        hasMore: this.hasMore,
        isLoading: this.isLoading,
        emailCount: this.emails.length,
        nextCursor: this.nextCursor
    });

    if (!emailList) {
        console.warn('[ensureScrollable] Email list element not found');
        return;
    }

    if (!this.hasMore) {
        debug('[ensureScrollable] No more emails to load (hasMore=false)');
        return;
    }

    if (this.isLoading) {
        debug('[ensureScrollable] Already loading, skipping');
        return;
    }

    // Check if content fills viewport (has scrollbar)
    const scrollHeight = emailList.scrollHeight;
    const clientHeight = emailList.clientHeight;
    const needsMore = scrollHeight <= clientHeight;

    debug('[ensureScrollable] Viewport check:', {
        scrollHeight,
        clientHeight,
        needsMore,
        hasScrollbar: scrollHeight > clientHeight
    });

    if (needsMore) {
        debug('[ensureScrollable] Loading more emails to fill viewport...');
        setTimeout(() => this.loadMore(), 100);
    } else {
        debug('[ensureScrollable] Viewport is full, stopping auto-load');
    }
},

renderEmails(append = false) {
    const emailList = document.querySelector('.email-list');
    if (!emailList) return;

    // Use filtered emails for display
    const displayEmails = this.getDisplayEmails();

    debug('[renderEmails]', {
        totalEmails: this.emails.length,
        filteredEmails: this.filteredEmails.length,
        currentFilter: this.currentFilter,
        displayCount: displayEmails.length,
        append,
        virtualScrollEnabled: this.virtualScroll.enabled
    });

    if (displayEmails.length === 0 && !append) {
        const isInbox = this.currentFolder === this.inboxFolderId ||
                       this.currentFolder === 'INBOX' ||
                       this.currentFolder === 'inbox';
        const emptyMessages = {
            'vip': { icon: '⭐', title: 'No VIP emails', message: 'Add VIP senders to see their emails here' },
            'unread': { icon: '✓', title: 'All caught up!', message: 'No unread emails', celebrate: isInbox },
            'all': isInbox
                ? { icon: '🎉', title: 'Inbox Zero!', message: 'You\'ve conquered your inbox. Take a moment to celebrate!', celebrate: true }
                : { icon: '📭', title: 'No emails', message: 'This folder is empty' }
        };
        const msg = emptyMessages[this.currentFilter] || emptyMessages.all;

        // DOM-construct the empty state so future copy changes can't
        // accidentally introduce HTML interpolation. icon/title/message
        // are static today but the dictionary is a tempting place to
        // start mixing in a folder name later.
        const empty = document.createElement('div');
        empty.className = 'empty-state inbox-zero' + (msg.celebrate ? ' celebration' : '');

        const iconEl = document.createElement('div');
        iconEl.className = 'empty-icon';
        iconEl.textContent = msg.icon;
        empty.appendChild(iconEl);

        const titleEl = document.createElement('div');
        titleEl.className = 'empty-title';
        titleEl.textContent = msg.title;
        empty.appendChild(titleEl);

        const messageEl = document.createElement('div');
        messageEl.className = 'empty-message';
        messageEl.textContent = msg.message;
        empty.appendChild(messageEl);

        if (msg.celebrate) {
            const sub = document.createElement('div');
            sub.className = 'inbox-zero-subtitle';
            sub.textContent = '✨ Enjoy the moment';
            empty.appendChild(sub);
        }

        emailList.replaceChildren(empty);

        // Trigger celebration confetti for Inbox Zero
        if (msg.celebrate && typeof window.celebrateInboxZero === 'function') {
            setTimeout(() => window.celebrateInboxZero(), 300);
        }
        return;
    }

    // Use virtual scrolling for large lists
    if (this.virtualScroll.enabled && displayEmails.length > 20) {
        if (!append) {
            emailList.innerHTML = '';
            this.initVirtualScroll();
        }

        // Calculate initial visible range
        const { itemHeight, bufferSize } = this.virtualScroll;
        const viewportHeight = emailList.clientHeight || 600;
        this.virtualScroll.visibleStart = 0;
        this.virtualScroll.visibleEnd = Math.min(
            displayEmails.length,
            Math.ceil(viewportHeight / itemHeight) + bufferSize * 2
        );

        this.renderVirtualEmails();
    } else {
        // Standard rendering for small lists
        if (!append) {
            emailList.innerHTML = '';
        }

        const fragment = document.createDocumentFragment();
        displayEmails.forEach(email => {
            const isSelected = email.id === this.selectedEmailId;
            const item = EmailRenderer.renderEmailItem(email, isSelected);
            fragment.appendChild(item);
        });

        if (append) {
            emailList.appendChild(fragment);
        } else {
            emailList.appendChild(fragment);
        }
    }
},
});
