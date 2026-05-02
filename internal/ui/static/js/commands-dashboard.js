// =============================================================================
// Dashboard Commands (Nylas Dashboard SaaS — register, SSO, apps, orgs)
// =============================================================================

const dashboardCommandSections = [
    {
        title: 'Status',
        commands: {
            'status': { title: 'Status', cmd: 'dashboard status', desc: 'Show Dashboard session status' },
            'refresh': { title: 'Refresh', cmd: 'dashboard refresh', desc: 'Refresh Dashboard credentials' }
        }
    },
    {
        title: 'Account',
        commands: {
            'register': {
                title: 'Register',
                cmd: 'dashboard register',
                desc: 'Register a new Dashboard account',
                flags: [
                    { name: 'google', type: 'checkbox', label: 'Google SSO' },
                    { name: 'microsoft', type: 'checkbox', label: 'Microsoft SSO' },
                    { name: 'github', type: 'checkbox', label: 'GitHub SSO' },
                    { name: 'accept-privacy-policy', type: 'checkbox', label: 'Accept privacy policy' }
                ]
            },
            'login': {
                title: 'Login',
                cmd: 'dashboard login',
                desc: 'Log in to the Nylas Dashboard',
                flags: [
                    { name: 'google', type: 'checkbox', label: 'Google SSO' },
                    { name: 'microsoft', type: 'checkbox', label: 'Microsoft SSO' },
                    { name: 'github', type: 'checkbox', label: 'GitHub SSO' },
                    { name: 'email', type: 'checkbox', label: 'Email + password' },
                    { name: 'org', type: 'text', label: 'Organization', placeholder: 'Organization public ID' },
                    { name: 'user', type: 'text', label: 'User email', placeholder: 'For non-interactive login' }
                ]
            },
            'logout': { title: 'Logout', cmd: 'dashboard logout', desc: 'Log out of the Dashboard' }
        }
    },
    {
        title: 'SSO',
        commands: {
            'sso-login': {
                title: 'SSO Login',
                cmd: 'dashboard sso login',
                desc: 'Sign in via SSO provider',
                flags: [
                    { name: 'provider', type: 'text', label: 'Provider', placeholder: 'google, microsoft, github', short: 'p' }
                ]
            },
            'sso-register': {
                title: 'SSO Register',
                cmd: 'dashboard sso register',
                desc: 'Register via SSO provider',
                flags: [
                    { name: 'provider', type: 'text', label: 'Provider', placeholder: 'google, microsoft, github', short: 'p' },
                    { name: 'accept-privacy-policy', type: 'checkbox', label: 'Accept privacy policy' }
                ]
            }
        }
    },
    {
        title: 'Apps',
        commands: {
            'apps-list': {
                title: 'List',
                cmd: 'dashboard apps list',
                desc: 'List Dashboard applications',
                flags: [
                    { name: 'region', type: 'text', label: 'Region', placeholder: 'us or eu', short: 'r' }
                ]
            },
            'apps-create': {
                title: 'Create',
                cmd: 'dashboard apps create',
                desc: 'Create a Dashboard application',
                flags: [
                    { name: 'name', type: 'text', label: 'Name', placeholder: 'Application name', required: true, short: 'n' },
                    { name: 'region', type: 'text', label: 'Region', placeholder: 'us or eu', required: true, short: 'r' },
                    { name: 'secret-delivery', type: 'text', label: 'Secret delivery', placeholder: 'clipboard or file' }
                ]
            },
            'apps-use': {
                title: 'Use',
                cmd: 'dashboard apps use',
                desc: 'Switch the active Dashboard application',
                param: { name: 'application-id', placeholder: 'Enter application ID...' },
                flags: [
                    { name: 'region', type: 'text', label: 'Region', placeholder: 'us or eu', short: 'r' }
                ]
            }
        }
    },
    {
        title: 'API Keys',
        commands: {
            'apikeys-list': {
                title: 'List',
                cmd: 'dashboard apikeys list',
                desc: 'List Dashboard API keys',
                flags: [
                    { name: 'app', type: 'text', label: 'Application', placeholder: 'Application ID (overrides active)' },
                    { name: 'region', type: 'text', label: 'Region', placeholder: 'us or eu', short: 'r' }
                ]
            },
            'apikeys-create': {
                title: 'Create',
                cmd: 'dashboard apikeys create',
                desc: 'Create a Dashboard API key',
                flags: [
                    { name: 'app', type: 'text', label: 'Application', placeholder: 'Application ID (overrides active)' },
                    { name: 'region', type: 'text', label: 'Region', placeholder: 'us or eu', short: 'r' },
                    { name: 'name', type: 'text', label: 'Name', placeholder: 'API key name', short: 'n' },
                    { name: 'expires', type: 'number', label: 'Expires (days)', placeholder: '0 (no expiration)' },
                    { name: 'delivery', type: 'text', label: 'Delivery', placeholder: 'activate, clipboard, or file' }
                ]
            }
        }
    },
    {
        title: 'Orgs',
        commands: {
            'orgs-list': { title: 'List', cmd: 'dashboard orgs list', desc: 'List Dashboard organizations' },
            'orgs-switch': {
                title: 'Switch',
                cmd: 'dashboard switch',
                desc: 'Switch the active organization',
                flags: [
                    { name: 'org', type: 'text', label: 'Organization', placeholder: 'Organization public ID' }
                ]
            }
        }
    }
];

const dashboardCommands = {};
dashboardCommandSections.forEach(section => {
    Object.assign(dashboardCommands, section.commands);
});

let currentDashboardCmd = '';

function showDashboardCmd(cmd) {
    const data = dashboardCommands[cmd];
    if (!data) return;

    currentDashboardCmd = cmd;

    document.querySelectorAll('#page-dashboard .cmd-item').forEach(item => {
        item.classList.toggle('active', item.dataset.cmd === cmd);
    });

    const detail = document.getElementById('dashboard-detail');
    detail.querySelector('.detail-placeholder').style.display = 'none';
    detail.querySelector('.detail-content').style.display = 'block';

    document.getElementById('dashboard-detail-title').textContent = data.title;
    document.getElementById('dashboard-detail-cmd').textContent = 'nylas ' + data.cmd;
    document.getElementById('dashboard-detail-desc').textContent = data.desc || '';
    document.getElementById('dashboard-output').textContent = 'Click "Run" to execute command...';
    document.getElementById('dashboard-output').className = 'output-pre';

    showParamInput('dashboard', data.param, data.flags);
}

async function runDashboardCmd() {
    if (!currentDashboardCmd) return;

    const data = dashboardCommands[currentDashboardCmd];
    const output = document.getElementById('dashboard-output');
    const btn = document.getElementById('dashboard-run-btn');
    const fullCmd = buildCommand(data.cmd, 'dashboard', data.flags);

    document.getElementById('dashboard-detail-cmd').textContent = 'nylas ' + fullCmd;

    setButtonLoading(btn, true);
    setOutputLoading(output);

    try {
        const res = await fetch('/api/exec', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ command: fullCmd })
        });
        const result = await res.json();

        if (result.error) {
            setOutputError(output, result.error);
            showToast('Command failed', 'error');
        } else {
            setOutputSuccess(output, result.output);
            showToast('Command completed', 'success');
        }

        updateTimestamp('dashboard');
    } catch (err) {
        setOutputError(output, 'Failed to execute command: ' + err.message);
        showToast('Connection error', 'error');
    } finally {
        setButtonLoading(btn, false);
    }
}

function refreshDashboardCmd() {
    if (currentDashboardCmd) runDashboardCmd();
}

function renderDashboardCommands() {
    renderCommandSections('dashboard-cmd-list', dashboardCommandSections, 'showDashboardCmd');
}
