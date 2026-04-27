/* Scheduled Send Manager - Send Later functionality */

const ScheduledSendManager = {
    scheduledMessages: [],
    dropdownOpen: false,

    async init() {
        await this.loadScheduledMessages();
        this.setupSendLaterButton();
        console.log('%c📤 Scheduled Send module loaded', 'color: #22c55e;');
    },

    toggleDropdown(event) {
        if (event) event.stopPropagation();
        const dropdown = document.querySelector('.send-dropdown');
        if (dropdown) {
            this.dropdownOpen = !this.dropdownOpen;
            dropdown.classList.toggle('open', this.dropdownOpen);
        }
    },

    closeDropdown() {
        const dropdown = document.querySelector('.send-dropdown');
        if (dropdown) {
            dropdown.classList.remove('open');
            this.dropdownOpen = false;
        }
    },

    async scheduleFromCompose() {
        this.closeDropdown();
        if (typeof ComposeManager === 'undefined') {
            if (typeof showToast === 'function') {
                showToast('error', 'Error', 'Compose not available');
            }
            return;
        }

        const data = ComposeManager.getFormData();
        if (!data.to || data.to.length === 0) {
            if (typeof showToast === 'function') {
                showToast('error', 'Error', 'Please add at least one recipient');
            }
            return;
        }

        this.showSchedulePicker(data);
    },

    setupSendLaterButton() {
        // Add "Send Later" option to compose modal
        const sendBtn = document.getElementById('composeSend');
        if (!sendBtn) return;

        // Create dropdown container
        const container = sendBtn.parentElement;
        if (!container || container.querySelector('.send-dropdown')) return;

        // Wrap send button in dropdown
        const dropdown = document.createElement('div');
        dropdown.className = 'send-dropdown';
        dropdown.innerHTML = `
            <button class="send-dropdown-toggle" title="Send options">▼</button>
            <div class="send-dropdown-menu">
                <button class="send-option" data-action="send-later">
                    <span>📅</span> Schedule send...
                </button>
                <button class="send-option" data-action="send-tomorrow">
                    <span>☀️</span> Send tomorrow 9 AM
                </button>
                <button class="send-option" data-action="send-monday">
                    <span>📆</span> Send Monday 9 AM
                </button>
            </div>
        `;

        container.appendChild(dropdown);

        // Toggle dropdown
        dropdown.querySelector('.send-dropdown-toggle').addEventListener('click', (e) => {
            e.stopPropagation();
            dropdown.classList.toggle('open');
        });

        // Handle options
        dropdown.querySelectorAll('.send-option').forEach(option => {
            option.addEventListener('click', (e) => {
                e.stopPropagation();
                dropdown.classList.remove('open');
                this.handleSendOption(option.dataset.action);
            });
        });

        // Close on click outside
        document.addEventListener('click', () => {
            dropdown.classList.remove('open');
        });
    },

    async handleSendOption(action) {
        if (typeof ComposeManager === 'undefined') return;

        const data = ComposeManager.getFormData();
        if (data.to.length === 0) {
            if (typeof showToast === 'function') {
                showToast('error', 'Error', 'Please add at least one recipient');
            }
            return;
        }

        let sendAt;
        switch (action) {
            case 'send-tomorrow':
                sendAt = 'tomorrow';
                break;
            case 'send-monday':
                sendAt = 'next monday';
                break;
            case 'send-later':
                this.showSchedulePicker(data);
                return;
            default:
                return;
        }

        await this.scheduleMessage(data, sendAt);
    },

    showSchedulePicker(data) {
        const picker = document.createElement('div');
        picker.className = 'schedule-picker-modal';
        picker.innerHTML = `
            <div class="schedule-picker">
                <h3>Schedule send</h3>
                <div class="schedule-options">
                    <input type="text" id="scheduleInput" placeholder="e.g., tomorrow 2pm, next friday, in 3 hours">
                </div>
                <div class="schedule-actions">
                    <button class="btn-secondary" data-action="schedule-picker-cancel">Cancel</button>
                    <button class="btn-primary" id="scheduleConfirm">Schedule</button>
                </div>
            </div>
        `;

        document.body.appendChild(picker);

        const input = picker.querySelector('#scheduleInput');
        const confirmBtn = picker.querySelector('#scheduleConfirm');

        input.focus();

        confirmBtn.addEventListener('click', async () => {
            const sendAt = input.value.trim();
            if (!sendAt) {
                if (typeof showToast === 'function') {
                    showToast('warning', 'Enter a time', 'Please enter when to send');
                }
                return;
            }
            picker.remove();
            await this.scheduleMessage(data, sendAt);
        });

        input.addEventListener('keypress', async (e) => {
            if (e.key === 'Enter') {
                confirmBtn.click();
            }
        });
    },

    async scheduleMessage(messageData, sendAt) {
        try {
            const result = await AirAPI.scheduleMessage(messageData, sendAt);

            if (typeof showToast === 'function') {
                showToast('success', 'Scheduled', result.message || 'Message scheduled');
            }

            if (typeof ComposeManager !== 'undefined') {
                ComposeManager.close();
            }
        } catch (error) {
            console.error('Failed to schedule message:', error);
            if (typeof showToast === 'function') {
                showToast('error', 'Error', 'Failed to schedule: ' + error.message);
            }
        }
    },

    async loadScheduledMessages() {
        try {
            const result = await AirAPI.getScheduledMessages();
            this.scheduledMessages = result.scheduled || result || [];
            if (!Array.isArray(this.scheduledMessages)) {
                this.scheduledMessages = [];
            }
        } catch (error) {
            // Silently fail - scheduled messages are optional
            this.scheduledMessages = [];
        }
    }
};

// =============================================================================
// UNDO SEND MANAGER
// =============================================================================

