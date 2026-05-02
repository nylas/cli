// =============================================================================
// Audit Commands
// =============================================================================

const auditCommandSections = [
    {
        title: 'Setup',
        commands: {
            'init': {
                title: 'Init',
                cmd: 'audit init',
                desc: 'Initialize audit logging configuration',
                flags: [
                    { name: 'path', type: 'text', label: 'Log directory', placeholder: '~/.config/nylas/audit' },
                    { name: 'retention', type: 'number', label: 'Retention (days)', placeholder: '90' },
                    { name: 'max-size', type: 'number', label: 'Max size (MB)', placeholder: '100' },
                    { name: 'format', type: 'text', label: 'Format', placeholder: 'jsonl or json' },
                    { name: 'enable', type: 'checkbox', label: 'Enable immediately' },
                    { name: 'no-prompt', type: 'checkbox', label: 'Skip prompts' }
                ]
            }
        }
    },
    {
        title: 'Logs',
        commands: {
            'logs-show': {
                title: 'Show',
                cmd: 'audit logs show',
                desc: 'Show recent audit log entries',
                flags: [
                    { name: 'limit', type: 'number', label: 'Limit', placeholder: '20', short: 'n' },
                    { name: 'since', type: 'text', label: 'Since', placeholder: 'YYYY-MM-DD' },
                    { name: 'until', type: 'text', label: 'Until', placeholder: 'YYYY-MM-DD' },
                    { name: 'command', type: 'text', label: 'Command', placeholder: 'Filter by command prefix' },
                    { name: 'status', type: 'text', label: 'Status', placeholder: 'success or error' },
                    { name: 'grant', type: 'text', label: 'Grant ID', placeholder: 'Filter by grant' },
                    { name: 'request-id', type: 'text', label: 'Request ID', placeholder: 'Filter by Nylas request ID' },
                    { name: 'invoker', type: 'text', label: 'Invoker', placeholder: 'Filter by username' },
                    { name: 'source', type: 'text', label: 'Source', placeholder: 'claude-code, github-actions, terminal' }
                ]
            },
            'logs-enable': { title: 'Enable', cmd: 'audit logs enable', desc: 'Enable audit logging' },
            'logs-disable': { title: 'Disable', cmd: 'audit logs disable', desc: 'Disable audit logging' },
            'logs-status': { title: 'Status', cmd: 'audit logs status', desc: 'Show audit logging status' },
            'logs-clear': { title: 'Clear', cmd: 'audit logs clear', desc: 'Clear all audit log entries' },
            'logs-summary': {
                title: 'Summary',
                cmd: 'audit logs summary',
                desc: 'Aggregated audit log summary',
                flags: [
                    { name: 'days', type: 'number', label: 'Days', placeholder: '7' }
                ]
            }
        }
    },
    {
        title: 'Config',
        commands: {
            'config-show': { title: 'Show', cmd: 'audit config show', desc: 'Show audit configuration' },
            'config-set': {
                title: 'Set',
                cmd: 'audit config set',
                desc: 'Set an audit config value',
                param: { name: 'key value', placeholder: 'e.g. retention 30' }
            }
        }
    },
    {
        title: 'Export',
        commands: {
            'export': {
                title: 'Export',
                cmd: 'audit export',
                desc: 'Export audit log entries',
                flags: [
                    { name: 'output', type: 'text', label: 'Output file', placeholder: 'audit-2026-05.json', short: 'o' },
                    { name: 'format', type: 'text', label: 'Format', placeholder: 'json or csv' },
                    { name: 'since', type: 'text', label: 'Since', placeholder: 'YYYY-MM-DD' },
                    { name: 'until', type: 'text', label: 'Until', placeholder: 'YYYY-MM-DD' },
                    { name: 'limit', type: 'number', label: 'Limit', placeholder: '10000', short: 'n' }
                ]
            }
        }
    }
];

const auditCommands = {};
auditCommandSections.forEach(section => {
    Object.assign(auditCommands, section.commands);
});

let currentAuditCmd = '';

function showAuditCmd(cmd) {
    const data = auditCommands[cmd];
    if (!data) return;

    currentAuditCmd = cmd;

    document.querySelectorAll('#page-audit .cmd-item').forEach(item => {
        item.classList.toggle('active', item.dataset.cmd === cmd);
    });

    const detail = document.getElementById('audit-detail');
    detail.querySelector('.detail-placeholder').style.display = 'none';
    detail.querySelector('.detail-content').style.display = 'block';

    document.getElementById('audit-detail-title').textContent = data.title;
    document.getElementById('audit-detail-cmd').textContent = 'nylas ' + data.cmd;
    document.getElementById('audit-detail-desc').textContent = data.desc || '';
    document.getElementById('audit-output').textContent = 'Click "Run" to execute command...';
    document.getElementById('audit-output').className = 'output-pre';

    showParamInput('audit', data.param, data.flags);
}

async function runAuditCmd() {
    if (!currentAuditCmd) return;

    const data = auditCommands[currentAuditCmd];
    const output = document.getElementById('audit-output');
    const btn = document.getElementById('audit-run-btn');
    const fullCmd = buildCommand(data.cmd, 'audit', data.flags);

    document.getElementById('audit-detail-cmd').textContent = 'nylas ' + fullCmd;

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

        updateTimestamp('audit');
    } catch (err) {
        setOutputError(output, 'Failed to execute command: ' + err.message);
        showToast('Connection error', 'error');
    } finally {
        setButtonLoading(btn, false);
    }
}

function refreshAuditCmd() {
    if (currentAuditCmd) runAuditCmd();
}

function renderAuditCommands() {
    renderCommandSections('audit-cmd-list', auditCommandSections, 'showAuditCmd');
}
