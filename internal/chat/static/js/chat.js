// chat.js — Main chat UI
const Chat = {
    conversationId: null,
    sending: false,
    streamingEl: null,
    streamingContent: '',

    init() {
        const form = document.getElementById('chat-form');
        const input = document.getElementById('chat-input');

        form.addEventListener('submit', (e) => {
            e.preventDefault();
            this.send();
        });

        // Auto-resize textarea
        input.addEventListener('input', () => {
            input.style.height = 'auto';
            input.style.height = Math.min(input.scrollHeight, 120) + 'px';
        });

        // Enter to send, Shift+Enter for newline; Tab for command completion
        input.addEventListener('keydown', (e) => {
            if (e.key === 'Tab' && input.value.startsWith('/')) {
                e.preventDefault();
                const completed = Commands.complete(input.value);
                if (completed) input.value = completed;
            } else if (e.key === 'Enter' && !e.shiftKey) {
                e.preventDefault();
                this.send();
            }
        });

        // Suggestion buttons
        document.querySelectorAll('.suggestion').forEach(btn => {
            btn.addEventListener('click', () => {
                input.value = btn.dataset.msg;
                this.send();
            });
        });
    },

    async send() {
        const input = document.getElementById('chat-input');
        const message = input.value.trim();
        if (!message || this.sending) return;

        // Check for slash commands
        const parsed = Commands.parse(message);
        if (parsed.isCommand) {
            input.value = '';
            input.style.height = 'auto';
            this.hideWelcome();

            if (parsed.type === 'client') {
                await Commands.executeClient(parsed.name, parsed.args);
            } else {
                await Commands.executeServer(parsed.name, parsed.args, this.conversationId);
            }
            return;
        }

        this.sending = true;
        this.hideWelcome();
        this.appendMessage('user', message);
        input.value = '';
        input.style.height = 'auto';
        document.getElementById('btn-send').disabled = true;

        const thinkingEl = this.showThinking();

        try {
            await ChatAPI.sendMessage(this.conversationId, message, (event, data) => {
                switch (event) {
                    case 'thinking':
                        // Already showing indicator
                        break;
                    case 'token':
                        thinkingEl.remove();
                        this.handleStreamToken(data.text);
                        break;
                    case 'stream_end':
                        this.finalizeStream();
                        break;
                    case 'stream_discard':
                        this.discardStream();
                        break;
                    case 'tool_call':
                        this.appendToolCall(data);
                        break;
                    case 'tool_result':
                        this.appendToolResult(data);
                        break;
                    case 'approval_required':
                        this.showApprovalCard(data);
                        break;
                    case 'approval_resolved':
                        this.resolveApprovalCard(data);
                        break;
                    case 'message':
                        thinkingEl.remove();
                        if (!this.streamingEl) {
                            this.appendMessage('assistant', data.content);
                        }
                        break;
                    case 'error':
                        thinkingEl.remove();
                        this.discardStream();
                        this.appendMessage('assistant', 'Error: ' + data.error);
                        break;
                    case 'done':
                        if (data.conversation_id) {
                            this.conversationId = data.conversation_id;
                        }
                        if (data.title && data.title !== 'New conversation') {
                            document.getElementById('chat-title').textContent = data.title;
                        }
                        Sidebar.activeId = this.conversationId;
                        Sidebar.refresh();
                        break;
                }
            });
        } catch (err) {
            thinkingEl.remove();
            this.discardStream();
            this.appendMessage('assistant', 'Connection error: ' + err.message);
        }

        this.sending = false;
        document.getElementById('btn-send').disabled = false;
        input.focus();
    },

    // Streaming support
    handleStreamToken(text) {
        if (!this.streamingEl) {
            this.streamingEl = this.createStreamingMessage();
            this.streamingContent = '';
        }
        this.streamingContent += text;
        this.renderStream();
    },

    createStreamingMessage() {
        const messages = document.getElementById('messages');
        const div = document.createElement('div');
        div.className = 'message assistant streaming';
        const bubble = document.createElement('div');
        bubble.className = 'message-content';
        div.appendChild(bubble);
        messages.appendChild(div);
        return div;
    },

    renderStream() {
        if (!this.streamingEl) return;
        const bubble = this.streamingEl.querySelector('.message-content');
        bubble.innerHTML = Markdown.render(this.streamingContent);
        this.scrollToBottom();
    },

    finalizeStream() {
        if (this.streamingEl) {
            this.streamingEl.classList.remove('streaming');
            this.renderStream();
            this.streamingEl = null;
            this.streamingContent = '';
        }
    },

    discardStream() {
        if (this.streamingEl) {
            this.streamingEl.remove();
            this.streamingEl = null;
            this.streamingContent = '';
        }
    },

    // Approval gating
    showApprovalCard(data) {
        const messages = document.getElementById('messages');
        const card = document.createElement('div');
        card.className = 'approval-card';
        card.id = 'approval-' + data.approval_id;

        let previewHtml = '<dl class="approval-preview">';
        for (const [key, val] of Object.entries(data.preview || {})) {
            previewHtml += '<dt>' + this.escapeHtml(key) + '</dt>' +
                '<dd>' + this.escapeHtml(String(val)) + '</dd>';
        }
        previewHtml += '</dl>';

        const header = document.createElement('div');
        header.className = 'approval-header';
        header.textContent = 'Confirm: ' + data.tool;

        const actionsDiv = document.createElement('div');
        actionsDiv.className = 'approval-actions';

        const approveBtn = document.createElement('button');
        approveBtn.className = 'btn-approve';
        approveBtn.textContent = 'Approve';
        approveBtn.dataset.action = 'approve';
        approveBtn.dataset.approvalId = data.approval_id;

        const rejectBtn = document.createElement('button');
        rejectBtn.className = 'btn-reject';
        rejectBtn.textContent = 'Reject';
        rejectBtn.dataset.action = 'reject';
        rejectBtn.dataset.approvalId = data.approval_id;

        actionsDiv.appendChild(approveBtn);
        actionsDiv.appendChild(rejectBtn);

        const statusDiv = document.createElement('div');
        statusDiv.className = 'approval-status';

        // Build card from safe DOM nodes; only previewHtml uses innerHTML
        // (it is constructed from escapeHtml-sanitised strings, never raw API data).
        card.innerHTML = previewHtml;
        card.insertBefore(header, card.firstChild);
        card.appendChild(actionsDiv);
        card.appendChild(statusDiv);

        messages.appendChild(card);
        this.scrollToBottom();
    },

    resolveApprovalCard(data) {
        const card = document.getElementById('approval-' + data.approval_id);
        if (!card) return;
        card.classList.add('resolved');
        const status = card.querySelector('.approval-status');
        status.textContent = data.approved ? 'Approved' : 'Rejected' + (data.reason ? ': ' + data.reason : '');
    },

    async approve(approvalId) {
        try { await ChatAPI.approveAction(approvalId); } catch { /* handled via SSE */ }
    },

    async reject(approvalId) {
        try { await ChatAPI.rejectAction(approvalId, 'rejected by user'); } catch { /* handled via SSE */ }
    },

    loadConversation(conv) {
        this.conversationId = conv.id;
        const messages = document.getElementById('messages');
        const welcome = document.getElementById('welcome');
        messages.innerHTML = '';
        if (welcome) messages.appendChild(welcome);
        this.hideWelcome();

        document.getElementById('chat-title').textContent = conv.title || 'New conversation';

        for (const msg of conv.messages) {
            if (msg.role === 'user' || msg.role === 'assistant') {
                this.appendMessage(msg.role, msg.content);
            } else if (msg.role === 'tool_call') {
                try {
                    const call = JSON.parse(msg.content);
                    this.appendToolCall(call);
                } catch { /* skip */ }
            } else if (msg.role === 'tool_result') {
                try {
                    const result = JSON.parse(msg.content);
                    this.appendToolResult(result);
                } catch { /* skip */ }
            }
        }

        this.scrollToBottom();
    },

    startNew(id) {
        this.conversationId = id;
        const messages = document.getElementById('messages');
        const welcome = document.getElementById('welcome');
        messages.innerHTML = '';
        if (welcome) messages.appendChild(welcome);
        document.getElementById('chat-title').textContent = 'New conversation';
        this.showWelcome();
    },

    appendMessage(role, content) {
        const messages = document.getElementById('messages');
        const div = document.createElement('div');
        div.className = 'message ' + role;

        const bubble = document.createElement('div');
        bubble.className = 'message-content';

        if (role === 'assistant') {
            bubble.innerHTML = Markdown.render(content);
        } else {
            bubble.textContent = content;
        }

        div.appendChild(bubble);
        messages.appendChild(div);
        this.scrollToBottom();
    },

    appendSystemMessage(content) {
        const messages = document.getElementById('messages');
        const div = document.createElement('div');
        div.className = 'message system';

        const bubble = document.createElement('div');
        bubble.className = 'message-content';
        bubble.innerHTML = Markdown.render(content);

        div.appendChild(bubble);
        messages.appendChild(div);
        this.scrollToBottom();
    },

    appendToolCall(data) {
        const messages = document.getElementById('messages');
        const div = document.createElement('div');
        div.className = 'tool-indicator';
        div.innerHTML = '🔧 Calling <span class="tool-name">' +
            this.escapeHtml(data.name) + '</span>...' +
            '<div class="tool-details">' + this.escapeHtml(JSON.stringify(data.args, null, 2)) + '</div>';
        div.addEventListener('click', () => div.classList.toggle('expanded'));
        messages.appendChild(div);
        this.scrollToBottom();
    },

    appendToolResult(data) {
        const messages = document.getElementById('messages');
        const div = document.createElement('div');
        div.className = 'tool-indicator';
        const label = data.error ? '❌ Error' : '✅ Result';
        const detail = data.error || JSON.stringify(data.data, null, 2);
        div.innerHTML = label + ': <span class="tool-name">' +
            this.escapeHtml(data.name) + '</span>' +
            '<div class="tool-details">' + this.escapeHtml(detail) + '</div>';
        div.addEventListener('click', () => div.classList.toggle('expanded'));
        messages.appendChild(div);
        this.scrollToBottom();
    },

    showThinking() {
        const messages = document.getElementById('messages');
        const div = document.createElement('div');
        div.className = 'thinking';
        div.innerHTML = '<div class="thinking-dots"><span></span><span></span><span></span></div> Thinking...';
        messages.appendChild(div);
        this.scrollToBottom();
        return div;
    },

    hideWelcome() {
        const welcome = document.getElementById('welcome');
        if (welcome) welcome.style.display = 'none';
    },

    showWelcome() {
        const welcome = document.getElementById('welcome');
        if (welcome) welcome.style.display = '';
    },

    scrollToBottom() {
        const messages = document.getElementById('messages');
        messages.scrollTop = messages.scrollHeight;
    },

    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }
};

// Initialize on DOM load
document.addEventListener('DOMContentLoaded', () => {
    Chat.init();
    Sidebar.init();
});

// Single delegated listener for approval card buttons built by showApprovalCard.
// Installed once at module load; the buttons carry data-action and data-approval-id.
document.addEventListener('click', function (e) {
    const target = e.target.closest('[data-action]');
    if (!target) return;
    const approvalId = target.dataset.approvalId;
    if (!approvalId) return;
    if (target.dataset.action === 'approve') {
        Chat.approve(approvalId);
    } else if (target.dataset.action === 'reject') {
        Chat.reject(approvalId);
    }
});
