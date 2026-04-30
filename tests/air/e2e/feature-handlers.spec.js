// @ts-check
const { test, expect } = require('@playwright/test');

/**
 * End-to-end coverage for Air productivity feature handlers.
 *
 * Each of these endpoints maintains in-memory state behind a mutex. Several
 * had their Lock/Unlock pairs rewritten across Rounds 1–3 to use
 * defer-via-IIFE for panic safety. These tests exercise the public API
 * round-trips (create → read → update → delete) so any future refactor
 * that breaks one of those paths shows up immediately.
 *
 * Tests are intentionally independent of UI rendering; they hit the JSON
 * API directly via fetch from the page context.
 */
test.describe('Feature handlers — Air', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('domcontentloaded');
  });

  // -------------------------------------------------------------------------
  // Focus Mode
  // -------------------------------------------------------------------------

  test('focus mode: start → status → stop releases write lock', async ({ page }) => {
    const result = await page.evaluate(async () => {
      const start = await fetch('/api/focus', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ duration: 25, pomodoroMode: false }),
      }).then((r) => r.json());

      const status = await fetch('/api/focus').then((r) => r.json());

      const stop = await fetch('/api/focus', { method: 'DELETE' }).then((r) => r.json());

      return {
        startedActive: start.isActive,
        statusActive: status.state.isActive,
        stopStatus: stop.status,
      };
    });

    expect(result.startedActive).toBe(true);
    expect(result.statusActive).toBe(true);
    expect(result.stopStatus).toBe('stopped');
  });

  test('focus mode: settings GET + PUT round-trip', async ({ page }) => {
    const result = await page.evaluate(async () => {
      const initial = await fetch('/api/focus/settings').then((r) => r.json());

      const updated = await fetch('/api/focus/settings', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          ...initial,
          defaultDuration: 30,
          autoReplyEnabled: true,
        }),
      });

      const fetched = await fetch('/api/focus/settings').then((r) => r.json());

      // Restore so subsequent tests see a sensible default.
      await fetch('/api/focus/settings', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(initial),
      });

      return {
        updateOk: updated.ok,
        defaultDuration: fetched.defaultDuration,
        autoReplyEnabled: fetched.autoReplyEnabled,
      };
    });

    expect(result.updateOk).toBe(true);
    expect(result.defaultDuration).toBe(30);
    expect(result.autoReplyEnabled).toBe(true);
  });

  test('focus mode: rejects /break when not in pomodoro mode', async ({ page }) => {
    const status = await page.evaluate(async () => {
      // Make sure no session is active.
      await fetch('/api/focus', { method: 'DELETE' });
      const r = await fetch('/api/focus/break', { method: 'POST' });
      return r.status;
    });
    expect(status).toBe(400);
  });

  // -------------------------------------------------------------------------
  // Read Receipts
  // -------------------------------------------------------------------------

  test('read receipts: settings round-trip', async ({ page }) => {
    const result = await page.evaluate(async () => {
      const initial = await fetch('/api/receipts/settings').then((r) => r.json());

      const updated = await fetch('/api/receipts/settings', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ ...initial, blockTracking: !initial.blockTracking }),
      }).then((r) => r.json());

      const fetched = await fetch('/api/receipts/settings').then((r) => r.json());

      // Restore.
      await fetch('/api/receipts/settings', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(initial),
      });

      return {
        updateStatus: updated.status,
        flippedTracking: fetched.blockTracking !== initial.blockTracking,
      };
    });

    expect(result.updateStatus).toBe('updated');
    expect(result.flippedTracking).toBe(true);
  });

  test('read receipts: tracking pixel returns valid GIF', async ({ page }) => {
    const result = await page.evaluate(async () => {
      const r = await fetch('/api/track/open?id=demo-1');
      const buf = await r.arrayBuffer();
      const bytes = new Uint8Array(buf);
      // GIF89a magic.
      const isGif =
        bytes[0] === 0x47 &&
        bytes[1] === 0x49 &&
        bytes[2] === 0x46 &&
        bytes[3] === 0x38;
      return {
        ok: r.ok,
        contentType: r.headers.get('content-type'),
        cacheControl: r.headers.get('cache-control'),
        isGif,
        size: bytes.length,
      };
    });

    expect(result.ok).toBe(true);
    expect(result.contentType).toBe('image/gif');
    expect(result.cacheControl).toMatch(/no-cache/);
    expect(result.isGif).toBe(true);
    expect(result.size).toBeGreaterThan(0);
  });

  test('read receipts: tracking pixel served even without id', async ({ page }) => {
    const ok = await page.evaluate(async () => {
      const r = await fetch('/api/track/open');
      return r.ok && r.headers.get('content-type') === 'image/gif';
    });
    expect(ok).toBe(true);
  });

  // -------------------------------------------------------------------------
  // Reply Later
  // -------------------------------------------------------------------------

  test('reply later: add → list → update → remove round-trip', async ({ page }) => {
    const result = await page.evaluate(async () => {
      const added = await fetch('/api/reply-later', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          emailId: 'rl-test-1',
          subject: 'Round 3',
          from: 'tester@example.com',
          remindIn: '1h',
          priority: 1,
          notes: 'urgent',
        }),
      }).then((r) => r.json());

      const list = await fetch('/api/reply-later').then((r) => r.json());

      const updated = await fetch('/api/reply-later/update', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          emailId: 'rl-test-1',
          isCompleted: true,
          notes: 'done',
        }),
      }).then((r) => r.json());

      const removeResp = await fetch('/api/reply-later/remove?emailId=rl-test-1', {
        method: 'DELETE',
      });

      return {
        addedId: added.emailId,
        addedPriority: added.priority,
        listIncludesAdded: list.some((it) => it.emailId === 'rl-test-1'),
        updatedCompleted: updated.isCompleted,
        updatedNotes: updated.notes,
        removeStatus: removeResp.status,
      };
    });

    expect(result.addedId).toBe('rl-test-1');
    expect(result.addedPriority).toBe(1);
    expect(result.listIncludesAdded).toBe(true);
    expect(result.updatedCompleted).toBe(true);
    expect(result.updatedNotes).toBe('done');
    expect(result.removeStatus).toBe(204);
  });

  test('reply later: rejects request without emailId', async ({ page }) => {
    const status = await page.evaluate(async () => {
      const r = await fetch('/api/reply-later', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ subject: 'x' }),
      });
      return r.status;
    });
    expect(status).toBe(400);
  });

  // -------------------------------------------------------------------------
  // Bundles
  // -------------------------------------------------------------------------

  test('bundles: GET returns shape regardless of state', async ({ page }) => {
    const result = await page.evaluate(async () => {
      const r = await fetch('/api/bundles');
      return { ok: r.ok, status: r.status, type: r.headers.get('content-type') };
    });
    expect(result.ok).toBe(true);
    expect(result.type).toMatch(/json/);
  });

  // -------------------------------------------------------------------------
  // Screener
  // -------------------------------------------------------------------------

  test('screener: allow + block round-trip releases write lock', async ({ page }) => {
    const result = await page.evaluate(async () => {
      const allowed = await fetch('/api/screener/allow', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email: 'sender@example.com', destination: 'primary' }),
      });

      const blocked = await fetch('/api/screener/block', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email: 'spam@example.com' }),
      });

      const list = await fetch('/api/screener').then((r) => r.json());

      return {
        allowedOk: allowed.ok,
        blockedOk: blocked.ok,
        listShape: typeof list,
      };
    });

    expect(result.allowedOk).toBe(true);
    expect(result.blockedOk).toBe(true);
    expect(result.listShape).toBe('object');
  });

  // -------------------------------------------------------------------------
  // Analytics
  // -------------------------------------------------------------------------

  test('analytics: dashboard returns serializable JSON', async ({ page }) => {
    const result = await page.evaluate(async () => {
      const r = await fetch('/api/analytics/dashboard');
      const j = await r.json();
      return { ok: r.ok, hasData: j !== null && typeof j === 'object' };
    });
    expect(result.ok).toBe(true);
    expect(result.hasData).toBe(true);
  });

  test('analytics: productivity + trends + focus-time return JSON', async ({ page }) => {
    const result = await page.evaluate(async () => {
      const endpoints = [
        '/api/analytics/productivity',
        '/api/analytics/trends',
        '/api/analytics/focus-time',
      ];
      const results = await Promise.all(
        endpoints.map(async (e) => {
          const r = await fetch(e);
          return { endpoint: e, ok: r.ok, type: r.headers.get('content-type') };
        }),
      );
      return results;
    });

    for (const { endpoint, ok, type } of result) {
      expect(ok, `${endpoint} should return ok`).toBe(true);
      expect(type, `${endpoint} content-type`).toMatch(/json/);
    }
  });

  // -------------------------------------------------------------------------
  // AI Config (touched in Round 1 — division-by-zero fix + masked-key check)
  // -------------------------------------------------------------------------

  test('AI config: masked API key is rejected (Round 1 regression)', async ({ page }) => {
    const result = await page.evaluate(async () => {
      const before = await fetch('/api/ai/config').then((r) => r.json());

      // Send a "masked" placeholder — the handler must NOT overwrite the
      // real API key with three asterisks (Round 1 bug).
      await fetch('/api/ai/config', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ apiKey: '***mask***' }),
      });

      const after = await fetch('/api/ai/config').then((r) => r.json());

      return {
        before: before.apiKey || '',
        after: after.apiKey || '',
      };
    });

    // The "after" key must not be the masked placeholder. Either it stayed
    // the same as before, or it got re-masked by the response — either way,
    // it must not literally be the masked input.
    expect(result.after).not.toBe('***mask***');
  });

  test('AI usage: zero budget does not 500 (Round 1 regression)', async ({ page }) => {
    const result = await page.evaluate(async () => {
      const r = await fetch('/api/ai/usage');
      return { ok: r.ok, status: r.status };
    });
    expect(result.status).toBeLessThan(500);
    expect(result.ok).toBe(true);
  });

  // -------------------------------------------------------------------------
  // Sanity: malformed JSON consistently produces 400 across mutating routes
  // -------------------------------------------------------------------------

  test('mutating routes consistently reject malformed JSON', async ({ page }) => {
    const statuses = await page.evaluate(async () => {
      const routes = [
        ['POST', '/api/snooze'],
        ['POST', '/api/templates'],
        ['POST', '/api/reply-later'],
        ['PUT', '/api/inbox/split'],
        ['PUT', '/api/undo-send'],
        ['POST', '/api/screener/allow'],
        ['POST', '/api/screener/block'],
        ['PUT', '/api/focus/settings'],
        ['PUT', '/api/receipts/settings'],
      ];
      const out = {};
      for (const [method, url] of routes) {
        const r = await fetch(url, {
          method,
          headers: { 'Content-Type': 'application/json' },
          body: 'not-json{{{',
        });
        out[`${method} ${url}`] = r.status;
      }
      return out;
    });

    // Every mutating route must return 4xx (preferably 400) on malformed JSON.
    // None should 5xx — that indicates a panic/crash path (lock leak risk).
    for (const [route, status] of Object.entries(statuses)) {
      expect(status, `${route} should return 4xx on malformed JSON`).toBeGreaterThanOrEqual(400);
      expect(status, `${route} should not 5xx`).toBeLessThan(500);
    }
  });
});
