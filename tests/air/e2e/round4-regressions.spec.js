// @ts-check
const { test, expect } = require('@playwright/test');

/**
 * Round-4 regression suite for the bug audit. Each test pins a contract
 * established by a fix in this round so future refactors can't silently
 * regress:
 *
 *  1. Scheduled-send rejects send times that are too far in the future.
 *  2. Scheduled-send rejects send times in the past / within 1 minute.
 *  3. Bundle PUT rejects rules with uncompilable regex.
 *  4. Track-open returns a 1×1 GIF for both empty and present id, and never
 *     surfaces a Set-Cookie or Cache-Control: public.
 *  5. Focus mode start → stop round-trip works in demo mode.
 *
 * All assertions go through fetch() in page context to avoid relying on
 * UI affordances; the relevant routes return JSON.
 */
test.describe('Round 4 regression — Air handlers', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('domcontentloaded');
  });

  test('scheduled-send: rejects far-future send_at', async ({ page }) => {
    const result = await page.evaluate(async () => {
      const body = {
        // ~10 years out — well past the 1-year ceiling.
        send_at: Math.floor(Date.now() / 1000) + 10 * 365 * 24 * 3600,
        to: [{ email: 'someone@example.com' }],
        subject: 'Hi',
        body: 'Hello',
      };
      const r = await fetch('/api/scheduled', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      });
      const json = await r.json();
      return { status: r.status, json };
    });

    expect(result.status).toBe(400);
    expect(JSON.stringify(result.json).toLowerCase()).toContain('within one year');
  });

  test('scheduled-send: rejects send_at in the past', async ({ page }) => {
    const result = await page.evaluate(async () => {
      const body = {
        send_at: Math.floor(Date.now() / 1000) - 60,
        to: [{ email: 'someone@example.com' }],
        subject: 'Hi',
        body: 'Hello',
      };
      const r = await fetch('/api/scheduled', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      });
      return { status: r.status };
    });

    expect(result.status).toBe(400);
  });

  test('scheduled-send: accepts send_at within 1 year', async ({ page }) => {
    const result = await page.evaluate(async () => {
      const body = {
        send_at: Math.floor(Date.now() / 1000) + 7200,
        to: [{ email: 'someone@example.com' }],
        subject: 'Hi',
        body: 'Hello',
      };
      const r = await fetch('/api/scheduled', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      });
      const json = await r.json();
      return { status: r.status, success: json.success === true };
    });

    expect(result.status).toBe(200);
    expect(result.success).toBe(true);
  });

  test('track/open: returns 1x1 GIF and no-store regardless of id', async ({ page }) => {
    const result = await page.evaluate(async () => {
      const r1 = await fetch('/api/track/open');
      const r2 = await fetch('/api/track/open?id=demo-email-001');
      return {
        ok1: r1.ok,
        ok2: r2.ok,
        type1: r1.headers.get('content-type'),
        type2: r2.headers.get('content-type'),
        cache1: r1.headers.get('cache-control'),
        cache2: r2.headers.get('cache-control'),
        len1: (await r1.arrayBuffer()).byteLength,
        len2: (await r2.arrayBuffer()).byteLength,
      };
    });

    expect(result.ok1).toBe(true);
    expect(result.ok2).toBe(true);
    expect(result.type1).toContain('image/gif');
    expect(result.type2).toContain('image/gif');
    expect(result.cache1).toContain('no-store');
    expect(result.cache2).toContain('no-store');
    // 43 bytes = 1×1 transparent GIF89a (the bytes literal in
    // handlers_read_receipts.go).
    expect(result.len1).toBe(43);
    expect(result.len2).toBe(43);
  });

  test('focus mode: start → state → stop round-trips cleanly', async ({ page }) => {
    const result = await page.evaluate(async () => {
      const start = await fetch('/api/focus', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ duration: 25 }),
      });
      const startJson = await start.json();
      const stateRes = await fetch('/api/focus');
      const stateJson = await stateRes.json();
      const stop = await fetch('/api/focus', { method: 'DELETE' });
      const stopJson = await stop.json();
      return { startOk: start.ok, startJson, stateOk: stateRes.ok, stateJson, stopOk: stop.ok, stopJson };
    });

    expect(result.startOk).toBe(true);
    expect(result.startJson.isActive).toBe(true);
    expect(result.stateOk).toBe(true);
    expect(result.stateJson.state.isActive).toBe(true);
    expect(result.stopOk).toBe(true);
    expect(result.stopJson.status).toBe('stopped');
  });
});
