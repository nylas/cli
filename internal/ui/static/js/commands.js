// =============================================================================
// Command System - Main Entry Point and Helpers
// =============================================================================

// Helper functions for output formatting (used by all command modules)

/**
 * Set button to loading state.
 * @param {HTMLElement} btn - The button element
 * @param {boolean} loading - Whether to show loading state
 */
function setButtonLoading(btn, loading) {
    if (loading) {
        btn.classList.add('loading');
        btn.textContent = 'Running...';
    } else {
        btn.classList.remove('loading');
        // Reset to Run button with play icon
        btn.textContent = '';
        const svg = document.createElementNS('http://www.w3.org/2000/svg', 'svg');
        svg.setAttribute('viewBox', '0 0 24 24');
        svg.setAttribute('fill', 'none');
        svg.setAttribute('stroke', 'currentColor');
        svg.setAttribute('stroke-width', '2');
        const polygon = document.createElementNS('http://www.w3.org/2000/svg', 'polygon');
        polygon.setAttribute('points', '5 3 19 12 5 21 5 3');
        svg.appendChild(polygon);
        btn.appendChild(svg);
        btn.appendChild(document.createTextNode(' Run'));
    }
}

/**
 * Set output to loading state.
 * @param {HTMLElement} output - The output element
 */
function setOutputLoading(output) {
    output.textContent = 'Running command...';
    output.className = 'output-pre loading';
}

/**
 * Set output to error state.
 * @param {HTMLElement} output - The output element
 * @param {string} message - The error message
 */
function setOutputError(output, message) {
    output.textContent = message;
    output.className = 'output-pre error';
}

/**
 * Set output to success state with formatted content.
 * Uses formatOutput which returns HTML for ANSI colors and table formatting.
 * Content is escaped via esc() before processing.
 * @param {HTMLElement} output - The output element
 * @param {string} content - The output content
 */
function setOutputSuccess(output, content) {
    if (!content) {
        output.textContent = 'Command completed successfully.';
    } else {
        // formatOutput returns safe HTML (content is escaped via esc())
        const formatted = formatOutput(content);
        if (formatted) {
            // Safe: formatOutput escapes content via esc() before processing
            output.innerHTML = formatted;
        } else {
            output.textContent = 'Command completed successfully.';
        }
    }
    output.className = 'output-pre';
}

// =============================================================================
// Initialization
// =============================================================================

document.addEventListener('DOMContentLoaded', () => {
    // Render all command sections
    renderAuthCommands();
    renderEmailCommands();
    renderCalendarCommands();
    renderContactsCommands();
    renderSchedulerCommands();
    renderTimezoneCommands();
    renderWebhookCommands();
    renderOtpCommands();
    renderAdminCommands();
    renderNotetakerCommands();
});
