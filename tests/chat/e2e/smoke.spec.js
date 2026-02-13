// @ts-check
const { test, expect } = require('@playwright/test');
const sel = require('../../shared/helpers/chat-selectors');

/**
 * Smoke tests for Nylas Chat.
 *
 * Verifies the application loads correctly and all major
 * UI elements are present and functional.
 */

test.describe('Smoke Tests', () => {
  test.beforeEach(async ({ page }) => {
    page.on('pageerror', (error) => {
      console.error('Page error:', error.message);
    });
    await page.goto('/');
    await expect(page.locator(sel.app)).toBeVisible();
  });

  test('page loads without JavaScript errors', async ({ page }) => {
    const errors = [];
    page.on('pageerror', (error) => errors.push(error.message));

    await page.waitForLoadState('domcontentloaded');
    await page.waitForTimeout(500);

    const criticalErrors = errors.filter((e) => {
      if (e.includes('Failed to load resource')) return false;
      if (e.includes('404')) return false;
      return true;
    });

    expect(criticalErrors).toHaveLength(0);
  });

  test('has correct page title', async ({ page }) => {
    await expect(page).toHaveTitle('Nylas Chat');
  });

  test('has viewport meta tag', async ({ page }) => {
    const viewport = page.locator('meta[name="viewport"]');
    await expect(viewport).toHaveAttribute(
      'content',
      expect.stringContaining('width=device-width')
    );
  });

  test('sidebar is visible with header', async ({ page }) => {
    const sidebar = page.locator(sel.sidebar.root);
    await expect(sidebar).toBeVisible();

    await expect(page.locator(sel.sidebar.title)).toHaveText('Nylas Chat');
    await expect(page.locator(sel.sidebar.newChatBtn)).toBeVisible();
  });

  test('agent selector is present in sidebar', async ({ page }) => {
    await expect(page.locator(sel.sidebar.agentLabel)).toBeVisible();
    await expect(page.locator(sel.sidebar.agentSelect)).toBeVisible();
  });

  test('chat main area is visible', async ({ page }) => {
    const main = page.locator(sel.chat.main);
    await expect(main).toBeVisible();

    await expect(page.locator(sel.chat.title)).toHaveText('New conversation');
    // Toggle button is in DOM but hidden at desktop width (shown on mobile via media query)
    await expect(page.locator(sel.chat.toggleSidebar)).toBeAttached();
  });

  test('welcome screen is displayed', async ({ page }) => {
    const welcome = page.locator(sel.chat.welcome);
    await expect(welcome).toBeVisible();

    await expect(page.locator(sel.chat.welcomeTitle)).toHaveText(
      'Welcome to Nylas Chat'
    );
  });

  test('suggestion buttons are present', async ({ page }) => {
    const suggestions = page.locator(sel.chat.suggestionBtn);
    await expect(suggestions).toHaveCount(3);

    await expect(suggestions.nth(0)).toContainText('Unread emails');
    await expect(suggestions.nth(1)).toContainText("Today's meetings");
    await expect(suggestions.nth(2)).toContainText('Search emails');
  });

  test('input area is present and functional', async ({ page }) => {
    const textarea = page.locator(sel.input.textarea);
    await expect(textarea).toBeVisible();
    await expect(textarea).toHaveAttribute(
      'placeholder',
      expect.stringContaining('emails')
    );

    const sendBtn = page.locator(sel.input.sendBtn);
    await expect(sendBtn).toBeVisible();
  });

  test('CSS files loaded correctly', async ({ page }) => {
    const chatCss = page.locator('link[href="/css/chat.css"]');
    await expect(chatCss).toBeAttached();

    const componentsCss = page.locator('link[href="/css/components.css"]');
    await expect(componentsCss).toBeAttached();
  });

  test('JavaScript files loaded correctly', async ({ page }) => {
    const scripts = ['markdown.js', 'api.js', 'commands.js', 'sidebar.js', 'chat.js'];
    for (const script of scripts) {
      const el = page.locator(`script[src="/js/${script}"]`);
      await expect(el).toBeAttached();
    }
  });
});
