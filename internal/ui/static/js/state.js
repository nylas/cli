// =============================================================================
// State Management
// =============================================================================

// Global state
let currentConfig = {};
let currentGrants = [];
let currentDefaultGrant = '';

// Initialize from server-provided state (no API calls needed)
function initFromServerState(state) {
    currentConfig = {
        region: state.region || 'us',
        client_id: state.clientID || ''
    };

    currentGrants = state.grants || [];
    currentDefaultGrant = state.defaultGrant || '';

    console.log('Initialized from server state:', state.configured ? 'configured' : 'needs setup');
}

// Fallback config check via API
async function checkConfig() {
    try {
        const res = await fetch('/api/config/status');
        const data = await res.json();
        if (data.configured) showDashboard(data);
    } catch (err) {
        console.error('Config check failed');
    }
}

function showDashboard(data) {
    document.getElementById('setup-view')?.classList.remove('active');
    document.getElementById('dashboard-view')?.classList.add('active');

    currentConfig = {
        region: data.region || 'us',
        client_id: data.client_id || ''
    };

    const region = currentConfig.region.toUpperCase();
    setText('config-region', region);
    setText('config-client', truncate(currentConfig.client_id, 16) || '-');

    loadAccounts();
}

async function loadAccounts() {
    try {
        const res = await fetch('/api/grants');
        const data = await res.json();
        currentGrants = data.grants || [];
        currentDefaultGrant = data.default_grant || '';

        renderAccounts(currentGrants, currentDefaultGrant);
        updateHeaderDropdowns();
    } catch (err) {
        console.error('Failed to load accounts');
    }
}

function renderAccounts(grants, defaultId) {
    const list = document.getElementById('accounts-list');
    if (!list) return;

    if (!grants.length) {
        list.innerHTML = '<div class="empty-state">No accounts connected yet</div>';
        return;
    }

    list.innerHTML = grants.map(g => `
        <div class="account-item">
            <div class="account-avatar">${g.email[0].toUpperCase()}</div>
            <div class="account-info">
                <div class="account-email">${esc(g.email)}</div>
                <div class="account-provider">${formatProvider(g.provider)}</div>
            </div>
            ${g.id === defaultId
                ? '<span class="account-badge">DEFAULT</span>'
                : `<button class="account-action" onclick="setDefault('${esc(g.id)}')">Set Default</button>`
            }
        </div>
    `).join('');
}

async function setDefault(id) {
    try {
        const res = await fetch('/api/grants/default', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ grant_id: id })
        });
        if ((await res.json()).success) {
            currentDefaultGrant = id;
            renderAccounts(currentGrants, currentDefaultGrant);
            updateAccountDropdown();
            showToast('Default account updated', 'success');
        }
    } catch (err) {
        console.error('Failed to set default');
        showToast('Failed to update default account', 'error');
    }
}

// Alias for template compatibility
const setDefaultGrant = setDefault;

// Select grant from header dropdown (used by template)
function selectGrant(id, email) {
    const grantValue = document.getElementById('selected-grant');
    if (grantValue) {
        grantValue.textContent = email || id;
    }

    document.querySelectorAll('#grant-menu .dropdown-item').forEach(item => {
        item.classList.toggle('active', item.dataset.grant === id);
    });

    setDefault(id);
    document.getElementById('grant-dropdown')?.classList.remove('open');
}

// Run a command from anywhere (used by Add Account button)
function runCommand(cmd) {
    if (cmd.startsWith('auth ')) {
        showPage('auth');
        const subCmd = cmd.replace('auth ', '');
        setTimeout(() => {
            showAuthCmd(subCmd);
            runAuthCmd();
        }, 100);
    }
}
