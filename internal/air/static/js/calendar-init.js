/**
 * Calendar Initialization - DOM ready setup
 */
// Data will load when user switches to calendar view (lazy loading)
document.addEventListener('DOMContentLoaded', () => {
    // Set up event listeners but don't load data yet
    if (document.getElementById('calendarView')) {
        CalendarManager.setupEventListeners();
    }

    // Wire up "New Event" button in calendar sidebar
    const newEventBtn = document.querySelector('#calendarView .compose-btn');
    if (newEventBtn) {
        newEventBtn.onclick = () => openEventModal();
    }

    // Wire up event card clicks for editing. Skip when the click originates
    // inside an interactive child (data-action button or link), so the
    // child's handler — e.g. join-meeting-link — runs alone instead of also
    // opening the edit modal. stopPropagation can't help here: both listeners
    // are on document, so they're siblings, not parent/child.
    document.addEventListener('click', (e) => {
        if (e.target.closest('[data-action]') || e.target.closest('a[href]')) {
            return;
        }
        const eventCard = e.target.closest('.event-card');
        if (eventCard) {
            const eventId = eventCard.getAttribute('data-event-id');
            if (eventId && CalendarManager) {
                const event = CalendarManager.events.find(ev => ev.id === eventId);
                if (event) {
                    openEventModal(event);
                }
            }
        }
    });

    // Close modal on Escape key
    document.addEventListener('keydown', (e) => {
        if (e.key === 'Escape' && EventModal.isOpen) {
            closeEventModal();
        }
    });
});
