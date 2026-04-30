/* Email Core - State and Initialization */

const EmailListManager = {
    currentFolder: 'INBOX',
    currentFilter: 'all',
    emails: [],
    filteredEmails: [],
    folders: [],
    inboxFolderId: null,
    vipSenders: [],
    selectedEmailId: null,
    selectedEmailFull: null,
    nextCursor: null,
    hasMore: false,
    isLoading: false,

    virtualScroll: {
        enabled: true,
        itemHeight: 76,
        bufferSize: 5,
        visibleStart: 0,
        visibleEnd: 0,
        scrollContainer: null,
        totalHeight: 0
    },

    cache: {
        emails: new Map(),
        folders: null,
        foldersTimestamp: 0,
        cacheDuration: 60000
    },

    pendingOperations: new Map(),

    async init() {
        // Set up event listeners first (UI is ready immediately)
        this.setupEventListeners();
        this.setupFilterTabs();

        // Load folders first, then emails from inbox (sequential to avoid rate limits)
        try {
            await this.loadFolders();
        } catch (error) {
            console.error('Failed to load folders:', error);
        }

        // Load VIP senders list in background
        this.loadVIPSenders().catch(err => console.error('Failed to load VIP senders:', err));

        try {
            // Load inbox emails by default - use actual folder ID if available
            const inboxId = this.inboxFolderId || 'INBOX';
            await this.loadEmails(inboxId);
        } catch (error) {
            console.error('Failed to load emails:', error);
            if (typeof showToast === 'function') {
                showToast('error', 'Error', 'Failed to load emails. Will retry...');
            }
            // Retry after delay
            const inboxId = this.inboxFolderId || 'INBOX';
            setTimeout(() => this.loadEmails(inboxId), 3000);
        }

        console.log('%c📧 Email module loaded', 'color: #22c55e;');
    },

    // Set up filter tab click handlers
    setupFilterTabs() {
        const filterTabs = document.querySelectorAll('.filter-tab');
        filterTabs.forEach(tab => {
            tab.addEventListener('click', () => {
                const filter = tab.textContent.toLowerCase().trim();
                this.setFilter(filter);
            });
        });
    },

    // Load VIP senders from backend
    async loadVIPSenders() {
        try {
            const response = await fetch('/api/inbox/vip');
            if (response.ok) {
                const data = await response.json();
                this.vipSenders = data.vip_senders || [];
                console.log('Loaded VIP senders:', this.vipSenders.length);
            }
        } catch (error) {
            console.error('Failed to load VIP senders:', error);
        }
    },

    // Set the current filter and update display
    setFilter(filter) {
        this.currentFilter = filter;

        // Update tab UI
        const filterTabs = document.querySelectorAll('.filter-tab');
        filterTabs.forEach(tab => {
            const tabFilter = tab.textContent.toLowerCase().trim();
            tab.classList.toggle('active', tabFilter === filter);
        });

        // Apply filter and re-render
        this.applyFilter();
        this.renderEmails();
    },

    // Check if email is from a VIP sender
    isVIP(email) {
        const senderEmail = email.from && email.from[0] ? email.from[0].email.toLowerCase() : '';
        return this.vipSenders.some(vip => senderEmail.includes(vip.toLowerCase()));
    },

    // Apply filter to emails (simplified: All, VIP, Unread)
    applyFilter() {
        switch (this.currentFilter) {
            case 'vip':
                this.filteredEmails = this.emails.filter(email => this.isVIP(email));
                break;
            case 'unread':
                this.filteredEmails = this.emails.filter(email => email.unread);
                break;
            default: // 'all'
                this.filteredEmails = [...this.emails];
                break;
        }

        // Update filter tab counts
        this.updateFilterCounts();
    },

    // Update counts on filter tabs
    updateFilterCounts() {
        const counts = {
            all: this.emails.length,
            vip: this.emails.filter(e => this.isVIP(e)).length,
            unread: this.emails.filter(e => e.unread).length
        };

        // Update DOM
        const tabs = document.querySelectorAll('.filter-tab');
        tabs.forEach(tab => {
            const filter = tab.dataset.filter || tab.textContent.toLowerCase().trim();
            const count = counts[filter];
            let countBadge = tab.querySelector('.filter-count');

            if (count > 0 && filter !== 'all') {
                if (!countBadge) {
                    countBadge = document.createElement('span');
                    countBadge.className = 'filter-count';
                    tab.appendChild(countBadge);
                }
                countBadge.textContent = count > 99 ? '99+' : count;
            } else if (countBadge) {
                countBadge.remove();
            }
        });

        this.updateActiveFolderBadge(counts.unread);
    },

    // Sync the sidebar badge for the active folder to the unread count we
    // actually observed in the loaded messages. Nylas's folder-level
    // unread_count sometimes lags per-message unread state (most visible on
    // Gmail labels like SENT, where the folder aggregate can read "1" while
    // 39 of the loaded messages are flagged unread). Without this, the
    // sidebar disagrees with the Unread filter pill and the unread dots
    // shown on each row.
    updateActiveFolderBadge(unreadCount) {
        this.setFolderBadge(this.currentFolder, unreadCount);
    },

    // Write or update the unread badge for a specific folder by id. Shared
    // between the active-folder sync (post-load) and the eager refresh that
    // runs at sidebar boot (so the initial paint isn't stuck on Nylas's
    // stale aggregate). Observing zero is treated as "no evidence" — we
    // leave any existing badge alone, because the loaded page might be a
    // partial view of a folder with messages on later pages.
    setFolderBadge(folderId, unreadCount) {
        if (!folderId) return;
        if (typeof CSS === 'undefined' || typeof CSS.escape !== 'function') return;
        if (unreadCount <= 0) return;

        const folderItem = document.querySelector(
            `.folder-item[data-folder-id="${CSS.escape(folderId)}"]`
        );
        if (!folderItem) return;

        const text = unreadCount > 99 ? '99+' : String(unreadCount);
        let badge = folderItem.querySelector('.folder-count');
        if (badge) {
            badge.textContent = text;
            badge.classList.add('unread');
            return;
        }
        badge = document.createElement('span');
        badge.className = 'folder-count unread';
        badge.textContent = text;
        folderItem.appendChild(badge);
    },

    // Best-effort fetch a folder's first page and update its badge from the
    // observed unread count. Errors are swallowed: if the request fails or
    // the API isn't available, the sidebar simply keeps whatever Nylas
    // returned for folder.unread_count — same fallback as before this fix.
    async refreshFolderBadge(folderId) {
        if (!folderId || typeof AirAPI === 'undefined') return;
        try {
            const data = await AirAPI.getEmails({ folder: folderId, limit: 50 });
            const emails = data && Array.isArray(data.emails) ? data.emails : [];
            const unreadCount = emails.reduce((acc, e) => acc + (e && e.unread ? 1 : 0), 0);
            this.setFolderBadge(folderId, unreadCount);
        } catch (_err) {
            /* leave existing badge alone */
        }
    },

    // After folders load, eagerly refresh badges for the system folders
    // most likely to disagree with Nylas's aggregate (Sent/Drafts/Archive,
    // and Inbox for completeness). Done in parallel and awaited only
    // best-effort, so a slow folder doesn't block the others. Caps the fan-
    // out so a custom-label-heavy Gmail account doesn't fire dozens of
    // simultaneous requests on startup.
    async refreshPrimaryFolderBadges() {
        if (!this.folders || !this.folders.length) return;
        const targetNames = new Set([
            'inbox', 'sent', 'sent items', 'draft', 'drafts',
            'archive', 'trash', 'deleted items', 'spam', 'junk', 'junk email',
        ]);
        const targets = this.folders.filter(f => {
            const name = (f.name || '').toLowerCase();
            const id = (f.id || '').toLowerCase();
            return targetNames.has(name) || targetNames.has(id);
        });
        await Promise.allSettled(targets.map(f => this.refreshFolderBadge(f.id)));
    }
};
