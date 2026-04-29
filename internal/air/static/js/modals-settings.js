// Settings modal helpers - extracted from modals_settings.gohtml inline script

// Snooze Modal Functions
function closeSnoozeModal() {
    document.getElementById('snoozePickerOverlay').classList.add('hidden');
}

function openSnoozeModal(emailId) {
    window.snoozeEmailId = emailId;
    updateSnoozeTimes();
    document.getElementById('snoozePickerOverlay').classList.remove('hidden');
}

function updateSnoozeTimes() {
    var now = new Date();

    // Later today (6 PM)
    var laterToday = new Date(now);
    laterToday.setHours(18, 0, 0, 0);
    if (laterToday <= now) laterToday.setDate(laterToday.getDate() + 1);
    document.getElementById('snoozeLaterToday').textContent = laterToday.toLocaleTimeString('en-US', { hour: 'numeric', minute: '2-digit' });

    // Tomorrow (9 AM)
    var tomorrow = new Date(now);
    tomorrow.setDate(tomorrow.getDate() + 1);
    tomorrow.setHours(9, 0, 0, 0);
    document.getElementById('snoozeTomorrow').textContent = 'Tomorrow ' + tomorrow.toLocaleTimeString('en-US', { hour: 'numeric', minute: '2-digit' });

    // Next week (Monday 9 AM)
    var nextWeek = new Date(now);
    var daysUntilMonday = (8 - now.getDay()) % 7 || 7;
    nextWeek.setDate(nextWeek.getDate() + daysUntilMonday);
    nextWeek.setHours(9, 0, 0, 0);
    document.getElementById('snoozeNextWeek').textContent = 'Monday ' + nextWeek.toLocaleTimeString('en-US', { hour: 'numeric', minute: '2-digit' });

    // Weekend (Saturday 10 AM)
    var weekend = new Date(now);
    var daysUntilSat = (6 - now.getDay() + 7) % 7 || 7;
    weekend.setDate(weekend.getDate() + daysUntilSat);
    weekend.setHours(10, 0, 0, 0);
    document.getElementById('snoozeWeekend').textContent = 'Sat ' + weekend.toLocaleTimeString('en-US', { hour: 'numeric', minute: '2-digit' });
}

function snoozeEmail(option) {
    console.log('Snoozing email:', window.snoozeEmailId, 'until:', option);
    showToast('Email snoozed', 'success');
    closeSnoozeModal();
}

function openCustomSnooze() {
    console.log('Opening custom snooze picker');
}

// Send Later Modal Functions
function closeSendLaterModal() {
    document.getElementById('sendLaterOverlay').classList.add('hidden');
}

function openSendLaterModal() {
    // Set default date to tomorrow
    var tomorrow = new Date();
    tomorrow.setDate(tomorrow.getDate() + 1);
    document.getElementById('sendLaterDate').value = tomorrow.toISOString().split('T')[0];
    document.getElementById('sendLaterOverlay').classList.remove('hidden');
}

function scheduleSend(option) {
    console.log('Scheduling send:', option);
    showToast('Email scheduled', 'success');
    closeSendLaterModal();
}

function scheduleCustomSend() {
    var date = document.getElementById('sendLaterDate').value;
    var time = document.getElementById('sendLaterTime').value;
    console.log('Scheduling custom send:', date, time);
    showToast('Email scheduled for ' + new Date(date + 'T' + time).toLocaleString(), 'success');
    closeSendLaterModal();
}
