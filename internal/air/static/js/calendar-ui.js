/**
 * Calendar UI - Rendering
 */
Object.assign(CalendarManager, {
renderCalendarGrid() {
    const container = document.querySelector('.calendar-grid');
    if (!container) return;

    const now = this.currentDate;
    const year = now.getFullYear();
    const month = now.getMonth();

    // Get first day of month and total days
    const firstDay = new Date(year, month, 1);
    const lastDay = new Date(year, month + 1, 0);
    const daysInMonth = lastDay.getDate();
    const startDayOfWeek = firstDay.getDay(); // 0 = Sunday

    // Get previous month's days to fill
    const prevMonthLastDay = new Date(year, month, 0).getDate();

    // Today's date for highlighting
    const today = new Date();
    const isCurrentMonth = today.getFullYear() === year && today.getMonth() === month;
    const todayDate = today.getDate();

    // Build grid HTML
    let html = `
        <div class="calendar-day-header">Sun</div>
        <div class="calendar-day-header">Mon</div>
        <div class="calendar-day-header">Tue</div>
        <div class="calendar-day-header">Wed</div>
        <div class="calendar-day-header">Thu</div>
        <div class="calendar-day-header">Fri</div>
        <div class="calendar-day-header">Sat</div>
    `;

    // Previous month days
    for (let i = startDayOfWeek - 1; i >= 0; i--) {
        const day = prevMonthLastDay - i;
        html += `<div class="calendar-day other-month" data-date="${year}-${month}-${day}">${day}</div>`;
    }

    // Current month days
    for (let day = 1; day <= daysInMonth; day++) {
        const isToday = isCurrentMonth && day === todayDate;
        const dateStr = `${year}-${month + 1}-${day}`;
        // Count events for this day
        const dayEvents = this.events.filter(e => {
            const eventDate = new Date(e.start_time * 1000);
            return eventDate.getFullYear() === year &&
                   eventDate.getMonth() === month &&
                   eventDate.getDate() === day;
        });
        const eventCount = dayEvents.length;

        let classes = 'calendar-day';
        if (isToday) classes += ' today';
        if (eventCount > 0) classes += ' has-event';

        html += `<div class="${classes}" data-date="${dateStr}">
            ${day}
            ${isToday ? '<span class="today-indicator"></span>' : ''}
            ${eventCount > 0 ? `<div class="event-count-badge">${eventCount}</div>` : ''}
        </div>`;
    }

    // Next month days to fill remaining cells (42 total = 6 weeks)
    const totalCells = startDayOfWeek + daysInMonth;
    const remainingCells = (7 - (totalCells % 7)) % 7;
    for (let day = 1; day <= remainingCells; day++) {
        html += `<div class="calendar-day other-month" data-date="${year}-${month + 2}-${day}">${day}</div>`;
    }

    container.innerHTML = html;

    // Add click handlers for calendar days
    container.querySelectorAll('.calendar-day:not(.calendar-day-header)').forEach(dayEl => {
        dayEl.addEventListener('click', () => {
            const dateStr = dayEl.dataset.date;
            if (dateStr) {
                this.onDayClick(dateStr, dayEl);
            }
        });
    });
},

onDayClick(dateStr, dayEl) {
    // Parse the date string and update currentDate
    const parts = dateStr.split('-').map(Number);
    const clickedDate = new Date(parts[0], parts[1] - 1, parts[2]);

    // Update current date
    this.currentDate = clickedDate;

    // Update selected state visually
    document.querySelectorAll('.calendar-day.selected').forEach(el => el.classList.remove('selected'));
    dayEl.classList.add('selected');

    // Check if clicked date is today
    const today = new Date();
    const isToday = clickedDate.getFullYear() === today.getFullYear() &&
                    clickedDate.getMonth() === today.getMonth() &&
                    clickedDate.getDate() === today.getDate();

    // Update events panel header title and date
    const headerEl = document.querySelector('.events-header h3');
    const dateEl = document.querySelector('.events-date');

    if (headerEl) {
        headerEl.textContent = isToday ? "Today's Schedule" : clickedDate.toLocaleDateString('en-US', {
            weekday: 'long',
            month: 'long',
            day: 'numeric'
        });
    }

    if (dateEl) {
        dateEl.textContent = clickedDate.toLocaleDateString('en-US', {
            weekday: 'short',
            month: 'short',
            day: 'numeric'
        });
    }

    // Filter and show events for clicked day
    this.renderEventsForDay(clickedDate);
},

renderEventsForDay(date) {
    const eventsContainer = document.querySelector('.events-list');
    if (!eventsContainer) return;

    // Filter events for the selected day
    const dayStart = new Date(date.getFullYear(), date.getMonth(), date.getDate()).getTime() / 1000;
    const dayEnd = dayStart + 86400; // 24 hours

    const dayEvents = this.events.filter(event => {
        return event.start_time >= dayStart && event.start_time < dayEnd;
    });

    if (dayEvents.length === 0) {
        eventsContainer.innerHTML = `
            <div class="empty-state">
                <div class="empty-icon">📅</div>
                <div class="empty-message">No events for this day</div>
            </div>
        `;
        return;
    }

    // Sort events by start time
    const sortedEvents = [...dayEvents].sort((a, b) => a.start_time - b.start_time);
    eventsContainer.innerHTML = sortedEvents.map(event => this.renderEventCard(event)).join('');
},

renderCalendarList() {
    const container = document.getElementById('calendarsList');
    if (!container) {
        console.error('Calendar list container not found');
        return;
    }

    // Clear skeleton loaders
    container.innerHTML = '';

    if (this.calendars.length === 0) {
        container.innerHTML = '<div class="folder-item"><span class="text-muted">No calendars found</span></div>';
        return;
    }

    this.calendars.forEach(cal => {
        const div = document.createElement('div');
        div.className = 'folder-item';
        div.setAttribute('data-calendar-id', cal.id);
        div.innerHTML = `
            <span class="label-dot" style="background: ${cal.hex_color || '#4285f4'}"></span>
            <span>${this.escapeHtml(cal.name)}</span>
        `;
        container.appendChild(div);
    });
},

renderEvents() {
    const eventsContainer = document.querySelector('.events-list');
    if (!eventsContainer) return;

    // Get filtered events based on current view
    const filteredEvents = this.getFilteredEventsForView();

    // Update header based on view
    const dateEl = document.querySelector('.events-date');
    const headerEl = document.querySelector('.events-header h3');
    if (dateEl && headerEl) {
        switch (this.currentView) {
            case 'today':
                headerEl.textContent = "Today's Schedule";
                dateEl.textContent = this.currentDate.toLocaleDateString('en-US', {
                    weekday: 'short',
                    month: 'short',
                    day: 'numeric'
                });
                break;
            case 'week':
                headerEl.textContent = "This Week";
                const weekStart = new Date(this.currentDate);
                weekStart.setDate(weekStart.getDate() - weekStart.getDay());
                const weekEnd = new Date(weekStart);
                weekEnd.setDate(weekEnd.getDate() + 6);
                dateEl.textContent = `${weekStart.toLocaleDateString('en-US', { month: 'short', day: 'numeric' })} - ${weekEnd.toLocaleDateString('en-US', { month: 'short', day: 'numeric' })}`;
                break;
            case 'month':
                headerEl.textContent = "This Month";
                dateEl.textContent = this.currentDate.toLocaleDateString('en-US', {
                    month: 'long',
                    year: 'numeric'
                });
                break;
            case 'agenda':
                headerEl.textContent = "Upcoming";
                dateEl.textContent = "Next 2 weeks";
                break;
        }
    }

    if (filteredEvents.length === 0) {
        eventsContainer.innerHTML = `
            <div class="empty-state">
                <div class="empty-icon">📅</div>
                <div class="empty-message">No events</div>
            </div>
        `;
        return;
    }

    // Sort events by start time
    const sortedEvents = [...filteredEvents].sort((a, b) => a.start_time - b.start_time);

    eventsContainer.innerHTML = sortedEvents.map(event => this.renderEventCard(event)).join('');
},

getFilteredEventsForView() {
    const now = this.currentDate;
    let start, end;

    switch (this.currentView) {
        case 'today':
            start = new Date(now.getFullYear(), now.getMonth(), now.getDate()).getTime() / 1000;
            end = start + 86400; // 24 hours
            break;
        case 'week':
            const weekStart = new Date(now);
            weekStart.setDate(weekStart.getDate() - weekStart.getDay());
            weekStart.setHours(0, 0, 0, 0);
            start = weekStart.getTime() / 1000;
            end = start + (7 * 86400); // 7 days
            break;
        case 'month':
            const monthStart = new Date(now.getFullYear(), now.getMonth(), 1);
            const monthEnd = new Date(now.getFullYear(), now.getMonth() + 1, 0);
            start = monthStart.getTime() / 1000;
            end = (monthEnd.getTime() / 1000) + 86400; // Include last day
            break;
        case 'agenda':
        default:
            start = new Date(now.getFullYear(), now.getMonth(), now.getDate()).getTime() / 1000;
            end = start + (14 * 86400); // 14 days
            break;
    }

    return this.events.filter(event => event.start_time >= start && event.start_time < end);
},

renderEventCard(event) {
    const startTime = this.formatEventTime(event.start_time);
    const endTime = this.formatEventTime(event.end_time);
    const isFocusTime = event.title?.toLowerCase().includes('focus');
    const hasConferencing = event.conferencing && event.conferencing.url;
    const relativeTime = this.getRelativeTime(event.start_time);

    const participantsHtml = event.participants && event.participants.length > 0
        ? `<div class="event-attendees">
            ${event.participants.slice(0, 3).map(p => `
                <div class="attendee-avatar" style="background: var(--gradient-${Math.floor(Math.random() * 5) + 1})" title="${this.escapeHtml(p.name || p.email)}">
                    ${(p.name || p.email || '?')[0].toUpperCase()}
                </div>
            `).join('')}
            ${event.participants.length > 3 ? `<div class="attendee-more">+${event.participants.length - 3}</div>` : ''}
           </div>`
        : '';

    return `
        <div class="event-card${isFocusTime ? ' focus-time' : ''}${relativeTime.class ? ' ' + relativeTime.class : ''}" data-event-id="${event.id}">
            <div class="event-time-row">
                <div class="event-time">${event.is_all_day ? 'All Day' : `${startTime} - ${endTime}`}</div>
                ${relativeTime.text ? `<div class="event-relative-time ${relativeTime.class}">${relativeTime.text}</div>` : ''}
            </div>
            <div class="event-title">${isFocusTime ? '🧘 ' : ''}${this.escapeHtml(event.title || '(No Title)')}</div>
            ${event.description ? `<div class="event-desc">${this.escapeHtml(this.stripHtml(event.description).substring(0, 100))}</div>` : ''}
            ${event.location ? `<div class="event-location">📍 ${this.escapeHtml(event.location)}</div>` : ''}
            ${participantsHtml}
            ${hasConferencing ? `
                <div class="event-actions">
                    <a href="${event.conferencing.url}" target="_blank" class="join-meeting-btn">
                        📹 Join Meeting
                    </a>
                </div>
            ` : ''}
        </div>
    `;
},

getRelativeTime(timestamp) {
    const now = Date.now() / 1000;
    const diff = timestamp - now;
    const diffMins = Math.floor(diff / 60);
    const diffHours = Math.floor(diff / 3600);

    // Past events
    if (diff < 0) {
        return { text: '', class: '' };
    }

    // Starting now (within 5 minutes)
    if (diffMins <= 5) {
        return { text: 'Starting now', class: 'starting-now' };
    }

    // Starting soon (within 30 minutes)
    if (diffMins <= 30) {
        return { text: `in ${diffMins} min`, class: 'starting-soon' };
    }

    // Within the hour
    if (diffMins < 60) {
        return { text: `in ${diffMins} min`, class: 'upcoming' };
    }

    // Within a few hours
    if (diffHours <= 3) {
        return { text: `in ${diffHours} hr${diffHours > 1 ? 's' : ''}`, class: 'upcoming' };
    }

    // Later today or tomorrow
    const eventDate = new Date(timestamp * 1000);
    const today = new Date();
    if (eventDate.toDateString() === today.toDateString()) {
        return { text: 'Today', class: '' };
    }

    const tomorrow = new Date(today);
    tomorrow.setDate(tomorrow.getDate() + 1);
    if (eventDate.toDateString() === tomorrow.toDateString()) {
        return { text: 'Tomorrow', class: '' };
    }

    return { text: '', class: '' };
},

formatEventTime(timestamp) {
    if (!timestamp) return '';
    const date = new Date(timestamp * 1000);
    return date.toLocaleTimeString('en-US', {
        hour: 'numeric',
        minute: '2-digit',
        hour12: true
    });
},

escapeHtml(str) {
    if (!str) return '';
    const div = document.createElement('div');
    div.textContent = str;
    return div.innerHTML;
},

stripHtml(html) {
    if (!html) return '';
    // Create a temporary element to parse HTML
    const tmp = document.createElement('div');
    tmp.innerHTML = html;
    // Get text content (strips all HTML tags)
    let text = tmp.textContent || tmp.innerText || '';
    // Clean up whitespace
    text = text.replace(/\s+/g, ' ').trim();
    return text;
},
});
