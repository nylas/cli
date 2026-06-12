/**
 * Safe DOM builders — all text goes through textContent, never innerHTML,
 * so resource names/IDs from the API can't inject markup.
 */
window.StudioDOM = {
    el(tag, className, text) {
        const node = document.createElement(tag);
        if (className) {
            node.className = className;
        }
        if (text !== undefined && text !== null && text !== '') {
            node.textContent = String(text);
        }
        return node;
    },

    add(parent, ...children) {
        for (const child of children) {
            if (child) {
                parent.appendChild(child);
            }
        }
        return parent;
    },

    clear(node) {
        while (node.firstChild) {
            node.removeChild(node.firstChild);
        }
        return node;
    },

    kv(label, value, mono) {
        const row = this.el('div', 'kv');
        this.add(row, this.el('span', '', label));
        const v = this.el('span', mono ? 'v mono' : 'v', value);
        this.add(row, v);
        return row;
    }
};
