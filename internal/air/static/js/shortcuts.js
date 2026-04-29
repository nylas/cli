/**
 * Keyboard Shortcuts Manager
 * Handles shortcut discovery, hints, and tooltips
 */

const ShortcutsManager = {
    // All available shortcuts
    shortcuts: {
        // Navigation
        'g i': { action: 'Go to Inbox', category: 'navigation' },
        'g s': { action: 'Go to Starred', category: 'navigation' },
        'g t': { action: 'Go to Sent', category: 'navigation' },
        'g d': { action: 'Go to Drafts', category: 'navigation' },

        // Email actions
        'j': { action: 'Next email', category: 'email' },
        'k': { action: 'Previous email', category: 'email' },
        'o': { action: 'Open email', category: 'email' },
        'Enter': { action: 'Open email', category: 'email' },
        'u': { action: 'Back to list', category: 'email' },
        'e': { action: 'Archive', category: 'email' },
        '#': { action: 'Delete', category: 'email' },
        's': { action: 'Toggle star', category: 'email' },
        'r': { action: 'Reply', category: 'email' },
        'a': { action: 'Reply all', category: 'email' },
        'f': { action: 'Forward', category: 'email' },
        'c': { action: 'Compose', category: 'email' },
        'n': { action: 'Compose new', category: 'email' },

        // Productivity
        'h': { action: 'Snooze', category: 'productivity' },
        'l': { action: 'Add label', category: 'productivity' },
        'v': { action: 'Move to folder', category: 'productivity' },

        // UI
        '/': { action: 'Search', category: 'ui' },
        '?': { action: 'Show shortcuts', category: 'ui' },
        'Escape': { action: 'Close / Cancel', category: 'ui' },
        '⌘K': { action: 'Command palette', category: 'ui', mac: true },
        'Ctrl+K': { action: 'Command palette', category: 'ui', mac: false },

        // Views
        '1': { action: 'Email view', category: 'views' },
        '2': { action: 'Calendar view', category: 'views' },
        '3': { action: 'Contacts view', category: 'views' },
    },

    // Pending key sequence for multi-key shortcuts (e.g., 'g i')
    pendingKey: null,
    pendingTimeout: null,

    // Hint display settings
    hintDelay: 800, // ms before showing hover hint
    proTipFrequency: 5, // Show pro tip every N mouse actions
    mouseActionCount: 0,
    shownProTips: new Set(),

    // Initialize shortcuts manager
    init() {
        this.setupKeySequenceDetection();
        this.setupHoverHints();
        this.setupProTips();
        this.createShortcutsModal();
        console.log('%c⌨️ Shortcuts manager initialized', 'color: #6366f1;');
    },

    // Setup multi-key sequence detection (e.g., 'g' then 'i')
    setupKeySequenceDetection() {
        document.addEventListener('keydown', (e) => {
            // Ignore if typing in input
            if (this.isTyping(e.target)) return;

            const key = e.key.toLowerCase();

            // Handle pending sequence
            if (this.pendingKey) {
                clearTimeout(this.pendingTimeout);
                const sequence = `${this.pendingKey} ${key}`;

                if (this.shortcuts[sequence]) {
                    this.showSequenceIndicator(null); // Clear indicator
                    this.pendingKey = null;
                    // Let the actual handler process this
                    return;
                }

                this.pendingKey = null;
                this.showSequenceIndicator(null);
            }

            // Check for sequence starters
            if (key === 'g') {
                this.pendingKey = 'g';
                this.showSequenceIndicator('g');
                this.pendingTimeout = setTimeout(() => {
                    this.pendingKey = null;
                    this.showSequenceIndicator(null);
                }, 1500);
            }
        });
    },

    // Show indicator for pending key sequence
    showSequenceIndicator(key) {
        let indicator = document.getElementById('shortcutSequenceIndicator');

        if (!key) {
            if (indicator) indicator.remove();
            return;
        }

        if (!indicator) {
            indicator = document.createElement('div');
            indicator.id = 'shortcutSequenceIndicator';
            indicator.className = 'shortcut-sequence-indicator';
            document.body.appendChild(indicator);
        }

        indicator.textContent = key + ' ...';
        indicator.classList.add('visible');
    },

    // Setup hover hints on actionable elements
    setupHoverHints() {
        // Map of selectors to their shortcuts
        const hintMappings = [
            { selector: '.compose-btn', shortcut: 'c', action: 'Compose' },
            { selector: '[data-action="archive"]', shortcut: 'e', action: 'Archive' },
            { selector: '[data-action="delete"]', shortcut: '#', action: 'Delete' },
            { selector: '[data-action="star"]', shortcut: 's', action: 'Star' },
            { selector: '[data-action="reply"]', shortcut: 'r', action: 'Reply' },
            { selector: '[data-action="reply-all"]', shortcut: 'a', action: 'Reply All' },
            { selector: '[data-action="forward"]', shortcut: 'f', action: 'Forward' },
            { selector: '[data-action="snooze"]', shortcut: 'h', action: 'Snooze' },
            { selector: '.search-input', shortcut: '/', action: 'Search' },
        ];

        // Add hover listeners with delay
        hintMappings.forEach(({ selector, shortcut, action }) => {
            document.querySelectorAll(selector).forEach(el => {
                let hintTimeout;

                el.addEventListener('mouseenter', () => {
                    hintTimeout = setTimeout(() => {
                        this.showHoverHint(el, shortcut, action);
                    }, this.hintDelay);
                });

                el.addEventListener('mouseleave', () => {
                    clearTimeout(hintTimeout);
                    this.hideHoverHint();
                });
            });
        });
    },

    // Show hover hint tooltip
    showHoverHint(element, shortcut, action) {
        this.hideHoverHint(); // Remove any existing

        const hint = document.createElement('div');
        hint.className = 'shortcut-hover-hint';
        hint.innerHTML = `
            <span class="hint-action">${action}</span>
            <kbd class="hint-key">${shortcut}</kbd>
        `;

        const rect = element.getBoundingClientRect();
        hint.style.top = `${rect.bottom + 8}px`;
        hint.style.left = `${rect.left + rect.width / 2}px`;

        document.body.appendChild(hint);

        // Animate in
        requestAnimationFrame(() => hint.classList.add('visible'));
    },

    // Hide hover hint
    hideHoverHint() {
        const existing = document.querySelector('.shortcut-hover-hint');
        if (existing) existing.remove();
    },

    // Setup pro tips after mouse actions
    setupProTips() {
        const actions = ['archive', 'delete', 'star', 'reply', 'forward'];

        actions.forEach(action => {
            document.addEventListener('click', (e) => {
                const button = e.target.closest(`[data-action="${action}"]`);
                if (button && !this.isTyping(e.target)) {
                    this.mouseActionCount++;

                    if (this.mouseActionCount % this.proTipFrequency === 0) {
                        this.maybeShowProTip(action);
                    }
                }
            });
        });
    },

    // Maybe show a pro tip (if not shown recently)
    maybeShowProTip(action) {
        const tips = {
            archive: { shortcut: 'e', tip: 'Press E to archive instantly' },
            delete: { shortcut: '#', tip: 'Press # to delete' },
            star: { shortcut: 's', tip: 'Press S to toggle star' },
            reply: { shortcut: 'r', tip: 'Press R to reply quickly' },
            forward: { shortcut: 'f', tip: 'Press F to forward' },
        };

        const tipData = tips[action];
        if (!tipData || this.shownProTips.has(action)) return;

        this.shownProTips.add(action);

        // Show pro tip toast
        if (typeof showToast === 'function') {
            showToast('info', 'Pro Tip', tipData.tip, {
                duration: 4000
            });
        }
    },

    // Create shortcuts help modal
    createShortcutsModal() {
        const modal = document.createElement('div');
        modal.id = 'shortcutsModal';
        modal.className = 'shortcuts-modal';
        modal.innerHTML = `
            <div class="shortcuts-modal-content">
                <div class="shortcuts-header">
                    <h2>Keyboard Shortcuts</h2>
                    <button class="shortcuts-close" data-action="shortcuts-modal-close">&times;</button>
                </div>
                <div class="shortcuts-body">
                    ${this.renderShortcutCategories()}
                </div>
                <div class="shortcuts-footer">
                    <span class="shortcuts-footer-hint">Press <kbd>?</kbd> anytime to show this</span>
                </div>
            </div>
        `;
        document.body.appendChild(modal);

        // Close on backdrop click
        modal.addEventListener('click', (e) => {
            if (e.target === modal) this.hideModal();
        });

        // Listen for ? key
        document.addEventListener('keydown', (e) => {
            if (e.key === '?' && !this.isTyping(e.target)) {
                e.preventDefault();
                this.toggleModal();
            }
        });
    },

    // Render shortcut categories for modal
    renderShortcutCategories() {
        const categories = {
            navigation: { title: 'Navigation', icon: '🧭' },
            email: { title: 'Email Actions', icon: '📧' },
            productivity: { title: 'Productivity', icon: '⚡' },
            ui: { title: 'Interface', icon: '🖥️' },
            views: { title: 'Views', icon: '👁️' },
        };

        let html = '';

        for (const [catKey, catInfo] of Object.entries(categories)) {
            const shortcuts = Object.entries(this.shortcuts)
                .filter(([_, data]) => data.category === catKey);

            if (shortcuts.length === 0) continue;

            html += `
                <div class="shortcuts-category">
                    <h3>${catInfo.icon} ${catInfo.title}</h3>
                    <div class="shortcuts-list">
                        ${shortcuts.map(([key, data]) => `
                            <div class="shortcut-item">
                                <span class="shortcut-action">${data.action}</span>
                                <kbd class="shortcut-key">${this.formatKey(key)}</kbd>
                            </div>
                        `).join('')}
                    </div>
                </div>
            `;
        }

        return html;
    },

    // Format key for display
    formatKey(key) {
        const isMac = navigator.platform.toUpperCase().indexOf('MAC') >= 0;

        return key
            .replace('⌘', isMac ? '⌘' : 'Ctrl')
            .replace('Ctrl+', isMac ? '⌘' : 'Ctrl+');
    },

    // Show shortcuts modal
    showModal() {
        const modal = document.getElementById('shortcutsModal');
        if (modal) {
            modal.classList.add('visible');
            document.body.style.overflow = 'hidden';
        }
    },

    // Hide shortcuts modal
    hideModal() {
        const modal = document.getElementById('shortcutsModal');
        if (modal) {
            modal.classList.remove('visible');
            document.body.style.overflow = '';
        }
    },

    // Toggle shortcuts modal
    toggleModal() {
        const modal = document.getElementById('shortcutsModal');
        if (modal && modal.classList.contains('visible')) {
            this.hideModal();
        } else {
            this.showModal();
        }
    },

    // Check if user is typing in an input
    isTyping(target) {
        return target.tagName === 'INPUT' ||
               target.tagName === 'TEXTAREA' ||
               target.isContentEditable ||
               target.closest('.compose-modal');
    },

    // Get shortcut for action (for external use)
    getShortcut(action) {
        for (const [key, data] of Object.entries(this.shortcuts)) {
            if (data.action.toLowerCase() === action.toLowerCase()) {
                return key;
            }
        }
        return null;
    }
};

// Initialize on DOM ready
document.addEventListener('DOMContentLoaded', () => {
    ShortcutsManager.init();
});

// Export for use
if (typeof window !== 'undefined') {
    window.ShortcutsManager = ShortcutsManager;
}
