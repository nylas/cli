// =============================================================================
// Delegated Click Actions (CSP-safe replacement for inline onclick handlers)
// =============================================================================
//
// Inline `onclick="..."` attributes are blocked by the strict Content
// Security Policy (script-src 'self', no 'unsafe-inline'). Clickable elements
// instead carry a `data-action` attribute (plus data-* parameters) handled by
// the single delegated listener below.

const UI_ACTIONS = {
    'run-cmd': (el) => invokeSectionCmd('run', el.dataset.section),
    'refresh-cmd': (el) => invokeSectionCmd('refresh', el.dataset.section),
    'copy-output': (el) => copyOutput(el.dataset.section, el),
    'clear-cache': () => clearCacheAndNotify(),
    'copy-text': (el) => copyText(el.dataset.copyText, el),
    'set-default-grant': (el) => setDefault(el.dataset.grantId),
    'select-grant': (el) => selectGrant(el.dataset.grant, el.dataset.email),
    'select-account': (el) => selectAccount(el.dataset.grantId),
    'show-add-account': () => showAddAccount(),
    'run-command': (el) => runCommand(el.dataset.command),
    'toggle-dropdown': (el) => toggleDropdown(el.dataset.target),
    'toggle-theme': () => toggleTheme(),
    'toggle-password': () => togglePassword(),
    'toggle-flags-panel': (el) => toggleFlagsPanel(el.dataset.section),
};

// invokeSectionCmd dispatches to the per-section run/refresh functions
// (runOtpCmd, refreshEmailCmd, ...) defined in commands-*.js.
function invokeSectionCmd(kind, section) {
    if (!section) return;
    const name = kind + section.charAt(0).toUpperCase() + section.slice(1) + 'Cmd';
    const fn = window[name];
    if (typeof fn === 'function') fn();
}

document.addEventListener('click', (e) => {
    const el = e.target.closest('[data-action]');
    if (!el) return;
    const handler = UI_ACTIONS[el.dataset.action];
    if (handler) handler(el);
});

if (typeof module !== 'undefined' && module.exports) {
    module.exports = {
        UI_ACTIONS,
    };
}
