/* Email Renderer - Utility for rendering email HTML */

const EmailRenderer = {
    // Format timestamp to relative time
    formatTime(timestamp) {
        const date = new Date(timestamp * 1000);
        const now = new Date();
        const diffMs = now - date;
        const diffMins = Math.floor(diffMs / 60000);
        const diffHours = Math.floor(diffMs / 3600000);
        const diffDays = Math.floor(diffMs / 86400000);

        if (diffMins < 1) return 'Just now';
        if (diffMins < 60) return `${diffMins}m`;
        if (diffHours < 24) return `${diffHours}h`;
        if (diffDays < 7) return `${diffDays}d`;

        return date.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
    },

    // Get sender display info
    getSenderInfo(from) {
        if (!from || from.length === 0) {
            return { name: 'Unknown', initials: '?', email: '' };
        }
        const sender = from[0];
        const name = sender.name || sender.email.split('@')[0];
        const initials = name.split(' ')
            .map(n => n[0])
            .join('')
            .substring(0, 2)
            .toUpperCase();
        return { name, initials, email: sender.email };
    },

    // Render a single email item for the list
    renderEmailItem(email, isSelected = false) {
        const sender = this.getSenderInfo(email.from);
        const time = this.formatTime(email.date);
        const hasAttachment = email.attachments && email.attachments.length > 0;

        const div = document.createElement('div');
        div.className = `email-item${isSelected ? ' selected' : ''}${email.unread ? ' unread' : ''}`;
        div.setAttribute('data-email-id', email.id);
        div.setAttribute('role', 'option');
        div.setAttribute('tabindex', '-1');
        div.setAttribute('aria-selected', isSelected ? 'true' : 'false');

        div.innerHTML = `
            <div class="email-avatar" style="background: ${gradientFor(sender.email || sender.name)}">
                ${this.escapeHtml(sender.initials)}
            </div>
            <div class="email-content">
                <div class="email-header">
                    <span class="email-sender">${this.escapeHtml(sender.name)}</span>
                    <span class="email-time">${time}</span>
                </div>
                <div class="email-subject">${this.escapeHtml(email.subject || '(No Subject)')}</div>
                <div class="email-preview">${this.escapeHtml(email.snippet || '')}</div>
            </div>
            <div class="email-actions-mini">
                ${email.starred ? '<span class="starred" title="Starred">&#9733;</span>' : ''}
                ${hasAttachment ? '<span class="attachment" title="Has attachments">&#128206;</span>' : ''}
            </div>
        `;

        return div;
    },

    // Render folder item
    renderFolderItem(folder, isActive = false) {
        const icons = {
            inbox: '<svg width="16" height="16" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><rect x="2" y="4" width="20" height="16" rx="2"/><path d="M2 12h6l2 2h4l2-2h6"/></svg>',
            sent: '<svg width="16" height="16" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><path d="M22 2L11 13M22 2l-7 20-4-9-9-4 20-7z"/></svg>',
            drafts: '<svg width="16" height="16" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><path d="M12 3H5a2 2 0 00-2 2v14a2 2 0 002 2h14a2 2 0 002-2v-7"/><path d="M18.5 2.5a2.121 2.121 0 013 3L12 15l-4 1 1-4 9.5-9.5z"/></svg>',
            trash: '<svg width="16" height="16" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><path d="M3 6h18M19 6v14a2 2 0 01-2 2H7a2 2 0 01-2-2V6m3 0V4a2 2 0 012-2h4a2 2 0 012 2v2"/></svg>',
            spam: '<svg width="16" height="16" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><circle cx="12" cy="12" r="10"/><path d="M12 8v4M12 16h.01"/></svg>',
            archive: '<svg width="16" height="16" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><rect x="2" y="4" width="20" height="5" rx="1"/><path d="M4 9v9a2 2 0 002 2h12a2 2 0 002-2V9M10 13h4"/></svg>'
        };

        const icon = icons[folder.system_folder] || icons.inbox;

        const li = document.createElement('li');
        li.className = `folder-item${isActive ? ' active' : ''}`;
        li.setAttribute('data-folder-id', folder.id);

        li.innerHTML = `
            <span class="folder-icon">${icon}</span>
            <span class="folder-name">${this.escapeHtml(folder.name)}</span>
            ${folder.unread_count > 0 ? `<span class="folder-count">${folder.unread_count}</span>` : ''}
        `;

        return li;
    },

    // Escape HTML to prevent XSS. Escapes &, <, >, ", and ' so the result
    // is safe in both element and attribute contexts.
    escapeHtml(str) {
        if (!str) return '';
        return String(str)
            .replaceAll('&', '&amp;')
            .replaceAll('<', '&lt;')
            .replaceAll('>', '&gt;')
            .replaceAll('"', '&quot;')
            .replaceAll("'", '&#39;');
    }
};

// Email List Manager with Virtual Scrolling & Optimistic Updates
