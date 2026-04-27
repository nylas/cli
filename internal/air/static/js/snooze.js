/* Snooze Manager - Email snoozing functionality */

const SnoozeManager = {
    snoozedEmails: [],

    async init() {
        this.setupSnoozePicker();
        await this.loadSnoozedEmails();
        console.log('%c⏰ Snooze module loaded', 'color: #22c55e;');
    },

    setupSnoozePicker() {
        const picker = document.getElementById('snoozePicker');
        if (!picker) return;

        // Update picker with actual functionality
        picker.innerHTML = `
            <div class="snooze-header">
                <span class="snooze-title">Snooze until...</span>
                <button class="snooze-close" data-action="snooze-picker-close">&times;</button>
            </div>
            <div class="snooze-options">
                <button class="snooze-option" data-duration="later">
                    <span class="snooze-icon">☀️</span>
                    <span class="snooze-label">Later today</span>
                    <span class="snooze-time">4:00 PM</span>
                </button>
                <button class="snooze-option" data-duration="tonight">
                    <span class="snooze-icon">🌙</span>
                    <span class="snooze-label">Tonight</span>
                    <span class="snooze-time">8:00 PM</span>
                </button>
                <button class="snooze-option" data-duration="tomorrow">
                    <span class="snooze-icon">📅</span>
                    <span class="snooze-label">Tomorrow</span>
                    <span class="snooze-time">9:00 AM</span>
                </button>
                <button class="snooze-option" data-duration="this weekend">
                    <span class="snooze-icon">🎉</span>
                    <span class="snooze-label">This weekend</span>
                    <span class="snooze-time">Saturday 9:00 AM</span>
                </button>
                <button class="snooze-option" data-duration="next week">
                    <span class="snooze-icon">📆</span>
                    <span class="snooze-label">Next week</span>
                    <span class="snooze-time">Monday 9:00 AM</span>
                </button>
                <button class="snooze-option" data-duration="next month">
                    <span class="snooze-icon">🗓️</span>
                    <span class="snooze-label">Next month</span>
                    <span class="snooze-time">1st of next month</span>
                </button>
            </div>
            <div class="snooze-custom">
                <input type="text" id="snoozeCustom" placeholder="Or type: 2h, 3d, next tuesday...">
                <button class="snooze-custom-btn" data-action="snooze-picker-custom">Snooze</button>
            </div>
        `;

        // Add click handlers
        picker.querySelectorAll('.snooze-option').forEach(option => {
            option.addEventListener('click', () => {
                this.snoozeSelected(option.dataset.duration);
            });
        });

        // Handle Enter key in custom input
        const customInput = picker.querySelector('#snoozeCustom');
        if (customInput) {
            customInput.addEventListener('keypress', (e) => {
                if (e.key === 'Enter') {
                    this.snoozeCustom();
                }
            });
        }
    },

    // Current email being snoozed
    currentEmailId: null,

    open() {
        this.showPicker();
    },

    close() {
        this.hidePicker();
    },

    openForEmail(emailId) {
        this.currentEmailId = emailId;
        this.showPicker();
    },

    showPicker() {
        const picker = document.getElementById('snoozePicker');
        if (picker) {
            picker.style.display = 'block';
            picker.classList.add('active');
        }
    },

    hidePicker() {
        const picker = document.getElementById('snoozePicker');
        if (picker) {
            picker.style.display = 'none';
            picker.classList.remove('active');
        }
        this.currentEmailId = null;
    },

    // Called from HTML template buttons
    async snooze(duration) {
        const emailId = this.currentEmailId || this.getSelectedEmailId();
        if (!emailId) {
            if (typeof showToast === 'function') {
                showToast('warning', 'No email selected', 'Select an email to snooze');
            }
            return;
        }
        await this.snoozeEmail(emailId, duration);
    },

    async snoozeSelected(duration) {
        const emailId = this.getSelectedEmailId();
        if (!emailId) {
            if (typeof showToast === 'function') {
                showToast('warning', 'No email selected', 'Select an email to snooze');
            }
            return;
        }

        await this.snoozeEmail(emailId, duration);
    },

    async snoozeCustom() {
        const input = document.getElementById('snoozeCustom');
        const duration = input?.value?.trim();

        if (!duration) {
            if (typeof showToast === 'function') {
                showToast('warning', 'Enter a time', 'Type a snooze time like "2h" or "tomorrow"');
            }
            return;
        }

        const emailId = this.getSelectedEmailId();
        if (!emailId) {
            if (typeof showToast === 'function') {
                showToast('warning', 'No email selected', 'Select an email to snooze');
            }
            return;
        }

        await this.snoozeEmail(emailId, duration);
        if (input) input.value = '';
    },

    async snoozeEmail(emailId, duration) {
        try {
            const result = await AirAPI.snoozeEmail(emailId, duration);
            this.hidePicker();

            if (typeof showToast === 'function') {
                showToast('success', 'Snoozed', result.message || 'Email snoozed');
            }

            // Remove from current view
            if (typeof EmailListManager !== 'undefined') {
                EmailListManager.emails = EmailListManager.emails.filter(e => e.id !== emailId);
                EmailListManager.applyFilter();
                EmailListManager.renderEmails();
            }
        } catch (error) {
            console.error('Failed to snooze email:', error);
            if (typeof showToast === 'function') {
                showToast('error', 'Error', 'Failed to snooze: ' + error.message);
            }
        }
    },

    async loadSnoozedEmails() {
        try {
            const result = await AirAPI.getSnoozedEmails();
            this.snoozedEmails = result.snoozed || result || [];
            if (!Array.isArray(this.snoozedEmails)) {
                this.snoozedEmails = [];
            }
        } catch (error) {
            // Silently fail - snoozed emails are optional
            this.snoozedEmails = [];
        }
    },

    getSelectedEmailId() {
        if (typeof EmailListManager !== 'undefined') {
            return EmailListManager.selectedEmailId;
        }
        return null;
    }
};

// =============================================================================
// SCHEDULED SEND MANAGER
// =============================================================================

