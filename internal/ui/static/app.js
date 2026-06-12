// =============================================================================
// Nylas CLI - Dashboard (Main Entry Point)
// =============================================================================

// readInitialState parses the server-rendered <script type="application/json">
// data block (CSP-safe replacement for an inline executable script).
function readInitialState() {
    const el = document.getElementById('initial-state');
    if (!el) return null;
    try {
        return JSON.parse(el.textContent);
    } catch (err) {
        console.error('Failed to parse initial state');
        return null;
    }
}

// Initialize application when DOM is ready
document.addEventListener('DOMContentLoaded', () => {
    // Initialize all modules
    initTheme();
    initForm();
    initDropdowns();
    initNavigation();
    initKeyboardShortcuts();
    initToast();

    // Use server-provided initial state (hybrid SSR)
    const initialState = readInitialState();
    if (initialState) {
        initFromServerState(initialState);
    } else {
        // Fallback to API call if no initial state
        checkConfig();
    }

    // Update timestamps every minute
    setInterval(updateAllTimestamps, 60000);
});
