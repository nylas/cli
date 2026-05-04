/* Email Helpers - Keyboard navigation and batch actions */

// ====================================

let selectedEmails = new Set();

// Virtual scrolling config
const ITEM_HEIGHT = 80;
const BUFFER_SIZE = 5;

// Get email items
function getEmailItems() {
    return document.querySelectorAll('.email-item');
}

// Resolve the current keyboard cursor from the DOM rather than a module
// global. The global drifted after archive/delete (the list shrinks but
// the index doesn't), so arrow keys could target a removed element. The
// keydown handler below already reads from DOM — these wrappers now
// match its policy so all keyboard nav agrees on a single source of
// truth.
function currentEmailItemIndex(items) {
    const list = items || getEmailItems();
    for (let i = 0; i < list.length; i++) {
        if (list[i].classList.contains('selected') || list[i].classList.contains('focused')) {
            return i;
        }
    }
    return -1;
}

// Select next email
function selectNextEmail() {
    const emailItems = getEmailItems();
    if (emailItems.length === 0) return;
    const current = currentEmailItemIndex(emailItems);
    const next = current < 0 ? 0 : current + 1;
    if (next >= emailItems.length) return;
    if (current >= 0) emailItems[current]?.classList.remove('selected');
    emailItems[next].classList.add('selected');
    emailItems[next].scrollIntoView({ behavior: 'smooth', block: 'nearest' });
    if (typeof announce === 'function') {
        announce(`Email ${next + 1} of ${emailItems.length}`);
    }
}

// Select previous email
function selectPrevEmail() {
    const emailItems = getEmailItems();
    if (emailItems.length === 0) return;
    const current = currentEmailItemIndex(emailItems);
    const prev = current <= 0 ? -1 : current - 1;
    if (prev < 0) return;
    emailItems[current]?.classList.remove('selected');
    emailItems[prev].classList.add('selected');
    emailItems[prev].scrollIntoView({ behavior: 'smooth', block: 'nearest' });
    if (typeof announce === 'function') {
        announce(`Email ${prev + 1} of ${emailItems.length}`);
    }
}

// Toggle email selection (for batch operations)
function toggleEmailSelection(index) {
    const emailItems = getEmailItems();
    if (selectedEmails.has(index)) {
        selectedEmails.delete(index);
        emailItems[index]?.classList.remove('batch-selected');
    } else {
        selectedEmails.add(index);
        emailItems[index]?.classList.add('batch-selected');
    }
    updateBatchActionsUI();
}

// Clear all selections
function clearEmailSelections() {
    const emailItems = getEmailItems();
    selectedEmails.clear();
    emailItems.forEach(item => item.classList.remove('batch-selected'));
    updateBatchActionsUI();
}

// Update batch actions UI
function updateBatchActionsUI() {
    const count = selectedEmails.size;
    const batchBar = document.getElementById('batchActionsBar');
    if (batchBar) {
        batchBar.style.display = count > 0 ? 'flex' : 'none';
        const countEl = batchBar.querySelector('.batch-count');
        if (countEl) countEl.textContent = `${count} selected`;
    }
}

// Note: previous versions of this file shipped five "fake action" helpers
// (archiveSelected, deleteSelected, markSelectedAsRead, sendEmail,
// generateAISummary) that showed success toasts without making any API
// calls. None of them were referenced from templates or other modules —
// they were dead code that misled audits — so they have been removed.
// The real implementations live in email-actions.js (CRUD + read/star)
// and email-ai.js (summarize) and run through AirAPI / fetch.

// Initialize email keyboard navigation
function initEmailKeyboard() {
    const emailList = document.querySelector('.email-list');
    if (!emailList) return;

    emailList.setAttribute('role', 'listbox');
    emailList.setAttribute('aria-label', 'Email messages');
    emailList.setAttribute('tabindex', '0');

    const items = emailList.querySelectorAll('.email-item');
    items.forEach((item, index) => {
        item.setAttribute('role', 'option');
        item.setAttribute('tabindex', '-1');
        item.setAttribute('aria-selected', item.classList.contains('selected') ? 'true' : 'false');
    });

    emailList.addEventListener('keydown', async function(e) {
        const items = emailList.querySelectorAll('.email-item');
        const currentIndex = Array.from(items).findIndex(item =>
            item.classList.contains('focused') || item.classList.contains('selected')
        );

        switch(e.key) {
            case 'ArrowDown':
                e.preventDefault();
                if (currentIndex < items.length - 1) {
                    items[currentIndex]?.classList.remove('focused');
                    items[currentIndex + 1].classList.add('focused');
                    items[currentIndex + 1].focus();
                    if (typeof announce === 'function') {
                        announce(`Email ${currentIndex + 2} of ${items.length}`);
                    }
                }
                break;
            case 'ArrowUp':
                e.preventDefault();
                if (currentIndex > 0) {
                    items[currentIndex]?.classList.remove('focused');
                    items[currentIndex - 1].classList.add('focused');
                    items[currentIndex - 1].focus();
                    if (typeof announce === 'function') {
                        announce(`Email ${currentIndex} of ${items.length}`);
                    }
                }
                break;
            case 'Enter':
            case ' ':
                e.preventDefault();
                items[currentIndex]?.click();
                if (typeof announce === 'function') {
                    announce('Email opened');
                }
                break;
            case 'Delete':
            case 'Backspace':
                e.preventDefault();
                // Route through EmailListManager so the keystroke
                // matches the click-Delete behaviour: optimistic
                // removal, real fetch, server-side persistence.
                //
                // Await the deletion before announcing — the previous
                // version fired the screen-reader announcement
                // synchronously, so users heard "Email deleted" even
                // when the underlying fetch failed and the email was
                // rolled back into the list. deleteEmail returns true
                // on a successful round-trip, false on rollback.
                {
                    const focused = items[currentIndex];
                    const id = focused && focused.dataset && focused.dataset.emailId;
                    if (id && typeof EmailListManager !== 'undefined' &&
                        typeof EmailListManager.deleteEmail === 'function') {
                        const ok = await EmailListManager.deleteEmail(id);
                        if (ok && typeof announce === 'function') {
                            announce('Email deleted');
                        } else if (!ok && typeof announce === 'function') {
                            announce('Delete failed');
                        }
                    }
                }
                break;
            case 'x':
                e.preventDefault();
                toggleEmailSelection(currentIndex);
                break;
        }
    });
}

// Initialize email module
document.addEventListener('DOMContentLoaded', () => {
    // Init keyboard navigation
    initEmailKeyboard();

    // Init email list manager if we have the email list element
    if (document.querySelector('.email-list')) {
        EmailListManager.init();
    }
});
