// @ts-check
const { test, expect } = require('@playwright/test');
const sel = require('../../shared/helpers/chat-selectors');

/**
 * UI interaction tests for Nylas Chat.
 *
 * Tests user interactions with the chat interface including
 * input, sidebar, and slash commands.
 */

test.describe('Chat Input', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await expect(page.locator(sel.app)).toBeVisible();
  });

  test('textarea is focused on load', async ({ page }) => {
    const textarea = page.locator(sel.input.textarea);
    await expect(textarea).toBeFocused();
  });

  test('textarea auto-resizes on input', async ({ page }) => {
    const textarea = page.locator(sel.input.textarea);

    const initialHeight = await textarea.evaluate((el) => el.offsetHeight);

    // Type multiple lines
    await textarea.fill('Line 1\nLine 2\nLine 3\nLine 4');
    await textarea.dispatchEvent('input');

    const expandedHeight = await textarea.evaluate((el) => el.offsetHeight);
    expect(expandedHeight).toBeGreaterThanOrEqual(initialHeight);
  });

  test('empty message is not sent', async ({ page }) => {
    const textarea = page.locator(sel.input.textarea);
    await textarea.fill('');

    await page.locator(sel.input.sendBtn).click();

    // Welcome should still be visible (no message sent)
    await expect(page.locator(sel.chat.welcome)).toBeVisible();
  });

  test('Shift+Enter creates newline without sending', async ({ page }) => {
    const textarea = page.locator(sel.input.textarea);
    await textarea.fill('Line 1');
    await textarea.press('Shift+Enter');
    await textarea.pressSequentially('Line 2');

    const value = await textarea.inputValue();
    expect(value).toContain('Line 1');
    expect(value).toContain('Line 2');

    // Welcome should still be visible (not sent)
    await expect(page.locator(sel.chat.welcome)).toBeVisible();
  });
});

test.describe('Sidebar', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await expect(page.locator(sel.app)).toBeVisible();
  });

  test('new chat button resets conversation', async ({ page }) => {
    // Click new chat
    await page.locator(sel.sidebar.newChatBtn).click();

    // Title should be reset
    await expect(page.locator(sel.chat.title)).toHaveText('New conversation');
  });

  test('sidebar toggle button is in DOM', async ({ page }) => {
    // Hidden at desktop width (display: none), visible only on mobile via media query
    const toggleBtn = page.locator(sel.chat.toggleSidebar);
    await expect(toggleBtn).toBeAttached();
    await expect(toggleBtn).toHaveText('\u2630'); // hamburger icon
  });

  test('agent selector has options', async ({ page }) => {
    const select = page.locator(sel.sidebar.agentSelect);
    await expect(select).toBeVisible();

    // Should have at least one option
    const options = select.locator('option');
    const count = await options.count();
    expect(count).toBeGreaterThan(0);
  });
});

test.describe('Slash Commands', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await expect(page.locator(sel.app)).toBeVisible();
  });

  test('/help shows available commands', async ({ page }) => {
    const textarea = page.locator(sel.input.textarea);
    await textarea.fill('/help');
    await textarea.press('Enter');

    // Should show a system message with help content
    const systemMsg = page.locator(sel.message.system);
    await expect(systemMsg.first()).toBeVisible({ timeout: 5000 });

    // Help content should mention available commands
    const content = await systemMsg.first().textContent();
    expect(content).toContain('help');
  });

  test('/clear clears the messages', async ({ page }) => {
    const textarea = page.locator(sel.input.textarea);

    // Send /help first to populate messages
    await textarea.fill('/help');
    await textarea.press('Enter');
    await expect(page.locator(sel.message.system).first()).toBeVisible({ timeout: 5000 });

    // Now clear
    await textarea.fill('/clear');
    await textarea.press('Enter');

    // Old messages should be gone; only "Messages cleared." system message remains
    await page.waitForTimeout(500);
    const userAndAssistant = page.locator(`${sel.message.user}, ${sel.message.assistant}`);
    await expect(userAndAssistant).toHaveCount(0);

    // The "Messages cleared." confirmation should be shown
    const systemMsg = page.locator(sel.message.system);
    await expect(systemMsg).toHaveCount(1);
    const content = await systemMsg.first().textContent();
    expect(content).toContain('cleared');
  });

  test('/new resets to new conversation', async ({ page }) => {
    const textarea = page.locator(sel.input.textarea);
    await textarea.fill('/new');
    await textarea.press('Enter');

    // /new calls async API to create conversation, then resets UI
    await expect(page.locator(sel.chat.title)).toHaveText('New conversation', { timeout: 5000 });
    await expect(page.locator(sel.chat.welcome)).toBeVisible({ timeout: 5000 });
  });

  test('Tab completes slash commands', async ({ page }) => {
    const textarea = page.locator(sel.input.textarea);
    await textarea.fill('/hel');
    await textarea.press('Tab');

    const value = await textarea.inputValue();
    // Tab completion appends a trailing space for args
    expect(value.trim()).toBe('/help');
  });

  test('Tab completes partial command names', async ({ page }) => {
    const textarea = page.locator(sel.input.textarea);
    await textarea.fill('/cal');
    await textarea.press('Tab');

    const value = await textarea.inputValue();
    expect(value.trim()).toBe('/calendar');
  });
});

test.describe('API Health', () => {
  test('health endpoint returns ok', async ({ request }) => {
    const resp = await request.get('/api/health');
    expect(resp.status()).toBe(200);

    const body = await resp.json();
    expect(body.status).toBe('ok');
  });

  test('config endpoint returns agent info', async ({ request }) => {
    const resp = await request.get('/api/config');
    expect(resp.status()).toBe(200);

    const body = await resp.json();
    expect(body.agent).toBeTruthy();
    expect(body.available).toBeInstanceOf(Array);
    expect(body.available.length).toBeGreaterThan(0);
  });

  test('conversations endpoint returns list', async ({ request }) => {
    const resp = await request.get('/api/conversations');
    expect(resp.status()).toBe(200);

    // API returns a raw array of conversation summaries
    const body = await resp.json();
    expect(body).toBeInstanceOf(Array);
  });

  test('command endpoint rejects GET', async ({ request }) => {
    const resp = await request.get('/api/command');
    expect(resp.status()).toBe(405);
  });

  test('chat endpoint rejects GET', async ({ request }) => {
    const resp = await request.get('/api/chat');
    expect(resp.status()).toBe(405);
  });
});
