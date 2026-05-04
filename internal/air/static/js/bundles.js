/**
 * Email Bundles Module
 * Smart email categorization inspired by Shortwave/Google Inbox
 * Groups emails by type: newsletters, receipts, social, etc.
 *
 * STATUS: This module is NOT currently included by base.gohtml — the
 * backend `/api/bundles*` routes ship and have integration tests, but
 * the front-end UI is gated on a future sidebar redesign. The defensive
 * cleanup in this file (fallback bundles when the API 5xx's, localStorage
 * collapse persistence, no PUT-to-GET-only route) is kept intact so the
 * module is ready to wire up via a single `<script>` include in
 * base.gohtml when the redesign lands. Until then, treat changes here
 * as exercising the defensive paths only — there is no live UX impact.
 */

const Bundles = {
    // Bundle state
    bundles: [],
    emailBundles: new Map(), // emailId -> bundleId
    isLoaded: false,

    // Bundle icons for UI
    icons: {
        newsletters: '📰',
        receipts: '🧾',
        social: '👥',
        updates: '🔔',
        promotions: '🏷️',
        finance: '💰',
        travel: '✈️',
        primary: '📥',
    },

    /**
     * Initialize bundles module
     */
    async init() {
        try {
            await this.loadBundles();
            this.setupUI();
            this.isLoaded = true;
            console.log('%c📦 Bundles module loaded', 'color: #22c55e;');
        } catch (error) {
            console.error('Failed to initialize bundles:', error);
        }
    },

    /**
     * Load bundles from server, then layer on per-user collapse state from
     * localStorage. Collapse is a UI preference that doesn't need a server
     * round-trip — keeping it client-side avoids the 405s the old code
     * silently hit when it tried to PUT /api/bundles.
     */
    async loadBundles() {
        try {
            const response = await fetch('/api/bundles');
            if (response.ok) {
                this.bundles = await response.json();
            } else {
                // fetch() does NOT throw on !response.ok (only on network
                // failure), so a 5xx from /api/bundles previously left
                // this.bundles as the empty initial value and the user
                // saw no fallback grouping at all. Surface the defaults
                // so the inbox is at least usable on partial outage.
                console.warn('[bundles] /api/bundles returned', response.status, '- using defaults');
                this.bundles = this.getDefaultBundles();
            }
        } catch (error) {
            console.error('Failed to load bundles:', error);
            this.bundles = this.getDefaultBundles();
        }
        this.applyStoredCollapseState();
    },

    /**
     * applyStoredCollapseState rehydrates bundle.collapsed from
     * localStorage. Tolerates corrupt JSON (e.g. user editing the value
     * in devtools) by falling through to the server-provided defaults.
     */
    applyStoredCollapseState() {
        const map = this.readCollapseMap();
        if (!map) return;
        this.bundles.forEach((bundle) => {
            if (Object.prototype.hasOwnProperty.call(map, bundle.id)) {
                bundle.collapsed = !!map[bundle.id];
            }
        });
    },

    readCollapseMap() {
        try {
            const raw = localStorage.getItem('air.bundles.collapsed');
            if (!raw) return null;
            const parsed = JSON.parse(raw);
            return parsed && typeof parsed === 'object' ? parsed : null;
        } catch (_) {
            return null;
        }
    },

    writeCollapseState(bundleId, collapsed) {
        const map = this.readCollapseMap() || {};
        map[bundleId] = !!collapsed;
        try {
            localStorage.setItem('air.bundles.collapsed', JSON.stringify(map));
        } catch (err) {
            // Quota exceeded / storage disabled — non-fatal; the toggle
            // still works for the current session.
            console.warn('[bundles] could not persist collapse state:', err);
        }
    },

    /**
     * Get default bundles (fallback)
     */
    getDefaultBundles() {
        return [
            { id: 'newsletters', name: 'Newsletters', icon: '📰', collapsed: true, count: 0 },
            { id: 'receipts', name: 'Receipts & Orders', icon: '🧾', collapsed: true, count: 0 },
            { id: 'social', name: 'Social', icon: '👥', collapsed: true, count: 0 },
            { id: 'updates', name: 'Updates', icon: '🔔', collapsed: true, count: 0 },
            { id: 'promotions', name: 'Promotions', icon: '🏷️', collapsed: true, count: 0 },
        ];
    },

    /**
     * Setup bundle UI in sidebar
     */
    setupUI() {
        const sidebar = document.querySelector('.sidebar-bundles');
        if (!sidebar) return;

        // Clear existing
        while (sidebar.firstChild) {
            sidebar.removeChild(sidebar.firstChild);
        }

        // Add bundle items
        this.bundles.forEach(bundle => {
            if (bundle.count > 0) {
                sidebar.appendChild(this.createBundleItem(bundle));
            }
        });
    },

    /**
     * Create bundle list item
     * @param {Object} bundle - Bundle data
     * @returns {HTMLElement}
     */
    createBundleItem(bundle) {
        const item = document.createElement('div');
        item.className = 'bundle-item';
        item.dataset.bundleId = bundle.id;

        const icon = document.createElement('span');
        icon.className = 'bundle-icon';
        icon.textContent = bundle.icon || this.icons[bundle.id] || '📁';
        item.appendChild(icon);

        const name = document.createElement('span');
        name.className = 'bundle-name';
        name.textContent = bundle.name;
        item.appendChild(name);

        if (bundle.count > 0) {
            const count = document.createElement('span');
            count.className = 'bundle-count';
            count.textContent = bundle.count;
            item.appendChild(count);
        }

        const toggle = document.createElement('button');
        toggle.className = 'bundle-toggle';
        toggle.setAttribute('aria-label', bundle.collapsed ? 'Expand' : 'Collapse');
        toggle.textContent = bundle.collapsed ? '▶' : '▼';
        item.appendChild(toggle);

        // Click to expand/view bundle
        item.addEventListener('click', () => this.viewBundle(bundle.id));
        toggle.addEventListener('click', (e) => {
            e.stopPropagation();
            this.toggleBundle(bundle.id);
        });

        return item;
    },

    /**
     * Categorize an email into a bundle
     * @param {Object} email - Email object with from, subject
     * @returns {string|null} Bundle ID or null
     */
    async categorize(email) {
        if (!email || !email.from) return null;

        try {
            const response = await fetch('/api/bundles/categorize', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    from: email.from,
                    subject: email.subject || '',
                    emailId: email.id,
                }),
            });

            if (response.ok) {
                const result = await response.json();
                if (result.bundleId) {
                    this.emailBundles.set(email.id, result.bundleId);
                    this.updateBundleCount(result.bundleId, 1);
                    return result.bundleId;
                }
            }
        } catch (error) {
            console.error('Failed to categorize email:', error);
        }

        return null;
    },

    /**
     * Categorize multiple emails
     * @param {Array} emails - Array of email objects
     */
    async categorizeAll(emails) {
        const promises = emails.map(email => this.categorize(email));
        await Promise.all(promises);
        this.setupUI();
    },

    /**
     * Update bundle count
     * @param {string} bundleId - Bundle ID
     * @param {number} delta - Change in count
     */
    updateBundleCount(bundleId, delta) {
        const bundle = this.bundles.find(b => b.id === bundleId);
        if (bundle) {
            bundle.count = (bundle.count || 0) + delta;
        }
    },

    /**
     * View emails in a bundle
     * @param {string} bundleId - Bundle ID
     */
    viewBundle(bundleId) {
        const bundle = this.bundles.find(b => b.id === bundleId);
        if (!bundle) return;

        // Dispatch event for email list to filter
        const event = new CustomEvent('bundleSelected', {
            detail: { bundleId, bundleName: bundle.name },
        });
        document.dispatchEvent(event);

        // Update active state
        document.querySelectorAll('.bundle-item').forEach(item => {
            item.classList.toggle('active', item.dataset.bundleId === bundleId);
        });
    },

    /**
     * Toggle bundle collapsed state and persist to localStorage so the
     * preference survives a reload. The previous implementation tried to
     * PUT /api/bundles, but that route only handles GET — every toggle
     * silently produced a 405 (caught by `.catch` but never reported), so
     * collapse state was effectively per-tab.
     * @param {string} bundleId - Bundle ID
     */
    toggleBundle(bundleId) {
        const bundle = this.bundles.find(b => b.id === bundleId);
        if (!bundle) return;

        bundle.collapsed = !bundle.collapsed;
        this.writeCollapseState(bundle.id, bundle.collapsed);
        this.setupUI();
    },

    /**
     * Get bundle for an email
     * @param {string} emailId - Email ID
     * @returns {Object|null} Bundle or null
     */
    getBundleForEmail(emailId) {
        const bundleId = this.emailBundles.get(emailId);
        return bundleId ? this.bundles.find(b => b.id === bundleId) : null;
    },

    /**
     * Check if email is in a collapsed bundle
     * @param {string} emailId - Email ID
     * @returns {boolean}
     */
    isInCollapsedBundle(emailId) {
        const bundle = this.getBundleForEmail(emailId);
        return bundle ? bundle.collapsed : false;
    },

    /**
     * Get all emails in a bundle
     * @param {string} bundleId - Bundle ID
     * @returns {string[]} Array of email IDs
     */
    getEmailsInBundle(bundleId) {
        const emailIds = [];
        this.emailBundles.forEach((bId, emailId) => {
            if (bId === bundleId) {
                emailIds.push(emailId);
            }
        });
        return emailIds;
    },
};

// Initialize on DOM ready
document.addEventListener('DOMContentLoaded', () => {
    Bundles.init();
});

// Export for use
if (typeof window !== 'undefined') {
    window.Bundles = Bundles;
}
