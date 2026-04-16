/**
 * App Keyboard - Primary keyboard shortcuts
 */
        // Keyboard Shortcuts
        document.addEventListener('keydown', function(e) {
            if ((e.ctrlKey || e.metaKey) && e.key === 'k') {
                e.preventDefault();
                toggleCommandPalette();
            }
            if (e.key === 'Escape') {
                document.getElementById('commandPalette').classList.add('hidden');
                document.getElementById('composeModal').classList.add('hidden');
            }
            // Skip shortcuts when modifier keys are pressed (allow Cmd+R refresh, etc.)
            if (!e.target.matches('input, textarea, [contenteditable]') && !e.metaKey && !e.ctrlKey && !e.altKey) {
                if (e.key === 'c') { e.preventDefault(); toggleCompose(); }
                const shortcutTab = document.querySelector(`.nav-tab[data-shortcut="${e.key}"]`);
                if (shortcutTab) {
                    e.preventDefault();
                    shortcutTab.click();
                    return;
                }
                if (e.key === 'e') { showToast('success', 'Archived', 'Moved to archive'); }
                if (e.key === 'r') {
                    // Reply to selected email
                    if (typeof EmailListManager !== 'undefined' && EmailListManager.selectedEmailFull) {
                        if (typeof ComposeManager !== 'undefined') {
                            ComposeManager.openReply(EmailListManager.selectedEmailFull);
                        }
                    } else {
                        showToast('info', 'No email selected', 'Select an email first to reply');
                    }
                }
                if (e.key === 's') { showToast('info', 'Starred', 'Conversation starred'); }
                if (e.key === '#') { showToast('warning', 'Deleted', 'Moved to trash'); }
                if (e.key === 'j') { selectNextEmail(); }
                if (e.key === 'k') { selectPrevEmail(); }
            }
            // Send email: Cmd+Enter (handled by ComposeManager in api.js)
        });

        // NOTE: Email navigation (selectNextEmail, selectPrevEmail, sendEmail)
        // is defined in js/email.js

        // Demo: Show toast on page load
        setTimeout(() => {
            showToast('info', 'Welcome back!', '3 new messages since you left');
        }, 1500);
