// =============================================================================
// Command System Core - ID Caching and Parsing
// =============================================================================

// Cached IDs from list commands (for suggestions in show/read/delete commands)
let cachedMessageIds = [];   // [{id: "abc123", label: "sender - subject"}, ...]
let cachedFolderIds = [];    // [{id: "folder-id", label: "INBOX (inbox)"}, ...]
let cachedScheduleIds = [];  // [{id: "schedule-id", label: "⏳ Jan 2, 2025 3:04 PM"}, ...]
let cachedThreadIds = [];    // [{id: "thread-id", label: "participants - subject"}, ...]
let cachedCalendarIds = [];  // [{id: "calendar-id", label: "Calendar Name"}, ...]
let cachedEventIds = [];     // [{id: "event-id", label: "Event Title"}, ...]
let cachedGrantIds = [];     // [{id: "grant-id", label: "email@example.com (Provider)"}, ...]
let cachedContactIds = [];   // [{id: "contact-id", label: "John Doe (john@example.com)"}, ...]
let cachedWebhookIds = [];   // [{id: "webhook-id", label: "https://example.com/webhook"}, ...]
let cachedNotetakerIds = []; // [{id: "notetaker-id", label: "Team Standup"}, ...]

// =============================================================================
// ID Parsing Functions
// =============================================================================

/**
 * Parse email list output to extract message IDs.
 * Format when --id flag is used:
 *   ● ★ sender@email.com    Subject line here...           2 hours ago
 *         ID: abc123def456...
 */
function parseMessageIdsFromOutput(output) {
    const ids = [];
    const lines = output.split('\n');
    let lastEmailInfo = null;

    for (const line of lines) {
        const idMatch = line.match(/^\s+ID:\s*(\S+)/);
        if (idMatch) {
            const id = idMatch[1];
            ids.push({
                id: id,
                label: lastEmailInfo || id.substring(0, 20) + '...'
            });
            lastEmailInfo = null;
            continue;
        }

        const trimmed = line.trim();
        if (trimmed && !trimmed.startsWith('Found') && !trimmed.startsWith('ID:')) {
            const cleaned = trimmed.replace(/^[●★\s]+/, '').trim();
            if (cleaned.length > 5) {
                lastEmailInfo = cleaned.length > 60 ? cleaned.substring(0, 57) + '...' : cleaned;
            }
        }
    }

    return ids;
}

/**
 * Parse folders list output to extract folder IDs.
 */
function parseFolderIdsFromOutput(output) {
    const ids = [];
    const lines = output.split('\n');
    let dataStarted = false;

    for (const line of lines) {
        if (line.includes('------')) {
            dataStarted = true;
            continue;
        }

        if (!dataStarted) continue;

        const trimmed = line.trim();
        if (!trimmed) continue;

        const parts = trimmed.split(/\s{2,}/);
        if (parts.length >= 2) {
            const id = parts[0].trim();
            const name = parts[1].trim();
            const type = parts.length > 2 ? parts[2].trim() : '';

            if (id && id.length > 10 && !id.includes('ID') && !id.includes('NAME')) {
                ids.push({
                    id: id,
                    label: `${name}${type ? ' (' + type + ')' : ''}`
                });
            }
        }
    }

    return ids;
}

/**
 * Parse scheduled list output to extract schedule IDs.
 */
function parseScheduleIdsFromOutput(output) {
    const ids = [];
    const lines = output.split('\n');
    let currentId = null;
    let currentStatus = '';
    let currentSendAt = '';

    for (const line of lines) {
        const idMatch = line.match(/Schedule ID:\s*(\S+)/);
        if (idMatch) {
            if (currentId) {
                ids.push({
                    id: currentId,
                    label: `${currentStatus} - ${currentSendAt}`.trim()
                });
            }
            currentId = idMatch[1];
            currentStatus = '';
            currentSendAt = '';
            continue;
        }

        const statusMatch = line.match(/Status:\s*(\S+)/);
        if (statusMatch && currentId) {
            const status = statusMatch[1];
            currentStatus = status === 'pending' ? '⏳' : status === 'sent' ? '✅' : '❌';
            continue;
        }

        const sendAtMatch = line.match(/Send at:\s*(.+)$/);
        if (sendAtMatch && currentId) {
            currentSendAt = sendAtMatch[1].trim();
            continue;
        }
    }

    if (currentId) {
        ids.push({
            id: currentId,
            label: `${currentStatus} - ${currentSendAt}`.trim()
        });
    }

    return ids;
}

/**
 * Parse threads list output to extract thread IDs.
 */
function parseThreadIdsFromOutput(output) {
    const ids = [];
    const lines = output.split('\n');
    let lastThreadInfo = null;

    for (const line of lines) {
        const idMatch = line.match(/^\s+ID:\s*(\S+)/);
        if (idMatch) {
            const id = idMatch[1];
            ids.push({
                id: id,
                label: lastThreadInfo || id.substring(0, 20) + '...'
            });
            lastThreadInfo = null;
            continue;
        }

        const trimmed = line.trim();
        if (trimmed && !trimmed.startsWith('Found') && !trimmed.startsWith('ID:')) {
            const cleaned = trimmed.replace(/^[●★📎\s]+/, '').trim();
            if (cleaned.length > 5) {
                lastThreadInfo = cleaned.length > 60 ? cleaned.substring(0, 57) + '...' : cleaned;
            }
        }
    }

    return ids;
}

/**
 * Parse calendar list output to extract calendar IDs.
 */
function parseCalendarIdsFromOutput(output) {
    const ids = [];
    const lines = output.split('\n');
    let dataStarted = false;

    for (const line of lines) {
        if (line.includes('───') || line.includes('---')) {
            dataStarted = true;
            continue;
        }

        if (!dataStarted) continue;

        const trimmed = line.trim();
        if (!trimmed) continue;

        const parts = trimmed.split(/\s{2,}/);
        if (parts.length >= 2) {
            const name = parts[0].trim();
            const id = parts[1].trim();

            if (id && (id.includes('@') || id.length > 10) && !id.includes('ID')) {
                const isPrimary = parts.length > 2 && parts[2].trim() === 'Yes';
                ids.push({
                    id: id,
                    label: `${name}${isPrimary ? ' (Primary)' : ''}`
                });
            }
        }
    }

    return ids;
}

/**
 * Parse events list output to extract event IDs.
 */
function parseEventIdsFromOutput(output) {
    const ids = [];
    const lines = output.split('\n');
    let currentTitle = null;

    for (const line of lines) {
        const idMatch = line.match(/^\s+ID:\s*(\S+)/);
        if (idMatch) {
            const id = idMatch[1];
            ids.push({
                id: id,
                label: currentTitle || id.substring(0, 20) + '...'
            });
            currentTitle = null;
            continue;
        }

        const trimmed = line.trim();
        if (trimmed && !line.startsWith(' ') && !trimmed.startsWith('Found') && !trimmed.startsWith('When:') && !trimmed.startsWith('Status:')) {
            currentTitle = trimmed.length > 50 ? trimmed.substring(0, 47) + '...' : trimmed;
        }
    }

    return ids;
}

/**
 * Parse auth list output to extract grant IDs.
 */
function parseGrantIdsFromOutput(output) {
    const ids = [];
    const lines = output.split('\n');

    for (const line of lines) {
        const trimmed = line.trim();
        if (!trimmed) continue;
        if (trimmed.startsWith('GRANT ID')) continue;

        const parts = trimmed.split(/\s{2,}/);
        if (parts.length >= 3) {
            const id = parts[0].trim();
            const email = parts[1].trim();
            const provider = parts[2].trim();

            if (id && id.length > 20 && email.includes('@')) {
                ids.push({
                    id: id,
                    label: `${email} (${provider})`
                });
            }
        }
    }

    return ids;
}

/**
 * Parse contacts list output to extract contact IDs.
 */
function parseContactIdsFromOutput(output) {
    const ids = [];
    const lines = output.split('\n');
    let dataStarted = false;

    for (const line of lines) {
        const trimmed = line.trim();
        if (!trimmed) continue;

        if (trimmed.startsWith('ID') && trimmed.includes('NAME')) {
            dataStarted = true;
            continue;
        }

        if (!dataStarted) continue;
        if (trimmed.includes('───') || trimmed.includes('---')) continue;

        const parts = trimmed.split(/\s{2,}/);
        if (parts.length >= 2) {
            const id = parts[0].trim();
            const name = parts[1].trim();
            const email = parts.length > 2 ? parts[2].trim() : '';

            if (id && id.length >= 10 && !id.includes('ID') && /^[a-zA-Z0-9_-]+$/.test(id)) {
                let label = name;
                if (email && email.includes('@')) {
                    label = `${name} (${email})`;
                }
                ids.push({
                    id: id,
                    label: label.length > 50 ? label.substring(0, 47) + '...' : label
                });
            }
        }
    }

    return ids;
}

/**
 * Parse webhook list output to extract webhook IDs.
 */
function parseWebhookIdsFromOutput(output) {
    const ids = [];
    const lines = output.split('\n');
    let dataStarted = false;

    for (const line of lines) {
        const trimmed = line.trim();
        if (!trimmed) continue;

        if (trimmed.startsWith('ID') && (trimmed.includes('CALLBACK') || trimmed.includes('URL'))) {
            dataStarted = true;
            continue;
        }

        if (!dataStarted) continue;
        if (trimmed.includes('───') || trimmed.includes('---') || trimmed.includes('webhooks')) continue;

        const parts = trimmed.split(/\s{2,}/);
        if (parts.length >= 2) {
            const id = parts[0].trim();
            const url = parts[1].trim();

            if (id && id.length >= 3 && !id.includes('ID')) {
                const label = url.length > 40 ? url.substring(0, 37) + '...' : url;
                ids.push({
                    id: id,
                    label: label
                });
            }
        }
    }

    return ids;
}

/**
 * Parse notetaker list output to extract notetaker IDs.
 */
function parseNotetakerIdsFromOutput(output) {
    const ids = [];
    const lines = output.split('\n');
    let dataStarted = false;

    for (const line of lines) {
        const trimmed = line.trim();
        if (!trimmed) continue;

        if (trimmed.startsWith('ID') && trimmed.includes('MEETING')) {
            dataStarted = true;
            continue;
        }

        if (!dataStarted) continue;
        if (trimmed.includes('───') || trimmed.includes('---') || trimmed.includes('notetakers')) continue;

        const parts = trimmed.split(/\s{2,}/);
        if (parts.length >= 2) {
            const id = parts[0].trim();
            const meeting = parts[1].trim();

            if (id && id.length >= 3 && !id.includes('ID')) {
                ids.push({
                    id: id,
                    label: meeting.length > 40 ? meeting.substring(0, 37) + '...' : meeting
                });
            }
        }
    }

    return ids;
}

// =============================================================================
// Cache Getters
// =============================================================================

function getCachedMessageIds() { return cachedMessageIds; }
function getCachedFolderIds() { return cachedFolderIds; }
function getCachedScheduleIds() { return cachedScheduleIds; }
function getCachedThreadIds() { return cachedThreadIds; }
function getCachedCalendarIds() { return cachedCalendarIds; }
function getCachedEventIds() { return cachedEventIds; }
function getCachedGrantIds() { return cachedGrantIds; }
function getCachedContactIds() { return cachedContactIds; }
function getCachedWebhookIds() { return cachedWebhookIds; }
function getCachedNotetakerIds() { return cachedNotetakerIds; }

// =============================================================================
// Cache Management
// =============================================================================

function clearAllCachedIds() {
    cachedMessageIds = [];
    cachedFolderIds = [];
    cachedScheduleIds = [];
    cachedThreadIds = [];
    cachedCalendarIds = [];
    cachedEventIds = [];
    cachedGrantIds = [];
    cachedContactIds = [];
    cachedWebhookIds = [];
    cachedNotetakerIds = [];
}

function getTotalCachedCount() {
    return cachedMessageIds.length + cachedFolderIds.length + cachedScheduleIds.length +
           cachedThreadIds.length + cachedCalendarIds.length + cachedEventIds.length +
           cachedGrantIds.length + cachedContactIds.length +
           cachedWebhookIds.length + cachedNotetakerIds.length;
}

function updateCacheCountBadge() {
    const count = getTotalCachedCount();
    const badgeIds = [
        'cache-count-badge',
        'calendar-cache-count-badge',
        'auth-cache-count-badge',
        'contacts-cache-count-badge',
        'webhook-cache-count-badge',
        'notetaker-cache-count-badge'
    ];

    badgeIds.forEach(id => {
        const badge = document.getElementById(id);
        if (badge) {
            if (count > 0) {
                badge.textContent = count;
                badge.style.display = 'inline-flex';
            } else {
                badge.style.display = 'none';
            }
        }
    });
}

function clearCacheAndNotify() {
    const count = getTotalCachedCount();
    if (count === 0) {
        showToast('No cached IDs to clear', 'info');
        return;
    }

    clearAllCachedIds();
    updateCacheCountBadge();
    showToast(`Cleared ${count} cached IDs`, 'success');

    // Refresh param inputs for active commands
    const commandRefreshers = [
        { current: 'currentEmailCmd', commands: 'emailCommands', prefix: 'email' },
        { current: 'currentCalendarCmd', commands: 'calendarCommands', prefix: 'calendar' },
        { current: 'currentAuthCmd', commands: 'authCommands', prefix: 'auth' },
        { current: 'currentContactsCmd', commands: 'contactsCommands', prefix: 'contacts' },
        { current: 'currentWebhookCmd', commands: 'webhookCommands', prefix: 'webhook' },
        { current: 'currentNotetakerCmd', commands: 'notetakerCommands', prefix: 'notetaker' }
    ];

    commandRefreshers.forEach(({ current, commands, prefix }) => {
        const currentCmd = window[current];
        const cmds = window[commands];
        if (currentCmd && cmds && cmds[currentCmd] && cmds[currentCmd].param) {
            showParamInput(prefix, cmds[currentCmd].param, cmds[currentCmd].flags);
        }
    });
}

// =============================================================================
// Shared Utilities
// =============================================================================

/**
 * Generic function to render command sections.
 * Note: Uses innerHTML with static command data defined in code (not user input).
 */
function renderCommandSections(containerId, sections, showFn) {
    const container = document.getElementById(containerId);
    if (!container) return;

    // Build HTML from static command definitions (safe - no user input)
    const fragment = document.createDocumentFragment();

    sections.forEach(section => {
        const titleDiv = document.createElement('div');
        titleDiv.className = 'cmd-section-title';
        titleDiv.textContent = section.title;
        fragment.appendChild(titleDiv);

        Object.entries(section.commands).forEach(([key, data]) => {
            const itemDiv = document.createElement('div');
            itemDiv.className = 'cmd-item';
            itemDiv.dataset.cmd = key;
            itemDiv.onclick = () => window[showFn](key);

            const nameSpan = document.createElement('span');
            nameSpan.className = 'cmd-name';
            nameSpan.textContent = data.title;
            itemDiv.appendChild(nameSpan);

            const copyBtn = document.createElement('button');
            copyBtn.className = 'cmd-copy';
            copyBtn.onclick = (e) => { e.stopPropagation(); copyText('nylas ' + data.cmd); };
            copyBtn.innerHTML = '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="9" y="9" width="13" height="13" rx="2"/><path d="M5 15H4a2 2 0 01-2-2V4a2 2 0 012-2h9a2 2 0 012 2v1"/></svg>';
            itemDiv.appendChild(copyBtn);

            fragment.appendChild(itemDiv);
        });
    });

    container.innerHTML = '';
    container.appendChild(fragment);
}
