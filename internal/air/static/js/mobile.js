// ====================================
// MOBILE MODULE
// ====================================

// Mobile state
let touchStartX = 0;
let touchStartY = 0;
let currentSwipeItem = null;
let pullStartY = 0;
let isPulling = false;

// Check if mobile
function isMobile() {
    return window.innerWidth <= 768;
}

// Toggle mobile sidebar
function toggleMobileSidebar() {
    const sidebar = document.querySelector('.sidebar');
    const overlay = document.getElementById('sidebarOverlay');

    if (sidebar && overlay) {
        sidebar.classList.toggle('open');
        overlay.classList.toggle('active');
    }
}

// Close mobile sidebar
function closeMobileSidebar() {
    const sidebar = document.querySelector('.sidebar');
    const overlay = document.getElementById('sidebarOverlay');

    if (sidebar && overlay) {
        sidebar.classList.remove('open');
        overlay.classList.remove('active');
    }
}

// Open email preview (mobile)
function openMobilePreview() {
    const preview = document.querySelector('.preview-pane');
    if (preview && isMobile()) {
        preview.classList.add('open');
    }
}

// Close email preview (mobile)
function closeMobilePreview() {
    const preview = document.querySelector('.preview-pane');
    if (preview) {
        preview.classList.remove('open');
    }
}

// Update mobile nav active state
function updateMobileNavActive(view) {
    document.querySelectorAll('.mobile-nav-item').forEach(item => {
        item.classList.remove('active');
    });

    const viewMap = {
        'email': 1,
        'calendar': 2,
        'contacts': 3
    };

    const index = viewMap[view];
    if (index !== undefined) {
        const items = document.querySelectorAll('.mobile-nav-item');
        items[index]?.classList.add('active');
    }
}

// Initialize swipe gestures
function initSwipeGestures() {
    const emailList = document.querySelector('.email-list');
    if (!emailList) return;

    emailList.addEventListener('touchstart', handleTouchStart, { passive: true });
    emailList.addEventListener('touchmove', handleTouchMove, { passive: false });
    emailList.addEventListener('touchend', handleTouchEnd, { passive: true });
}

function handleTouchStart(e) {
    const emailItem = e.target.closest('.email-item');
    if (!emailItem) return;

    touchStartX = e.touches[0].clientX;
    touchStartY = e.touches[0].clientY;
    currentSwipeItem = emailItem;
}

function handleTouchMove(e) {
    if (!currentSwipeItem) return;

    const touchX = e.touches[0].clientX;
    const touchY = e.touches[0].clientY;
    const diffX = touchX - touchStartX;
    const diffY = touchY - touchStartY;

    // If vertical scroll, don't handle swipe
    if (Math.abs(diffY) > Math.abs(diffX)) {
        currentSwipeItem = null;
        return;
    }

    // Prevent page scroll during swipe
    if (Math.abs(diffX) > 10) {
        e.preventDefault();
    }

    // Apply transform
    const maxSwipe = 100;
    const clampedDiff = Math.max(-maxSwipe, Math.min(maxSwipe, diffX));
    currentSwipeItem.style.transform = `translateX(${clampedDiff}px)`;

    // Show swipe indicator
    if (diffX > 50) {
        currentSwipeItem.classList.add('swipe-archive');
    } else if (diffX < -50) {
        currentSwipeItem.classList.add('swipe-delete');
    } else {
        currentSwipeItem.classList.remove('swipe-archive', 'swipe-delete');
    }
}

async function handleTouchEnd(e) {
    if (!currentSwipeItem) return;

    const transform = currentSwipeItem.style.transform;
    const match = transform.match(/translateX\((-?\d+)px\)/);
    const swipeDistance = match ? parseInt(match[1]) : 0;
    const emailId = currentSwipeItem.dataset.emailId;
    const swipedItem = currentSwipeItem;

    // Reset transient state up-front so the EmailListManager can
    // re-render without our half-finished gesture sticking around.
    swipedItem.classList.remove('swipe-archive', 'swipe-delete');
    currentSwipeItem = null;

    if (swipeDistance > 80) {
        // Archive — animate, then dispatch the same handler the desktop
        // archive button uses so the swipe actually moves the email.
        swipedItem.style.transform = 'translateX(100%)';
        swipedItem.style.opacity = '0';
        let archived = false;
        if (emailId && typeof EmailListManager !== 'undefined' &&
            typeof EmailListManager.archiveEmail === 'function') {
            archived = await EmailListManager.archiveEmail(emailId);
        }
        if (!archived && swipedItem.isConnected) {
            swipedItem.style.transform = '';
            swipedItem.style.opacity = '';
        }
    } else if (swipeDistance < -80) {
        // Delete — same routing as archive, but to the delete handler.
        swipedItem.style.transform = 'translateX(-100%)';
        swipedItem.style.opacity = '0';
        let deleted = false;
        if (emailId && typeof EmailListManager !== 'undefined' &&
            typeof EmailListManager.deleteEmail === 'function') {
            deleted = await EmailListManager.deleteEmail(emailId);
        }
        if (!deleted && swipedItem.isConnected) {
            swipedItem.style.transform = '';
            swipedItem.style.opacity = '';
        }
    } else {
        // Insufficient distance — snap back.
        swipedItem.style.transform = '';
        swipedItem.style.opacity = '';
    }
}

// Initialize pull to refresh
function initPullToRefresh() {
    const emailList = document.querySelector('.email-list-container');
    if (!emailList) return;

    let pullIndicator = document.querySelector('.pull-to-refresh');
    if (!pullIndicator) {
        pullIndicator = document.createElement('div');
        pullIndicator.className = 'pull-to-refresh';
        // Decorative refresh icon — built via createElementNS so we
        // don't need innerHTML for a static SVG. aria-hidden because
        // the indicator is visual feedback, not a labelled control.
        const svgNS = 'http://www.w3.org/2000/svg';
        const svg = document.createElementNS(svgNS, 'svg');
        svg.setAttribute('aria-hidden', 'true');
        svg.setAttribute('width', '24');
        svg.setAttribute('height', '24');
        svg.setAttribute('fill', 'none');
        svg.setAttribute('stroke', 'currentColor');
        svg.setAttribute('stroke-width', '2');
        svg.setAttribute('viewBox', '0 0 24 24');
        const p1 = document.createElementNS(svgNS, 'path');
        p1.setAttribute('d', 'M23 4v6h-6M1 20v-6h6');
        const p2 = document.createElementNS(svgNS, 'path');
        p2.setAttribute('d', 'M3.51 9a9 9 0 0114.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0020.49 15');
        svg.appendChild(p1);
        svg.appendChild(p2);
        pullIndicator.appendChild(svg);
        emailList.style.position = 'relative';
        emailList.insertBefore(pullIndicator, emailList.firstChild);
    }

    emailList.addEventListener('touchstart', (e) => {
        if (emailList.scrollTop === 0) {
            pullStartY = e.touches[0].clientY;
            isPulling = true;
        }
    }, { passive: true });

    emailList.addEventListener('touchmove', (e) => {
        if (!isPulling) return;

        const pullY = e.touches[0].clientY;
        const pullDistance = pullY - pullStartY;

        if (pullDistance > 0 && pullDistance < 100) {
            pullIndicator.style.top = `${pullDistance - 50}px`;
            pullIndicator.style.opacity = pullDistance / 100;
        }

        if (pullDistance > 80) {
            pullIndicator.classList.add('active');
        }
    }, { passive: true });

    emailList.addEventListener('touchend', async () => {
        if (pullIndicator.classList.contains('active')) {
            // Trigger a real refresh — the previous version showed a
            // canned "Updated" toast on a 1.5s timer without ever
            // re-fetching, so the inbox could be stale by minutes.
            showToast('info', 'Refreshing', 'Checking for new emails…');
            // Re-load WITH the user's current view restored. Calling
            // loadEmails() with no args used to drop the active folder
            // filter (Sent / Archive / search results would silently swap
            // to an unfiltered list while the sidebar still highlighted
            // the wrong folder). Pass the cached selection through.
            const opts = {};
            if (typeof EmailListManager !== 'undefined') {
                if (EmailListManager.currentFolder) opts.folder = EmailListManager.currentFolder;
                if (EmailListManager.currentSearch) opts.search = EmailListManager.currentSearch;
            }
            // loadEmails returns one of:
            //   'loaded'      → fetch succeeded
            //   'in-progress' → another load was already running; treat
            //                   as a benign no-op, NOT a failure
            //   'failed'      → fetch raised; an error toast was already
            //                   shown internally
            // Older bundles may still return a boolean — coerce so the
            // contract stays backwards-compatible during rollout.
            let outcome = 'failed';
            if (typeof EmailListManager !== 'undefined' &&
                typeof EmailListManager.loadEmails === 'function') {
                const result = await EmailListManager.loadEmails(opts);
                if (typeof result === 'string') {
                    outcome = result;
                } else if (result === true) {
                    outcome = 'loaded';
                } else if (result === false) {
                    outcome = 'failed';
                }
            }
            if (outcome === 'loaded') {
                showToast('success', 'Updated', 'Inbox is up to date');
            } else if (outcome === 'in-progress') {
                // Another load is in flight — keep the existing pull
                // indicator UX silent so users don't see a confusing
                // toast just because they pulled twice quickly.
            } else {
                showToast('error', 'Refresh failed', 'Could not load new emails');
            }
        }

        pullIndicator.style.top = '-50px';
        pullIndicator.style.opacity = '0';
        pullIndicator.classList.remove('active');
        isPulling = false;
    }, { passive: true });
}

// Handle orientation change
function handleOrientationChange() {
    // Close modals on orientation change
    closeMobileSidebar();
    closeMobilePreview();
}

// Handle resize
const handleResize = debounce(() => {
    if (!isMobile()) {
        closeMobileSidebar();
        closeMobilePreview();
    }
}, 250);

// Initialize mobile features
function initMobile() {
    if (!isMobile()) return;

    initSwipeGestures();
    initPullToRefresh();

    // Override showView to update mobile nav
    const originalShowView = window.showView;
    window.showView = function(view, event) {
        if (typeof originalShowView === 'function') {
            originalShowView(view, event);
        }
        updateMobileNavActive(view);
        closeMobileSidebar();
    };

    // Handle email item clicks on mobile
    document.querySelectorAll('.email-item').forEach(item => {
        item.addEventListener('click', () => {
            if (isMobile()) {
                openMobilePreview();
            }
        });
    });
}

// Event listeners
window.addEventListener('resize', handleResize);
window.addEventListener('orientationchange', handleOrientationChange);

// Initialize on DOM ready
document.addEventListener('DOMContentLoaded', () => {
    initMobile();
});

console.log('%c📱 Mobile module loaded', 'color: #f59e0b;');
