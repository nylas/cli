/**
 * Touch Gestures Module
 * Mobile-first swipe and gesture support for email actions
 * Implements swipe-to-archive, swipe-to-delete patterns
 */

const TouchGestures = {
    // Configuration
    config: {
        swipeThreshold: 80,
        velocityThreshold: 0.5,
        resistanceFactor: 0.4,
    },

    // Gesture state
    state: {
        startX: 0,
        startY: 0,
        currentX: 0,
        currentY: 0,
        startTime: 0,
        isTracking: false,
        activeElement: null,
        direction: null,
    },

    /**
     * Initialize touch gestures
     */
    init() {
        if (!this.isTouchDevice()) {
            console.log('Touch gestures: Not a touch device');
            return;
        }

        this.setupEmailListGestures();
        this.setupPullToRefresh();
        console.log('%c👆 Touch gestures initialized', 'color: #f59e0b;');
    },

    /**
     * Check if device supports touch
     * @returns {boolean}
     */
    isTouchDevice() {
        return 'ontouchstart' in window || navigator.maxTouchPoints > 0;
    },

    /**
     * Setup swipe gestures on email list items
     */
    setupEmailListGestures() {
        const emailList = document.querySelector('.email-list');
        if (!emailList) return;

        emailList.addEventListener('touchstart', (e) => this.handleTouchStart(e), { passive: true });
        emailList.addEventListener('touchmove', (e) => this.handleTouchMove(e), { passive: false });
        emailList.addEventListener('touchend', (e) => this.handleTouchEnd(e));
        emailList.addEventListener('touchcancel', () => this.resetState());
    },

    /**
     * Handle touch start
     * @param {TouchEvent} e
     */
    handleTouchStart(e) {
        const emailItem = e.target.closest('.email-item');
        if (!emailItem) return;

        const touch = e.touches[0];
        this.state = {
            startX: touch.clientX,
            startY: touch.clientY,
            currentX: touch.clientX,
            currentY: touch.clientY,
            startTime: Date.now(),
            isTracking: true,
            activeElement: emailItem,
            direction: null,
        };

        emailItem.classList.add('touch-active');
    },

    /**
     * Handle touch move
     * @param {TouchEvent} e
     */
    handleTouchMove(e) {
        if (!this.state.isTracking || !this.state.activeElement) return;

        const touch = e.touches[0];
        const deltaX = touch.clientX - this.state.startX;
        const deltaY = touch.clientY - this.state.startY;

        // Determine direction if not set
        if (!this.state.direction) {
            if (Math.abs(deltaX) > Math.abs(deltaY) && Math.abs(deltaX) > 10) {
                this.state.direction = 'horizontal';
            } else if (Math.abs(deltaY) > 10) {
                this.state.direction = 'vertical';
                this.resetState();
                return;
            }
        }

        if (this.state.direction !== 'horizontal') return;

        e.preventDefault();

        this.state.currentX = touch.clientX;
        this.state.currentY = touch.clientY;

        // Apply transform with resistance
        let translateX = deltaX;
        if (Math.abs(deltaX) > this.config.swipeThreshold) {
            const excess = Math.abs(deltaX) - this.config.swipeThreshold;
            translateX = Math.sign(deltaX) * (this.config.swipeThreshold + excess * this.config.resistanceFactor);
        }

        this.state.activeElement.style.transform = `translateX(${translateX}px)`;
        this.updateSwipeIndicator(deltaX);
    },

    /**
     * Handle touch end
     * @param {TouchEvent} e
     */
    handleTouchEnd(e) {
        if (!this.state.isTracking || !this.state.activeElement) return;

        const deltaX = this.state.currentX - this.state.startX;
        const deltaTime = Date.now() - this.state.startTime;
        const velocity = Math.abs(deltaX) / deltaTime;

        const shouldComplete = Math.abs(deltaX) > this.config.swipeThreshold ||
                              velocity > this.config.velocityThreshold;

        if (shouldComplete) {
            this.completeSwipe(deltaX > 0 ? 'right' : 'left');
        } else {
            this.cancelSwipe();
        }
    },

    /**
     * Update swipe indicator
     * @param {number} deltaX
     */
    updateSwipeIndicator(deltaX) {
        const element = this.state.activeElement;
        if (!element) return;

        // Remove existing indicators
        element.querySelectorAll('.swipe-indicator').forEach(el => el.remove());

        if (Math.abs(deltaX) < 20) return;

        const indicator = document.createElement('div');
        indicator.className = 'swipe-indicator';

        if (deltaX > 0) {
            indicator.classList.add('swipe-right');
            const icon = document.createElement('span');
            icon.textContent = '📥';
            indicator.appendChild(icon);
            const text = document.createElement('span');
            text.textContent = 'Archive';
            indicator.appendChild(text);
        } else {
            indicator.classList.add('swipe-left');
            const icon = document.createElement('span');
            icon.textContent = '🗑️';
            indicator.appendChild(icon);
            const text = document.createElement('span');
            text.textContent = 'Delete';
            indicator.appendChild(text);
        }

        element.appendChild(indicator);
    },

    /**
     * Complete swipe action
     * @param {string} direction - 'left' or 'right'
     */
    completeSwipe(direction) {
        const element = this.state.activeElement;
        if (!element) return;

        const emailId = element.dataset.emailId;
        const action = direction === 'right' ? 'archive' : 'delete';

        // Animate out
        element.style.transition = 'transform 0.2s ease-out, opacity 0.2s ease-out';
        element.style.transform = `translateX(${direction === 'right' ? '100%' : '-100%'})`;
        element.style.opacity = '0';

        // Perform action after animation
        setTimeout(() => {
            this.performAction(emailId, action);
            element.remove();
        }, 200);

        this.resetState();
    },

    /**
     * Cancel swipe and reset
     */
    cancelSwipe() {
        const element = this.state.activeElement;
        if (!element) return;

        element.style.transition = 'transform 0.2s ease-out';
        element.style.transform = 'translateX(0)';

        element.querySelectorAll('.swipe-indicator').forEach(el => el.remove());

        setTimeout(() => {
            element.style.transition = '';
        }, 200);

        this.resetState();
    },

    /**
     * Reset gesture state
     */
    resetState() {
        if (this.state.activeElement) {
            this.state.activeElement.classList.remove('touch-active');
        }

        this.state = {
            startX: 0,
            startY: 0,
            currentX: 0,
            currentY: 0,
            startTime: 0,
            isTracking: false,
            activeElement: null,
            direction: null,
        };
    },

    /**
     * Perform email action
     * @param {string} emailId
     * @param {string} action
     */
    async performAction(emailId, action) {
        try {
            const safeId = encodeURIComponent(emailId);
            if (action === 'archive') {
                await fetch(`/api/emails/${safeId}`, {
                    method: 'PUT',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ folder: 'archive' }),
                });
            } else if (action === 'delete') {
                await fetch(`/api/emails/${safeId}`, {
                    method: 'DELETE',
                });
            }

            // Dispatch event for UI update
            document.dispatchEvent(new CustomEvent('emailAction', {
                detail: { emailId, action },
            }));
        } catch (error) {
            console.error(`Failed to ${action} email:`, error);
        }
    },

    /**
     * Setup pull to refresh
     */
    setupPullToRefresh() {
        const container = document.querySelector('.email-list-container');
        if (!container) return;

        let startY = 0;
        let pullDistance = 0;
        let isPulling = false;

        container.addEventListener('touchstart', (e) => {
            if (container.scrollTop === 0) {
                startY = e.touches[0].clientY;
                isPulling = true;
            }
        }, { passive: true });

        container.addEventListener('touchmove', (e) => {
            if (!isPulling) return;

            pullDistance = e.touches[0].clientY - startY;

            if (pullDistance > 0 && pullDistance < 150) {
                container.style.transform = `translateY(${pullDistance * 0.5}px)`;
                this.showPullIndicator(pullDistance);
            }
        }, { passive: true });

        container.addEventListener('touchend', () => {
            if (pullDistance > 80) {
                this.triggerRefresh();
            }
            container.style.transform = '';
            this.hidePullIndicator();
            isPulling = false;
            pullDistance = 0;
        });
    },

    /**
     * Show pull to refresh indicator
     * @param {number} distance
     */
    showPullIndicator(distance) {
        let indicator = document.querySelector('.pull-indicator');
        if (!indicator) {
            indicator = document.createElement('div');
            indicator.className = 'pull-indicator';
            const spinner = document.createElement('div');
            spinner.className = 'pull-spinner';
            indicator.appendChild(spinner);
            const text = document.createElement('span');
            text.textContent = 'Pull to refresh';
            indicator.appendChild(text);
            document.querySelector('.email-list-container')?.prepend(indicator);
        }

        indicator.style.opacity = Math.min(distance / 80, 1);
        if (distance > 80) {
            indicator.querySelector('span').textContent = 'Release to refresh';
        }
    },

    /**
     * Hide pull indicator
     */
    hidePullIndicator() {
        const indicator = document.querySelector('.pull-indicator');
        if (indicator) {
            indicator.remove();
        }
    },

    /**
     * Trigger refresh
     */
    triggerRefresh() {
        document.dispatchEvent(new CustomEvent('refreshRequested'));
    },
};

// Initialize on DOM ready
document.addEventListener('DOMContentLoaded', () => {
    TouchGestures.init();
});

// Export for use
if (typeof window !== 'undefined') {
    window.TouchGestures = TouchGestures;
}
