// ====================================
// SETTINGS MODULE
// ====================================

// Settings state
const settingsState = {
    aiProvider: null,  // No default - user must select in Settings
    theme: 'dark',
    accentColor: 'purple',
    threading: true,
    avatars: true,
    previewPane: true,
    refreshInterval: 60, // Default 60 seconds
    // Notetaker sources - array of {from, subject, linkDomain}
    // No default - user must configure in Settings
    notetakerSources: []
};

// Refresh interval timer
let refreshTimer = null;
let lastRefreshTime = Date.now();

// Color values
const accentColors = {
    purple: '#8b5cf6',
    blue: '#3b82f6',
    green: '#22c55e',
    orange: '#f59e0b',
    pink: '#ec4899',
    red: '#ef4444'
};

// Load settings from localStorage
function loadSettings() {
    const saved = storage.get('nylasClientSettings');
    if (saved) {
        Object.assign(settingsState, saved);
        applySettings();
    }
}

// Save settings to localStorage
function saveSettings() {
    // Collect toggle states
    settingsState.threading = document.getElementById('threadingToggle')?.checked ?? true;
    settingsState.avatars = document.getElementById('avatarsToggle')?.checked ?? true;
    settingsState.previewPane = document.getElementById('previewToggle')?.checked ?? true;

    storage.set('nylasClientSettings', settingsState);
    showToast('success', 'Settings Saved', 'Your preferences have been saved');
    toggleSettings();
}

// Reset settings to defaults
function resetSettings() {
    settingsState.aiProvider = null;  // No default - user must select
    settingsState.theme = 'dark';
    settingsState.accentColor = 'purple';
    settingsState.threading = true;
    settingsState.avatars = true;
    settingsState.previewPane = true;

    applySettings();
    updateSettingsUI();
    showToast('info', 'Settings Reset', 'All settings restored to defaults');
}

// Apply settings to UI
function applySettings() {
    setAccentColor(settingsState.accentColor, false);
    setTheme(settingsState.theme, false);
}

// Update settings UI to reflect current state
function updateSettingsUI() {
    // Update AI provider selection
    document.querySelectorAll('.settings-option').forEach(opt => {
        opt.classList.remove('selected');
        const input = opt.querySelector('input');
        if (input && input.value === settingsState.aiProvider) {
            opt.classList.add('selected');
            input.checked = true;
        }
    });

    // Update theme buttons
    document.querySelectorAll('.theme-btn').forEach(btn => {
        btn.classList.toggle('active', btn.dataset.theme === settingsState.theme);
    });

    // Update color options
    document.querySelectorAll('.color-option').forEach(opt => {
        opt.classList.toggle('active', opt.dataset.color === settingsState.accentColor);
    });

    // Update toggles
    const threadingToggle = document.getElementById('threadingToggle');
    const avatarsToggle = document.getElementById('avatarsToggle');
    const previewToggle = document.getElementById('previewToggle');

    if (threadingToggle) threadingToggle.checked = settingsState.threading;
    if (avatarsToggle) avatarsToggle.checked = settingsState.avatars;
    if (previewToggle) previewToggle.checked = settingsState.previewPane;
}

// Toggle settings modal
function toggleSettings() {
    const overlay = document.getElementById('settingsOverlay');
    if (!overlay) return;

    if (overlay.classList.contains('active')) {
        overlay.classList.remove('active');
        setTimeout(() => overlay.style.display = 'none', 200);
    } else {
        overlay.style.display = 'flex';
        updateSettingsUI();
        requestAnimationFrame(() => overlay.classList.add('active'));
    }
}

// Close settings when clicking overlay
function closeSettingsOnOverlay(event) {
    if (event.target.id === 'settingsOverlay') {
        toggleSettings();
    }
}

// Set theme mode
function setTheme(theme, notify = true) {
    settingsState.theme = theme;

    document.querySelectorAll('.theme-btn').forEach(btn => {
        btn.classList.toggle('active', btn.dataset.theme === theme);
    });

    if (theme === 'light') {
        document.body.classList.add('light-theme');
        document.body.classList.remove('dark-theme');
    } else if (theme === 'dark') {
        document.body.classList.remove('light-theme');
        document.body.classList.add('dark-theme');
    } else {
        const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
        document.body.classList.toggle('dark-theme', prefersDark);
        document.body.classList.toggle('light-theme', !prefersDark);
    }

    if (notify) {
        showToast('info', 'Theme Updated', `Switched to ${theme} mode`);
    }
}

// Set accent color
function setAccentColor(color, notify = true) {
    settingsState.accentColor = color;

    document.querySelectorAll('.color-option').forEach(opt => {
        opt.classList.toggle('active', opt.dataset.color === color);
    });

    const colorValue = accentColors[color] || accentColors.purple;
    document.documentElement.style.setProperty('--accent', colorValue);
    document.documentElement.style.setProperty('--gradient-accent',
        `linear-gradient(135deg, ${colorValue} 0%, ${adjustColor(colorValue, -20)} 100%)`);

    if (notify) {
        showToast('info', 'Color Updated', `Accent color set to ${color}`);
    }
}

// Helper to adjust color brightness
function adjustColor(hex, percent) {
    const num = parseInt(hex.slice(1), 16);
    const amt = Math.round(2.55 * percent);
    const R = Math.min(255, Math.max(0, (num >> 16) + amt));
    const G = Math.min(255, Math.max(0, ((num >> 8) & 0x00FF) + amt));
    const B = Math.min(255, Math.max(0, (num & 0x0000FF) + amt));
    return `#${(0x1000000 + R * 0x10000 + G * 0x100 + B).toString(16).slice(1)}`;
}

// Initialize AI provider selection listeners
function initSettingsListeners() {
    document.querySelectorAll('.settings-option input[name="ai-provider"]').forEach(input => {
        input.addEventListener('change', function() {
            settingsState.aiProvider = this.value;
            document.querySelectorAll('.settings-option').forEach(opt => opt.classList.remove('selected'));
            this.closest('.settings-option').classList.add('selected');
        });
    });

    // Listen for system theme changes
    window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', (e) => {
        if (settingsState.theme === 'system') {
            setTheme('system', false);
        }
    });
}

// ====================================
// REFRESH INTERVAL FUNCTIONALITY
// ====================================

// Set refresh interval
function setRefreshInterval(seconds) {
    settingsState.refreshInterval = seconds;

    // Update UI
    document.querySelectorAll('.interval-btn').forEach(btn => {
        btn.classList.toggle('active', parseInt(btn.dataset.interval) === seconds);
    });

    // Update status text
    updateRefreshStatus();

    // Restart timer with new interval
    startRefreshTimer();
}

// Update refresh status display
function updateRefreshStatus() {
    const statusEl = document.getElementById('refreshStatus');
    if (!statusEl) return;

    const indicator = statusEl.querySelector('.refresh-indicator');
    const text = statusEl.querySelector('.refresh-text');

    if (settingsState.refreshInterval === 0) {
        indicator.classList.add('paused');
        text.textContent = 'Manual refresh only';
    } else {
        indicator.classList.remove('paused');
        const interval = settingsState.refreshInterval;
        if (interval < 60) {
            text.textContent = `Auto-refresh every ${interval} seconds`;
        } else if (interval === 60) {
            text.textContent = 'Auto-refresh every 1 minute';
        } else {
            text.textContent = `Auto-refresh every ${interval / 60} minutes`;
        }
    }
}

// Start the refresh timer
function startRefreshTimer() {
    // Clear existing timer
    if (refreshTimer) {
        clearInterval(refreshTimer);
        refreshTimer = null;
    }

    // Don't start if manual mode
    if (settingsState.refreshInterval === 0) {
        console.log('%c⏸️ Auto-refresh disabled', 'color: #f59e0b;');
        return;
    }

    // Start new timer
    refreshTimer = setInterval(() => {
        refreshEmails();
    }, settingsState.refreshInterval * 1000);

    console.log(`%c🔄 Auto-refresh started: every ${settingsState.refreshInterval}s`, 'color: #22c55e;');
}

// Refresh emails function. Re-fetches the currently visible folder so newly
// arrived messages show up without requiring a full page reload.
function refreshEmails() {
    lastRefreshTime = Date.now();

    const syncStatus = document.querySelector('.sync-status');
    if (syncStatus) {
        syncStatus.classList.add('syncing');
    }

    const clearSpinner = () => {
        if (syncStatus) syncStatus.classList.remove('syncing');
    };

    if (typeof EmailListManager === 'undefined' || typeof EmailListManager.loadEmails !== 'function') {
        clearSpinner();
        return;
    }

    if (EmailListManager.isLoading) {
        clearSpinner();
        return;
    }

    const folder = EmailListManager.currentFolder || 'INBOX';
    Promise.resolve(EmailListManager.loadEmails(folder))
        .catch(err => console.error('[refreshEmails] failed:', err))
        .finally(clearSpinner);
}

// Manual refresh triggered by a user action (e.g. pull-to-refresh, button).
function manualRefresh() {
    refreshEmails();
    if (typeof showToast === 'function') {
        showToast('info', 'Refreshing', 'Checking for new emails...');
    }
}

// Update settings UI to include refresh interval
const originalUpdateSettingsUI = updateSettingsUI;
updateSettingsUI = function() {
    originalUpdateSettingsUI();

    // Update refresh interval buttons
    document.querySelectorAll('.interval-btn').forEach(btn => {
        btn.classList.toggle('active', parseInt(btn.dataset.interval) === settingsState.refreshInterval);
    });

    updateRefreshStatus();
};

// Initialize settings on load
document.addEventListener('DOMContentLoaded', () => {
    loadSettings();
    initSettingsListeners();
    startRefreshTimer();
});

// ====================================
// NOTETAKER SOURCES MANAGEMENT
// ====================================

// Create element helper
function createElement(tag, className, textContent) {
    const el = document.createElement(tag);
    if (className) el.className = className;
    if (textContent) el.textContent = textContent;
    return el;
}

// Render notetaker sources list in settings
function renderNotetakerSources() {
    const container = document.getElementById('notetakerSourcesList');
    if (!container) return;

    // Clear existing content
    container.textContent = '';

    if (!settingsState.notetakerSources || settingsState.notetakerSources.length === 0) {
        const emptyDiv = createElement('div', 'notetaker-sources-empty');
        const emptyText = createElement('p', null, 'No sources configured. Add a source to fetch meeting recordings from email.');
        emptyDiv.appendChild(emptyText);
        container.appendChild(emptyDiv);
        return;
    }

    settingsState.notetakerSources.forEach((source, index) => {
        const card = createElement('div', 'notetaker-source-card');
        card.onclick = () => editNotetakerSource(index);

        const icon = createElement('div', 'notetaker-source-icon', '📧');
        card.appendChild(icon);

        const info = createElement('div', 'notetaker-source-info');
        const fromDiv = createElement('div', 'notetaker-source-from', source.from);
        info.appendChild(fromDiv);

        const domainDiv = createElement('div', 'notetaker-source-domain', '🔗 ' + source.linkDomain);
        info.appendChild(domainDiv);

        if (source.subject) {
            const subjectDiv = createElement('div', 'notetaker-source-subject', '📌 Subject: ' + source.subject);
            info.appendChild(subjectDiv);
        }
        card.appendChild(info);

        const deleteBtn = createElement('button', 'notetaker-source-delete');
        deleteBtn.title = 'Delete source';
        deleteBtn.onclick = (e) => {
            e.stopPropagation();
            deleteNotetakerSource(index);
        };
        const svg = document.createElementNS('http://www.w3.org/2000/svg', 'svg');
        svg.setAttribute('width', '16');
        svg.setAttribute('height', '16');
        svg.setAttribute('fill', 'none');
        svg.setAttribute('stroke', 'currentColor');
        svg.setAttribute('stroke-width', '2');
        svg.setAttribute('viewBox', '0 0 24 24');
        const path1 = document.createElementNS('http://www.w3.org/2000/svg', 'path');
        path1.setAttribute('d', 'M18 6L6 18');
        const path2 = document.createElementNS('http://www.w3.org/2000/svg', 'path');
        path2.setAttribute('d', 'M6 6l12 12');
        svg.appendChild(path1);
        svg.appendChild(path2);
        deleteBtn.appendChild(svg);
        card.appendChild(deleteBtn);

        container.appendChild(card);
    });
}

// Open add notetaker source modal
function openAddNotetakerSourceModal() {
    document.getElementById('notetakerSourceModalTitle').textContent = 'Add Notetaker Source';
    document.getElementById('notetakerSourceEditIndex').value = '-1';
    document.getElementById('notetakerSourceFrom').value = '';
    document.getElementById('notetakerSourceSubject').value = '';
    document.getElementById('notetakerSourceLink').value = '';
    document.getElementById('notetakerSourceModal').classList.remove('hidden');
}

// Edit existing notetaker source
function editNotetakerSource(index) {
    const source = settingsState.notetakerSources[index];
    if (!source) return;

    document.getElementById('notetakerSourceModalTitle').textContent = 'Edit Notetaker Source';
    document.getElementById('notetakerSourceEditIndex').value = index;
    document.getElementById('notetakerSourceFrom').value = source.from || '';
    document.getElementById('notetakerSourceSubject').value = source.subject || '';
    document.getElementById('notetakerSourceLink').value = source.linkDomain || '';
    document.getElementById('notetakerSourceModal').classList.remove('hidden');
}

// Close notetaker source modal
function closeNotetakerSourceModal() {
    document.getElementById('notetakerSourceModal').classList.add('hidden');
}

// Save notetaker source
function saveNotetakerSource() {
    const fromEmail = document.getElementById('notetakerSourceFrom').value.trim();
    const subject = document.getElementById('notetakerSourceSubject').value.trim();
    const linkDomain = document.getElementById('notetakerSourceLink').value.trim();
    const editIndex = parseInt(document.getElementById('notetakerSourceEditIndex').value);

    // Validate required fields
    if (!fromEmail) {
        showToast('error', 'Validation Error', 'From Email is required');
        return;
    }
    if (!linkDomain) {
        showToast('error', 'Validation Error', 'External Link Domain is required');
        return;
    }

    const source = { from: fromEmail, subject: subject, linkDomain: linkDomain };

    if (!settingsState.notetakerSources) {
        settingsState.notetakerSources = [];
    }

    if (editIndex >= 0 && editIndex < settingsState.notetakerSources.length) {
        // Edit existing
        settingsState.notetakerSources[editIndex] = source;
        showToast('success', 'Source Updated', 'Notetaker source has been updated');
    } else {
        // Add new
        settingsState.notetakerSources.push(source);
        showToast('success', 'Source Added', 'New notetaker source has been added');
    }

    closeNotetakerSourceModal();
    renderNotetakerSources();
}

// Delete notetaker source
function deleteNotetakerSource(index) {
    if (!settingsState.notetakerSources || index < 0 || index >= settingsState.notetakerSources.length) {
        return;
    }

    settingsState.notetakerSources.splice(index, 1);
    renderNotetakerSources();
    showToast('info', 'Source Removed', 'Notetaker source has been deleted');
}

// Update the updateSettingsUI to also render notetaker sources
const originalUpdateSettingsUIForNotetaker = updateSettingsUI;
updateSettingsUI = function() {
    originalUpdateSettingsUIForNotetaker();
    renderNotetakerSources();
};

console.log('%c⚙️ Settings module loaded', 'color: #8b5cf6;');
