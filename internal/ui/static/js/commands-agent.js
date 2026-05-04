// =============================================================================
// Agent Commands
// =============================================================================

const agentCommandSections = [
    {
        title: 'Status',
        commands: {
            'status': { title: 'Status', cmd: 'agent status', desc: 'Check connector and account readiness' }
        }
    },
    {
        title: 'Account',
        commands: {
            'account-list': { title: 'List', cmd: 'agent account list', desc: 'List agent accounts' },
            'account-get': { title: 'Get', cmd: 'agent account get', desc: 'Show an agent account', param: { name: 'agent-id-or-email', placeholder: 'agent-id or email' } },
            'account-create': {
                title: 'Create',
                cmd: 'agent account create',
                desc: 'Create a new agent account',
                param: { name: 'email', placeholder: 'me@yourapp.nylas.email' },
                flags: [
                    { name: 'app-password', type: 'text', label: 'App password', placeholder: 'Optional IMAP/SMTP app password' },
                    { name: 'policy-id', type: 'text', label: 'Policy ID', placeholder: 'Attach a policy by ID (optional)' }
                ]
            },
            'account-update': {
                title: 'Update',
                cmd: 'agent account update',
                desc: 'Update an agent account',
                param: { name: 'agent-id-or-email', placeholder: 'agent-id or email' },
                flags: [
                    { name: 'app-password', type: 'text', label: 'App password', placeholder: 'Rotate or add the app password' }
                ]
            },
            'account-delete': { title: 'Delete', cmd: 'agent account delete', desc: 'Delete an agent account', param: { name: 'agent-id-or-email', placeholder: 'agent-id or email' } }
        }
    },
    {
        title: 'Policy',
        commands: {
            'policy-list': { title: 'List', cmd: 'agent policy list', desc: 'List agent policies' },
            'policy-get': { title: 'Get', cmd: 'agent policy get', desc: 'Show a policy summary', param: { name: 'policy-id', placeholder: 'Enter policy ID...' } },
            'policy-read': { title: 'Read', cmd: 'agent policy read', desc: 'Read a full policy document', param: { name: 'policy-id', placeholder: 'Enter policy ID...' } },
            'policy-create': {
                title: 'Create',
                cmd: 'agent policy create',
                desc: 'Create a policy',
                flags: [
                    { name: 'name', type: 'text', label: 'Name', placeholder: 'Policy name' },
                    { name: 'data-file', type: 'text', label: 'Data file', placeholder: 'Path to JSON request body' }
                ]
            },
            'policy-update': {
                title: 'Update',
                cmd: 'agent policy update',
                desc: 'Update a policy',
                param: { name: 'policy-id', placeholder: 'Enter policy ID...' },
                flags: [
                    { name: 'name', type: 'text', label: 'Name', placeholder: 'Updated policy name' },
                    { name: 'data-file', type: 'text', label: 'Data file', placeholder: 'Path to JSON request body' }
                ]
            },
            'policy-delete': { title: 'Delete', cmd: 'agent policy delete', desc: 'Delete a policy', param: { name: 'policy-id', placeholder: 'Enter policy ID...' } }
        }
    },
    {
        title: 'Rule',
        commands: {
            'rule-list': { title: 'List', cmd: 'agent rule list', desc: 'List agent rules' },
            'rule-get': { title: 'Get', cmd: 'agent rule get', desc: 'Show a rule summary', param: { name: 'rule-id', placeholder: 'Enter rule ID...' } },
            'rule-read': { title: 'Read', cmd: 'agent rule read', desc: 'Read a full rule document', param: { name: 'rule-id', placeholder: 'Enter rule ID...' } },
            'rule-create': {
                title: 'Create',
                cmd: 'agent rule create',
                desc: 'Create a rule',
                flags: [
                    { name: 'name', type: 'text', label: 'Name', placeholder: 'Rule name' },
                    { name: 'description', type: 'text', label: 'Description', placeholder: 'Rule description' },
                    { name: 'priority', type: 'number', label: 'Priority', placeholder: '0' },
                    { name: 'trigger', type: 'text', label: 'Trigger', placeholder: 'inbound or outbound' },
                    { name: 'enabled', type: 'checkbox', label: 'Enabled' },
                    { name: 'policy-id', type: 'text', label: 'Policy ID', placeholder: 'Attach to policy ID' },
                    { name: 'data-file', type: 'text', label: 'Data file', placeholder: 'Path to JSON request body' }
                ]
            },
            'rule-update': {
                title: 'Update',
                cmd: 'agent rule update',
                desc: 'Update a rule',
                param: { name: 'rule-id', placeholder: 'Enter rule ID...' },
                flags: [
                    { name: 'name', type: 'text', label: 'Name', placeholder: 'Updated rule name' },
                    { name: 'description', type: 'text', label: 'Description', placeholder: 'Updated description' },
                    { name: 'priority', type: 'number', label: 'Priority', placeholder: '0' },
                    { name: 'enabled', type: 'checkbox', label: 'Enabled' },
                    { name: 'data-file', type: 'text', label: 'Data file', placeholder: 'Path to JSON request body' }
                ]
            },
            'rule-delete': { title: 'Delete', cmd: 'agent rule delete', desc: 'Delete a rule', param: { name: 'rule-id', placeholder: 'Enter rule ID...' } }
        }
    }
];

const agentCommands = {};
agentCommandSections.forEach(section => {
    Object.assign(agentCommands, section.commands);
});

let currentAgentCmd = '';

function showAgentCmd(cmd) {
    const data = agentCommands[cmd];
    if (!data) return;

    currentAgentCmd = cmd;

    document.querySelectorAll('#page-agent .cmd-item').forEach(item => {
        item.classList.toggle('active', item.dataset.cmd === cmd);
    });

    const detail = document.getElementById('agent-detail');
    detail.querySelector('.detail-placeholder').style.display = 'none';
    detail.querySelector('.detail-content').style.display = 'block';

    document.getElementById('agent-detail-title').textContent = data.title;
    document.getElementById('agent-detail-cmd').textContent = 'nylas ' + data.cmd;
    document.getElementById('agent-detail-desc').textContent = data.desc || '';
    document.getElementById('agent-output').textContent = 'Click "Run" to execute command...';
    document.getElementById('agent-output').className = 'output-pre';

    showParamInput('agent', data.param, data.flags);
}

async function runAgentCmd() {
    if (!currentAgentCmd) return;

    const data = agentCommands[currentAgentCmd];
    const output = document.getElementById('agent-output');
    const btn = document.getElementById('agent-run-btn');
    const fullCmd = buildCommand(data.cmd, 'agent', data.flags);

    document.getElementById('agent-detail-cmd').textContent = 'nylas ' + fullCmd;

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

        updateTimestamp('agent');
    } catch (err) {
        setOutputError(output, 'Failed to execute command: ' + err.message);
        showToast('Connection error', 'error');
    } finally {
        setButtonLoading(btn, false);
    }
}

function refreshAgentCmd() {
    if (currentAgentCmd) runAgentCmd();
}

function renderAgentCommands() {
    renderCommandSections('agent-cmd-list', agentCommandSections, 'showAgentCmd');
}
