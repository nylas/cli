// sidebar.js — Conversation list management
const Sidebar = {
    activeId: null,
    currentAgent: null,

    async init() {
        await this.refresh();
        document.getElementById('btn-new-chat').addEventListener('click', () => this.newChat());
        document.getElementById('btn-toggle-sidebar').addEventListener('click', () => this.toggle());

        // Load agent info and populate dropdown
        try {
            const config = await ChatAPI.getConfig();
            this.currentAgent = config.agent;
            this.populateAgentDropdown(config.agent, config.available);
        } catch { /* ignore */ }

        // Agent switch handler
        document.getElementById('agent-select').addEventListener('change', (e) => {
            this.switchAgent(e.target.value);
        });
    },

    populateAgentDropdown(active, available) {
        const select = document.getElementById('agent-select');
        select.innerHTML = '';

        for (const agent of available) {
            const option = document.createElement('option');
            option.value = agent;
            option.textContent = agent.charAt(0).toUpperCase() + agent.slice(1);
            if (agent === active) option.selected = true;
            select.appendChild(option);
        }

        // Hide dropdown if only one agent
        select.parentElement.classList.toggle('single-agent', available.length <= 1);
    },

    async switchAgent(agent) {
        if (agent === this.currentAgent) return;
        try {
            const config = await ChatAPI.switchAgent(agent);
            this.currentAgent = config.agent;
            this.populateAgentDropdown(config.agent, config.available);
        } catch (err) {
            // Revert dropdown on failure
            const select = document.getElementById('agent-select');
            select.value = this.currentAgent;
        }
    },

    async refresh() {
        const conversations = await ChatAPI.listConversations();
        const list = document.getElementById('conversation-list');
        list.innerHTML = '';

        for (const conv of conversations) {
            const el = document.createElement('div');
            el.className = 'conv-item' + (conv.id === this.activeId ? ' active' : '');
            el.dataset.id = conv.id;

            const content = document.createElement('div');
            content.className = 'conv-item-content';

            const title = document.createElement('div');
            title.className = 'conv-title';
            title.textContent = conv.title || 'New conversation';
            content.appendChild(title);

            if (conv.preview) {
                const preview = document.createElement('div');
                preview.className = 'conv-preview';
                preview.textContent = conv.preview;
                content.appendChild(preview);
            }

            el.appendChild(content);

            const delBtn = document.createElement('button');
            delBtn.className = 'conv-delete';
            delBtn.textContent = '✕';
            delBtn.title = 'Delete conversation';
            delBtn.addEventListener('click', (e) => {
                e.stopPropagation();
                this.deleteConversation(conv.id);
            });
            el.appendChild(delBtn);

            el.addEventListener('click', () => this.loadConversation(conv.id));
            list.appendChild(el);
        }
    },

    async loadConversation(id) {
        this.activeId = id;
        const conv = await ChatAPI.getConversation(id);
        Chat.loadConversation(conv);
        this.highlightActive();
        this.closeMobile();
    },

    async newChat() {
        const conv = await ChatAPI.createConversation();
        this.activeId = conv.id;
        Chat.startNew(conv.id);
        await this.refresh();
        this.closeMobile();
    },

    async deleteConversation(id) {
        await ChatAPI.deleteConversation(id);
        if (this.activeId === id) {
            this.activeId = null;
            Chat.startNew(null);
        }
        await this.refresh();
    },

    highlightActive() {
        document.querySelectorAll('.conv-item').forEach(el => {
            el.classList.toggle('active', el.dataset.id === this.activeId);
        });
    },

    toggle() {
        document.getElementById('sidebar').classList.toggle('open');
    },

    closeMobile() {
        document.getElementById('sidebar').classList.remove('open');
    }
};
