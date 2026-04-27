/**
 * Calendar Find Time - Modal for finding available time slots
 */
const FindTimeModal = {
    isOpen: false,
    selectedSlot: null,

    open() {
        const overlay = document.getElementById('findTimeModalOverlay');
        if (!overlay) {
            this.createModal();
        }
        document.getElementById('findTimeModalOverlay').classList.remove('hidden');
        this.isOpen = true;
        this.reset();
    },

    close() {
        const overlay = document.getElementById('findTimeModalOverlay');
        if (overlay) {
            overlay.classList.add('hidden');
        }
        this.isOpen = false;
    },

    reset() {
        this.selectedSlot = null;
        const participantsInput = document.getElementById('findTimeParticipants');
        const durationSelect = document.getElementById('findTimeDuration');
        const resultsContainer = document.getElementById('findTimeResults');

        if (participantsInput) participantsInput.value = '';
        if (durationSelect) durationSelect.value = '30';
        if (resultsContainer) resultsContainer.innerHTML = `
            <div class="find-time-hint">
                Enter participant emails and click "Find Available Times"
            </div>
        `;
    },

    async search() {
        const participantsInput = document.getElementById('findTimeParticipants');
        const durationSelect = document.getElementById('findTimeDuration');
        const resultsContainer = document.getElementById('findTimeResults');
        const searchBtn = document.getElementById('findTimeSearchBtn');

        if (!participantsInput || !resultsContainer) return;

        const participants = participantsInput.value
            .split(',')
            .map(e => e.trim())
            .filter(e => e.length > 0);

        if (participants.length === 0) {
            if (typeof showToast === 'function') {
                showToast('error', 'Error', 'Please enter at least one participant email');
            }
            return;
        }

        // Show loading state
        if (searchBtn) {
            searchBtn.disabled = true;
            searchBtn.textContent = 'Searching...';
        }
        resultsContainer.innerHTML = '<div class="loading">Finding available times...</div>';

        try {
            const duration = parseInt(durationSelect?.value || '30', 10);
            const slots = await AirAPI.findTime(participants, duration);

            CalendarManager.availabilitySlots = slots.slots || [];
            CalendarManager.renderAvailabilitySlots(resultsContainer);

            // Add click handlers for slot selection
            resultsContainer.querySelectorAll('.availability-slot').forEach(btn => {
                btn.addEventListener('click', () => {
                    this.selectSlot(btn);
                });
            });

        } catch (error) {
            console.error('Find time error:', error);
            resultsContainer.innerHTML = `
                <div class="error-state">
                    <div class="error-icon">❌</div>
                    <div class="error-message">${error.message || 'Failed to find available times'}</div>
                </div>
            `;
        } finally {
            if (searchBtn) {
                searchBtn.disabled = false;
                searchBtn.textContent = 'Find Available Times';
            }
        }
    },

    selectSlot(btn) {
        // Remove previous selection
        document.querySelectorAll('.availability-slot.selected').forEach(el => {
            el.classList.remove('selected');
        });

        btn.classList.add('selected');
        this.selectedSlot = {
            start_time: parseInt(btn.dataset.start, 10),
            end_time: parseInt(btn.dataset.end, 10)
        };

        // Enable create event button
        const createBtn = document.getElementById('findTimeCreateBtn');
        if (createBtn) {
            createBtn.disabled = false;
        }
    },

    createEvent() {
        if (!this.selectedSlot) {
            if (typeof showToast === 'function') {
                showToast('error', 'Error', 'Please select a time slot first');
            }
            return;
        }

        const participantsInput = document.getElementById('findTimeParticipants');
        const participants = participantsInput?.value
            .split(',')
            .map(e => e.trim())
            .filter(e => e.length > 0) || [];

        // Close find time modal
        this.close();

        // Open event modal with pre-filled data
        const eventData = {
            start_time: this.selectedSlot.start_time,
            end_time: this.selectedSlot.end_time,
            participants: participants.map(email => ({ email }))
        };

        EventModal.open(eventData);
    },

    createModal() {
        const overlay = document.createElement('div');
        overlay.id = 'findTimeModalOverlay';
        overlay.className = 'modal-overlay hidden';
        overlay.innerHTML = `
            <div class="modal find-time-modal">
                <div class="modal-header">
                    <h2>Find Available Time</h2>
                    <button class="modal-close" data-action="find-time-close">&times;</button>
                </div>
                <div class="modal-body">
                    <div class="form-group">
                        <label for="findTimeParticipants">Participants (comma-separated emails)</label>
                        <input type="text" id="findTimeParticipants"
                               placeholder="alice@example.com, bob@example.com">
                    </div>
                    <div class="form-group">
                        <label for="findTimeDuration">Meeting Duration</label>
                        <select id="findTimeDuration">
                            <option value="15">15 minutes</option>
                            <option value="30" selected>30 minutes</option>
                            <option value="45">45 minutes</option>
                            <option value="60">1 hour</option>
                            <option value="90">1.5 hours</option>
                            <option value="120">2 hours</option>
                        </select>
                    </div>
                    <button id="findTimeSearchBtn" class="btn btn-primary" data-action="find-time-search">
                        Find Available Times
                    </button>
                    <div id="findTimeResults" class="find-time-results">
                        <div class="find-time-hint">
                            Enter participant emails and click "Find Available Times"
                        </div>
                    </div>
                </div>
                <div class="modal-footer">
                    <button class="btn btn-secondary" data-action="find-time-close">Cancel</button>
                    <button id="findTimeCreateBtn" class="btn btn-primary" data-action="find-time-create-event" disabled>
                        Create Event
                    </button>
                </div>
            </div>
        `;
        document.body.appendChild(overlay);

        // Close on backdrop click
        overlay.addEventListener('click', (e) => {
            if (e.target === overlay) {
                this.close();
            }
        });
    }
};

// Global function for Find Time
function openFindTimeModal() {
    FindTimeModal.open();
}
