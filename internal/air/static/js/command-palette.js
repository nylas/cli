/* ====================================
   COMMAND PALETTE - Obsidian Velocity
   Keyboard-first navigation system
   ==================================== */

const CommandPalette = {
    isOpen: false,
    selectedIndex: 0,
    commands: [],
    filteredCommands: [],
    recentCommands: [],

    // Command definitions organized by category
    allCommands: {
        actions: [
            { id: 'compose', icon: '✉️', title: 'Compose new email', description: 'Start writing a new message', shortcut: ['C'], action: () => typeof toggleCompose === 'function' && toggleCompose() },
            { id: 'create-event', icon: '📅', title: 'Create event', description: 'Add a new calendar event', shortcut: ['⇧', 'C'], action: () => typeof openCreateEventModal === 'function' && openCreateEventModal() },
            { id: 'search', icon: '🔍', title: 'Search', description: 'Search emails, events, contacts', shortcut: ['/', '⌘', 'F'], action: () => typeof openSearch === 'function' && openSearch() },
            { id: 'refresh', icon: '🔄', title: 'Refresh', description: 'Reload current view', shortcut: ['R'], action: () => location.reload() },
        ],
        navigation: [
            { id: 'goto-email', icon: '📧', title: 'Go to Email', description: 'Switch to email view', shortcut: ['G', 'E'], action: () => typeof switchView === 'function' && switchView('email') },
            { id: 'goto-calendar', icon: '📅', title: 'Go to Calendar', description: 'Switch to calendar view', shortcut: ['G', 'C'], action: () => typeof switchView === 'function' && switchView('calendar') },
            { id: 'goto-contacts', icon: '👤', title: 'Go to Contacts', description: 'Switch to contacts view', shortcut: ['G', 'O'], action: () => typeof switchView === 'function' && switchView('contacts') },
            { id: 'goto-inbox', icon: '📥', title: 'Go to Inbox', description: 'View inbox folder', shortcut: ['G', 'I'], action: () => typeof EmailListManager !== 'undefined' && EmailListManager.loadFolder && EmailListManager.loadFolder('inbox') },
            { id: 'goto-sent', icon: '📤', title: 'Go to Sent', description: 'View sent folder', shortcut: ['G', 'S'], action: () => typeof EmailListManager !== 'undefined' && EmailListManager.loadFolder && EmailListManager.loadFolder('sent') },
            { id: 'goto-drafts', icon: '📝', title: 'Go to Drafts', description: 'View drafts folder', shortcut: ['G', 'D'], action: () => typeof EmailListManager !== 'undefined' && EmailListManager.loadFolder && EmailListManager.loadFolder('drafts') },
        ],
        email: [
            { id: 'reply', icon: '↩️', title: 'Reply', description: 'Reply to selected email', shortcut: ['R'], context: 'email', action: () => typeof EmailListManager !== 'undefined' && EmailListManager.replyToEmail && EmailListManager.replyToEmail() },
            { id: 'reply-all', icon: '↩️', title: 'Reply All', description: 'Reply to all recipients', shortcut: ['⇧', 'R'], context: 'email', action: () => typeof EmailListManager !== 'undefined' && EmailListManager.replyAllToEmail && EmailListManager.replyAllToEmail() },
            { id: 'forward', icon: '➡️', title: 'Forward', description: 'Forward selected email', shortcut: ['F'], context: 'email', action: () => typeof EmailListManager !== 'undefined' && EmailListManager.forwardEmail && EmailListManager.forwardEmail() },
            { id: 'archive', icon: '📁', title: 'Archive', description: 'Archive selected email', shortcut: ['E'], context: 'email', action: () => typeof EmailListManager !== 'undefined' && EmailListManager.archiveEmail && EmailListManager.archiveEmail() },
            { id: 'delete', icon: '🗑️', title: 'Delete', description: 'Delete selected email', shortcut: ['#'], context: 'email', action: () => typeof EmailListManager !== 'undefined' && EmailListManager.deleteEmail && EmailListManager.deleteEmail() },
            { id: 'star', icon: '⭐', title: 'Star/Unstar', description: 'Toggle star on email', shortcut: ['S'], context: 'email', action: () => typeof EmailListManager !== 'undefined' && EmailListManager.toggleStar && EmailListManager.toggleStar() },
            { id: 'mark-read', icon: '👁️', title: 'Mark as Read', description: 'Mark email as read', shortcut: ['⇧', 'I'], context: 'email', action: () => typeof EmailListManager !== 'undefined' && EmailListManager.markAsRead && EmailListManager.markAsRead() },
            { id: 'mark-unread', icon: '📩', title: 'Mark as Unread', description: 'Mark email as unread', shortcut: ['⇧', 'U'], context: 'email', action: () => typeof EmailListManager !== 'undefined' && EmailListManager.markAsUnread && EmailListManager.markAsUnread() },
            { id: 'snooze', icon: '⏰', title: 'Snooze', description: 'Snooze email for later', shortcut: ['B'], context: 'email', action: () => typeof openSnoozeModal === 'function' && openSnoozeModal() },
        ],
        ai: [
            { id: 'ai-summarize', icon: '✨', title: 'AI Summarize', description: 'Get AI summary of email', shortcut: ['⌘', 'S'], action: () => typeof EmailListManager !== 'undefined' && EmailListManager.summarizeWithAI && EmailListManager.summarizeWithAI(EmailListManager.selectedEmail) },
            { id: 'ai-smart-reply', icon: '💬', title: 'Smart Reply', description: 'Get AI reply suggestions', action: () => typeof EmailListManager !== 'undefined' && EmailListManager.loadSmartReplies && EmailListManager.loadSmartReplies(EmailListManager.selectedEmail) },
            { id: 'ai-compose', icon: '🪄', title: 'AI Compose', description: 'AI-powered email drafting', action: () => typeof toggleCompose === 'function' && toggleCompose() },
        ],
        productivity: [
            { id: 'find-time', icon: '🎯', title: 'Find Meeting Time', description: 'Find optimal time across timezones', action: () => typeof showFindTimeModal === 'function' && showFindTimeModal() },
            { id: 'focus-time', icon: '🧘', title: 'Block Focus Time', description: 'Protect time for deep work', action: () => typeof showFocusTimeModal === 'function' && showFocusTimeModal() },
            { id: 'send-later', icon: '📆', title: 'Schedule Send', description: 'Schedule email for later', shortcut: ['⇧', '⌘', 'S'], action: () => typeof openSendLaterModal === 'function' && openSendLaterModal() },
        ],
        settings: [
            { id: 'settings', icon: '⚙️', title: 'Settings', description: 'Open settings panel', shortcut: ['⌘', ','], action: () => typeof toggleSettings === 'function' && toggleSettings() },
            { id: 'keyboard-shortcuts', icon: '⌨️', title: 'Keyboard Shortcuts', description: 'View all shortcuts', shortcut: ['?'], action: () => CommandPalette.showShortcutsHelp() },
            { id: 'cycle-theme', icon: '🎨', title: 'Cycle Theme', description: 'Dark → Light → OLED → System', shortcut: ['⌘', '⇧', 'T'], action: () => typeof ThemeManager !== 'undefined' && ThemeManager.cycleTheme() },
            { id: 'dark-mode', icon: '🌙', title: 'Dark Mode', description: 'Switch to dark theme', action: () => typeof ThemeManager !== 'undefined' && ThemeManager.setTheme('dark') },
            { id: 'light-mode', icon: '☀️', title: 'Light Mode', description: 'Switch to light theme', action: () => typeof ThemeManager !== 'undefined' && ThemeManager.setTheme('light') },
            { id: 'oled-mode', icon: '🖤', title: 'OLED Mode', description: 'True black for OLED displays', action: () => typeof ThemeManager !== 'undefined' && ThemeManager.setTheme('oled') },
            { id: 'system-theme', icon: '💻', title: 'System Theme', description: 'Follow OS preference', action: () => typeof ThemeManager !== 'undefined' && ThemeManager.setTheme('system') },
        ]
    },

    init() {
        this.buildCommandsList();
        this.loadRecentCommands();
        this.setupEventListeners();
        this.registerKeyboardShortcuts();
    },

    buildCommandsList() {
        this.commands = [];
        for (const [category, commands] of Object.entries(this.allCommands)) {
            commands.forEach(cmd => {
                this.commands.push({ ...cmd, category });
            });
        }
    },

    loadRecentCommands() {
        try {
            const stored = localStorage.getItem('nylas_recent_commands');
            this.recentCommands = stored ? JSON.parse(stored) : [];
        } catch {
            this.recentCommands = [];
        }
    },

    saveRecentCommand(commandId) {
        // Add to front, remove duplicates, keep last 5
        this.recentCommands = [
            commandId,
            ...this.recentCommands.filter(id => id !== commandId)
        ].slice(0, 5);
        try {
            localStorage.setItem('nylas_recent_commands', JSON.stringify(this.recentCommands));
        } catch {
            // Ignore storage errors
        }
    },

    setupEventListeners() {
        const palette = document.getElementById('commandPalette');
        if (!palette) return;

        const input = palette.querySelector('.command-input');
        if (input) {
            input.addEventListener('input', (e) => this.handleSearch(e.target.value));
            input.addEventListener('keydown', (e) => this.handleKeydown(e));
        }

        // Click outside to close
        palette.addEventListener('click', (e) => {
            if (e.target === palette) this.close();
        });
    },

    registerKeyboardShortcuts() {
        // Track pending key combo for sequences like "g e"
        let pendingKey = null;
        let pendingTimeout = null;

        document.addEventListener('keydown', (e) => {
            // Don't intercept when typing in inputs
            if (e.target.matches('input, textarea, [contenteditable]')) {
                // Only handle Escape
                if (e.key === 'Escape') {
                    this.close();
                }
                return;
            }

            // Cmd/Ctrl + K - Toggle command palette
            if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
                e.preventDefault();
                this.toggle();
                return;
            }

            // Escape - Close command palette
            if (e.key === 'Escape') {
                this.close();
                return;
            }

            // Skip if modifier keys for most shortcuts
            if (e.metaKey || e.ctrlKey || e.altKey) return;

            // Handle key sequences like "g e" for Go to Email
            if (pendingKey === 'g') {
                clearTimeout(pendingTimeout);
                pendingKey = null;

                switch (e.key.toLowerCase()) {
                    case 'e': e.preventDefault(); this.executeCommand('goto-email'); return;
                    case 'c': e.preventDefault(); this.executeCommand('goto-calendar'); return;
                    case 'o': e.preventDefault(); this.executeCommand('goto-contacts'); return;
                    case 'i': e.preventDefault(); this.executeCommand('goto-inbox'); return;
                    case 's': e.preventDefault(); this.executeCommand('goto-sent'); return;
                    case 'd': e.preventDefault(); this.executeCommand('goto-drafts'); return;
                }
            }

            // Start key sequence
            if (e.key.toLowerCase() === 'g') {
                pendingKey = 'g';
                pendingTimeout = setTimeout(() => { pendingKey = null; }, 1000);
                return;
            }

            // Single key shortcuts (only when palette is closed)
            if (this.isOpen) return;

            switch (e.key.toLowerCase()) {
                case 'c': e.preventDefault(); this.executeCommand('compose'); break;
                case '/': e.preventDefault(); this.executeCommand('search'); break;
                case '?': e.preventDefault(); this.showShortcutsHelp(); break;
                // Email-specific shortcuts (when email is selected)
                case 'r':
                    if (e.shiftKey) {
                        e.preventDefault();
                        this.executeCommand('reply-all');
                    } else {
                        e.preventDefault();
                        this.executeCommand('reply');
                    }
                    break;
                case 'f': e.preventDefault(); this.executeCommand('forward'); break;
                case 'e': e.preventDefault(); this.executeCommand('archive'); break;
                case 's': e.preventDefault(); this.executeCommand('star'); break;
                case 'b': e.preventDefault(); this.executeCommand('snooze'); break;
            }
        });
    },

    toggle() {
        if (this.isOpen) {
            this.close();
        } else {
            this.open();
        }
    },

    open() {
        const palette = document.getElementById('commandPalette');
        if (!palette) return;

        this.isOpen = true;
        this.selectedIndex = 0;
        this.filteredCommands = this.getDefaultCommands();

        palette.classList.remove('hidden');

        const input = palette.querySelector('.command-input');
        if (input) {
            input.value = '';
            input.focus();
        }

        this.renderCommands();
    },

    close() {
        const palette = document.getElementById('commandPalette');
        if (palette) {
            palette.classList.add('hidden');
        }
        this.isOpen = false;
    },

    getDefaultCommands() {
        // Show recent commands first, then top actions
        const recent = this.recentCommands
            .map(id => this.commands.find(c => c.id === id))
            .filter(Boolean);

        const actions = this.commands
            .filter(c => c.category === 'actions' && !this.recentCommands.includes(c.id))
            .slice(0, 4);

        return [...recent, ...actions];
    },

    handleSearch(query) {
        if (!query.trim()) {
            this.filteredCommands = this.getDefaultCommands();
        } else {
            const lowerQuery = query.toLowerCase();
            this.filteredCommands = this.commands.filter(cmd => {
                return cmd.title.toLowerCase().includes(lowerQuery) ||
                       cmd.description.toLowerCase().includes(lowerQuery) ||
                       cmd.category.toLowerCase().includes(lowerQuery);
            });
        }

        this.selectedIndex = 0;
        this.renderCommands();
    },

    handleKeydown(e) {
        switch (e.key) {
            case 'ArrowDown':
                e.preventDefault();
                this.selectedIndex = Math.min(this.selectedIndex + 1, this.filteredCommands.length - 1);
                this.updateSelection();
                break;
            case 'ArrowUp':
                e.preventDefault();
                this.selectedIndex = Math.max(this.selectedIndex - 1, 0);
                this.updateSelection();
                break;
            case 'Enter':
                e.preventDefault();
                if (this.filteredCommands[this.selectedIndex]) {
                    this.executeCommand(this.filteredCommands[this.selectedIndex].id);
                }
                break;
            case 'Escape':
                e.preventDefault();
                this.close();
                break;
        }
    },

    executeCommand(commandId) {
        const command = this.commands.find(c => c.id === commandId);
        if (!command) return;

        this.saveRecentCommand(commandId);
        this.close();

        if (typeof command.action === 'function') {
            try {
                command.action();
            } catch (err) {
                console.error('Command execution error:', err);
            }
        }
    },

    renderCommands() {
        const palette = document.getElementById('commandPalette');
        if (!palette) return;

        const results = palette.querySelector('.command-results');
        if (!results) return;

        if (this.filteredCommands.length === 0) {
            results.innerHTML = `
                <div class="command-empty">
                    <div class="command-empty-icon">🔍</div>
                    <div class="command-empty-title">No commands found</div>
                    <div class="command-empty-description">Try a different search term</div>
                </div>
            `;
            return;
        }

        // Group commands by category
        const grouped = {};
        this.filteredCommands.forEach((cmd, index) => {
            const category = cmd.category;
            if (!grouped[category]) grouped[category] = [];
            grouped[category].push({ ...cmd, index });
        });

        const categoryTitles = {
            actions: 'Quick Actions',
            navigation: 'Navigation',
            email: 'Email Actions',
            ai: 'AI Features',
            productivity: 'Productivity',
            settings: 'Settings'
        };

        let html = '';
        for (const [category, commands] of Object.entries(grouped)) {
            html += `
                <div class="command-section">
                    <div class="command-section-title">${categoryTitles[category] || category}</div>
                    ${commands.map(cmd => this.renderCommandItem(cmd, cmd.index)).join('')}
                </div>
            `;
        }

        results.innerHTML = html;

        // Add click handlers
        results.querySelectorAll('.command-item').forEach((item, i) => {
            item.addEventListener('click', () => {
                this.selectedIndex = parseInt(item.dataset.index);
                this.executeCommand(this.filteredCommands[this.selectedIndex].id);
            });
        });
    },

    renderCommandItem(cmd, index) {
        const isSelected = index === this.selectedIndex;
        const shortcutHtml = cmd.shortcut
            ? `<div class="command-shortcut">${cmd.shortcut.map(k => `<kbd>${k}</kbd>`).join('')}</div>`
            : '';

        return `
            <div class="command-item ${isSelected ? 'selected' : ''}" data-index="${index}">
                <div class="command-icon-wrapper">${cmd.icon}</div>
                <div class="command-text">
                    <div class="command-title">${this.escapeHtml(cmd.title)}</div>
                    <div class="command-description">${this.escapeHtml(cmd.description)}</div>
                </div>
                ${shortcutHtml}
            </div>
        `;
    },

    updateSelection() {
        const results = document.querySelector('.command-results');
        if (!results) return;

        results.querySelectorAll('.command-item').forEach((item, i) => {
            item.classList.toggle('selected', i === this.selectedIndex);
        });

        // Scroll selected into view
        const selected = results.querySelector('.command-item.selected');
        if (selected) {
            selected.scrollIntoView({ block: 'nearest', behavior: 'smooth' });
        }
    },

    showShortcutsHelp() {
        // Create shortcuts help modal
        let modal = document.getElementById('shortcutsHelpModal');
        if (!modal) {
            modal = document.createElement('div');
            modal.id = 'shortcutsHelpModal';
            modal.className = 'modal-overlay';
            document.body.appendChild(modal);
        }

        const shortcutGroups = [
            {
                title: 'Navigation',
                shortcuts: [
                    { keys: ['G', 'E'], description: 'Go to Email' },
                    { keys: ['G', 'C'], description: 'Go to Calendar' },
                    { keys: ['G', 'O'], description: 'Go to Contacts' },
                    { keys: ['G', 'I'], description: 'Go to Inbox' },
                    { keys: ['G', 'S'], description: 'Go to Sent' },
                    { keys: ['G', 'D'], description: 'Go to Drafts' },
                ]
            },
            {
                title: 'Actions',
                shortcuts: [
                    { keys: ['C'], description: 'Compose email' },
                    { keys: ['/'], description: 'Search' },
                    { keys: ['⌘', 'K'], description: 'Command palette' },
                    { keys: ['R'], description: 'Refresh' },
                ]
            },
            {
                title: 'Email',
                shortcuts: [
                    { keys: ['R'], description: 'Reply' },
                    { keys: ['⇧', 'R'], description: 'Reply All' },
                    { keys: ['F'], description: 'Forward' },
                    { keys: ['E'], description: 'Archive' },
                    { keys: ['S'], description: 'Star/Unstar' },
                    { keys: ['B'], description: 'Snooze' },
                    { keys: ['#'], description: 'Delete' },
                ]
            }
        ];

        modal.innerHTML = `
            <div class="modal shortcuts-modal" data-action="cmd-shortcuts-stop-propagation">
                <div class="modal-header">
                    <h3>⌨️ Keyboard Shortcuts</h3>
                    <button class="close-btn" data-action="cmd-shortcuts-close">&times;</button>
                </div>
                <div class="modal-body shortcuts-body">
                    ${shortcutGroups.map(group => `
                        <div class="shortcuts-group">
                            <div class="shortcuts-group-title">${group.title}</div>
                            <div class="shortcuts-list">
                                ${group.shortcuts.map(s => `
                                    <div class="shortcut-item">
                                        <div class="shortcut-keys">
                                            ${s.keys.map(k => `<kbd>${k}</kbd>`).join('')}
                                        </div>
                                        <div class="shortcut-desc">${s.description}</div>
                                    </div>
                                `).join('')}
                            </div>
                        </div>
                    `).join('')}
                </div>
            </div>
        `;

        modal.style.display = 'flex';
        modal.classList.remove('hidden');

        modal.addEventListener('click', (e) => {
            if (e.target === modal) {
                modal.classList.add('hidden');
            }
        });
    },

    escapeHtml(text) {
        if (text == null) return '';
        return String(text)
            .replaceAll('&', '&amp;')
            .replaceAll('<', '&lt;')
            .replaceAll('>', '&gt;')
            .replaceAll('"', '&quot;')
            .replaceAll("'", '&#39;');
    }
};

// Initialize when DOM is ready
document.addEventListener('DOMContentLoaded', () => {
    CommandPalette.init();
});

// Export for global access
window.CommandPalette = CommandPalette;
window.toggleCommandPalette = () => CommandPalette.toggle();
