const test = require('node:test');
const assert = require('node:assert/strict');

const EmailFolders = require('./email-folders.js');

function escapeHtml(value) {
    return String(value)
        .replaceAll('&', '&amp;')
        .replaceAll('<', '&lt;')
        .replaceAll('>', '&gt;')
        .replaceAll('"', '&quot;')
        .replaceAll("'", '&#39;');
}

function createElement(tagName = 'div') {
    const element = {
        tagName,
        children: [],
        attributes: {},
        className: '',
        _innerHTML: '',
        _textContent: '',
        appendChild(child) {
            this.children.push(child);
            return child;
        },
        setAttribute(name, value) {
            this.attributes[name] = String(value);
        },
        getAttribute(name) {
            return this.attributes[name] || null;
        },
        removeAttribute(name) {
            delete this.attributes[name];
        },
    };

    Object.defineProperty(element, 'innerHTML', {
        get() {
            return this._innerHTML;
        },
        set(value) {
            this._innerHTML = String(value);
            if (value === '') {
                this.children = [];
            }
        },
    });

    Object.defineProperty(element, 'textContent', {
        get() {
            return this._textContent;
        },
        set(value) {
            this._textContent = String(value);
            this._innerHTML = escapeHtml(value);
        },
    });

    return element;
}

function createDocument(folderList) {
    return {
        getElementById(id) {
            return id === 'folderList' ? folderList : null;
        },
        querySelector(selector) {
            return selector === '.folder-group' ? folderList : null;
        },
        createElement() {
            return createElement();
        },
    };
}

test.afterEach(() => {
    delete global.document;
    EmailFolders.currentFolder = 'INBOX';
});

test('getVisibleFolders keeps archive and junk visible while filtering provider pseudo-folders', () => {
    const folders = [
        { id: 'CATEGORY_SOCIAL', name: 'Social' },
        { id: 'UNREAD', name: 'Unread' },
        { id: 'scheduled', name: 'Scheduled' },
        { id: 'outbox', name: 'Outbox' },
        { id: 'projects', name: 'Projects' },
        { id: 'archive', name: 'Archive' },
        { id: 'junk', name: 'Junk' },
        { id: 'trash', name: 'Trash' },
        { id: 'drafts', name: 'Drafts' },
        { id: 'sent', name: 'Sent' },
        { id: 'starred', name: 'Starred' },
        { id: 'inbox', name: 'Inbox' },
    ];

    const visibleNames = EmailFolders.getVisibleFolders(folders).map((folder) => folder.name);

    assert.deepEqual(visibleNames, [
        'Inbox',
        'Starred',
        'Sent',
        'Drafts',
        'Archive',
        'Trash',
        'Junk',
        'Projects',
    ]);
});

test('renderFolders renders all visible folders inline without a More entry', () => {
    const folderList = createElement('div');
    global.document = createDocument(folderList);
    EmailFolders.currentFolder = 'archive';

    EmailFolders.renderFolders([
        { id: 'inbox', name: 'Inbox', unread_count: 3 },
        { id: 'archive', name: 'Archive', total_count: 10 },
        { id: 'projects', name: 'Projects', total_count: 4 },
        { id: 'scheduled', name: 'Scheduled', total_count: 1 },
    ]);

    const renderedNames = folderList.children.map((child) => child.getAttribute('data-folder-name'));

    assert.deepEqual(renderedNames, ['Inbox', 'Archive', 'Projects']);
    assert.equal(renderedNames.includes('More'), false);
    assert.equal(folderList.children[1].className.includes('active'), true);
});
