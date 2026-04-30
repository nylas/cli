/* Email Selection - Detail view */
Object.assign(EmailListManager, {
async selectEmail(emailId) {
    this.selectedEmailId = emailId;

    // Update list selection
    document.querySelectorAll('.email-item').forEach(item => {
        const isSelected = item.getAttribute('data-email-id') === emailId;
        item.classList.toggle('selected', isSelected);
        item.setAttribute('aria-selected', isSelected ? 'true' : 'false');
    });

    // Load full email
    try {
        const email = await AirAPI.getEmail(emailId);
        // Store full email for reply/forward (includes thread_id)
        this.selectedEmailFull = email;
        this.renderEmailDetail(email);

        // Render email body in sandboxed iframe (async, after DOM update)
        requestAnimationFrame(() => {
            this.renderEmailBodyIframe(email.id, email.body);
        });

        // Trigger an invite card render when the email looks like an
        // invitation: either it carries a real text/calendar attachment,
        // or its subject matches the patterns Google/Microsoft use for
        // calendar invitations. The subject heuristic is needed because
        // Gmail ships the ICS as an inline body part — Nylas does not
        // surface those in attachments[], so the attachment-only check
        // misses them. The /api/emails/{id}/invite endpoint then walks
        // raw_mime as a fallback.
        if (this.hasCalendarAttachment(email) || this.looksLikeInviteSubject(email)) {
            this.loadAndRenderInvite(email.id).catch((err) => {
                console.warn('[invite] render failed:', err);
            });
        }

        // Mark as read if unread
        if (email.unread) {
            await this.markAsRead(emailId);
        }
    } catch (error) {
        console.error('Failed to load email:', error);
        if (typeof showToast === 'function') {
            showToast('error', 'Error', 'Failed to load email');
        }
    }
},

renderEmailDetail(email) {
    const detailPane = document.querySelector('.email-detail');
    if (!detailPane) return;

    const sender = EmailRenderer.getSenderInfo(email.from);
    const time = new Date(email.date * 1000).toLocaleString();

    const toList = (email.to || [])
        .map(p => p.name || p.email)
        .join(', ');

    detailPane.innerHTML = `
        <div class="email-detail-header">
            <div class="email-detail-subject">${EmailRenderer.escapeHtml(email.subject || '(No Subject)')}</div>
            <div class="email-detail-meta">
                <div class="email-detail-avatar" style="background: var(--gradient-1)">${sender.initials}</div>
                <div class="email-detail-info">
                    <div class="email-detail-from">${EmailRenderer.escapeHtml(sender.name)} <span class="email-detail-email">&lt;${EmailRenderer.escapeHtml(sender.email)}&gt;</span></div>
                    <div class="email-detail-to">To: ${EmailRenderer.escapeHtml(toList || 'me')}</div>
                </div>
                <div class="email-detail-time">${time}</div>
            </div>
        </div>
        <div class="email-detail-actions">
            <button class="action-btn" data-action="reply-email" data-email-id="${escapeHtml(email.id)}" title="Reply">
                <svg width="16" height="16" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24">
                    <path d="M9 17H4a2 2 0 01-2-2V5a2 2 0 012-2h16a2 2 0 012 2v10a2 2 0 01-2 2h-5l-5 5v-5z"/>
                </svg>
                Reply
            </button>
            <button class="action-btn" data-action="toggle-star" data-email-id="${escapeHtml(email.id)}" title="Star">
                <svg width="16" height="16" fill="${email.starred ? 'currentColor' : 'none'}" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24">
                    <polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2"/>
                </svg>
                ${email.starred ? 'Starred' : 'Star'}
            </button>
            <button class="action-btn" data-action="archive-email" data-email-id="${escapeHtml(email.id)}" title="Archive">
                <svg width="16" height="16" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24">
                    <rect x="2" y="4" width="20" height="5" rx="1"/>
                    <path d="M4 9v9a2 2 0 002 2h12a2 2 0 002-2V9M10 13h4"/>
                </svg>
                Archive
            </button>
            <button class="action-btn" data-action="delete-email" data-email-id="${escapeHtml(email.id)}" title="Delete">
                <svg width="16" height="16" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24">
                    <path d="M3 6h18M19 6v14a2 2 0 01-2 2H7a2 2 0 01-2-2V6m3 0V4a2 2 0 012-2h4a2 2 0 012 2v2"/>
                </svg>
                Delete
            </button>
            <button class="action-btn ai-btn" id="summarizeBtn-${escapeHtml(email.id)}" data-action="summarize-email" data-email-id="${escapeHtml(email.id)}" title="Summarize with AI">
                <svg class="ai-icon" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24">
                    <path d="M12 2a10 10 0 100 20 10 10 0 000-20z"/>
                    <path d="M12 6v6l4 2"/>
                </svg>
                <svg class="ai-spinner" width="16" height="16" viewBox="0 0 24 24" style="display:none">
                    <circle cx="12" cy="12" r="10" stroke="currentColor" stroke-width="2" fill="none" stroke-dasharray="31.4" stroke-dashoffset="10">
                        <animateTransform attributeName="transform" type="rotate" from="0 12 12" to="360 12 12" dur="1s" repeatCount="indefinite"/>
                    </circle>
                </svg>
                <span class="ai-btn-text">✨ Summarize</span>
            </button>
        </div>
        <div class="smart-replies-container" id="smartReplies-${escapeHtml(email.id)}">
            <button class="smart-replies-trigger" data-action="load-smart-replies" data-email-id="${escapeHtml(email.id)}">
                <span class="smart-replies-icon">💬</span>
                <span>Get smart reply suggestions</span>
            </button>
        </div>
        <div class="calendar-invite-card-slot" id="inviteSlot-${escapeHtml(email.id)}" hidden></div>
        ${email.attachments && email.attachments.length > 0 ? `
            <div class="email-detail-attachments">
                <div class="attachments-header">Attachments (${email.attachments.length})</div>
                <div class="attachments-list">
                    ${email.attachments.map(a => `
                        <div class="attachment-item">
                            <span class="attachment-icon">&#128206;</span>
                            <span class="attachment-name">${EmailRenderer.escapeHtml(a.filename)}</span>
                            <span class="attachment-size">${this.formatSize(a.size)}</span>
                        </div>
                    `).join('')}
                </div>
            </div>
        ` : ''}
        <div class="email-detail-body">
            <div class="email-iframe-container" id="emailBodyContainer-${email.id}">
                <div class="email-loading-state">
                    <div class="email-loading-spinner"></div>
                    <span>Loading email...</span>
                </div>
            </div>
        </div>
    `;
},

// Render email body into a sandboxed iframe for security and proper HTML rendering
renderEmailBodyIframe(emailId, bodyHtml) {
    const container = document.getElementById(`emailBodyContainer-${emailId}`);
    if (!container) return;

    // Create sandboxed iframe - no scripts, no forms, no popups
    const iframe = document.createElement('iframe');
    iframe.className = 'email-body-iframe';
    iframe.setAttribute('sandbox', 'allow-same-origin'); // Minimal permissions - no scripts
    iframe.setAttribute('title', 'Email content');
    iframe.setAttribute('loading', 'lazy');

    // Build the email content with embedded styles for proper rendering
    const emailContent = this.buildEmailIframeContent(bodyHtml);

    // Use srcdoc for security - content is isolated
    iframe.srcdoc = emailContent;

    // Handle iframe load
    iframe.onload = () => {
        // Process and hide broken/tracking images first
        this.processIframeImages(iframe);
        // Auto-resize iframe to content height
        this.resizeIframeToContent(iframe);
        // Add loaded class for fade-in animation
        container.classList.add('loaded');
        // Make links open in new tab
        this.processIframeLinks(iframe);
    };

    iframe.onerror = () => {
        container.innerHTML = `
            <div class="email-error-state">
                <span class="error-icon">⚠️</span>
                <span>Failed to load email content</span>
            </div>
        `;
    };

    // Clear loading state and add iframe
    container.innerHTML = '';
    container.appendChild(iframe);
},

// Build the HTML content for the email iframe with embedded styles
buildEmailIframeContent(bodyHtml) {
    // Default to empty paragraph if no content
    const content = bodyHtml || '<p style="color: #71717a; font-style: italic;">No content</p>';

    return `<!DOCTYPE html>
<html>
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<style>
    /* Reset and base styles */
    *, *::before, *::after {
        box-sizing: border-box;
    }

    html, body {
        margin: 0;
        padding: 0;
        font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
        font-size: 15px;
        line-height: 1.65;
        color: #1a1a2e;
        background: transparent;
        -webkit-font-smoothing: antialiased;
        -moz-osx-font-smoothing: grayscale;
    }

    body {
        padding: 4px;
    }

    /* Typography */
    p {
        margin: 0 0 1em 0;
    }

    p:last-child {
        margin-bottom: 0;
    }

    h1, h2, h3, h4, h5, h6 {
        margin: 0 0 0.5em 0;
        font-weight: 600;
        line-height: 1.3;
    }

    h1 { font-size: 1.75em; }
    h2 { font-size: 1.5em; }
    h3 { font-size: 1.25em; }

    /* Links */
    a {
        color: #6366f1;
        text-decoration: none;
        transition: color 0.15s ease;
    }

    a:hover {
        color: #4f46e5;
        text-decoration: underline;
    }

    /* Images */
    img {
        max-width: 100%;
        height: auto;
    }

    /* Hide broken images - applied via JS */
    img.broken-image,
    img.tracking-pixel {
        display: none !important;
        visibility: hidden !important;
        width: 0 !important;
        height: 0 !important;
        opacity: 0 !important;
    }

    /* Tables - common in HTML emails */
    table {
        border-collapse: collapse;
        max-width: 100%;
        width: auto;
    }

    td, th {
        padding: 8px 12px;
        text-align: left;
        vertical-align: top;
    }

    /* Blockquotes - for email threads */
    blockquote {
        margin: 1em 0;
        padding: 0.5em 0 0.5em 1em;
        border-left: 3px solid #e5e7eb;
        color: #52525b;
    }

    /* Code blocks */
    pre, code {
        font-family: 'SF Mono', Monaco, 'Cascadia Code', monospace;
        font-size: 0.9em;
        background: #f4f4f5;
        border-radius: 4px;
    }

    code {
        padding: 0.15em 0.4em;
    }

    pre {
        padding: 1em;
        overflow-x: auto;
        white-space: pre-wrap;
        word-wrap: break-word;
    }

    pre code {
        padding: 0;
        background: transparent;
    }

    /* Lists */
    ul, ol {
        margin: 0.5em 0;
        padding-left: 1.5em;
    }

    li {
        margin-bottom: 0.25em;
    }

    /* Horizontal rules */
    hr {
        border: none;
        border-top: 1px solid #e5e7eb;
        margin: 1.5em 0;
    }

    /* Hide tracking pixels and invisible images */
    img[width="1"], img[height="1"],
    img[width="0"], img[height="0"],
    img[width="1px"], img[height="1px"],
    img[style*="display:none"],
    img[style*="display: none"],
    img[style*="width: 1px"],
    img[style*="height: 1px"],
    img[style*="width:1px"],
    img[style*="height:1px"],
    img[src*="tracking"],
    img[src*="beacon"],
    img[src*="pixel"],
    img[src*="open."],
    img[src*="/o/"],
    img[src*="mailtrack"] {
        display: none !important;
        width: 0 !important;
        height: 0 !important;
    }

    /* Force word wrapping for long URLs/text */
    * {
        word-wrap: break-word;
        overflow-wrap: break-word;
    }
</style>
</head>
<body>${content}</body>
</html>`;
},

// Resize iframe to fit its content
resizeIframeToContent(iframe) {
    try {
        const doc = iframe.contentDocument || iframe.contentWindow?.document;
        if (doc && doc.body) {
            // Get the actual content height
            const height = Math.max(
                doc.body.scrollHeight,
                doc.body.offsetHeight,
                doc.documentElement?.scrollHeight || 0,
                doc.documentElement?.offsetHeight || 0
            );
            // Set minimum height and add small buffer
            iframe.style.height = Math.max(height + 20, 100) + 'px';
        }
    } catch (e) {
        // Cross-origin restrictions - use default height
        console.warn('Could not resize iframe:', e);
        iframe.style.height = '400px';
    }
},

// Process links in iframe to open in new tab
processIframeLinks(iframe) {
    try {
        const doc = iframe.contentDocument || iframe.contentWindow?.document;
        if (doc) {
            const links = doc.querySelectorAll('a[href]');
            links.forEach(link => {
                link.setAttribute('target', '_blank');
                link.setAttribute('rel', 'noopener noreferrer');
            });
        }
    } catch (e) {
        console.warn('Could not process iframe links:', e);
    }
},

// Process images in iframe - hide broken and tracking images
processIframeImages(iframe) {
    try {
        const doc = iframe.contentDocument || iframe.contentWindow?.document;
        if (!doc) return;

        const images = doc.querySelectorAll('img');
        images.forEach(img => {
            // Check if image is a tracking pixel by size
            const isTrackingPixel = (
                img.naturalWidth <= 3 ||
                img.naturalHeight <= 3 ||
                img.width <= 3 ||
                img.height <= 3 ||
                (img.getAttribute('width') && parseInt(img.getAttribute('width')) <= 3) ||
                (img.getAttribute('height') && parseInt(img.getAttribute('height')) <= 3)
            );

            if (isTrackingPixel) {
                img.classList.add('tracking-pixel');
                img.style.display = 'none';
                return;
            }

            // Handle broken images
            img.onerror = () => {
                img.classList.add('broken-image');
                img.style.display = 'none';
            };

            // Check if already broken (naturalWidth is 0 for broken images)
            if (img.complete && img.naturalWidth === 0) {
                img.classList.add('broken-image');
                img.style.display = 'none';
            }
        });

        // Re-run check after a short delay for images still loading
        setTimeout(() => {
            images.forEach(img => {
                if (img.complete && img.naturalWidth === 0 && !img.classList.contains('broken-image')) {
                    img.classList.add('broken-image');
                    img.style.display = 'none';
                }
                // Final check for tiny images
                if (img.complete && (img.naturalWidth <= 3 || img.naturalHeight <= 3)) {
                    img.classList.add('tracking-pixel');
                    img.style.display = 'none';
                }
            });
            // Resize iframe after hiding images
            this.resizeIframeToContent(iframe);
        }, 500);

    } catch (e) {
        console.warn('Could not process iframe images:', e);
    }
},

formatSize(bytes) {
    if (bytes < 1024) return bytes + ' B';
    if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB';
    return (bytes / (1024 * 1024)).toFixed(1) + ' MB';
},

// hasCalendarAttachment returns true if the email has at least one
// attachment that looks like an iCalendar invite (.ics or text/calendar).
hasCalendarAttachment(email) {
    if (!email || !Array.isArray(email.attachments)) return false;
    return email.attachments.some((a) => {
        const ct = (a.content_type || a.contentType || '').toLowerCase();
        const fn = (a.filename || '').toLowerCase();
        return ct.startsWith('text/calendar') ||
               ct.startsWith('application/ics') ||
               fn.endsWith('.ics');
    });
},

// looksLikeInviteSubject is a cheap heuristic for "should we ask the
// /invite endpoint about this email even though attachments[] has no
// ICS?". Matches Google Calendar's "Invitation: …" / "Event
// Invitation: …" / "Updated invitation: …" / "Canceled event: …"
// patterns and Outlook's "Updated event: …". Returns false on any
// non-string subject so we never trip on cached or partial responses.
looksLikeInviteSubject(email) {
    if (!email || typeof email.subject !== 'string') return false;
    return /\b(invitation|invite|event invitation|calendar invitation|updated event|canceled event|cancelled event)\b/i.test(email.subject);
},

// loadAndRenderInvite fetches /api/emails/{id}/invite and renders the
// returned event into the calendar-invite-card-slot. Silently no-ops if
// the slot is missing (e.g., user navigated away) or the response says
// has_invite=false.
async loadAndRenderInvite(emailId) {
    const slot = document.getElementById(`inviteSlot-${emailId}`);
    if (!slot) return;

    let invite;
    try {
        const resp = await fetch(`/api/emails/${encodeURIComponent(emailId)}/invite`);
        if (!resp.ok) return;
        invite = await resp.json();
    } catch (err) {
        console.warn('[invite] fetch failed:', err);
        return;
    }
    if (!invite || !invite.has_invite) return;

    // Skip if user has navigated to a different email since the fetch
    // was kicked off.
    if (this.selectedEmailId !== emailId) return;

    // Replace existing children safely, then insert sanitised markup.
    // All interpolated strings pass through EmailRenderer.escapeHtml; URL
    // is screened by isSafeUrl. We use insertAdjacentHTML rather than
    // direct DOM construction to keep the markup colocated and readable.
    slot.replaceChildren();
    slot.insertAdjacentHTML('beforeend', this.renderCalendarInviteCard(invite, emailId));
    slot.removeAttribute('hidden');

    // Mirror Gmail's behaviour by surfacing the ICS as a regular
    // attachment row when the invite came from an inline calendar part.
    // For real attachments the email-detail render already shows them,
    // so we only need to inject when the row isn't there yet.
    this.ensureInviteAttachmentRow(emailId, invite);
},

// ensureInviteAttachmentRow appends a calendar-attachment row to the
// email detail view when one isn't already rendered. Used for inline
// calendar parts that Nylas does not surface in attachments[].
ensureInviteAttachmentRow(emailId, invite) {
    if (!invite || !invite.filename) return;
    if (this.selectedEmailId !== emailId) return;

    const detail = document.querySelector('.email-detail');
    if (!detail) return;

    // Skip if any existing attachment-name already matches the invite
    // filename — avoids double-rendering when Nylas DID return the ICS
    // as a real attachment.
    const existing = detail.querySelectorAll('.email-detail-attachments .attachment-name');
    for (const el of existing) {
        if ((el.textContent || '').toLowerCase() === invite.filename.toLowerCase()) {
            return;
        }
    }

    const esc = EmailRenderer.escapeHtml;
    const rowHTML = `
        <div class="attachment-item" data-inline-calendar="true">
            <span class="attachment-icon">&#128206;</span>
            <span class="attachment-name">${esc(invite.filename)}</span>
            <span class="attachment-size">Calendar invitation</span>
        </div>
    `;

    const list = detail.querySelector('.email-detail-attachments .attachments-list');
    if (list) {
        list.insertAdjacentHTML('beforeend', rowHTML);
        const header = detail.querySelector('.email-detail-attachments .attachments-header');
        if (header) {
            const count = list.querySelectorAll('.attachment-item').length;
            header.textContent = `Attachments (${count})`;
        }
        return;
    }

    // No attachments section exists yet — render one and place it
    // directly after the invite card so it sits where the user expects.
    const slot = document.getElementById(`inviteSlot-${emailId}`);
    if (!slot) return;
    const sectionHTML = `
        <div class="email-detail-attachments">
            <div class="attachments-header">Attachments (1)</div>
            <div class="attachments-list">${rowHTML}</div>
        </div>
    `;
    slot.insertAdjacentHTML('afterend', sectionHTML);
},

// renderCalendarInviteCard returns the HTML for a Gmail-style invite
// card. All untrusted strings are escaped; the URL is also screened by
// isSafeUrl before being placed in href.
renderCalendarInviteCard(invite, emailId) {
    const esc = EmailRenderer.escapeHtml;
    const fmtRange = this.formatInviteRange(invite);
    const safeURL = (typeof isSafeUrl === 'function' && invite.conferencing_url &&
        isSafeUrl(invite.conferencing_url)) ? invite.conferencing_url : '';

    const orgLine = invite.organizer_email
        ? `${esc(invite.organizer_name || invite.organizer_email)} · Organizer`
        : '';

    const isCancelled = String(invite.method || '').toUpperCase() === 'CANCEL' ||
        String(invite.status || '').toUpperCase() === 'CANCELLED';
    const banner = isCancelled
        ? `<div class="calendar-invite-banner calendar-invite-banner-cancel" role="alert">This event was cancelled</div>`
        : '';

    const attendeesBlock = this.renderInviteAttendees(invite.attendees);
    const actions = isCancelled
        ? ''
        : `<div class="calendar-invite-actions">
                <button type="button" class="calendar-invite-btn primary"
                        data-action="invite-rsvp" data-email-id="${esc(emailId)}" data-rsvp="yes">Yes</button>
                <button type="button" class="calendar-invite-btn"
                        data-action="invite-rsvp" data-email-id="${esc(emailId)}" data-rsvp="maybe">Maybe</button>
                <button type="button" class="calendar-invite-btn"
                        data-action="invite-rsvp" data-email-id="${esc(emailId)}" data-rsvp="no">No</button>
            </div>`;

    return `
        <section class="calendar-invite-card${isCancelled ? ' is-cancelled' : ''}" role="region" aria-label="Calendar invitation">
            ${banner}
            <header class="calendar-invite-header">
                <div class="calendar-invite-icon" aria-hidden="true">📅</div>
                <div class="calendar-invite-when">
                    <div class="calendar-invite-time">${esc(fmtRange)}</div>
                    <div class="calendar-invite-title">${esc(invite.title || 'Untitled event')}</div>
                </div>
            </header>
            ${invite.location ? `<div class="calendar-invite-location">📍 ${esc(invite.location)}</div>` : ''}
            ${orgLine ? `<div class="calendar-invite-org">${orgLine}</div>` : ''}
            ${safeURL ? `<a class="calendar-invite-link" href="${esc(safeURL)}" target="_blank" rel="noopener noreferrer">Join with conferencing</a>` : ''}
            ${attendeesBlock}
            ${actions}
        </section>
    `;
},

// renderInviteAttendees produces the attendee summary that mirrors
// Gmail's "3 going, 1 declined" line and the per-attendee chip list.
// Returns an empty string when no attendees are present so the card
// stays compact for invitations without an explicit attendee list
// (Outlook sometimes omits ATTENDEE on REQUEST).
renderInviteAttendees(attendees) {
    if (!Array.isArray(attendees) || attendees.length === 0) return '';

    const esc = EmailRenderer.escapeHtml;
    const counts = { ACCEPTED: 0, DECLINED: 0, TENTATIVE: 0, OTHER: 0 };
    attendees.forEach((a) => {
        const status = String(a.status || '').toUpperCase();
        if (status in counts) counts[status]++;
        else counts.OTHER++;
    });

    const parts = [];
    if (counts.ACCEPTED > 0) parts.push(`${counts.ACCEPTED} going`);
    if (counts.DECLINED > 0) parts.push(`${counts.DECLINED} declined`);
    if (counts.TENTATIVE > 0) parts.push(`${counts.TENTATIVE} maybe`);
    if (counts.OTHER > 0) parts.push(`${counts.OTHER} no response`);

    const summary = parts.length > 0
        ? `<div class="calendar-invite-summary">${esc(parts.join(' · '))}</div>`
        : '';

    const chips = attendees.slice(0, 8).map((a) => {
        const label = a.name || a.email || '';
        if (!label) return '';
        const status = String(a.status || '').toUpperCase();
        const cls = status === 'ACCEPTED' ? 'is-accepted'
            : status === 'DECLINED' ? 'is-declined'
            : status === 'TENTATIVE' ? 'is-tentative'
            : 'is-pending';
        const role = a.is_organizer ? ' · Organizer' : '';
        return `<span class="calendar-invite-attendee ${cls}" title="${esc(a.email || '')}${esc(role)}">${esc(label)}</span>`;
    }).filter(Boolean).join('');

    const overflow = attendees.length > 8
        ? `<span class="calendar-invite-attendee is-overflow">+${attendees.length - 8} more</span>`
        : '';

    return `
        <div class="calendar-invite-attendees">
            ${summary}
            <div class="calendar-invite-attendee-list">${chips}${overflow}</div>
        </div>
    `;
},

// formatInviteRange turns Unix-second start/end into a human string.
// Falls back gracefully if either side is missing.
formatInviteRange(invite) {
    const start = invite.start_time ? new Date(invite.start_time * 1000) : null;
    const end = invite.end_time ? new Date(invite.end_time * 1000) : null;

    if (invite.is_all_day && start) {
        return `${start.toLocaleDateString(undefined, { weekday: 'short', year: 'numeric', month: 'short', day: 'numeric' })} · All day`;
    }
    if (start && end) {
        const sameDay = start.toDateString() === end.toDateString();
        const dateOpts = { weekday: 'short', month: 'short', day: 'numeric' };
        const timeOpts = { hour: 'numeric', minute: '2-digit' };
        if (sameDay) {
            return `${start.toLocaleDateString(undefined, dateOpts)} · ${start.toLocaleTimeString(undefined, timeOpts)} – ${end.toLocaleTimeString(undefined, timeOpts)}`;
        }
        return `${start.toLocaleString(undefined, { ...dateOpts, ...timeOpts })} – ${end.toLocaleString(undefined, { ...dateOpts, ...timeOpts })}`;
    }
    if (start) return start.toLocaleString();
    return 'Time not specified';
},

// rsvpToInvite handles a Yes/No/Maybe click. Currently a stub — shows a
// toast confirming the choice and updates button state. Backend wiring
// will plug into PUT /api/events/{id}/rsvp once the embedded event ID is
// surfaced from the parser.
rsvpToInvite(emailId, response) {
    const valid = new Set(['yes', 'no', 'maybe']);
    const choice = String(response || '').toLowerCase();
    if (!valid.has(choice)) return;

    if (typeof showToast === 'function') {
        const labels = { yes: 'Accepted', no: 'Declined', maybe: 'Tentative' };
        showToast('success', labels[choice], `Invite ${labels[choice].toLowerCase()}`);
    }

    // Visual feedback — mark the chosen button as active.
    const slot = document.getElementById(`inviteSlot-${emailId}`);
    if (slot) {
        slot.querySelectorAll('.calendar-invite-btn').forEach((btn) => {
            btn.classList.toggle('active', btn.dataset.rsvp === choice);
        });
    }
},

});

// Single delegated listener for all email-detail action buttons and smart-reply triggers.
// Installed once at module load; covers buttons rendered by renderEmailDetail and email-ai.js.
document.addEventListener('click', function (e) {
    const target = e.target.closest('[data-action]');
    if (!target) return;
    const action = target.dataset.action;
    const emailId = target.dataset.emailId;
    switch (action) {
        case 'reply-email':
            EmailListManager.replyToEmail(emailId);
            break;
        case 'toggle-star':
            EmailListManager.toggleStar(emailId);
            break;
        case 'archive-email':
            EmailListManager.archiveEmail(emailId);
            break;
        case 'delete-email':
            EmailListManager.deleteEmail(emailId);
            break;
        case 'summarize-email':
            EmailListManager.summarizeWithAI(emailId);
            break;
        case 'load-smart-replies':
            EmailListManager.loadSmartReplies(emailId);
            break;
        case 'use-smart-reply': {
            const replyIndex = parseInt(target.dataset.replyIndex, 10);
            EmailListManager.useSmartReply(emailId, replyIndex);
            break;
        }
        case 'invite-rsvp':
            EmailListManager.rsvpToInvite(emailId, target.dataset.rsvp);
            break;
    }
});
