// Actions registry - registers all data-action handlers for event delegation.
// This file must load after all domain modules (email, calendar, contacts, etc.)
// and after event-delegation.js.

document.addEventListener('DOMContentLoaded', function() {

    // ========== NAVIGATION ==========
    Actions.register('nav-email', function(target, e) {
        if (typeof showView === 'function') showView('email', e);
    });
    Actions.register('nav-calendar', function(target, e) {
        if (typeof showView === 'function') showView('calendar', e);
    });
    Actions.register('nav-contacts', function(target, e) {
        if (typeof showView === 'function') showView('contacts', e);
    });
    Actions.register('nav-notetaker', function(target, e) {
        if (typeof showView === 'function') showView('notetaker', e);
    });
    Actions.register('show-email-view', function() {
        if (typeof showView === 'function') showView('email');
    });
    Actions.register('show-rules-policy', function() {
        if (typeof showView === 'function') showView('rulesPolicy');
    });

    // ========== COMMAND PALETTE / SEARCH ==========
    Actions.register('toggle-command-palette', function() {
        if (typeof toggleCommandPalette === 'function') toggleCommandPalette();
    });
    Actions.register('stop-propagation', function(target, e) {
        e.stopPropagation();
    });
    Actions.register('close-search-overlay', function(target, e) {
        if (e.target === target && typeof closeSearch === 'function') closeSearch();
    });
    Actions.register('search-filter', function(target) {
        if (typeof toggleSearchFilter === 'function') toggleSearchFilter(target);
    });
    Actions.register('clear-recent-searches', function() {
        if (typeof RecentSearches !== 'undefined') RecentSearches.clear();
    });

    // ========== SETTINGS ==========
    Actions.register('toggle-settings', function() {
        if (typeof toggleSettings === 'function') toggleSettings();
    });
    Actions.register('close-settings-overlay', function(target, e) {
        if (typeof closeSettingsOnOverlay === 'function') closeSettingsOnOverlay(e);
    });
    Actions.register('reset-settings', function() {
        if (typeof resetSettings === 'function') resetSettings();
    });
    Actions.register('save-settings', function() {
        if (typeof saveSettings === 'function') saveSettings();
    });
    Actions.register('set-theme', function(target) {
        if (typeof setTheme === 'function') setTheme(target.dataset.theme);
    });

    // ========== ACCOUNT ==========
    Actions.register('toggle-account-dropdown', function() {
        if (typeof toggleAccountDropdown === 'function') toggleAccountDropdown();
    });
    Actions.register('switch-account', function(target) {
        if (typeof switchAccount === 'function') switchAccount(target.dataset.grantId);
    });
    Actions.register('add-account', function() {
        if (typeof addAccount === 'function') addAccount();
    });
    Actions.register('show-setup-instructions', function() {
        if (typeof showSetupInstructions === 'function') showSetupInstructions();
    });
    Actions.register('close-setup-banner', function() {
        if (typeof closeSetupBanner === 'function') closeSetupBanner();
    });

    // ========== COMPOSE ==========
    Actions.register('toggle-compose', function() {
        if (typeof toggleCompose === 'function') toggleCompose();
    });
    Actions.register('toggle-cc-bcc', function() {
        if (typeof toggleCcBcc === 'function') toggleCcBcc();
    });

    // ========== FOCUS MODE ==========
    Actions.register('toggle-focus-mode', function() {
        if (typeof toggleFocusMode === 'function') toggleFocusMode();
    });

    // ========== SHORTCUT OVERLAY ==========
    Actions.register('close-shortcut-overlay', function(target, e) {
        // If the overlay itself was clicked (backdrop), close; button click also closes
        if (e.target === target || target.classList.contains('shortcut-close')) {
            if (typeof closeShortcutOverlay === 'function') closeShortcutOverlay();
        }
    });

    // ========== CONTEXT MENU ==========
    Actions.register('context-reply', function() {
        if (typeof handleContextAction === 'function') handleContextAction('reply');
    });
    Actions.register('context-forward', function() {
        if (typeof handleContextAction === 'function') handleContextAction('forward');
    });
    Actions.register('context-archive', function() {
        if (typeof handleContextAction === 'function') handleContextAction('archive');
    });
    Actions.register('context-star', function() {
        if (typeof handleContextAction === 'function') handleContextAction('star');
    });
    Actions.register('context-delete', function() {
        if (typeof handleContextAction === 'function') handleContextAction('delete');
    });
    Actions.register('context-snooze', function() {
        if (typeof SnoozeManager !== 'undefined') SnoozeManager.openForEmail(window.contextMenuEmailId);
    });

    // ========== CALENDAR ==========
    Actions.register('create-event', function() {
        if (typeof EventModal !== 'undefined' && typeof EventModal.openNew === 'function') {
            EventModal.openNew();
        } else {
            console.warn('create-event: EventModal.openNew is not available');
        }
    });
    Actions.register('open-find-time-modal', function() {
        if (typeof openFindTimeModal === 'function') openFindTimeModal();
    });
    Actions.register('close-event-modal-overlay', function(target, e) {
        if (e.target === target && typeof closeEventModal === 'function') closeEventModal();
    });
    Actions.register('close-event-modal', function() {
        if (typeof closeEventModal === 'function') closeEventModal();
    });
    Actions.register('delete-event', function() {
        if (typeof deleteEvent === 'function') deleteEvent();
    });
    Actions.register('save-event', function() {
        if (typeof saveEvent === 'function') saveEvent();
    });
    Actions.register('toggle-all-day', function() {
        if (typeof toggleAllDay === 'function') toggleAllDay();
    });

    // ========== CONTACTS ==========
    Actions.register('close-contact-modal-overlay', function(target, e) {
        if (e.target === target && typeof ContactsManager !== 'undefined') ContactsManager.closeModal();
    });
    Actions.register('close-contact-modal', function() {
        if (typeof ContactsManager !== 'undefined') ContactsManager.closeModal();
    });
    Actions.register('contact-add-email-row', function() {
        if (typeof ContactsManager !== 'undefined') ContactsManager.addEmailRow();
    });
    Actions.register('contact-add-phone-row', function() {
        if (typeof ContactsManager !== 'undefined') ContactsManager.addPhoneRow();
    });
    Actions.register('save-contact', function() {
        if (typeof ContactsManager !== 'undefined') ContactsManager.saveContact();
    });

    // ========== NOTETAKER ==========
    Actions.register('filter-notetakers', function(target) {
        if (typeof filterNotetakers === 'function') filterNotetakers(target.dataset.filter, target);
    });
    Actions.register('refresh-notetakers', function() {
        if (typeof refreshNotetakers === 'function') refreshNotetakers();
    });
    Actions.register('open-join-meeting-modal', function() {
        if (typeof openJoinMeetingModal === 'function') openJoinMeetingModal();
    });
    Actions.register('close-join-meeting-modal', function() {
        if (typeof closeJoinMeetingModal === 'function') closeJoinMeetingModal();
    });
    Actions.register('join-meeting', function() {
        if (typeof joinMeeting === 'function') joinMeeting();
    });

    // ========== RULES / POLICY ==========
    Actions.register('rules-refresh-policies', function() {
        if (typeof RulesPolicyManager !== 'undefined') RulesPolicyManager.refreshPolicies();
    });
    Actions.register('rules-refresh-rules', function() {
        if (typeof RulesPolicyManager !== 'undefined') RulesPolicyManager.refreshRules();
    });

    // ========== SNOOZE ==========
    Actions.register('close-snooze-overlay', function(target, e) {
        if (e.target === target && typeof closeSnoozeModal === 'function') closeSnoozeModal();
    });
    Actions.register('close-snooze-modal', function() {
        if (typeof closeSnoozeModal === 'function') closeSnoozeModal();
    });
    Actions.register('snooze-email', function(target) {
        if (typeof snoozeEmail === 'function') snoozeEmail(target.dataset.snoozeOption);
    });
    Actions.register('open-custom-snooze', function() {
        if (typeof openCustomSnooze === 'function') openCustomSnooze();
    });

    // ========== SEND LATER ==========
    Actions.register('close-send-later-overlay', function(target, e) {
        if (e.target === target && typeof closeSendLaterModal === 'function') closeSendLaterModal();
    });
    Actions.register('close-send-later-modal', function() {
        if (typeof closeSendLaterModal === 'function') closeSendLaterModal();
    });
    Actions.register('schedule-send', function(target) {
        if (typeof scheduleSend === 'function') scheduleSend(target.dataset.sendOption);
    });
    Actions.register('schedule-custom-send', function() {
        if (typeof scheduleCustomSend === 'function') scheduleCustomSend();
    });

    // ========== NOTETAKER SOURCE MODAL ==========
    Actions.register('open-notetaker-source-modal', function() {
        if (typeof openAddNotetakerSourceModal === 'function') openAddNotetakerSourceModal();
    });
    Actions.register('close-notetaker-source-overlay', function(target, e) {
        if (e.target === target && typeof closeNotetakerSourceModal === 'function') closeNotetakerSourceModal();
    });
    Actions.register('close-notetaker-source-modal', function() {
        if (typeof closeNotetakerSourceModal === 'function') closeNotetakerSourceModal();
    });
    Actions.register('save-notetaker-source', function() {
        if (typeof saveNotetakerSource === 'function') saveNotetakerSource();
    });

    // ========== JS-BUILT MODAL ACTIONS ==========
    // AI Summary Modal
    Actions.register('ai-summary-close', function() {
        if (typeof EmailListManager !== 'undefined') EmailListManager.closeAISummaryModal();
    });
    Actions.register('ai-summary-copy', function() {
        if (typeof EmailListManager !== 'undefined') EmailListManager.copyAISummary();
    });

    // Find Time Modal
    Actions.register('find-time-close', function() {
        if (typeof FindTimeModal !== 'undefined') FindTimeModal.close();
    });
    Actions.register('find-time-search', function() {
        if (typeof FindTimeModal !== 'undefined') FindTimeModal.search();
    });
    Actions.register('find-time-create-event', function() {
        if (typeof FindTimeModal !== 'undefined') FindTimeModal.createEvent();
    });

    // Calendar event edit (event card)
    Actions.register('calendar-edit-event', function(target, e) {
        e.stopPropagation();
        if (typeof CalendarManager !== 'undefined') CalendarManager.openEditModal(target.dataset.eventId);
    });

    // Calendar join meeting link (stop propagation only; href handles navigation)
    Actions.register('join-meeting-link', function(target, e) {
        e.stopPropagation();
    });

    // Contacts load more
    Actions.register('contacts-load-more', function() {
        if (typeof ContactsManager !== 'undefined') ContactsManager.loadContacts(true);
    });
    Actions.register('contacts-retry', function() {
        if (typeof ContactsManager !== 'undefined') ContactsManager.loadContacts();
    });

    // Contact row management (add/remove rows in JS-built modals)
    Actions.register('contact-remove-row', function(target) {
        var row = target.parentElement;
        if (row) row.remove();
    });

    // Notetaker action buttons (JS-built cards). The external URL comes
    // from the Nylas API response; escapeHtml at the writer side stops
    // attribute-context injection but does NOT block javascript:/data:
    // schemes. Validate the scheme here before calling window.open so an
    // attacker-influenced API response cannot run script in our origin.
    Actions.register('notetaker-open-external', function(target) {
        var url = target.dataset.externalUrl;
        if (url && /^https?:\/\//i.test(url)) {
            window.open(url, '_blank', 'noopener,noreferrer');
        } else if (url) {
            console.warn('notetaker-open-external: refusing non-http(s) URL', url);
        }
    });
    Actions.register('notetaker-play', function(target) {
        if (typeof NotetakerModule !== 'undefined') NotetakerModule.playRecording(target.dataset.notId);
    });
    Actions.register('notetaker-transcript', function(target) {
        if (typeof NotetakerModule !== 'undefined') NotetakerModule.viewTranscript(target.dataset.notId);
    });
    Actions.register('notetaker-summarize', function(target) {
        if (typeof NotetakerModule !== 'undefined') NotetakerModule.summarize(target.dataset.notId);
    });
    Actions.register('notetaker-cancel', function(target) {
        if (typeof NotetakerModule !== 'undefined') NotetakerModule.cancel(target.dataset.notId);
    });

    // Snooze picker (JS-built) close button
    Actions.register('snooze-picker-close', function() {
        if (typeof SnoozeManager !== 'undefined') SnoozeManager.hidePicker();
    });
    Actions.register('snooze-picker-custom', function() {
        if (typeof SnoozeManager !== 'undefined') SnoozeManager.snoozeCustom();
    });

    // Shortcuts modal (JS-built) close button
    Actions.register('shortcuts-modal-close', function() {
        if (typeof ShortcutsManager !== 'undefined') ShortcutsManager.hideModal();
    });

    // Command palette shortcuts modal close
    Actions.register('cmd-shortcuts-close', function() {
        var modal = document.getElementById('shortcutsHelpModal');
        if (modal) modal.classList.add('hidden');
    });
    Actions.register('cmd-shortcuts-stop-propagation', function(target, e) {
        e.stopPropagation();
    });

    // Scheduled send cancel picker
    Actions.register('schedule-picker-cancel', function(target) {
        var modal = target.closest('.schedule-picker-modal');
        if (modal) modal.remove();
    });

    // Undo send
    Actions.register('undo-send', function() {
        if (typeof UndoSendManager !== 'undefined') UndoSendManager.undo();
    });

    // Templates manager (JS-built)
    Actions.register('templates-close', function() {
        if (typeof TemplatesManager !== 'undefined') TemplatesManager.close();
    });
    Actions.register('templates-show-create', function() {
        if (typeof TemplatesManager !== 'undefined') TemplatesManager.showCreate();
    });
    Actions.register('templates-cancel-variables', function() {
        if (typeof TemplatesManager !== 'undefined') TemplatesManager.cancelVariables();
    });
    Actions.register('templates-apply-variables', function() {
        if (typeof TemplatesManager !== 'undefined') TemplatesManager.applyVariables();
    });
    Actions.register('templates-hide-create', function() {
        if (typeof TemplatesManager !== 'undefined') TemplatesManager.hideCreate();
    });

    // Recent search item click
    Actions.register('execute-search', function(target) {
        var query = target.dataset.searchQuery;
        if (query && typeof executeSearch === 'function') executeSearch(query);
    });
    Actions.register('remove-recent-search', function(target, e) {
        e.stopPropagation();
        var query = target.dataset.searchQuery;
        if (query && typeof RecentSearches !== 'undefined') RecentSearches.remove(query);
    });

});
