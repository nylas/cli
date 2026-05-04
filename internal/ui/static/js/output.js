// =============================================================================
// Output Formatting (ANSI parsing, table parsing)
// =============================================================================
//
// formatOutput / parseTable / parseAnsi return DOM nodes (or null), never
// HTML strings. Every cell, header, and text run is set via textContent
// so the call site can always use replaceChildren without needing to
// think about escaping. Mirrors the Air UI's "interpolation goes through
// textContent" doctrine.

// Table Parser - converts CLI table output to a <table> element, or
// null when the input doesn't look like a table.
function parseTable(text) {
    if (!text) return null;

    const lines = text.trim().split('\n');
    if (lines.length < 2) return null;

    const headerLine = lines[0].trim();
    const tablePatterns = [
        /^\s*(GRANT\s*ID|ID|EMAIL|NAME|SUBJECT|TITLE|CALENDAR)/i,
        /^\s*\w+\s{2,}\w+/
    ];

    const looksLikeTable = tablePatterns.some(p => p.test(headerLine));
    if (!looksLikeTable) return null;

    const headerParts = headerLine.split(/\s{2,}/).filter(h => h.trim());
    if (headerParts.length < 2) return null;

    const colPositions = [];
    let pos = 0;
    for (const header of headerParts) {
        const idx = headerLine.indexOf(header, pos);
        colPositions.push(idx);
        pos = idx + header.length;
    }

    const rows = [];
    for (let i = 1; i < lines.length; i++) {
        const line = lines[i];
        if (!line.trim()) continue;

        const cells = [];
        for (let j = 0; j < colPositions.length; j++) {
            const start = colPositions[j];
            const end = j < colPositions.length - 1 ? colPositions[j + 1] : line.length;
            const cell = line.substring(start, end).trim();
            cells.push(cell);
        }

        if (cells.some(c => c)) {
            rows.push(cells);
        }
    }

    if (rows.length === 0) return null;

    const table = document.createElement('table');
    table.className = 'formatted-table';

    const thead = document.createElement('thead');
    const headerRow = document.createElement('tr');
    for (const header of headerParts) {
        const th = document.createElement('th');
        th.textContent = header;
        headerRow.appendChild(th);
    }
    thead.appendChild(headerRow);
    table.appendChild(thead);

    const tbody = document.createElement('tbody');
    for (const row of rows) {
        const tr = document.createElement('tr');
        for (let i = 0; i < headerParts.length; i++) {
            const cell = row[i] || '';
            const headerLower = headerParts[i].toLowerCase();
            const td = document.createElement('td');
            if (headerLower.includes('id') || headerLower.includes('grant')) {
                td.className = 'cell-id';
            } else if (headerLower.includes('email')) {
                td.className = 'cell-email';
            } else if (cell === '✓' || cell === '✔') {
                td.className = 'cell-check';
            }
            td.textContent = cell;
            tr.appendChild(td);
        }
        tbody.appendChild(tr);
    }
    table.appendChild(tbody);
    return table;
}

// formatOutput - try table first, then ANSI. Returns a Node (table or
// DocumentFragment) or null when there is no content to render.
function formatOutput(text) {
    if (!text) return null;

    const table = parseTable(text);
    if (table) {
        return table;
    }

    return parseAnsi(text);
}

// ANSI class mapping. Each entry is the digits between CSI `[` and `m`.
const ANSI_CLASS_BY_CODE = {
    '1': 'ansi-bold',
    '2': 'ansi-dim',
    '4': 'ansi-underline',
    '30': 'ansi-gray', '90': 'ansi-gray',
    '31': 'ansi-red', '91': 'ansi-red',
    '32': 'ansi-green', '92': 'ansi-green',
    '33': 'ansi-yellow', '93': 'ansi-yellow',
    '34': 'ansi-blue', '94': 'ansi-blue',
    '35': 'ansi-magenta', '95': 'ansi-magenta',
    '36': 'ansi-cyan', '96': 'ansi-cyan',
    '37': 'ansi-white', '97': 'ansi-white',
    '1;32': 'ansi-bold ansi-green',
    '1;31': 'ansi-bold ansi-red',
    '1;33': 'ansi-bold ansi-yellow',
    '1;34': 'ansi-bold ansi-blue',
    '1;36': 'ansi-bold ansi-cyan',
};

// Matches a real ANSI CSI SGR sequence plus the two literal-escape forms
// that occasionally arrive after JSON / HTML round-trips.
const ANSI_SEQUENCE_RE = /\x1b\[([0-9;]*)m|\\x1b\[([0-9;]*)m|&#x1b;\[([0-9;]*)m/g;

// parseAnsi walks the input and returns a DocumentFragment whose text
// runs are plain text nodes wrapped in <span class="ansi-..."> for each
// active style. Resets pop one level off the style stack — matches the
// original string parser's per-reset behavior. Codes outside the
// allow-list (and stray escape forms) are silently dropped.
function parseAnsi(text) {
    const fragment = document.createDocumentFragment();
    if (!text) return fragment;

    const stack = [fragment];
    let lastIndex = 0;
    let match;

    ANSI_SEQUENCE_RE.lastIndex = 0;
    while ((match = ANSI_SEQUENCE_RE.exec(text)) !== null) {
        const parent = stack[stack.length - 1];

        if (match.index > lastIndex) {
            parent.appendChild(document.createTextNode(text.substring(lastIndex, match.index)));
        }

        const code = match[1] ?? match[2] ?? match[3] ?? '';
        if (code === '' || code === '0') {
            if (stack.length > 1) {
                stack.pop();
            }
        } else {
            const className = ANSI_CLASS_BY_CODE[code];
            if (className) {
                const span = document.createElement('span');
                span.className = className;
                parent.appendChild(span);
                stack.push(span);
            }
        }

        lastIndex = ANSI_SEQUENCE_RE.lastIndex;
    }

    if (lastIndex < text.length) {
        stack[stack.length - 1].appendChild(document.createTextNode(text.substring(lastIndex)));
    }

    return fragment;
}
