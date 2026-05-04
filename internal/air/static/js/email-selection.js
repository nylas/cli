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

// renderEmailDetail builds the right-pane email view. We use DOM
// construction (createElement + textContent + setAttribute) instead of a
// single innerHTML template so:
//   1. There is no template-literal interpolation of user data, so a
//      future change can't accidentally introduce an XSS vector by
//      forgetting to escape one field.
//   2. IDs derived from email.id round-trip cleanly between write
//      (setAttribute stores the raw string) and lookup (getElementById
//      reads the raw string) — no HTML-encoding dance required.
//   3. Static SVG markup is the only innerHTML use, kept inline because
//      DOM-constructing each path tag would dwarf the actual logic.
renderEmailDetail(email) {
    const detailPane = document.querySelector('.email-detail');
    if (!detailPane) return;

    const sender = EmailRenderer.getSenderInfo(email.from);
    const time = new Date(email.date * 1000).toLocaleString();
    const toList = (email.to || []).map(p => p.name || p.email).join(', ');

    detailPane.replaceChildren(
        this.buildDetailHeader(email, sender, toList, time),
        this.buildDetailActions(email),
        this.buildSmartRepliesSlot(email.id),
        this.buildInviteSlot(email.id),
    );

    if (email.attachments && email.attachments.length > 0) {
        detailPane.appendChild(this.buildAttachmentsSection(email.attachments));
    }

    detailPane.appendChild(this.buildBodyContainer(email.id));

    // Move keyboard focus to the subject heading so screen-reader and
    // keyboard users land on the new email instead of staying on the
    // previous list-item context. tabindex=-1 makes the heading
    // programmatically focusable without entering the tab order.
    const subjectEl = detailPane.querySelector('.email-detail-subject');
    if (subjectEl) {
        subjectEl.setAttribute('tabindex', '-1');
        try { subjectEl.focus({ preventScroll: true }); } catch (_) { /* older browsers */ }
    }
},

// buildDetailHeader renders the subject + sender + recipient + timestamp
// block. All text uses textContent so subject/from/to interpolation is
// safe by construction.
buildDetailHeader(email, sender, toList, time) {
    const header = document.createElement('div');
    header.className = 'email-detail-header';

    const subjectEl = document.createElement('div');
    subjectEl.className = 'email-detail-subject';
    subjectEl.textContent = email.subject || '(No Subject)';
    header.appendChild(subjectEl);

    const meta = document.createElement('div');
    meta.className = 'email-detail-meta';

    const avatar = document.createElement('div');
    avatar.className = 'email-detail-avatar';
    avatar.style.background = 'var(--gradient-1)';
    avatar.textContent = sender.initials || '';
    meta.appendChild(avatar);

    const info = document.createElement('div');
    info.className = 'email-detail-info';

    const fromEl = document.createElement('div');
    fromEl.className = 'email-detail-from';
    fromEl.appendChild(document.createTextNode((sender.name || '') + ' '));
    const fromAddr = document.createElement('span');
    fromAddr.className = 'email-detail-email';
    fromAddr.textContent = '<' + (sender.email || '') + '>';
    fromEl.appendChild(fromAddr);
    info.appendChild(fromEl);

    const toEl = document.createElement('div');
    toEl.className = 'email-detail-to';
    toEl.textContent = 'To: ' + (toList || 'me');
    info.appendChild(toEl);

    meta.appendChild(info);

    const timeEl = document.createElement('div');
    timeEl.className = 'email-detail-time';
    timeEl.textContent = time;
    meta.appendChild(timeEl);

    header.appendChild(meta);
    return header;
},

// buildDetailActions returns the button row (reply/star/archive/delete +
// summarize). The SVG icon strings are static literals from this file,
// so insertAdjacentHTML is safe; everything else uses setAttribute /
// textContent so email.id round-trips without HTML encoding tricks.
buildDetailActions(email) {
    const actions = document.createElement('div');
    actions.className = 'email-detail-actions';

    // Decorative SVG icons — labelled by the trailing text node, so
    // aria-hidden keeps screen readers from announcing the path data
    // alongside the button text. STAR_FILL is interpolated; the rest
    // of the SVG bodies are static literals owned by this file.
    const REPLY_SVG = '<svg aria-hidden="true" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><path d="M9 17H4a2 2 0 01-2-2V5a2 2 0 012-2h16a2 2 0 012 2v10a2 2 0 01-2 2h-5l-5 5v-5z"/></svg>';
    const ARCHIVE_SVG = '<svg aria-hidden="true" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><rect x="2" y="4" width="20" height="5" rx="1"/><path d="M4 9v9a2 2 0 002 2h12a2 2 0 002-2V9M10 13h4"/></svg>';
    const DELETE_SVG = '<svg aria-hidden="true" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><path d="M3 6h18M19 6v14a2 2 0 01-2 2H7a2 2 0 01-2-2V6m3 0V4a2 2 0 012-2h4a2 2 0 012 2v2"/></svg>';
    const STAR_FILL = email.starred ? 'currentColor' : 'none';
    const STAR_SVG = `<svg aria-hidden="true" width="16" height="16" fill="${STAR_FILL}" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2"/></svg>`;

    actions.appendChild(this.buildActionButton(email.id, 'reply-email', 'Reply', REPLY_SVG));
    actions.appendChild(this.buildActionButton(email.id, 'toggle-star', email.starred ? 'Starred' : 'Star', STAR_SVG));
    actions.appendChild(this.buildActionButton(email.id, 'archive-email', 'Archive', ARCHIVE_SVG));
    actions.appendChild(this.buildActionButton(email.id, 'delete-email', 'Delete', DELETE_SVG));
    actions.appendChild(this.buildSummarizeButton(email.id));

    return actions;
},

buildActionButton(emailId, action, label, iconHTML) {
    const btn = document.createElement('button');
    btn.className = 'action-btn';
    btn.setAttribute('data-action', action);
    btn.setAttribute('data-email-id', emailId);
    btn.title = label;
    // iconHTML is a static SVG literal owned by this file — never
    // touched by user input. Trailing label uses textContent.
    btn.insertAdjacentHTML('beforeend', iconHTML);
    btn.appendChild(document.createTextNode(' ' + label));
    return btn;
},

buildSummarizeButton(emailId) {
    const SUMMARIZE_ICON = '<svg aria-hidden="true" class="ai-icon" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><path d="M12 2a10 10 0 100 20 10 10 0 000-20z"/><path d="M12 6v6l4 2"/></svg>';
    const SPINNER_ICON = '<svg aria-hidden="true" class="ai-spinner" width="16" height="16" viewBox="0 0 24 24" style="display:none"><circle cx="12" cy="12" r="10" stroke="currentColor" stroke-width="2" fill="none" stroke-dasharray="31.4" stroke-dashoffset="10"><animateTransform attributeName="transform" type="rotate" from="0 12 12" to="360 12 12" dur="1s" repeatCount="indefinite"/></circle></svg>';

    const btn = document.createElement('button');
    btn.className = 'action-btn ai-btn';
    btn.setAttribute('id', 'summarizeBtn-' + emailId);
    btn.setAttribute('data-action', 'summarize-email');
    btn.setAttribute('data-email-id', emailId);
    btn.title = 'Summarize with AI';
    btn.insertAdjacentHTML('beforeend', SUMMARIZE_ICON + SPINNER_ICON);
    const text = document.createElement('span');
    text.className = 'ai-btn-text';
    text.textContent = '✨ Summarize';
    btn.appendChild(text);
    return btn;
},

buildSmartRepliesSlot(emailId) {
    const wrap = document.createElement('div');
    wrap.className = 'smart-replies-container';
    wrap.setAttribute('id', 'smartReplies-' + emailId);

    const trigger = document.createElement('button');
    trigger.className = 'smart-replies-trigger';
    trigger.setAttribute('data-action', 'load-smart-replies');
    trigger.setAttribute('data-email-id', emailId);
    const icon = document.createElement('span');
    icon.className = 'smart-replies-icon';
    icon.textContent = '💬'; // 💬
    trigger.appendChild(icon);
    const label = document.createElement('span');
    label.textContent = 'Get smart reply suggestions';
    trigger.appendChild(label);
    wrap.appendChild(trigger);
    return wrap;
},

buildInviteSlot(emailId) {
    const slot = document.createElement('div');
    slot.className = 'calendar-invite-card-slot';
    slot.setAttribute('id', 'inviteSlot-' + emailId);
    slot.hidden = true;
    return slot;
},

buildAttachmentsSection(attachments) {
    const wrap = document.createElement('div');
    wrap.className = 'email-detail-attachments';

    const header = document.createElement('div');
    header.className = 'attachments-header';
    header.textContent = 'Attachments (' + attachments.length + ')';
    wrap.appendChild(header);

    const list = document.createElement('div');
    list.className = 'attachments-list';
    attachments.forEach((a) => {
        const item = document.createElement('div');
        item.className = 'attachment-item';

        const icon = document.createElement('span');
        icon.className = 'attachment-icon';
        icon.textContent = '📎'; // 📎
        item.appendChild(icon);

        const name = document.createElement('span');
        name.className = 'attachment-name';
        name.textContent = a.filename || '';
        item.appendChild(name);

        const size = document.createElement('span');
        size.className = 'attachment-size';
        size.textContent = this.formatSize(a.size);
        item.appendChild(size);

        list.appendChild(item);
    });
    wrap.appendChild(list);
    return wrap;
},

buildBodyContainer(emailId) {
    const wrap = document.createElement('div');
    wrap.className = 'email-detail-body';

    const container = document.createElement('div');
    container.className = 'email-iframe-container';
    container.setAttribute('id', 'emailBodyContainer-' + emailId);

    const loading = document.createElement('div');
    loading.className = 'email-loading-state';
    const spinner = document.createElement('div');
    spinner.className = 'email-loading-spinner';
    loading.appendChild(spinner);
    const loadingText = document.createElement('span');
    loadingText.textContent = 'Loading email...';
    loading.appendChild(loadingText);
    container.appendChild(loading);

    wrap.appendChild(container);
    return wrap;
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
        // DOM-construct the error state for consistency with the rest
        // of the file. Strings are static literals, but using
        // replaceChildren keeps every container mutation in this file
        // free of innerHTML so no future contributor lands user data
        // through a "just like this one" copy.
        const errState = document.createElement('div');
        errState.className = 'email-error-state';
        const icon = document.createElement('span');
        icon.className = 'error-icon';
        icon.textContent = '⚠️';
        const label = document.createElement('span');
        label.textContent = 'Failed to load email content';
        errState.appendChild(icon);
        errState.appendChild(label);
        container.replaceChildren(errState);
    };

    // Clear loading state and add iframe
    container.replaceChildren(iframe);
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

    // Construct the card via DOM nodes (createElement + textContent +
    // setAttribute) so every interpolation of upstream-provided strings
    // (title, organizer name/email, attendees, location) is safe by
    // construction. Mirrors the renderEmailDetail rewrite — keeping
    // both blocks consistent prevents an "escapeHtml drift" where a
    // future template-literal contributor forgets to escape one field.
    slot.replaceChildren(this.buildCalendarInviteCard(invite, emailId));
    slot.removeAttribute('hidden');

    // Mirror Gmail's behaviour by surfacing the ICS as a regular
    // attachment row when the invite came from an inline calendar part.
    // For real attachments the email-detail render already shows them,
    // so we only need to inject when the row isn't there yet.
    this.ensureInviteAttachmentRow(emailId, invite);
},

// ensureInviteAttachmentRow appends a calendar-attachment row to the
// email detail view when one isn't already rendered. Used for inline
// calendar parts that Nylas does not surface in attachments[]. All
// nodes are constructed via createElement+textContent so an attacker-
// controlled invite.filename can never inject markup.
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

    const list = detail.querySelector('.email-detail-attachments .attachments-list');
    if (list) {
        list.appendChild(this.buildInviteAttachmentRow(invite));
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
    slot.after(this.buildInviteAttachmentSection(invite));
},

// buildInviteAttachmentRow constructs a single .attachment-item row for
// an inline calendar part. The filename is set via textContent, so any
// shape of invite.filename is safe.
buildInviteAttachmentRow(invite) {
    const item = document.createElement('div');
    item.className = 'attachment-item';
    item.setAttribute('data-inline-calendar', 'true');

    const icon = document.createElement('span');
    icon.className = 'attachment-icon';
    icon.textContent = '📎';
    item.appendChild(icon);

    const name = document.createElement('span');
    name.className = 'attachment-name';
    name.textContent = invite.filename || '';
    item.appendChild(name);

    const size = document.createElement('span');
    size.className = 'attachment-size';
    size.textContent = 'Calendar invitation';
    item.appendChild(size);

    return item;
},

// buildInviteAttachmentSection builds a fresh
// .email-detail-attachments section containing exactly the inline-
// calendar row. Used when no attachments section exists yet.
buildInviteAttachmentSection(invite) {
    const wrap = document.createElement('div');
    wrap.className = 'email-detail-attachments';

    const header = document.createElement('div');
    header.className = 'attachments-header';
    header.textContent = 'Attachments (1)';
    wrap.appendChild(header);

    const list = document.createElement('div');
    list.className = 'attachments-list';
    list.appendChild(this.buildInviteAttachmentRow(invite));
    wrap.appendChild(list);

    return wrap;
},

// buildCalendarInviteCard returns the DOM node for a Gmail-style invite
// card. Builds the entire tree with createElement + textContent +
// setAttribute so untrusted invite fields (title, location, organizer,
// attendees) can never inject markup — there is no template-literal
// interpolation to forget to escape. Conferencing URL is screened via
// isSafeUrl before being assigned to <a>.href.
buildCalendarInviteCard(invite, emailId) {
    const fmtRange = this.formatInviteRange(invite);
    const safeURL = (typeof isSafeUrl === 'function' && invite.conferencing_url &&
        isSafeUrl(invite.conferencing_url)) ? invite.conferencing_url : '';
    const isCancelled = String(invite.method || '').toUpperCase() === 'CANCEL' ||
        String(invite.status || '').toUpperCase() === 'CANCELLED';

    const card = document.createElement('section');
    card.className = 'calendar-invite-card' + (isCancelled ? ' is-cancelled' : '');
    card.setAttribute('role', 'region');
    card.setAttribute('aria-label', 'Calendar invitation');

    if (isCancelled) {
        const banner = document.createElement('div');
        banner.className = 'calendar-invite-banner calendar-invite-banner-cancel';
        banner.setAttribute('role', 'alert');
        banner.textContent = 'This event was cancelled';
        card.appendChild(banner);
    }

    const header = document.createElement('header');
    header.className = 'calendar-invite-header';
    const iconEl = document.createElement('div');
    iconEl.className = 'calendar-invite-icon';
    iconEl.setAttribute('aria-hidden', 'true');
    iconEl.textContent = '📅';
    header.appendChild(iconEl);

    const when = document.createElement('div');
    when.className = 'calendar-invite-when';
    const timeEl = document.createElement('div');
    timeEl.className = 'calendar-invite-time';
    timeEl.textContent = fmtRange;
    when.appendChild(timeEl);
    const titleEl = document.createElement('div');
    titleEl.className = 'calendar-invite-title';
    titleEl.textContent = invite.title || 'Untitled event';
    when.appendChild(titleEl);
    header.appendChild(when);
    card.appendChild(header);

    if (invite.location) {
        const loc = document.createElement('div');
        loc.className = 'calendar-invite-location';
        // 📍 is a static literal owned by this file — keep as a separate
        // text node so the user-supplied location text uses textContent.
        loc.appendChild(document.createTextNode('📍 '));
        loc.appendChild(document.createTextNode(invite.location));
        card.appendChild(loc);
    }

    if (invite.organizer_email) {
        const org = document.createElement('div');
        org.className = 'calendar-invite-org';
        org.textContent = (invite.organizer_name || invite.organizer_email) + ' · Organizer';
        card.appendChild(org);
    }

    if (safeURL) {
        const link = document.createElement('a');
        link.className = 'calendar-invite-link';
        // Assigning to .href via property does NOT HTML-escape the URL,
        // but isSafeUrl already screened the scheme/host above.
        link.href = safeURL;
        link.target = '_blank';
        link.rel = 'noopener noreferrer';
        link.textContent = 'Join with conferencing';
        card.appendChild(link);
    }

    const attendees = this.buildInviteAttendees(invite.attendees);
    if (attendees) card.appendChild(attendees);

    if (!isCancelled) {
        card.appendChild(this.buildInviteActions(emailId));
    }
    return card;
},

// buildInviteActions constructs the Yes/Maybe/No RSVP button row.
buildInviteActions(emailId) {
    const wrap = document.createElement('div');
    wrap.className = 'calendar-invite-actions';
    [
        { rsvp: 'yes',   label: 'Yes',   primary: true },
        { rsvp: 'maybe', label: 'Maybe', primary: false },
        { rsvp: 'no',    label: 'No',    primary: false },
    ].forEach(({ rsvp, label, primary }) => {
        const btn = document.createElement('button');
        btn.type = 'button';
        btn.className = primary ? 'calendar-invite-btn primary' : 'calendar-invite-btn';
        btn.setAttribute('data-action', 'invite-rsvp');
        btn.setAttribute('data-email-id', emailId);
        btn.setAttribute('data-rsvp', rsvp);
        btn.textContent = label;
        wrap.appendChild(btn);
    });
    return wrap;
},

// buildInviteAttendees produces the attendee summary that mirrors
// Gmail's "3 going, 1 declined" line and the per-attendee chip list.
// Returns null when no attendees are present so the card stays compact
// for invitations without an explicit attendee list (Outlook sometimes
// omits ATTENDEE on REQUEST). Every interpolation of attendee data
// (name, email, role) goes through textContent / setAttribute, never
// HTML-string concatenation.
buildInviteAttendees(attendees) {
    if (!Array.isArray(attendees) || attendees.length === 0) return null;

    const counts = { ACCEPTED: 0, DECLINED: 0, TENTATIVE: 0, OTHER: 0 };
    attendees.forEach((a) => {
        const status = String(a.status || '').toUpperCase();
        if (status in counts) counts[status]++;
        else counts.OTHER++;
    });

    const wrap = document.createElement('div');
    wrap.className = 'calendar-invite-attendees';

    const parts = [];
    if (counts.ACCEPTED > 0) parts.push(`${counts.ACCEPTED} going`);
    if (counts.DECLINED > 0) parts.push(`${counts.DECLINED} declined`);
    if (counts.TENTATIVE > 0) parts.push(`${counts.TENTATIVE} maybe`);
    if (counts.OTHER > 0) parts.push(`${counts.OTHER} no response`);
    if (parts.length > 0) {
        const summary = document.createElement('div');
        summary.className = 'calendar-invite-summary';
        summary.textContent = parts.join(' · ');
        wrap.appendChild(summary);
    }

    const list = document.createElement('div');
    list.className = 'calendar-invite-attendee-list';
    attendees.slice(0, 8).forEach((a) => {
        const label = a.name || a.email || '';
        if (!label) return;
        const status = String(a.status || '').toUpperCase();
        const cls = status === 'ACCEPTED' ? 'is-accepted'
            : status === 'DECLINED' ? 'is-declined'
            : status === 'TENTATIVE' ? 'is-tentative'
            : 'is-pending';
        const chip = document.createElement('span');
        chip.className = 'calendar-invite-attendee ' + cls;
        // setAttribute does not HTML-encode; the title is read by
        // browsers as a plain string. Concatenation of the email and
        // role is safe because both are textual values, never HTML.
        const role = a.is_organizer ? ' · Organizer' : '';
        chip.setAttribute('title', (a.email || '') + role);
        chip.textContent = label;
        list.appendChild(chip);
    });
    if (attendees.length > 8) {
        const overflow = document.createElement('span');
        overflow.className = 'calendar-invite-attendee is-overflow';
        overflow.textContent = `+${attendees.length - 8} more`;
        list.appendChild(overflow);
    }
    wrap.appendChild(list);
    return wrap;
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

// rsvpToInvite forwards the user's choice to POST /api/emails/{id}/rsvp
// which resolves the invite to a Nylas event and calls send-rsvp.
//
// Buttons are disabled for the duration of the request so a frustrated
// user doesn't double-fire (each call sends a real email to the
// organiser). The active state is only applied after the server confirms
// — a click that fails leaves the previous selection alone.
async rsvpToInvite(emailId, response) {
    const valid = new Set(['yes', 'no', 'maybe']);
    const choice = String(response || '').toLowerCase();
    if (!valid.has(choice)) return;

    const slot = document.getElementById(`inviteSlot-${emailId}`);
    const buttons = slot ? Array.from(slot.querySelectorAll('.calendar-invite-btn')) : [];
    const previouslyActive = buttons.find((btn) => btn.classList.contains('active')) || null;

    // Disable all RSVP buttons and mark the clicked one as in-flight so
    // the user gets immediate visual feedback while we wait on Nylas.
    buttons.forEach((btn) => {
        btn.disabled = true;
        btn.classList.toggle('is-loading', btn.dataset.rsvp === choice);
    });

    try {
        // No CSRF token: Air binds the listener to localhost via the
        // shared internal/webguard package, and cross-origin requests
        // to localhost are blocked by the browser's PNA / origin
        // checks. If Air is ever ported to a hosted URL, this fetch
        // (and the matching /rsvp handler) MUST gain a CSRF token.
        const resp = await fetch(`/api/emails/${encodeURIComponent(emailId)}/rsvp`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ status: choice }),
        });

        if (!resp.ok) {
            let message = 'Failed to send RSVP';
            try {
                const errBody = await resp.json();
                if (errBody && typeof errBody.error === 'string' && errBody.error) {
                    // Cap the upstream message so a 200 KB HTML body
                    // parsed as JSON, or a maliciously long Nylas error,
                    // can't blow up the toast layout. showToast also
                    // truncates defensively, but we narrow at the source.
                    message = errBody.error.slice(0, 200);
                }
            } catch (e) {
                // Body wasn't JSON — fall back to the generic message.
                // Log the parse failure so degraded errors (server
                // returned HTML, body too large, etc.) are debuggable.
                console.warn('[invite] RSVP error body parse failed:', e);
            }
            if (typeof showToast === 'function') {
                showToast('error', 'RSVP failed', message);
            }
            return;
        }

        if (typeof showToast === 'function') {
            const labels = { yes: 'Accepted', no: 'Declined', maybe: 'Tentative' };
            showToast('success', labels[choice], `Invite ${labels[choice].toLowerCase()}`);
        }

        // Apply the new active state only if the user is STILL on this
        // email AND the slot still exists. Without these guards, a user
        // who clicks RSVP, navigates away mid-flight, then returns,
        // would see success applied to a card that's now showing a
        // different invite — a confusing reality drift.
        if (this.selectedEmailId === emailId) {
            const liveSlot = document.getElementById(`inviteSlot-${emailId}`);
            if (liveSlot) {
                liveSlot.querySelectorAll('.calendar-invite-btn').forEach((btn) => {
                    btn.classList.toggle('active', btn.dataset.rsvp === choice);
                });
            }
        }
    } catch (err) {
        console.warn('[invite] RSVP failed:', err);
        if (typeof showToast === 'function') {
            showToast('error', 'RSVP failed', 'Could not reach the server. Check your connection.');
        }
        // Restore prior selection so the UI doesn't lie about state.
        // Skip the restore if the user has already navigated away (the
        // node is no longer connected to the document) — touching a
        // detached element is harmless but pollutes the heap.
        if (previouslyActive && previouslyActive.isConnected) {
            previouslyActive.classList.add('active');
        }
    } finally {
        // Only re-enable the buttons that are still live — captured
        // references can be detached if the user navigated away, in
        // which case the toggles are no-ops on a stale reference.
        buttons.forEach((btn) => {
            if (!btn.isConnected) return;
            btn.disabled = false;
            btn.classList.remove('is-loading');
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
