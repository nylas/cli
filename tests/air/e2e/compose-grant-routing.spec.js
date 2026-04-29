// @ts-check
const { test, expect } = require('@playwright/test');
const selectors = require('../../shared/helpers/air-selectors');

/**
 * Verifies that the compose flow pins the send to the page's rendered grant
 * (so the user's visible "from" account matches the backend grant) instead of
 * relying on whatever the persisted default happens to be at send time.
 *
 * Regression test for: Air sending the wrong account when the persisted
 * default grant drifts out of sync with the displayed account.
 */
test.describe('Compose grant routing', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await expect(page.locator(selectors.general && selectors.general.app ? selectors.general.app : 'body')).toBeVisible();
    await page.waitForLoadState('domcontentloaded');
  });

  test('body exposes default grant id, provider, and email as data attrs', async ({ page }) => {
    const grantId = await page.evaluate(() => document.body.dataset.defaultGrantId);
    const provider = await page.evaluate(() => document.body.dataset.defaultGrantProvider);
    const email = await page.evaluate(() => document.body.dataset.defaultGrantEmail);

    expect(grantId, 'data-default-grant-id must be present so compose can pin to it').toBeTruthy();
    expect(provider, 'data-default-grant-provider must be present').toBeTruthy();
    expect(email, 'data-default-grant-email must be present').toBeTruthy();
  });

  test('send request includes grant_id from body data-attr', async ({ page }) => {
    const expectedGrantId = await page.evaluate(() => document.body.dataset.defaultGrantId);
    expect(expectedGrantId, 'page must render with a grant id').toBeTruthy();

    let capturedBody = null;
    await page.route('**/api/send', async (route) => {
      try {
        capturedBody = JSON.parse(route.request().postData() || '{}');
      } catch (_) {
        capturedBody = null;
      }
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ success: true, message_id: 'test-msg-1', message: 'ok' }),
      });
    });

    await page.click(selectors.email.composeBtn);
    await expect(page.locator(selectors.compose.modal)).toBeVisible();
    await page.fill(selectors.compose.to, 'recipient@example.com');
    await page.fill(selectors.compose.subject, 'Routing test');
    await page.fill(selectors.compose.body, 'Verifying grant_id propagation.');
    await page.click(selectors.compose.sendBtn);

    await expect.poll(() => capturedBody, {
      message: 'POST /api/send was never intercepted',
      timeout: 5000,
    }).not.toBeNull();

    expect(capturedBody.grant_id, 'send payload must include grant_id').toBe(expectedGrantId);
    expect(capturedBody.to).toEqual([{ email: 'recipient@example.com' }]);
    expect(capturedBody.subject).toBe('Routing test');
  });
});
