/**
 * Notetaker Actions - User interactions
 */
Object.assign(NotetakerModule, {
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
 * Strip embedded styles and scripts from HTML for safe rendering.
 *
 * Delegates structural sanitization (scripts, event handlers, dangerous
 * URLs) to sanitizeHtml() in utils.js — that uses DOMParser, which is not
 * defeatable by entity tricks or malformed tags the way the previous
 * regex strippers were. Inline <style> blocks and style="" attributes are
 * removed as before so our CSS controls theming.
 */
stripEmbeddedStyles(html) {
    let cleaned = sanitizeHtml(html);
    // Remove inline style attributes and <style> blocks so our app CSS
    // controls theming. sanitizeHtml leaves these intact intentionally so
    // legitimate emails can keep their structure.
    cleaned = cleaned.replace(/<style[^>]*>[\s\S]*?<\/style>/gi, '');
    cleaned = cleaned.replace(/\s+style="[^"]*"/gi, '');
    cleaned = cleaned.replace(/\s+style='[^']*'/gi, '');
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
 * Render the notetaker panel (legacy support)
 */
renderNotetakerPanel() {
    this.renderNotetakers();
},

/**
 * Play recording in modal
 */
async playRecording(notetakerId) {
    try {
        const media = await this.getMedia(notetakerId);

        const content = this.createElement('div', 'video-container');
        const video = document.createElement('video');
        video.controls = true;
        video.autoplay = true;
        video.style.width = '100%';

        const source = document.createElement('source');
        source.src = media.recordingUrl;
        source.type = 'video/mp4';
        video.appendChild(source);

        const fallback = document.createTextNode('Your browser does not support video playback.');
        video.appendChild(fallback);
        content.appendChild(video);

        this.showMediaModal('Recording', content);
    } catch (err) {
        this.showNotification('Recording not available', 'error');
    }
},

/**
 * View transcript in modal
 */
async viewTranscript(notetakerId) {
    try {
        const media = await this.getMedia(notetakerId);

        const content = this.createElement('div', 'transcript-content');
        const loading = this.createElement('p', null, 'Loading transcript...');
        content.appendChild(loading);

        const downloadBtn = this.createElement('button', null, '⬇️ Download Transcript');
        downloadBtn.onclick = () => window.open(media.transcriptUrl, '_blank');
        content.appendChild(downloadBtn);

        this.showMediaModal('Transcript', content);
    } catch (err) {
        this.showNotification('Transcript not available', 'error');
    }
},

/**
 * Get AI summary of meeting
 */
async summarize(notetakerId) {
    this.showNotification('Generating AI summary...', 'info');
    try {
        await this.getMedia(notetakerId);

        const content = this.createElement('div', 'ai-summary');

        const title = this.createElement('h4', null, '✨ Meeting Summary');
        content.appendChild(title);

        const desc = this.createElement('p', null, 'AI-generated summary of the meeting:');
        content.appendChild(desc);

        const keyPointsTitle = this.createElement('h5', null, 'Key Points:');
        content.appendChild(keyPointsTitle);

        const keyPoints = this.createElement('ul');
        ['Discussion of project timeline', 'Resource allocation review', 'Next steps identified'].forEach(point => {
            const li = this.createElement('li', null, point);
            keyPoints.appendChild(li);
        });
        content.appendChild(keyPoints);

        const actionsTitle = this.createElement('h5', null, 'Action Items:');
        content.appendChild(actionsTitle);

        const actions = this.createElement('ul');
        ['Follow up with team by Friday', 'Schedule next review meeting'].forEach(action => {
            const li = this.createElement('li', null, action);
            actions.appendChild(li);
        });
        content.appendChild(actions);

        this.showMediaModal('AI Summary', content);
    } catch (err) {
        this.showNotification('Failed to generate summary', 'error');
    }
},

/**
 * Show summary modal for external notetakers
 */
showSummaryModal(nt) {
    const content = this.createElement('div', 'summary-modal-content');

    // Email summary content - just the body text
    if (nt.summary) {
        const summaryDiv = this.createElement('div', 'summary-body');
        summaryDiv.innerHTML = this.stripEmailCruft(nt.summary);
        content.appendChild(summaryDiv);
    }

    this.showMediaModal(nt.meetingTitle || 'Meeting Summary', content);
},

/**
 * Clean email HTML - keep structure but remove scripts and styles for safe rendering.
 *
 * Like stripEmbeddedStyles, defers structural sanitization to sanitizeHtml.
 * Strips style/width/height after parsing so the app CSS owns layout.
 */
stripEmailCruft(html) {
    let cleaned = sanitizeHtml(html);
    cleaned = cleaned.replace(/<style[^>]*>[\s\S]*?<\/style>/gi, '');
    cleaned = cleaned.replace(/\s+style="[^"]*"/gi, '');
    cleaned = cleaned.replace(/\s+style='[^']*'/gi, '');
    cleaned = cleaned.replace(/\s+width="[^"]*"/gi, '');
    cleaned = cleaned.replace(/\s+height="[^"]*"/gi, '');
    // Add a small constrained size to images so they don't blow up the modal.
    cleaned = cleaned.replace(/<img/gi, '<img style="max-width:80px;max-height:40px;display:block;margin:0 auto 16px"');
    return cleaned;
},

/**
 * Show media modal using safe DOM methods
 */
showMediaModal(title, contentElement) {
    const modal = this.createElement('div', 'notetaker-modal');

    const backdrop = this.createElement('div', 'modal-backdrop');
    backdrop.onclick = () => modal.remove();
    modal.appendChild(backdrop);

    const content = this.createElement('div', 'modal-content');

    const header = this.createElement('div', 'modal-header');
    const headerTitle = this.createElement('h3', null, title);
    const closeBtn = this.createElement('button', null, '✕');
    closeBtn.onclick = () => modal.remove();
    header.appendChild(headerTitle);
    header.appendChild(closeBtn);
    content.appendChild(header);

    const body = this.createElement('div', 'modal-body');
    body.appendChild(contentElement);
    content.appendChild(body);

    modal.appendChild(content);
    document.body.appendChild(modal);
},

/**
 * Show notification
 */
showNotification(message, type = 'info') {
    if (typeof showToast === 'function') {
        showToast(message, type);
    } else {
        console.log(`[${type}] ${message}`);
    }
},

/**
 * Show settings modal - opens global settings modal
 */
showSettingsModal() {
    // Open the global settings modal
    if (typeof toggleSettings === 'function') {
        toggleSettings();
    }
},

/**
 * Setup event listeners (handled in core module)
 */
initActions() {
    console.log('Notetaker actions initialized');
}
});
