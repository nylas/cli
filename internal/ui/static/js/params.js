// =============================================================================
// Parameter Input Handling
// =============================================================================

function getParamValue(section) {
    const input = document.getElementById(`${section}-param-input`);
    return input ? input.value.trim() : '';
}

/**
 * Get suggestions for a parameter based on cached data.
 * Returns array of {id, label} objects.
 */
function getParamSuggestions(section, paramName) {
    // Map param names to their cache getter functions by section
    const emailParamMap = {
        'message-id': 'getCachedMessageIds',   // email read, delete, mark, metadata, attachments-list
        'folder-id': 'getCachedFolderIds',     // folders-show, folders-rename, folders-delete
        'schedule-id': 'getCachedScheduleIds', // scheduled-show, scheduled-cancel
        'thread-id': 'getCachedThreadIds'      // threads-show, threads-delete, threads-mark
    };

    const calendarParamMap = {
        'calendar-id': 'getCachedCalendarIds', // calendar show, update, delete
        'event-id': 'getCachedEventIds'        // events-show, events-update, events-delete, events-rsvp
    };

    const authParamMap = {
        'grant-id': 'getCachedGrantIds'        // auth show, switch, remove, revoke, scopes
    };

    const contactsParamMap = {
        'contact-id': 'getCachedContactIds'    // contacts show, update, delete
    };

    const webhookParamMap = {
        'webhook-id': 'getCachedWebhookIds'    // webhook show, update, delete
    };

    const notetakerParamMap = {
        'notetaker-id': 'getCachedNotetakerIds' // notetaker show, delete, media
    };

    let getterName = null;

    if (section === 'email') {
        getterName = emailParamMap[paramName];
    } else if (section === 'calendar') {
        getterName = calendarParamMap[paramName];
    } else if (section === 'auth') {
        getterName = authParamMap[paramName];
    } else if (section === 'contacts') {
        getterName = contactsParamMap[paramName];
    } else if (section === 'webhook') {
        getterName = webhookParamMap[paramName];
    } else if (section === 'notetaker') {
        getterName = notetakerParamMap[paramName];
    }

    if (getterName && typeof window[getterName] === 'function') {
        return window[getterName]();
    }

    return [];
}

/**
 * Render a datalist element with suggestions.
 */
function renderDatalist(id, suggestions) {
    if (!suggestions || suggestions.length === 0) return '';

    let options = suggestions.map(s => {
        // Escape HTML in label
        const safeLabel = (s.label || s.id).replace(/</g, '&lt;').replace(/>/g, '&gt;');
        return `<option value="${s.id}">${safeLabel}</option>`;
    }).join('');

    return `<datalist id="${id}">${options}</datalist>`;
}

function showParamInput(section, param, flags) {
    const container = document.getElementById(`${section}-param-container`);
    if (!container) return;

    let html = '';

    // Render flags panel if flags exist
    if (flags && flags.length > 0) {
        html += `<div class="flags-panel">
            <div class="flags-header" onclick="toggleFlagsPanel('${section}')">
                <span>Options</span>
                <svg class="flags-chevron" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <polyline points="6 9 12 15 18 9"/>
                </svg>
            </div>
            <div class="flags-content" id="${section}-flags-content">
                <div class="flags-grid">`;

        flags.forEach(flag => {
            const flagId = `${section}-flag-${flag.name}`;
            if (flag.type === 'checkbox') {
                const checked = flag.default ? 'checked' : '';
                html += `
                    <label class="flag-item flag-checkbox">
                        <input type="checkbox" id="${flagId}" data-flag="${flag.name}" ${checked}>
                        <span class="flag-checkmark"></span>
                        <span class="flag-label">${flag.label}</span>
                    </label>`;
            } else {
                html += `
                    <div class="flag-item flag-input">
                        <label for="${flagId}">${flag.label}</label>
                        <input type="${flag.type}" id="${flagId}" data-flag="${flag.name}"
                               placeholder="${flag.placeholder || ''}" />
                    </div>`;
            }
        });

        html += `</div></div></div>`;
    }

    // Render simple param input if param exists
    if (param) {
        // Check if we should show a datalist with suggestions
        const suggestions = getParamSuggestions(section, param.name);
        const datalistId = `${section}-param-datalist`;
        const listAttr = suggestions.length > 0 ? `list="${datalistId}"` : '';

        html += `
            <div class="param-field ${suggestions.length > 0 ? 'has-suggestions' : ''}">
                <label for="${section}-param-input">
                    ${param.name}
                    ${suggestions.length > 0 ? `<span class="suggestions-badge">${suggestions.length} suggestions</span>` : ''}
                </label>
                <input type="text" id="${section}-param-input" placeholder="${param.placeholder}" ${listAttr} autocomplete="off" />
                ${suggestions.length > 0 ? renderDatalist(datalistId, suggestions) : ''}
            </div>`;
    }

    if (html) {
        container.innerHTML = html;
        container.style.display = 'block';
    } else {
        container.style.display = 'none';
        container.innerHTML = '';
    }
}

function toggleFlagsPanel(section) {
    const content = document.getElementById(`${section}-flags-content`);
    const panel = content.closest('.flags-panel');
    if (content && panel) {
        panel.classList.toggle('collapsed');
    }
}

function buildCommand(baseCmd, section, flags) {
    let cmd = baseCmd;

    // Add flags if they exist
    if (flags && flags.length > 0) {
        flags.forEach(flag => {
            const el = document.getElementById(`${section}-flag-${flag.name}`);
            if (!el) return;

            if (flag.type === 'checkbox') {
                if (el.checked) {
                    cmd += ` --${flag.name}`;
                }
            } else {
                const val = el.value.trim();
                if (val) {
                    cmd += ` --${flag.name} ${val}`;
                }
            }
        });
    }

    // Add simple param if exists
    const param = getParamValue(section);
    if (param) {
        cmd += ' ' + param;
    }

    return cmd;
}
