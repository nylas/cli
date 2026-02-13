/**
 * Semantic selectors for Nylas Chat E2E tests.
 * Uses IDs and semantic attributes from index.gohtml and chat.js.
 */
module.exports = {
  app: '.app',
  sidebar: {
    root: '#sidebar',
    header: '.sidebar-header',
    title: '.sidebar-header h2',
    newChatBtn: '#btn-new-chat',
    conversationList: '#conversation-list',
    agentSelect: '#agent-select',
    agentLabel: '.agent-label',
  },
  chat: {
    main: '.chat-main',
    header: '#chat-header',
    title: '#chat-title',
    toggleSidebar: '#btn-toggle-sidebar',
    messages: '#messages',
    welcome: '#welcome',
    welcomeTitle: '#welcome h2',
    suggestions: '.suggestions',
    suggestionBtn: '.suggestion',
  },
  input: {
    form: '#chat-form',
    textarea: '#chat-input',
    sendBtn: '#btn-send',
  },
  message: {
    user: '.message.user',
    assistant: '.message.assistant',
    system: '.message.system',
    content: '.message-content',
    streaming: '.message.streaming',
  },
  tool: {
    indicator: '.tool-indicator',
    name: '.tool-name',
    details: '.tool-details',
  },
  approval: {
    card: '.approval-card',
    header: '.approval-header',
    preview: '.approval-preview',
    approveBtn: '.btn-approve',
    rejectBtn: '.btn-reject',
    status: '.approval-status',
    resolved: '.approval-card.resolved',
  },
  thinking: '.thinking',
};
