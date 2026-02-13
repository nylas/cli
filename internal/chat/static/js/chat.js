// chat.js — Main chat UI
const Chat = {
    conversationId: null,
    sending: false,

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

        // Enter to send, Shift+Enter for newline
        input.addEventListener('keydown', (e) => {
            if (e.key === 'Enter' && !e.shiftKey) {
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
                    case 'tool_call':
                        this.appendToolCall(data);
                        break;
                    case 'tool_result':
                        this.appendToolResult(data);
                        break;
                    case 'message':
                        thinkingEl.remove();
                        this.appendMessage('assistant', data.content);
                        break;
                    case 'error':
                        thinkingEl.remove();
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
            this.appendMessage('assistant', 'Connection error: ' + err.message);
        }

        this.sending = false;
        document.getElementById('btn-send').disabled = false;
        input.focus();
    },

    loadConversation(conv) {
        this.conversationId = conv.id;
        const messages = document.getElementById('messages');
        messages.innerHTML = '';
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
        messages.innerHTML = '';
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
