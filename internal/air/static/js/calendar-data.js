/**
 * Calendar Data - Loading and navigation
 */
Object.assign(CalendarManager, {
async loadCalendars() {
    try {
        const data = await AirAPI.getCalendars();
        this.calendars = data.calendars || [];
        this.renderCalendarList();
    } catch (error) {
        console.error('Failed to load calendars:', error);
    }
},

async loadEvents() {
    if (this.isLoading) return;
    this.isLoading = true;

    try {
        const { start, end } = this.getDateRange();

        const data = await AirAPI.getEvents({
            start: Math.floor(start.getTime() / 1000),
            end: Math.floor(end.getTime() / 1000),
            limit: 200  // Fetch more events to cover the full month
        });

        this.events = data.events || [];
        this.renderCalendarGrid(); // Re-render grid to show event dots
        this.renderEvents();
    } catch (error) {
        console.error('Failed to load events:', error);
        if (typeof showToast === 'function') {
            showToast('error', 'Error', 'Failed to load events');
        }
    } finally {
        this.isLoading = false;
    }
},

getDateRange() {
    const now = this.currentDate;

    // Always load events for the full visible month in the calendar grid
    // This includes padding days from prev/next months that appear in the grid
    const firstOfMonth = new Date(now.getFullYear(), now.getMonth(), 1);
    const lastOfMonth = new Date(now.getFullYear(), now.getMonth() + 1, 0);

    // Start from the Sunday of the week containing the 1st
    const start = new Date(firstOfMonth);
    start.setDate(start.getDate() - firstOfMonth.getDay());
    start.setHours(0, 0, 0, 0);

    // End at the Saturday of the week containing the last day
    const end = new Date(lastOfMonth);
    const daysUntilSaturday = 6 - lastOfMonth.getDay();
    end.setDate(end.getDate() + daysUntilSaturday + 1); // +1 to include the full day
    end.setHours(23, 59, 59, 999);

    return { start, end };
},

navigate(direction) {
    switch (this.currentView) {
        case 'today':
            this.currentDate.setDate(this.currentDate.getDate() + direction);
            break;
        case 'week':
            this.currentDate.setDate(this.currentDate.getDate() + (direction * 7));
            break;
        case 'month':
            this.currentDate.setMonth(this.currentDate.getMonth() + direction);
            break;
        case 'agenda':
            this.currentDate.setDate(this.currentDate.getDate() + (direction * 14));
            break;
    }

    this.updateTitle();
    this.loadEvents();
},

goToToday() {
    this.currentDate = new Date();
    this.updateTitle();
    this.loadEvents();
},

setView(view) {
    this.currentView = view;

    // Update sidebar active state
    const folderItems = document.querySelectorAll('#calendarView .sidebar .folder-item');
    folderItems.forEach(item => {
        const text = item.textContent.trim().toLowerCase();
        const isActive = text.includes(view) ||
            (view === 'today' && text.includes('today')) ||
            (view === 'week' && text.includes('week')) ||
            (view === 'month' && text.includes('month')) ||
            (view === 'agenda' && text.includes('agenda'));
        item.classList.toggle('active', isActive);
    });

    this.updateTitle();
    this.loadEvents();
},

updateTitle() {
    const titleEl = document.querySelector('.calendar-title');
    if (!titleEl) return;

    const options = { month: 'long', year: 'numeric' };
    if (this.currentView === 'today') {
        options.day = 'numeric';
    }

    titleEl.textContent = this.currentDate.toLocaleDateString('en-US', options);
},
});
