// commands.js — Slash command handling
const Commands = {
    // Registry of all commands
    registry: [
        { name: 'help', type: 'client', description: 'Show available commands' },
        { name: 'new', type: 'client', description: 'Start a new conversation' },
        { name: 'reset', type: 'client', description: 'Start a new conversation' },
        { name: 'clear', type: 'client', description: 'Clear current messages' },
        { name: 'model', type: 'client', description: 'Switch AI agent (e.g. /model claude)', args: '<name>' },
        { name: 'agent', type: 'client', description: 'Switch AI agent (e.g. /agent ollama)', args: '<name>' },
        { name: 'status', type: 'server', description: 'Show current session status' },
        { name: 'email', type: 'server', description: 'Quick email lookup (e.g. /email from:sarah)', args: '[query]' },
        { name: 'calendar', type: 'server', description: 'Show upcoming events (e.g. /calendar 7)', args: '[days]' },
        { name: 'contacts', type: 'server', description: 'Search contacts (e.g. /contacts john)', args: '[query]' },
    ],

    // Parse input to check if it's a command
    parse(input) {
        const trimmed = input.trim();
        if (!trimmed.startsWith('/')) {
            return { isCommand: false };
        }

        const parts = trimmed.slice(1).split(/\s+/);
        const name = parts[0].toLowerCase();
        const args = parts.slice(1).join(' ');

        const cmd = this.registry.find(c => c.name === name);
        if (!cmd) {
            return { isCommand: false };
        }

        return { isCommand: true, name, args, type: cmd.type };
    },

    // Execute a client-side command. Returns true if handled.
    async executeClient(name, args) {
        switch (name) {
            case 'help':
                this.showHelp();
                return true;
            case 'new':
            case 'reset':
                await Sidebar.newChat();
                return true;
            case 'clear':
                this.clearMessages();
                return true;
            case 'model':
            case 'agent':
                this.switchAgent(args);
                return true;
            default:
                return false;
        }
    },

    // Execute a server-side command via API
    async executeServer(name, args, conversationId) {
        try {
            const result = await ChatAPI.executeCommand(name, args, conversationId);
            Chat.appendSystemMessage(result.content || result.error || 'Command executed.');
        } catch (err) {
            Chat.appendSystemMessage('Command error: ' + err.message);
        }
    },

    // Show help with all available commands
    showHelp() {
        const lines = ['**Available Commands:**', ''];
        for (const cmd of this.registry) {
            const argHint = cmd.args ? ' ' + cmd.args : '';
            lines.push('`/' + cmd.name + argHint + '` — ' + cmd.description);
        }
        lines.push('', 'Type a command or just chat normally.');
        Chat.appendSystemMessage(lines.join('\n'));
    },

    // Clear messages in current conversation view
    clearMessages() {
        const messages = document.getElementById('messages');
        const welcome = document.getElementById('welcome');
        messages.innerHTML = '';
        if (welcome) messages.appendChild(welcome);
        Chat.showWelcome();
        Chat.appendSystemMessage('Messages cleared.');
    },

    // Switch agent via sidebar
    switchAgent(name) {
        if (!name) {
            Chat.appendSystemMessage('Usage: `/model <name>` — Available agents are in the sidebar dropdown.');
            return;
        }
        const select = document.getElementById('agent-select');
        const options = Array.from(select.options);
        const match = options.find(o => o.value.toLowerCase() === name.toLowerCase());
        if (!match) {
            Chat.appendSystemMessage('Unknown agent: `' + name + '`. Available: ' +
                options.map(o => '`' + o.value + '`').join(', '));
            return;
        }
        select.value = match.value;
        Sidebar.switchAgent(match.value);
        Chat.appendSystemMessage('Switched to agent: `' + match.value + '`');
    },

    // Tab completion for command names
    complete(partial) {
        if (!partial.startsWith('/')) return null;

        const typed = partial.slice(1).toLowerCase();
        if (!typed) return null;

        const matches = this.registry
            .filter(c => c.name.startsWith(typed))
            .map(c => '/' + c.name);

        if (matches.length === 1) {
            return matches[0] + ' ';
        }
        return null;
    }
};
