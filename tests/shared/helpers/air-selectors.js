/**
 * Shared selectors for Nylas Air E2E tests.
 *
 * Using data-testid attributes for stability.
 * Falls back to semantic selectors where data-testid is not available.
 */

// Navigation
exports.nav = {
  main: '[data-testid="main-nav"]',
  tabEmail: '[data-testid="nav-tab-email"]',
  tabCalendar: '[data-testid="nav-tab-calendar"]',
  tabContacts: '[data-testid="nav-tab-contacts"]',
  tabNotetaker: '.nav-tab:has-text("Notetaker")',
  logo: '.logo',
  searchTrigger: '.search-trigger',
  settingsBtn: '.settings-btn',
  accountSwitcher: '[data-testid="account-switcher"]',
  accountDropdown: '#accountDropdown',
};

// Views
exports.views = {
  email: '[data-testid="email-view"]',
  calendar: '[data-testid="calendar-view"]',
  contacts: '[data-testid="contacts-view"]',
  notetaker: '#notetakerView',
};

// Email View
exports.email = {
  view: '[data-testid="email-view"]',
  folderSidebar: '[data-testid="folder-sidebar"]',
  folderList: '#folderList',
  folderItem: '.folder-item',
  composeBtn: '[data-testid="compose-btn"]',
  emailListContainer: '[data-testid="email-list-container"]',
  emailList: '[data-testid="email-list"]',
  emailItem: '.email-item',
  emailSkeleton: '.email-skeleton',
  preview: '[data-testid="email-preview"]',
  emptyState: '.empty-state',
  filterTabs: '#emailFilterTabs',
  filterTab: '.filter-tab',
};

// Compose Modal
exports.compose = {
  modal: '[data-testid="compose-modal"]',
  header: '.compose-header',
  title: '.compose-title',
  closeBtn: '[data-testid="compose-close"]',
  to: '[data-testid="compose-to"]',
  cc: '[data-testid="compose-cc"]',
  bcc: '[data-testid="compose-bcc"]',
  ccField: '[data-testid="compose-cc-field"]',
  bccField: '[data-testid="compose-bcc-field"]',
  ccBccToggle: '[data-testid="compose-cc-bcc-toggle"]',
  subject: '[data-testid="compose-subject"]',
  body: '[data-testid="compose-body"]',
  sendBtn: '[data-testid="compose-send"]',
  saveBtn: '#composeSave',
  discardBtn: '#composeDiscard',
};

// Command Palette
exports.commandPalette = {
  overlay: '[data-testid="command-palette"]',
  input: '[data-testid="command-palette-input"]',
  results: '.command-results',
  item: '.command-item',
  section: '.command-section',
};

// Search Overlay
exports.search = {
  overlay: '[data-testid="search-overlay"]',
  input: '[data-testid="search-input"]',
  filters: '.search-filters',
  filterChip: '.search-filter-chip',
  suggestions: '#searchSuggestions',
  recentGroup: '#recentSearchesGroup',
  resultsSection: '#searchResultsSection',
};

// Calendar View
exports.calendar = {
  view: '[data-testid="calendar-view"]',
  sidebar: '.calendar-view .sidebar',
  newEventBtn: '[data-testid="new-event-btn"]',
  calendarsList: '#calendarsList',
  grid: '[data-testid="calendar-grid"]',
  dayHeader: '.calendar-day-header',
  day: '.calendar-day',
  today: '.calendar-day.today',
  eventsPanel: '[data-testid="events-panel"]',
  eventsList: '#eventsList',
  conflictsPanel: '#conflictsPanel',
  // Event card elements
  eventCard: '.event-card',
  eventEditBtn: '.event-edit-btn',
  joinMeetingBtn: '.join-meeting-btn',
  eventCountBadge: '.event-count-badge',
  eventRelativeTime: '.event-relative-time',
  todayIndicator: '.today-indicator',
};

// Contacts View
exports.contacts = {
  view: '[data-testid="contacts-view"]',
  newContactBtn: '.contacts-view .compose-btn',
  list: '[data-testid="contacts-list"]',
  item: '.contact-item',
  detail: '[data-testid="contact-detail"]',
};

// Settings Modal
exports.settings = {
  overlay: '[data-testid="settings-overlay"]',
  modal: '.settings-modal',
  closeBtn: '[data-testid="settings-close"]',
  aiSection: '[data-testid="settings-ai-section"]',
  themeSection: '[data-testid="settings-theme-section"]',
  themeBtn: '.theme-btn',
  saveBtn: '.settings-save-btn',
  resetBtn: '.settings-reset-btn',
};

// Event Modal
exports.eventModal = {
  overlay: '#eventModalOverlay',
  modal: '.event-modal',
  title: '#eventTitle',
  startDate: '#eventStartDate',
  startTime: '#eventStartTime',
  endDate: '#eventEndDate',
  endTime: '#eventEndTime',
  allDay: '#eventAllDay',
  location: '#eventLocation',
  description: '#eventDescription',
  participants: '#eventParticipants',
  busy: '#eventBusy',
  saveBtn: '#eventSaveBtn',
  deleteBtn: '#eventDeleteBtn',
  closeBtn: '.event-modal-close',
};

// Contact Modal
exports.contactModal = {
  overlay: '#contactModalOverlay',
  modal: '.contact-modal',
  givenName: '#contactGivenName',
  surname: '#contactSurname',
  emailInput: '.contact-email-input',
  phoneInput: '.contact-phone-input',
  company: '#contactCompany',
  jobTitle: '#contactJobTitle',
  notes: '#contactNotes',
  saveBtn: '#contactSaveBtn',
  closeBtn: '.contact-modal-close',
};

// Snooze Modal
exports.snooze = {
  overlay: '#snoozePickerOverlay',
  modal: '.snooze-picker-modal',
  option: '.snooze-option',
  closeBtn: '.snooze-picker-close',
};

// Keyboard Shortcut Overlay
exports.shortcuts = {
  overlay: '[data-testid="shortcut-overlay"]',
  modal: '.shortcut-modal',
  closeBtn: '[data-testid="shortcut-close"]',
  group: '.shortcut-group',
  item: '.shortcut-item',
};

// Context Menu
exports.contextMenu = {
  menu: '[data-testid="context-menu"]',
  item: '.context-menu-item',
};

// Toast System
exports.toast = {
  container: '[data-testid="toast-container"]',
  toast: '.toast',
  success: '.toast.success',
  error: '.toast.error',
  info: '.toast.info',
  warning: '.toast.warning',
};

// Focus Mode
exports.focusMode = {
  toggle: '#focusModeToggle',
};

// Status Bar
exports.statusBar = {
  bar: '.status-bar',
};

// General
exports.general = {
  app: '.app',
  mainLayout: '.main-layout',
  liveRegion: '#announcer',
  skipLink: '.skip-link',
};
