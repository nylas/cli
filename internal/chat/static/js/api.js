// api.js — Chat API client
const ChatAPI = {
    async sendMessage(convId, message, onEvent) {
        const body = { message };
        if (convId) body.conversation_id = convId;

        const resp = await fetch('/api/chat', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(body),
        });

        if (!resp.ok) throw new Error(await resp.text());

        const reader = resp.body.getReader();
        const decoder = new TextDecoder();
        let buffer = '';

        while (true) {
            const { done, value } = await reader.read();
            if (done) break;

            buffer += decoder.decode(value, { stream: true });
            const lines = buffer.split('\n');
            buffer = lines.pop() || '';

            let currentEvent = '';
            for (const line of lines) {
                if (line.startsWith('event: ')) {
                    currentEvent = line.slice(7);
                } else if (line.startsWith('data: ') && currentEvent) {
                    try {
                        const data = JSON.parse(line.slice(6));
                        onEvent(currentEvent, data);
                    } catch { /* skip malformed */ }
                    currentEvent = '';
                }
            }
        }
    },

    async listConversations() {
        const resp = await fetch('/api/conversations');
        return resp.json();
    },

    async createConversation() {
        const resp = await fetch('/api/conversations', { method: 'POST' });
        return resp.json();
    },

    async getConversation(id) {
        const resp = await fetch('/api/conversations/' + id);
        return resp.json();
    },

    async deleteConversation(id) {
        const resp = await fetch('/api/conversations/' + id, { method: 'DELETE' });
        return resp.json();
    },

    async getConfig() {
        const resp = await fetch('/api/config');
        return resp.json();
    },

    async switchAgent(agent) {
        const resp = await fetch('/api/config', {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ agent }),
        });
        if (!resp.ok) throw new Error(await resp.text());
        return resp.json();
    },
};
