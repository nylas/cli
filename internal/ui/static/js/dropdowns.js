// =============================================================================
// Dropdowns
// =============================================================================

function initDropdowns() {
    document.addEventListener('click', (e) => {
        if (!e.target.closest('.dropdown')) {
            document.querySelectorAll('.dropdown.open').forEach(d => d.classList.remove('open'));
        }
    });
}

function toggleDropdown(id) {
    const dropdown = document.getElementById(id);
    const wasOpen = dropdown.classList.contains('open');

    // Close all dropdowns
    document.querySelectorAll('.dropdown.open').forEach(d => d.classList.remove('open'));

    // Toggle this one
    if (!wasOpen) dropdown.classList.add('open');
}

function updateHeaderDropdowns() {
    const controls = document.getElementById('header-controls');
    if (!controls) return;

    // Show header controls on dashboard
    controls.style.display = 'flex';

    // Update app dropdown
    const appSpan = document.getElementById('selected-client');
    if (appSpan && currentConfig.client_id) {
        appSpan.textContent = currentConfig.client_id;
    }

    // Update app menu
    const appMenu = document.getElementById('client-menu');
    if (appMenu && currentConfig.client_id) {
        appMenu.innerHTML = `
            <button class="dropdown-item active">
                <div class="item-info">
                    <span class="item-title">${truncate(currentConfig.client_id, 20)}</span>
                    <span class="item-subtitle">${currentConfig.region?.toUpperCase() || 'US'} Region</span>
                </div>
                <svg class="item-check" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M20 6L9 17l-5-5"/></svg>
            </button>
        `;
    }

    // Update account dropdown
    updateAccountDropdown();
}

function updateAccountDropdown() {
    const accountSpan = document.getElementById('selected-grant');
    const accountMenu = document.getElementById('grant-menu');

    if (!accountSpan || !accountMenu) return;

    // Find default account
    const defaultAccount = currentGrants.find(g => g.id === currentDefaultGrant);

    if (defaultAccount) {
        accountSpan.textContent = defaultAccount.email || defaultAccount.id;
    } else if (currentGrants.length > 0) {
        accountSpan.textContent = currentGrants[0].email || currentGrants[0].id;
    } else {
        accountSpan.textContent = 'None';
    }

    // Build menu
    let menuHTML = currentGrants.map(g => `
        <button class="dropdown-item ${g.id === currentDefaultGrant ? 'active' : ''}" onclick="selectAccount('${esc(g.id)}')">
            <div class="item-avatar">${g.email[0].toUpperCase()}</div>
            <div class="item-info">
                <span class="item-title">${esc(g.email)}</span>
                <span class="item-subtitle">${formatProvider(g.provider)}</span>
            </div>
            <svg class="item-check" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M20 6L9 17l-5-5"/></svg>
        </button>
    `).join('');

    if (currentGrants.length > 0) {
        menuHTML += '<div class="dropdown-divider"></div>';
    }

    menuHTML += `
        <button class="dropdown-item add-new" onclick="showAddAccount()">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>
            <span class="item-title">Add Account</span>
        </button>
    `;

    accountMenu.innerHTML = menuHTML;
}

async function selectAccount(id) {
    try {
        const res = await fetch('/api/grants/default', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ grant_id: id })
        });
        if ((await res.json()).success) {
            currentDefaultGrant = id;
            updateAccountDropdown();
            loadAccounts();
        }
    } catch (err) {
        console.error('Failed to set default');
    }

    // Close dropdown
    document.querySelectorAll('.dropdown.open').forEach(d => d.classList.remove('open'));
}

function showAddAccount() {
    // Close dropdown
    document.querySelectorAll('.dropdown.open').forEach(d => d.classList.remove('open'));

    // Show a simple alert with the command
    alert('Run this command in your terminal:\n\nnylas auth login');
}

if (typeof module !== 'undefined' && module.exports) {
    module.exports = {
        toggleDropdown,
        updateAccountDropdown,
        updateHeaderDropdowns,
    };
}
