// markdown.js — Lightweight markdown renderer
const Markdown = {
    render(text) {
        if (!text) return '';
        let html = this.escape(text);

        // Code blocks
        html = html.replace(/```(\w*)\n([\s\S]*?)```/g, '<pre><code class="lang-$1">$2</code></pre>');

        // Inline code
        html = html.replace(/`([^`]+)`/g, '<code>$1</code>');

        // Bold
        html = html.replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>');

        // Italic
        html = html.replace(/\*(.+?)\*/g, '<em>$1</em>');

        // Headers
        html = html.replace(/^### (.+)$/gm, '<h4>$1</h4>');
        html = html.replace(/^## (.+)$/gm, '<h3>$1</h3>');
        html = html.replace(/^# (.+)$/gm, '<h2>$1</h2>');

        // Unordered lists
        html = html.replace(/^[\s]*[-*] (.+)$/gm, '<li>$1</li>');
        html = html.replace(/(<li>.*<\/li>\n?)+/g, '<ul>$&</ul>');

        // Ordered lists
        html = html.replace(/^\d+\. (.+)$/gm, '<li>$1</li>');

        // Links — scheme-validate the URL so attacker-controlled markdown
        // coming from agent output cannot inject `javascript:` URLs. The URL
        // is already HTML-escaped by escape() above (the regex runs against
        // the escaped html), so it is already safe to drop into a
        // double-quoted attribute.
        html = html.replace(
            /\[([^\]]+)\]\(([^)]+)\)/g,
            (_, label, url) => {
                const safe = Markdown.safeUrl(url);
                return '<a href="' + safe + '" target="_blank" rel="noopener noreferrer">' + label + '</a>';
            }
        );

        // Paragraphs
        html = html.replace(/\n\n/g, '</p><p>');
        html = '<p>' + html + '</p>';
        html = html.replace(/<p>\s*<(h[234]|ul|ol|pre|li)/g, '<$1');
        html = html.replace(/<\/(h[234]|ul|ol|pre|li)>\s*<\/p>/g, '</$1>');

        // Line breaks
        html = html.replace(/\n/g, '<br>');

        return html;
    },

    escape(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    },

    // safeUrl returns the URL if it uses an http(s) or mailto: scheme; otherwise
    // it returns "#". Relative URLs (no scheme) and anchor links pass through.
    // This blocks javascript:, data:, vbscript: and similar dangerous schemes.
    safeUrl(rawUrl) {
        const url = String(rawUrl).trim();
        if (url === '') return '#';
        // Anchor / relative path / explicit scheme.
        const schemeMatch = url.match(/^([a-z][a-z0-9+.-]*):/i);
        if (!schemeMatch) {
            // No scheme — relative or anchor link, allow.
            return url;
        }
        const scheme = schemeMatch[1].toLowerCase();
        if (scheme === 'http' || scheme === 'https' || scheme === 'mailto') {
            return url;
        }
        return '#';
    },

};
