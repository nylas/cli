// =============================================================================
// Scheduler Commands
// =============================================================================

const schedulerCommandSections = [
    {
        title: 'Configurations',
        commands: {
            'config-list': { title: 'List', cmd: 'scheduler configurations list', desc: 'List scheduler configurations' },
            'config-show': { title: 'Show', cmd: 'scheduler configurations show', desc: 'Show configuration details', param: { name: 'config-id', placeholder: 'Enter configuration ID...' } },
            'config-create': {
                title: 'Create',
                cmd: 'scheduler configurations create',
                desc: 'Create a scheduler configuration',
                flags: [
                    { name: 'name', type: 'text', label: 'Name', placeholder: 'Configuration name', required: true },
                    { name: 'title', type: 'text', label: 'Title', placeholder: 'Event title', required: true },
                    { name: 'participants', type: 'text', label: 'Participants', placeholder: 'email1@example.com,email2@example.com', required: true },
                    { name: 'duration', type: 'number', label: 'Duration (min)', placeholder: '30' },
                    { name: 'location', type: 'text', label: 'Location', placeholder: 'Meeting location' }
                ]
            }
        }
    },
    {
        title: 'Pages',
        commands: {
            'page-list': { title: 'List', cmd: 'scheduler pages list', desc: 'List scheduler pages' },
            'page-show': { title: 'Show', cmd: 'scheduler pages show', desc: 'Show page details', param: { name: 'page-id', placeholder: 'Enter page ID...' } },
            'page-create': {
                title: 'Create',
                cmd: 'scheduler pages create',
                desc: 'Create a scheduler page',
                flags: [
                    { name: 'config-id', type: 'text', label: 'Config ID', placeholder: 'Configuration ID', required: true },
                    { name: 'name', type: 'text', label: 'Name', placeholder: 'Page name', required: true },
                    { name: 'slug', type: 'text', label: 'Slug', placeholder: 'URL slug (optional)' }
                ]
            }
        }
    },
    {
        title: 'Sessions',
        commands: {
            'session-create': {
                title: 'Create',
                cmd: 'scheduler sessions create',
                desc: 'Create a scheduling session',
                flags: [
                    { name: 'config-id', type: 'text', label: 'Config ID', placeholder: 'Configuration ID', required: true },
                    { name: 'ttl', type: 'number', label: 'TTL (min)', placeholder: '30' }
                ]
            },
            'session-show': { title: 'Show', cmd: 'scheduler sessions show', desc: 'Show session details', param: { name: 'session-id', placeholder: 'Enter session ID...' } }
        }
    },
    {
        title: 'Bookings',
        commands: {
            'booking-show': { title: 'Show', cmd: 'scheduler bookings show', desc: 'Show booking details', flags: [{ name: 'configuration-id', label: 'Configuration ID', type: 'text', placeholder: 'Enter configuration ID...', required: true }], param: { name: 'booking-id', placeholder: 'Enter booking ID...' } },
            'booking-confirm': { title: 'Confirm', cmd: 'scheduler bookings confirm', desc: 'Confirm a booking', flags: [{ name: 'configuration-id', label: 'Configuration ID', type: 'text', placeholder: 'Enter configuration ID...', required: true }, { name: 'salt', label: 'Salt (from confirmation link)', type: 'text', placeholder: 'Enter salt...', required: true }], param: { name: 'booking-id', placeholder: 'Enter booking ID...' } },
            'booking-cancel': { title: 'Cancel', cmd: 'scheduler bookings cancel', desc: 'Cancel a booking', flags: [{ name: 'configuration-id', label: 'Configuration ID', type: 'text', placeholder: 'Enter configuration ID...', required: true }], param: { name: 'booking-id', placeholder: 'Enter booking ID...' } }
        }
    }
];

const schedulerCommands = {};
schedulerCommandSections.forEach(section => {
    Object.assign(schedulerCommands, section.commands);
});

let currentSchedulerCmd = '';

function showSchedulerCmd(cmd) {
    const data = schedulerCommands[cmd];
    if (!data) return;

    currentSchedulerCmd = cmd;

    document.querySelectorAll('#page-scheduler .cmd-item').forEach(item => {
        item.classList.toggle('active', item.dataset.cmd === cmd);
    });

    const detail = document.getElementById('scheduler-detail');
    detail.querySelector('.detail-placeholder').style.display = 'none';
    detail.querySelector('.detail-content').style.display = 'block';

    document.getElementById('scheduler-detail-title').textContent = data.title;
    document.getElementById('scheduler-detail-cmd').textContent = 'nylas ' + data.cmd;
    document.getElementById('scheduler-detail-desc').textContent = data.desc || '';
    document.getElementById('scheduler-output').textContent = 'Click "Run" to execute command...';
    document.getElementById('scheduler-output').className = 'output-pre';

    showParamInput('scheduler', data.param, data.flags);
}

async function runSchedulerCmd() {
    if (!currentSchedulerCmd) return;

    const data = schedulerCommands[currentSchedulerCmd];
    const output = document.getElementById('scheduler-output');
    const btn = document.getElementById('scheduler-run-btn');
    const fullCmd = buildCommand(data.cmd, 'scheduler', data.flags);

    document.getElementById('scheduler-detail-cmd').textContent = 'nylas ' + fullCmd;

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

        updateTimestamp('scheduler');
    } catch (err) {
        setOutputError(output, 'Failed to execute command: ' + err.message);
        showToast('Connection error', 'error');
    } finally {
        setButtonLoading(btn, false);
    }
}

function refreshSchedulerCmd() {
    if (currentSchedulerCmd) runSchedulerCmd();
}

function renderSchedulerCommands() {
    renderCommandSections('scheduler-cmd-list', schedulerCommandSections, 'showSchedulerCmd');
}
