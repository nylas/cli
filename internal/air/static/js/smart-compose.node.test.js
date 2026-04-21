const test = require('node:test');
const assert = require('node:assert/strict');

const SmartCompose = require('./smart-compose.js');

function resetSmartCompose() {
    SmartCompose.config.enabled = true;
    SmartCompose.state.isActive = false;
    SmartCompose.state.currentSuggestion = '';
    SmartCompose.state.lastText = '';
    SmartCompose.state.debounceTimer = null;
    SmartCompose.state.textarea = {
        value: '',
        dispatchEvent() {},
    };
    SmartCompose.state.abortController = null;
    SmartCompose.state.requestId = 0;
    SmartCompose.overlay = {
        classList: {
            add() {},
            remove() {},
        },
    };
    SmartCompose.ghostEl = { textContent: '' };
    SmartCompose.suggestionEl = { textContent: '' };
}

test.beforeEach(() => {
    resetSmartCompose();
});

test.afterEach(() => {
    delete global.fetch;
});

test('fetchSuggestion aborts the previous request and keeps the latest suggestion', async () => {
    let firstRequestAborted = false;
    let callCount = 0;

    global.fetch = (url, init) => {
        callCount += 1;
        if (callCount === 1) {
            return new Promise((resolve, reject) => {
                init.signal.addEventListener('abort', () => {
                    firstRequestAborted = true;
                    const err = new Error('aborted');
                    err.name = 'AbortError';
                    reject(err);
                });
            });
        }

        return Promise.resolve({
            ok: true,
            json: async () => ({ suggestion: ' latest suggestion' }),
        });
    };

    SmartCompose.state.textarea.value = 'drafting a long enough message';
    const first = SmartCompose.fetchSuggestion('drafting a long enough message');

    SmartCompose.state.textarea.value = 'drafting a longer replacement message';
    const second = SmartCompose.fetchSuggestion('drafting a longer replacement message');

    await Promise.all([first, second]);

    assert.equal(firstRequestAborted, true);
    assert.equal(SmartCompose.state.currentSuggestion, ' latest suggestion');
    assert.equal(SmartCompose.suggestionEl.textContent, ' latest suggestion');
});

test('fetchSuggestion ignores a response for stale textarea content', async () => {
    global.fetch = async () => ({
        ok: true,
        json: async () => ({ suggestion: ' stale suggestion' }),
    });

    SmartCompose.state.textarea.value = 'drafting a long enough message';
    const request = SmartCompose.fetchSuggestion('drafting a long enough message');
    SmartCompose.state.textarea.value = 'edited before response returned';

    await request;

    assert.equal(SmartCompose.state.currentSuggestion, '');
    assert.equal(SmartCompose.suggestionEl.textContent, '');
});

test('clearSuggestion invalidates a late response from an already-cleared request', async () => {
    let resolveFetch;

    global.fetch = () => new Promise((resolve) => {
        resolveFetch = resolve;
    });

    SmartCompose.state.textarea.value = 'drafting a long enough message';
    const request = SmartCompose.fetchSuggestion('drafting a long enough message');

    SmartCompose.clearSuggestion();

    resolveFetch({
        ok: true,
        json: async () => ({ suggestion: ' should stay hidden' }),
    });

    await request;

    assert.equal(SmartCompose.state.currentSuggestion, '');
    assert.equal(SmartCompose.suggestionEl.textContent, '');
});

test('clearSuggestion cancels a queued debounce before fetch starts', () => {
    const originalSetTimeout = global.setTimeout;
    const originalClearTimeout = global.clearTimeout;

    let scheduled = null;
    let clearedToken = null;
    let fetchCalled = false;

    global.setTimeout = (fn) => {
        scheduled = fn;
        return 42;
    };
    global.clearTimeout = (token) => {
        clearedToken = token;
        scheduled = null;
    };
    global.fetch = async () => {
        fetchCalled = true;
        return {
            ok: true,
            json: async () => ({ suggestion: ' should not run' }),
        };
    };

    try {
        SmartCompose.state.textarea.value = 'drafting a long enough message';
        SmartCompose.handleInput();
        SmartCompose.clearSuggestion();

        assert.equal(clearedToken, 42);
        assert.equal(scheduled, null);
        assert.equal(fetchCalled, false);
    } finally {
        global.setTimeout = originalSetTimeout;
        global.clearTimeout = originalClearTimeout;
    }
});

test('handleInput aborts an in-flight request immediately while typing', async () => {
    const originalSetTimeout = global.setTimeout;
    const originalClearTimeout = global.clearTimeout;

    let aborted = false;
    let scheduled = null;

    global.setTimeout = (fn) => {
        scheduled = fn;
        return 99;
    };
    global.clearTimeout = () => {};
    global.fetch = (url, init) => new Promise((resolve, reject) => {
        init.signal.addEventListener('abort', () => {
            aborted = true;
            const err = new Error('aborted');
            err.name = 'AbortError';
            reject(err);
        });
    });

    try {
        SmartCompose.state.textarea.value = 'drafting a long enough message';
        const request = SmartCompose.fetchSuggestion('drafting a long enough message');

        SmartCompose.state.textarea.value = 'drafting a slightly longer message';
        SmartCompose.handleInput();

        await request;

        assert.equal(aborted, true);
        assert.equal(typeof scheduled, 'function');
    } finally {
        global.setTimeout = originalSetTimeout;
        global.clearTimeout = originalClearTimeout;
    }
});
