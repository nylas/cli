/**
 * Notetaker UI - Rendering and display
 */
Object.assign(NotetakerModule, {
renderNotetakers() {
    const list = document.getElementById('notetakerList');
    const empty = document.getElementById('notetakerEmpty');
    if (!list) return;

    // Filter by past/upcoming
    const now = Date.now();
    let filtered = this.notetakers.filter(nt => {
        const ntTime = nt.createdAt ? new Date(nt.createdAt).getTime() : now;
        if (this.currentFilter === 'past') {
            return nt.state === 'complete' || nt.state === 'completed' || nt.state === 'failed' || nt.state === 'cancelled' || nt.isExternal;
        }
        if (this.currentFilter === 'upcoming') {
            return nt.state === 'scheduled' || nt.state === 'connecting' || nt.state === 'waiting_for_entry' || nt.state === 'attending';
        }
        return true;
    });

    // Clear existing cards
    list.querySelectorAll('.nt-card').forEach(c => c.remove());

    // Show/hide empty state
    if (empty) empty.style.display = filtered.length === 0 ? 'flex' : 'none';

    // Render cards
    filtered.forEach(nt => list.appendChild(this.buildNotetakerItem(nt)));
},

/**
 * Update sidebar counts
 */
updateCounts() {
    const counts = {
        all: this.notetakers.length,
        scheduled: this.notetakers.filter(n => n.state === 'scheduled').length,
        attending: this.notetakers.filter(n => ['connecting', 'waiting_for_entry', 'attending'].includes(n.state)).length,
        complete: this.notetakers.filter(n => n.state === 'complete' || n.state === 'completed').length
    };

    Object.entries(counts).forEach(([key, value]) => {
        const el = document.getElementById(`notetakerCount${key.charAt(0).toUpperCase() + key.slice(1)}`);
        if (el) el.textContent = value;
    });
},

/**
 * Select a notetaker and show details
 */
selectNotetaker(notetakerId) {
    this.selectedNotetaker = this.notetakers.find(n => n.id === notetakerId);

    // Update active state in list
    document.querySelectorAll('.notetaker-item').forEach(item => {
        item.classList.toggle('active', item.dataset.id === notetakerId);
    });

    this.renderDetail();
},

/**
 * Render notetaker detail panel
 */
renderDetail() {
    const detail = document.getElementById('notetakerDetail');
    if (!detail) return;

    if (!this.selectedNotetaker) {
        detail.innerHTML = `
            <div class="notetaker-detail-empty">
                <div class="detail-empty-icon">🎬</div>
                <h3>Select a recording</h3>
                <p>Click on a recording to view details, playback, and transcript</p>
            </div>
        `;
        return;
    }

    const nt = this.selectedNotetaker;
    const statusClass = nt.state === 'attending' ? 'recording' : nt.state;
    const isCompleted = nt.state === 'complete' || nt.state === 'completed';

    // Determine body content
    let bodyContent;
    if (nt.isExternal) {
        bodyContent = this.renderExternalContent(nt);
    } else if (isCompleted) {
        bodyContent = this.renderCompleteContent(nt);
    } else {
        bodyContent = this.renderPendingContent(nt);
    }

    // Build status display
    const statusDisplay = nt.isExternal
        ? '🔗 External'
        : this.getStatusIcon(nt.state) + ' ' + this.getStatusText(nt.state);

    // Build attendees line
    const attendeesLine = nt.attendees
        ? '<p class="notetaker-detail-attendees">👥 ' + nt.attendees + '</p>'
        : '';

    detail.innerHTML = `
        <div class="notetaker-detail-header">
            <div class="notetaker-detail-status ${statusClass}">
                ${statusDisplay}
            </div>
            <h2>${nt.meetingTitle || 'Meeting Recording'}</h2>
            <p class="notetaker-detail-meta">
                ${this.getProviderIcon(nt.provider)} ${this.getProviderName(nt.provider)}
                ${nt.createdAt ? ' • ' + new Date(nt.createdAt).toLocaleString() : ''}
            </p>
            ${attendeesLine}
        </div>
        <div class="notetaker-detail-body">
            ${bodyContent}
        </div>
        <div class="notetaker-detail-actions">
            ${this.renderActions(nt)}
        </div>
    `;
},

/**
 * Get provider display name
 */
getProviderName(provider) {
    const names = {
        'zoom': 'Zoom',
        'google_meet': 'Google Meet',
        'teams': 'Microsoft Teams',
        'nylas_notebook': 'Nylas Notebook (External)'
    };
    return names[provider] || provider || 'Unknown';
},

/**
 * Strip embedded styles from HTML to allow our CSS to take control
 */
stripEmbeddedStyles(html) {
    // Remove <style> tags and their content
    let cleaned = html.replace(/<style[^>]*>[\s\S]*?<\/style>/gi, '');
    // Remove inline style attributes
    cleaned = cleaned.replace(/\s+style="[^"]*"/gi, '');
    // Remove <html>, <head>, <body> tags but keep their content
    cleaned = cleaned.replace(/<\/?html[^>]*>/gi, '');
    cleaned = cleaned.replace(/<head[^>]*>[\s\S]*?<\/head>/gi, '');
    cleaned = cleaned.replace(/<\/?body[^>]*>/gi, '');
    return cleaned;
},

/**
 * Render content for external recording (from Nylas Notebook emails)
 */
renderExternalContent(nt) {
    const container = this.createElement('div', 'external-content');

    // If there's a summary from the email, show it
    if (nt.summary) {
        const summarySection = this.createElement('div', 'detail-section summary-section');
        const summaryContent = this.createElement('div', 'summary-content');
        // Strip embedded styles to let our CSS control theming
        summaryContent.innerHTML = this.stripEmbeddedStyles(nt.summary);
        summarySection.appendChild(summaryContent);
        container.appendChild(summarySection);
    } else {
        const content = this.createElement('div', 'detail-section');
        const title = this.createElement('h3', null, '🔗 External Recording');
        const desc = this.createElement('p', null, 'This recording is available in Nylas Notebook.');
        const note = this.createElement('p', 'external-note', 'Click the button below to open in a new tab.');
        content.appendChild(title);
        content.appendChild(desc);
        content.appendChild(note);
        container.appendChild(content);
    }

    return container.outerHTML;
},

/**
 * Render content for completed recording
 */
renderCompleteContent(nt) {
    return `
        <div class="detail-section">
            <h3>📹 Recording</h3>
            <p>Video recording available for playback</p>
        </div>
        <div class="detail-section">
            <h3>📝 Transcript</h3>
            <p>Full meeting transcript with speaker labels</p>
        </div>
        <div class="detail-section">
            <h3>✨ AI Summary</h3>
            <p>Get key points and action items from this meeting</p>
        </div>
    `;
},

/**
 * Render content for pending/in-progress recording
 */
renderPendingContent(nt) {
    if (nt.state === 'scheduled') {
        return `
            <div class="detail-section">
                <h3>⏰ Scheduled</h3>
                <p>The bot will join the meeting at the scheduled time.</p>
                <p>Meeting link: <a href="${nt.meetingLink}" target="_blank">${nt.meetingLink || 'N/A'}</a></p>
            </div>
        `;
    }
    if (['connecting', 'waiting_for_entry', 'attending'].includes(nt.state)) {
        return `
            <div class="detail-section recording-indicator">
                <div class="recording-dot"></div>
                <h3>Recording in Progress</h3>
                <p>The bot is currently recording this meeting.</p>
            </div>
        `;
    }
    return `<div class="detail-section"><p>Status: ${this.getStatusText(nt.state)}</p></div>`;
},

/**
 * Render action buttons based on state
 */
renderActions(nt) {
    if (nt.isExternal && nt.externalUrl) {
        return `
            <button class="btn-primary" data-action="notetaker-open-external" data-external-url="${this.escapeHtml(nt.externalUrl)}">
                🔗 Open in Nylas Notebook
            </button>
        `;
    }
    if (nt.state === 'complete' || nt.state === 'completed') {
        return `
            <button class="btn-primary" data-action="notetaker-play" data-not-id="${this.escapeHtml(nt.id)}">
                <svg width="16" height="16" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24">
                    <polygon points="5 3 19 12 5 21 5 3"/>
                </svg>
                Play Recording
            </button>
            <button class="btn-secondary" data-action="notetaker-transcript" data-not-id="${this.escapeHtml(nt.id)}">
                📝 View Transcript
            </button>
            <button class="btn-secondary" data-action="notetaker-summarize" data-not-id="${this.escapeHtml(nt.id)}">
                ✨ AI Summary
            </button>
        `;
    }
    if (nt.state === 'scheduled') {
        return `
            <button class="btn-danger" data-action="notetaker-cancel" data-not-id="${this.escapeHtml(nt.id)}">
                ❌ Cancel Recording
            </button>
        `;
    }
    return '';
},

/**
 * Render the notetaker list as modern cards
 */
renderNotetakerPanel() {
    this.renderNotetakers();
    this.updateCounts();
}
});
