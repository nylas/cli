/**
 * Notetaker Helpers - Icons, status, and utilities
 */
Object.assign(NotetakerModule, {
getStatusIcon(state) {
    const icons = {
        'scheduled': '🟡',
        'connecting': '🟠',
        'waiting_for_entry': '🟠',
        'attending': '🟢',
        'media_processing': '🔵',
        'complete': '✅',
        'cancelled': '⚪',
        'failed': '🔴'
    };
    return icons[state] || '⚪';
},

/**
 * Get human-readable status text
 */
getStatusText(state) {
    const texts = {
        'scheduled': 'Scheduled',
        'connecting': 'Connecting...',
        'waiting_for_entry': 'Waiting to join',
        'attending': 'Recording',
        'media_processing': 'Processing',
        'complete': 'Complete',
        'cancelled': 'Cancelled',
        'failed': 'Failed'
    };
    return texts[state] || state;
},

/**
 * Get provider icon
 */
getProviderIcon(provider) {
    const icons = {
        'zoom': '📹',
        'google_meet': '🎥',
        'teams': '💼',
        'nylas_notebook': '📓'
    };
    return icons[provider] || '📹';
},

/**
 * Build empty state element
 */
buildEmptyState() {
    const container = this.createElement('div', 'empty-state');
    const icon = this.createElement('span', 'icon', '🤖');
    const title = this.createElement('h3', null, 'No Recordings');
    const desc = this.createElement('p', null, 'Schedule a bot to record your meetings');
    container.appendChild(icon);
    container.appendChild(title);
    container.appendChild(desc);
    return container;
},

/**
 * Build notetaker card element
 */
buildNotetakerItem(nt) {
    const card = this.createElement('div', 'nt-card');
    card.dataset.id = nt.id;

    // Click handler to show summary for external notetakers
    if (nt.isExternal && nt.summary) {
        card.style.cursor = 'pointer';
        card.onclick = (e) => {
            if (e.target.closest('.nt-card-btn') || e.target.closest('.nt-card-toggle')) return;
            this.showSummaryModal(nt);
        };
    }

    // Banner with provider icon
    const banner = this.createElement('div', 'nt-card-banner');
    const providerIcon = this.createElement('div', 'nt-card-provider');
    providerIcon.innerHTML = this.getProviderSVG(nt.provider);
    banner.appendChild(providerIcon);
    card.appendChild(banner);

    // Card body
    const body = this.createElement('div', 'nt-card-body');

    // Title row with badge
    const titleRow = this.createElement('div', 'nt-card-title-row');
    const title = this.createElement('h4', 'nt-card-title', nt.meetingTitle || 'Meeting Recording');
    titleRow.appendChild(title);

    // Status badge - show "External" for external sources
    const badge = this.createElement('span', 'nt-card-badge');
    if (nt.isExternal) {
        badge.classList.add('external');
        badge.textContent = 'External';
    } else {
        badge.classList.add(this.getBadgeClass(nt.state));
        badge.textContent = this.getStatusText(nt.state);
    }
    titleRow.appendChild(badge);
    body.appendChild(titleRow);

    // Meta info (date/time)
    const meta = this.createElement('div', 'nt-card-meta');
    if (nt.createdAt) {
        const d = new Date(nt.createdAt);
        meta.innerHTML = `<span>📅 ${d.toLocaleDateString()}</span><span>🕐 ${d.toLocaleTimeString([], {hour:'2-digit',minute:'2-digit'})}</span>`;
    }
    body.appendChild(meta);

    // Details section (collapsed by default)
    const details = this.createElement('div', 'nt-card-details');
    details.style.display = 'none';
    const detailsLink = nt.meetingLink && isSafeUrl(nt.meetingLink)
        ? `<p><a href="${escapeHtml(nt.meetingLink)}" target="_blank" rel="noopener noreferrer">🔗 Meeting Link</a></p>`
        : '';
    details.innerHTML = `<p>${this.getProviderName(nt.provider)}</p>${detailsLink}`;
    body.appendChild(details);

    // Toggle details button
    const toggleBtn = this.createElement('button', 'nt-card-toggle', '▼ Meeting Details');
    toggleBtn.onclick = (e) => {
        e.stopPropagation();
        const open = details.style.display !== 'none';
        details.style.display = open ? 'none' : 'block';
        toggleBtn.textContent = open ? '▼ Meeting Details' : '▲ Hide Details';
    };
    body.appendChild(toggleBtn);

    // Action button
    const btn = this.createElement('button', 'nt-card-btn');
    if (nt.isExternal && nt.externalUrl) {
        btn.textContent = '🔗 Open Recording';
        btn.onclick = () => window.open(nt.externalUrl, '_blank');
    } else if (nt.state === 'complete' || nt.state === 'completed') {
        btn.textContent = '▶️ Watch Now';
        btn.onclick = () => this.playRecording(nt.id);
    } else if (nt.state === 'scheduled') {
        btn.textContent = '❌ Cancel';
        btn.classList.add('danger');
        btn.onclick = () => this.cancel(nt.id);
    } else {
        btn.textContent = this.getStatusText(nt.state);
        btn.disabled = true;
    }
    body.appendChild(btn);

    card.appendChild(body);
    return card;
},

/**
 * Get badge CSS class for state
 */
getBadgeClass(state) {
    const classes = {
        'complete': 'complete', 'completed': 'complete',
        'failed': 'failed', 'failed_entry': 'failed', 'cancelled': 'failed',
        'attending': 'active', 'connecting': 'pending', 'waiting_for_entry': 'pending',
        'scheduled': 'pending', 'media_processing': 'pending'
    };
    return classes[state] || 'pending';
},

/**
 * Get provider SVG icon
 */
getProviderSVG(provider) {
    if (provider === 'google_meet') return '<svg viewBox="0 0 24 24" width="48" height="48"><rect fill="#00897B" width="24" height="24" rx="4"/><path fill="#fff" d="M12 6l6 4v4l-6 4-6-4v-4z"/></svg>';
    if (provider === 'zoom') return '<svg viewBox="0 0 24 24" width="48" height="48"><rect fill="#2D8CFF" width="24" height="24" rx="4"/><path fill="#fff" d="M4 8h10v8H4z"/><path fill="#fff" d="M14 10l4-2v8l-4-2z"/></svg>';
    if (provider === 'teams') return '<svg viewBox="0 0 24 24" width="48" height="48"><rect fill="#5059C9" width="24" height="24" rx="4"/><path fill="#fff" d="M6 8h8v8H6z"/></svg>';
    return '<svg viewBox="0 0 24 24" width="48" height="48"><rect fill="#8b5cf6" width="24" height="24" rx="4"/><text x="12" y="16" text-anchor="middle" fill="#fff" font-size="10">N</text></svg>';
},

/**
 * Render the notetaker list as cards
 */
renderNotetakerCards() {
    this.renderNotetakers();
}
});
