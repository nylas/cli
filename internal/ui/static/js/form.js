// =============================================================================
// Setup Form
// =============================================================================

function getSetupErrorElement(doc = document) {
    return doc.getElementById('setup-error') || doc.getElementById('error-msg');
}

function initForm() {
    const form = document.getElementById('setup-form');
    if (!form) return;

    document.getElementById('api-key')?.addEventListener('input', () => {
        resetApplicationSelection();
    });
    document.getElementById('region')?.addEventListener('change', () => {
        resetApplicationSelection();
    });

    form.addEventListener('submit', async (e) => {
        e.preventDefault();
        const btn = form.querySelector('.btn-primary');
        const error = getSetupErrorElement();

        error?.classList.remove('visible');
        btn.classList.add('loading');
        btn.disabled = true;

        try {
            const res = await fetch('/api/config/setup', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(buildSetupPayload())
            });
            const data = await res.json();
            handleSetupResult(data, res.status);
        } catch (err) {
            showFormError('Connection failed. Please try again.');
        } finally {
            btn.classList.remove('loading');
            btn.disabled = false;
        }
    });
}

function buildSetupPayload(doc = document) {
    const payload = {
        api_key: doc.getElementById('api-key')?.value.trim() || '',
        region: doc.getElementById('region')?.value || 'us',
    };

    const clientID = getSelectedClientID(doc);
    if (clientID) {
        payload.client_id = clientID;
    }

    return payload;
}

function getSelectedClientID(doc = document) {
    const field = doc.getElementById('client-id-field');
    const select = doc.getElementById('client-id');
    if (!field || !select || field.classList.contains('is-hidden')) return '';
    return select.value.trim();
}

function shouldShowApplicationSelection(status, data) {
    return status === 409 && Array.isArray(data?.applications) && data.applications.length > 0;
}

function formatApplicationLabel(app) {
    const name = app?.name || app?.id || 'Unknown application';
    const environment = app?.environment || 'production';
    return `${name} (${environment})`;
}

function handleSetupResult(data, status, deps = {}) {
    const showDashboardFn = deps.showDashboard || (typeof showDashboard === 'function' ? showDashboard : () => {});
    const showToastFn = deps.showToast || (typeof showToast === 'function' ? showToast : () => {});

    if (data?.success) {
        clearFormError();
        resetApplicationSelection();
        showDashboardFn(data);
        syncDashboardSummary(data);

        if (data.warning) {
            displaySetupWarning(data.warning);
            showToastFn(data.warning, 'info');
        } else {
            clearSetupWarning();
        }
        return;
    }

    clearSetupWarning();
    if (shouldShowApplicationSelection(status, data)) {
        showApplicationSelection(data.applications, data.error || 'Choose an application to continue.');
        return;
    }

    showFormError(data?.error || 'Setup failed');
}

function showApplicationSelection(applications, message, doc = document) {
    const field = doc.getElementById('client-id-field');
    const select = doc.getElementById('client-id');
    const btn = doc.getElementById('setup-btn');
    if (!field || !select) {
        showFormError(message);
        return;
    }

    select.innerHTML = applications.map((app) => (
        `<option value="${esc(app.id)}">${esc(formatApplicationLabel(app))}</option>`
    )).join('');

    if (applications[0]?.id) {
        select.value = applications[0].id;
    }

    field.classList.remove('is-hidden');
    btn && (btn.textContent = 'Connect Selected Application');
    showFormError(message);
    if (typeof select.focus === 'function') {
        select.focus();
    }
}

function resetApplicationSelection(doc = document) {
    const field = doc.getElementById('client-id-field');
    const select = doc.getElementById('client-id');
    const btn = doc.getElementById('setup-btn');

    if (field) {
        field.classList.add('is-hidden');
    }
    if (select) {
        select.innerHTML = '';
        select.value = '';
    }
    if (btn) {
        btn.textContent = 'Connect Account';
    }
    clearFormError(doc);
}

function syncDashboardSummary(data, doc = document) {
    const regionEl = doc.getElementById('config-region');
    const clientEl = doc.getElementById('config-client');
    if (regionEl) {
        regionEl.textContent = (data?.region || 'us').toUpperCase();
    }
    if (clientEl) {
        const clientID = data?.client_id || '';
        clientEl.textContent = clientID ? truncate(clientID, 16) : '-';
    }
}

function showFormError(msg, doc = document) {
    const error = getSetupErrorElement(doc);
    if (error) {
        error.textContent = msg;
        error.classList.add('visible');
    }
}

function clearFormError(doc = document) {
    const error = getSetupErrorElement(doc);
    if (error) {
        error.textContent = '';
        error.classList.remove('visible');
    }
}

function displaySetupWarning(msg, doc = document) {
    const warning = doc.getElementById('setup-warning-banner');
    if (!warning) return;

    warning.textContent = msg;
    warning.classList.add('visible');
}

function clearSetupWarning(doc = document) {
    const warning = doc.getElementById('setup-warning-banner');
    if (!warning) return;

    warning.textContent = '';
    warning.classList.remove('visible');
}

function togglePassword() {
    const input = document.getElementById('api-key');
    const btn = input.parentElement.querySelector('.input-btn');
    const isPassword = input.type === 'password';
    input.type = isPassword ? 'text' : 'password';
    btn?.classList.toggle('visible', isPassword);
}

if (typeof module !== 'undefined' && module.exports) {
    module.exports = {
        buildSetupPayload,
        clearFormError,
        clearSetupWarning,
        displaySetupWarning,
        formatApplicationLabel,
        getSelectedClientID,
        handleSetupResult,
        resetApplicationSelection,
        shouldShowApplicationSelection,
        showApplicationSelection,
        showFormError,
        syncDashboardSummary,
        togglePassword,
    };
}
