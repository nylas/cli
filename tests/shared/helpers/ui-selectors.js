/**
 * Shared selectors for Nylas UI (Web CLI Interface) E2E tests.
 *
 * The UI is the web-based interface for Nylas CLI (port 7363).
 * Different from Air (email client) and TUI (terminal).
 */

// General / App
exports.general = {
  app: '.app',
  toastContainer: '#toast-container',
  toast: '.toast',
};

// Header
exports.header = {
  header: '.header',
  logo: '.logo',
  brandText: '.brand-text',
  controls: '#header-controls',
  clientDropdown: '#client-dropdown',
  grantDropdown: '#grant-dropdown',
  themeBtn: '.theme-btn',
  selectedClient: '#selected-client',
  selectedGrant: '#selected-grant',
};

// Dropdowns
exports.dropdown = {
  btn: '.dropdown-btn',
  menu: '.dropdown-menu',
  item: '.dropdown-item',
  activeItem: '.dropdown-item.active',
  addNew: '.dropdown-item.add-new',
};

// Setup View (unconfigured state)
exports.setup = {
  view: '#setup-view',
  card: '.setup-card',
  apiKeyInput: '#api-key',
  regionSelect: '#region',
  submitBtn: '#setup-btn',
  errorMsg: '#setup-error',
};

// Dashboard View (configured state)
exports.dashboard = {
  view: '#dashboard-view',
  sidebar: '.sidebar',
  content: '.dashboard-content',
  header: '.dashboard-header',
  title: '.dashboard-title',
  subtitle: '.dashboard-subtitle',
  statusBadge: '.status-badge',
};

// Sidebar Navigation
exports.nav = {
  sidebar: '.sidebar-nav',
  section: '.nav-section',
  sectionTitle: '.nav-section-title',
  item: '.nav-item',
  activeItem: '.nav-item.active',
  // Specific nav items
  overview: '[data-page="overview"]',
  admin: '[data-page="admin"]',
  auth: '[data-page="auth"]',
  calendar: '[data-page="calendar"]',
  contacts: '[data-page="contacts"]',
  email: '[data-page="email"]',
  notetaker: '[data-page="notetaker"]',
  otp: '[data-page="otp"]',
  scheduler: '[data-page="scheduler"]',
  timezone: '[data-page="timezone"]',
  webhook: '[data-page="webhook"]',
};

// Pages
exports.pages = {
  overview: '#page-overview',
  admin: '#page-admin',
  auth: '#page-auth',
  calendar: '#page-calendar',
  contacts: '#page-contacts',
  email: '#page-email',
  notetaker: '#page-notetaker',
  otp: '#page-otp',
  scheduler: '#page-scheduler',
  timezone: '#page-timezone',
  webhook: '#page-webhook',
  activePage: '.page.active',
};

// Overview Page
exports.overview = {
  page: '#page-overview',
  configCard: '.card:has(.card-title:text("CONFIGURATION"))',
  accountsCard: '.card:has(.card-title:text("CONNECTED ACCOUNTS"))',
  resourcesCard: '.card:has(.card-title:text("RESOURCES"))',
  commandsCard: '.commands-card',
  configRegion: '#config-region',
  configClient: '#config-client',
  accountsList: '#accounts-list',
  accountItem: '.account-item',
  cmdCard: '.cmd-card',
};

// Cards (shared)
exports.card = {
  glass: '.glass-card',
  title: '.card-title',
  divider: '.card-divider',
};

// Form Elements
exports.form = {
  field: '.field',
  label: 'label',
  input: 'input',
  select: 'select',
  textarea: 'textarea',
  btnPrimary: '.btn-primary',
  btnSecondary: '.btn-secondary',
  errorMsg: '.error-msg',
};

// Command Panels (on command pages)
exports.commandPanel = {
  panel: '.command-panel',
  title: '.panel-title',
  form: '.panel-form',
  output: '.output-panel',
  outputContent: '.output-content',
  runBtn: '.run-btn',
  copyBtn: '.copy-btn',
};

// Empty States
exports.emptyState = {
  container: '.empty-state',
};

// Resources
exports.resources = {
  list: '.resources-list',
  item: '.resource-item',
  title: '.resource-title',
  desc: '.resource-desc',
};

// Command List (on command pages like Auth, Email, Calendar, etc.)
exports.commandList = {
  container: '[id$="-cmd-list"]',
  section: '.cmd-section',
  sectionTitle: '.cmd-section-title',
  item: '.cmd-item',
  activeItem: '.cmd-item.active',
  itemName: '.cmd-name',
  itemAction: '.cmd-action',
};

// Command Detail Panel
exports.commandDetail = {
  panel: '.command-detail',
  header: '.cmd-header',
  title: '.cmd-title',
  description: '.cmd-description',
  codeBlock: '.cmd-code',
  code: '.cmd-code code',
  actions: '.cmd-actions',
  runBtn: '.run-btn',
  rerunBtn: '.rerun-btn',
  outputPanel: '.output-panel',
  outputHeader: '.output-header',
  outputTitle: '.output-title',
  outputContent: '.output-content',
  outputPlaceholder: '.output-placeholder',
  copyBtn: '.copy-btn',
  clearBtn: '.clear-btn',
  lastRun: '.last-run',
};

// Quick Commands (on overview page)
exports.quickCommands = {
  container: '.commands-card',
  title: '.commands-title',
  list: '.cmd-cards',
  card: '.cmd-card',
  cardCode: '.cmd-card code',
  cardDescription: '.cmd-card-desc',
};

// Configuration Card (on overview page)
exports.config = {
  card: '.config-card',
  item: '.config-item',
  label: '.config-label',
  value: '.config-value',
  region: '#config-region',
  clientId: '#config-client',
  apiKey: '#config-api-key',
};

// Connected Accounts (on overview page)
exports.accounts = {
  card: '.accounts-card',
  list: '#accounts-list',
  item: '.account-item',
  avatar: '.account-avatar',
  email: '.account-email',
  provider: '.account-provider',
  badge: '.account-badge',
  defaultBadge: '.default-badge',
  hint: '.account-hint',
};

// Toast Notifications
exports.toast = {
  container: '#toast-container',
  toast: '.toast',
  success: '.toast-success',
  error: '.toast-error',
  info: '.toast-info',
  message: '.toast-message',
  icon: '.toast-icon',
};

// Loading States
exports.loading = {
  spinner: '.spinner',
  skeleton: '.skeleton',
  loadingText: '.loading-text',
};

// Status Indicators
exports.status = {
  badge: '.status-badge',
  connected: '.status-connected',
  disconnected: '.status-disconnected',
  indicator: '.status-indicator',
};
