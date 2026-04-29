/* Email AI Features - Smart replies and summarization */

Object.assign(EmailListManager, {
async loadSmartReplies(emailId) {
    const email = this.selectedEmailFull && this.selectedEmailFull.id === emailId
        ? this.selectedEmailFull
        : this.emails.find(e => e.id === emailId);

    if (!email) return;

    const container = document.getElementById(`smartReplies-${emailId}`);
    if (!container) return;

    // Show loading state
    container.innerHTML = `
        <div class="smart-replies-loading">
            <span class="loading-spinner"></span>
            <span>Generating smart replies...</span>
        </div>
    `;

    // Extract plain text from email body
    const parser = new DOMParser();
    const doc = parser.parseFromString(email.body || '', 'text/html');
    const plainText = doc.body?.textContent || '';

    try {
        const response = await fetch('/api/ai/smart-replies', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                email_id: emailId,
                subject: email.subject || '',
                from: email.from?.map(f => f.name || f.email).join(', ') || '',
                body: plainText.substring(0, 2000)
            })
        });

        const result = await response.json();

        if (result.success && result.replies && result.replies.length > 0) {
            container.innerHTML = `
                <div class="smart-replies-header">
                    <span class="smart-replies-icon">💬</span>
                    <span>Smart Replies</span>
                </div>
                <div class="smart-replies-list">
                    ${result.replies.map((reply, i) => `
                        <button class="smart-reply-chip" data-action="use-smart-reply" data-email-id="${escapeHtml(emailId)}" data-reply-index="${i}" data-reply="${this.escapeHtml(reply)}">
                            ${this.escapeHtml(reply)}
                        </button>
                    `).join('')}
                </div>
            `;
            // Store replies for use
            this.smartReplies = result.replies;
        } else {
            container.innerHTML = `
                <button class="smart-replies-trigger" data-action="load-smart-replies" data-email-id="${escapeHtml(emailId)}">
                    <span class="smart-replies-icon">💬</span>
                    <span>Get smart reply suggestions</span>
                </button>
            `;
            if (result.error) {
                if (typeof showToast === 'function') {
                    showToast('error', 'AI Error', result.error);
                }
            }
        }
    } catch (err) {
        console.error('Smart replies error:', err);
        container.innerHTML = `
            <button class="smart-replies-trigger" data-action="load-smart-replies" data-email-id="${escapeHtml(emailId)}">
                <span class="smart-replies-icon">💬</span>
                <span>Get smart reply suggestions</span>
            </button>
        `;
    }
},

// Use a smart reply suggestion
useSmartReply(emailId, replyIndex) {
    if (!this.smartReplies || !this.smartReplies[replyIndex]) return;

    const reply = this.smartReplies[replyIndex];
    const email = this.selectedEmailFull && this.selectedEmailFull.id === emailId
        ? this.selectedEmailFull
        : this.emails.find(e => e.id === emailId);

    if (email && typeof ComposeManager !== 'undefined') {
        ComposeManager.openReply(email, reply);
    }
},

// Summarize email with AI (enhanced version)
async summarizeWithAI(emailId) {
    const email = this.selectedEmailFull && this.selectedEmailFull.id === emailId
        ? this.selectedEmailFull
        : this.emails.find(e => e.id === emailId);

    if (!email) {
        if (typeof showToast === 'function') {
            showToast('error', 'Error', 'Email not found');
        }
        return;
    }

    // Get button and show loading state
    const btn = document.getElementById(`summarizeBtn-${emailId}`);
    if (btn) {
        btn.classList.add('loading');
        btn.disabled = true;
        const icon = btn.querySelector('.ai-icon');
        const spinner = btn.querySelector('.ai-spinner');
        const text = btn.querySelector('.ai-btn-text');
        if (icon) icon.style.display = 'none';
        if (spinner) spinner.style.display = 'block';
        if (text) text.textContent = 'Analyzing...';
    }

    // Extract plain text from email body safely using DOMParser
    const parser = new DOMParser();
    const doc = parser.parseFromString(email.body || '', 'text/html');
    const plainText = doc.body?.textContent || '';

    try {
        // Use enhanced summary endpoint
        const response = await fetch('/api/ai/enhanced-summary', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                email_id: emailId,
                subject: email.subject || '',
                from: email.from?.map(f => f.name || f.email).join(', ') || '',
                body: plainText.substring(0, 3000)
            })
        });

        const result = await response.json();

        if (result.success) {
            // Show enhanced summary modal
            this.showEnhancedSummaryModal(email.subject, result);
        } else {
            if (typeof showToast === 'function') {
                showToast('error', 'AI Error', result.error || 'Failed to summarize');
            }
        }
    } catch (err) {
        console.error('AI summarize error:', err);
        if (typeof showToast === 'function') {
            showToast('error', 'Error', 'Failed to connect to Claude Code');
        }
    } finally {
        // Reset button state
        if (btn) {
            btn.classList.remove('loading');
            btn.disabled = false;
            const icon = btn.querySelector('.ai-icon');
            const spinner = btn.querySelector('.ai-spinner');
            const text = btn.querySelector('.ai-btn-text');
            if (icon) icon.style.display = 'block';
            if (spinner) spinner.style.display = 'none';
            if (text) text.textContent = '✨ Summarize';
        }
    }
},

// Show enhanced AI summary modal with action items and sentiment
showEnhancedSummaryModal(subject, result) {
    const sentimentIcons = {
        positive: '😊',
        neutral: '😐',
        negative: '😟',
        urgent: '🚨'
    };

    const categoryIcons = {
        meeting: '📅',
        task: '✅',
        fyi: 'ℹ️',
        question: '❓',
        social: '👋'
    };

    const sentimentIcon = sentimentIcons[result.sentiment] || '😐';
    const categoryIcon = categoryIcons[result.category] || 'ℹ️';

    // Build action items HTML
    const actionItemsHtml = result.action_items && result.action_items.length > 0
        ? `<div class="summary-section">
            <div class="summary-section-title">📋 Action Items</div>
            <ul class="action-items-list">
                ${result.action_items.map(item => `<li>${this.escapeHtml(item)}</li>`).join('')}
            </ul>
           </div>`
        : '';

    let modal = document.getElementById('aiSummaryModal');
    if (!modal) {
        modal = document.createElement('div');
        modal.id = 'aiSummaryModal';
        modal.className = 'modal-overlay';
        document.body.appendChild(modal);
    }

    modal.innerHTML = `
        <div class="modal ai-summary-modal">
            <div class="modal-header">
                <h3>✨ AI Analysis</h3>
                <button class="close-btn" data-action="ai-summary-close">&times;</button>
            </div>
            <div class="modal-body">
                <div class="summary-subject">${this.escapeHtml(subject || '(No Subject)')}</div>
                <div class="summary-badges">
                    <span class="summary-badge sentiment-${this.escapeHtml(result.sentiment)}">${sentimentIcon} ${this.escapeHtml(result.sentiment)}</span>
                    <span class="summary-badge category-${this.escapeHtml(result.category)}">${categoryIcon} ${this.escapeHtml(result.category)}</span>
                </div>
                <div class="summary-section">
                    <div class="summary-section-title">📝 Summary</div>
                    <div class="summary-content">${this.escapeHtml(result.summary)}</div>
                </div>
                ${actionItemsHtml}
            </div>
            <div class="modal-footer">
                <button class="btn btn-secondary" data-action="ai-summary-copy">Copy Summary</button>
                <button class="btn btn-primary" data-action="ai-summary-close">Close</button>
            </div>
        </div>
    `;

    modal.style.display = 'flex';
    modal.classList.add('active');

    // Store summary for copying
    this.currentSummary = result.summary;
    if (result.action_items && result.action_items.length > 0) {
        this.currentSummary += '\n\nAction Items:\n' + result.action_items.map(item => '• ' + item).join('\n');
    }
},

// Show AI summary in a modal (legacy - for basic summarize)
showAISummaryModal(subject, summary) {
    // Check if modal already exists
    let modal = document.getElementById('aiSummaryModal');
    if (!modal) {
        modal = document.createElement('div');
        modal.id = 'aiSummaryModal';
        modal.className = 'modal-overlay';
        modal.innerHTML = `
            <div class="modal ai-summary-modal">
                <div class="modal-header">
                    <h3>✨ AI Summary</h3>
                    <button class="close-btn" data-action="ai-summary-close">&times;</button>
                </div>
                <div class="modal-body">
                    <div class="summary-subject"></div>
                    <div class="summary-content"></div>
                </div>
                <div class="modal-footer">
                    <button class="btn btn-secondary" data-action="ai-summary-copy">Copy Summary</button>
                    <button class="btn btn-primary" data-action="ai-summary-close">Close</button>
                </div>
            </div>
        `;
        document.body.appendChild(modal);
    }

    // Update content
    modal.querySelector('.summary-subject').textContent = subject || '(No Subject)';
    modal.querySelector('.summary-content').textContent = summary;
    modal.style.display = 'flex';
    modal.classList.add('active');

    // Store summary for copying
    this.currentSummary = summary;
},

// Close AI summary modal
closeAISummaryModal() {
    const modal = document.getElementById('aiSummaryModal');
    if (modal) {
        modal.classList.remove('active');
        setTimeout(() => modal.style.display = 'none', 200);
    }
},

// Copy AI summary to clipboard
async copyAISummary() {
    if (this.currentSummary) {
        try {
            await navigator.clipboard.writeText(this.currentSummary);
            if (typeof showToast === 'function') {
                showToast('success', 'Copied', 'Summary copied to clipboard');
            }
        } catch (err) {
            console.error('Copy error:', err);
        }
    }
}
});
