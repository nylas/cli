// @ts-check
const { test, expect } = require('@playwright/test');

/**
 * Round 3 regression tests for Air handler robustness.
 *
 * These exercise paths whose mutex Lock/Unlock pairs were rewritten to use
 * defer-via-IIFE in Round 3. The tests don't try to *prove* the locks are
 * panic-safe (that's a Go-language guarantee) — they verify the endpoints
 * behave correctly under normal AND malformed inputs, which is enough to
 * catch any accidental regressions in those handler refactors.
 *
 * Tests use AirAPI directly so they're independent of UI rendering.
 */
test.describe('Handler robustness — Air (Round 3)', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('domcontentloaded');
  });

  test('templates: create + fetch + delete round-trip releases locks', async ({ page }) => {
    // Run the full lifecycle and ensure each call returns. If any of the
    // Lock/Unlock pairs leaks the lock, the second call hangs forever.
    const result = await page.evaluate(async () => {
      const created = await fetch('/api/templates', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          name: 'Round3 Test',
          subject: 'Hi {{name}}',
          body: 'Hello {{name}}, this is a test.',
        }),
      }).then((r) => r.json());

      // Immediate fetch — would hang if write-lock leaked.
      const fetched = await fetch(`/api/templates/${encodeURIComponent(created.id)}`).then((r) =>
        r.json(),
      );

      // Expand — increments usage count under another Lock/Unlock.
      const expanded = await fetch(
        `/api/templates/${encodeURIComponent(created.id)}/expand`,
        {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ variables: { name: 'World' } }),
        },
      ).then((r) => r.json());

      // Delete — also under Lock/Unlock.
      const delResp = await fetch(`/api/templates/${encodeURIComponent(created.id)}`, {
        method: 'DELETE',
      });

      return {
        createdId: created.id,
        createdName: created.name,
        fetchedId: fetched.id,
        expandedSubject: expanded.subject,
        expandedBody: expanded.body,
        deleteOk: delResp.ok,
      };
    });

    expect(result.createdId).toMatch(/^tmpl-/);
    expect(result.createdName).toBe('Round3 Test');
    expect(result.fetchedId).toBe(result.createdId);
    expect(result.expandedSubject).toBe('Hi World');
    expect(result.expandedBody).toContain('Hello World');
    expect(result.deleteOk).toBe(true);
  });

  test('snooze: snooze + unsnooze round-trip releases locks', async ({ page }) => {
    const result = await page.evaluate(async () => {
      const future = Math.floor(Date.now() / 1000) + 3600; // +1h

      const snoozeResp = await fetch('/api/snooze', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email_id: 'msg-round3-1', snooze_until: future }),
      }).then((r) => r.json());

      // Listing reads under RLock — should not deadlock against the writer.
      const list = await fetch('/api/snooze').then((r) => r.json());

      // Unsnooze — second writer Lock/Unlock IIFE.
      const unsnoozeResp = await fetch('/api/snooze?email_id=msg-round3-1', {
        method: 'DELETE',
      }).then((r) => r.json());

      return {
        snoozeSuccess: snoozeResp.success,
        snoozeUntil: snoozeResp.snooze_until,
        listCount: list.count,
        listed: list.snoozed.some((s) => s.email_id === 'msg-round3-1'),
        unsnoozeSuccess: unsnoozeResp.success,
      };
    });

    expect(result.snoozeSuccess).toBe(true);
    expect(typeof result.snoozeUntil).toBe('number');
    expect(result.listCount).toBeGreaterThan(0);
    expect(result.listed).toBe(true);
    expect(result.unsnoozeSuccess).toBe(true);
  });

  test('snooze: rejects past timestamps with 400 (no lock acquired)', async ({ page }) => {
    const status = await page.evaluate(async () => {
      const past = Math.floor(Date.now() / 1000) - 3600;
      const r = await fetch('/api/snooze', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email_id: 'msg-past', snooze_until: past }),
      });
      return r.status;
    });
    expect(status).toBe(400);
  });

  test('split inbox: PUT config + immediate GET reflects update', async ({ page }) => {
    const result = await page.evaluate(async () => {
      const updateResp = await fetch('/api/inbox/split', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          enabled: true,
          categories: ['primary', 'vip'],
          vip_senders: ['boss@example.com'],
          rules: [],
        }),
      }).then((r) => r.json());

      // Immediate GET — would hang if PUT path leaked the write lock.
      const fetched = await fetch('/api/inbox/split').then((r) => r.json());

      return {
        updateSuccess: updateResp.success,
        fetchedEnabled: fetched.config.enabled,
        fetchedVIPs: fetched.config.vip_senders,
      };
    });

    expect(result.updateSuccess).toBe(true);
    expect(result.fetchedEnabled).toBe(true);
    expect(result.fetchedVIPs).toContain('boss@example.com');
  });

  test('VIP senders: add + remove round-trip releases write lock', async ({ page }) => {
    const result = await page.evaluate(async () => {
      const addResp = await fetch('/api/inbox/vip', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email: 'newvip@example.com' }),
      }).then((r) => r.json());

      const listAfterAdd = await fetch('/api/inbox/vip').then((r) => r.json());

      const delResp = await fetch(
        '/api/inbox/vip?email=' + encodeURIComponent('newvip@example.com'),
        { method: 'DELETE' },
      ).then((r) => r.json());

      const listAfterDel = await fetch('/api/inbox/vip').then((r) => r.json());

      return {
        addSuccess: addResp.success,
        afterAddIncludes: listAfterAdd.vip_senders.includes('newvip@example.com'),
        delSuccess: delResp.success,
        afterDelExcludes: !listAfterDel.vip_senders.includes('newvip@example.com'),
      };
    });

    expect(result.addSuccess).toBe(true);
    expect(result.afterAddIncludes).toBe(true);
    expect(result.delSuccess).toBe(true);
    expect(result.afterDelExcludes).toBe(true);
  });

  test('undo send config: PUT clamps grace period and persists', async ({ page }) => {
    const result = await page.evaluate(async () => {
      // Deliberately out-of-range value (>60). Backend clamps to 60.
      const putResp = await fetch('/api/undo-send', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ enabled: true, grace_period_sec: 9999 }),
      }).then((r) => r.json());

      const getResp = await fetch('/api/undo-send').then((r) => r.json());

      return {
        putGrace: putResp.config.grace_period_sec,
        getGrace: getResp.grace_period_sec,
        getEnabled: getResp.enabled,
      };
    });

    expect(result.putGrace).toBe(60);
    expect(result.getGrace).toBe(60);
    expect(result.getEnabled).toBe(true);
  });

  test('split inbox config: malformed JSON is rejected with 400', async ({ page }) => {
    const status = await page.evaluate(async () => {
      const r = await fetch('/api/inbox/split', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: 'not-json{{{',
      });
      return r.status;
    });
    // Either 400 or 405 (depending on routing); both indicate the handler
    // did NOT acquire a lock and write garbage state.
    expect([400, 405]).toContain(status);
  });

  test('email row spinner clears after optimistic markAsRead succeeds', async ({ page }) => {
    // Regression: optimisticUpdate used to delete the pending operation
    // from the map without re-rendering, leaving the .pending-update
    // CSS class (and its rotating ::after spinner) on the row forever.
    const result = await page.evaluate(async () => {
      // Wait for the EmailListManager to be present and to have at least one email.
      const start = Date.now();
      while (
        (typeof EmailListManager === 'undefined' ||
          !EmailListManager ||
          !Array.isArray(EmailListManager.emails) ||
          EmailListManager.emails.length === 0) &&
        Date.now() - start < 5000
      ) {
        await new Promise((r) => setTimeout(r, 100));
      }
      if (
        typeof EmailListManager === 'undefined' ||
        !EmailListManager ||
        !EmailListManager.emails ||
        EmailListManager.emails.length === 0
      ) {
        return { skipped: true };
      }
      const target = EmailListManager.emails[0];
      // Force unread so markAsRead has work to do.
      target.unread = true;
      EmailListManager.updateEmailInUI(target.id);

      // Stub the network call so the test is deterministic.
      const origFetch = window.fetch;
      window.fetch = async (input, init) => {
        const url = typeof input === 'string' ? input : input.url;
        if (url.includes(`/api/emails/${encodeURIComponent(target.id)}`)) {
          return new Response('{}', {
            status: 200,
            headers: { 'Content-Type': 'application/json' },
          });
        }
        return origFetch.call(window, input, init);
      };

      try {
        await EmailListManager.markAsRead(target.id);
      } finally {
        window.fetch = origFetch;
      }

      const item = document.querySelector(
        `.email-item[data-email-id="${CSS.escape(target.id)}"]`,
      );
      const stillPending = item ? item.classList.contains('pending-update') : null;
      const stillInMap = EmailListManager.hasPendingOperation(target.id);

      return {
        skipped: false,
        stillPending,
        stillInMap,
      };
    });

    if (result.skipped) {
      test.skip(true, 'No emails available in this build to exercise markAsRead');
      return;
    }

    expect(result.stillPending).toBe(false);
    expect(result.stillInMap).toBe(false);
  });

  test('email row spinner clears after optimistic markAsRead fails', async ({ page }) => {
    // The error-path symmetric of the above. On failure, rollback should
    // restore state AND clear the spinner class.
    const result = await page.evaluate(async () => {
      const start = Date.now();
      while (
        (typeof EmailListManager === 'undefined' ||
          !EmailListManager ||
          !Array.isArray(EmailListManager.emails) ||
          EmailListManager.emails.length === 0) &&
        Date.now() - start < 5000
      ) {
        await new Promise((r) => setTimeout(r, 100));
      }
      if (
        typeof EmailListManager === 'undefined' ||
        !EmailListManager ||
        !EmailListManager.emails ||
        EmailListManager.emails.length === 0
      ) {
        return { skipped: true };
      }
      const target = EmailListManager.emails[0];
      target.unread = true;
      EmailListManager.updateEmailInUI(target.id);

      const origFetch = window.fetch;
      window.fetch = async (input, init) => {
        const url = typeof input === 'string' ? input : input.url;
        if (url.includes(`/api/emails/${encodeURIComponent(target.id)}`)) {
          return new Response('boom', { status: 500 });
        }
        return origFetch.call(window, input, init);
      };

      try {
        try {
          await EmailListManager.markAsRead(target.id);
        } catch {
          // optimisticUpdate swallows the error internally; nothing to do.
        }
      } finally {
        window.fetch = origFetch;
      }

      const item = document.querySelector(
        `.email-item[data-email-id="${CSS.escape(target.id)}"]`,
      );
      return {
        skipped: false,
        stillPending: item ? item.classList.contains('pending-update') : null,
        stillInMap: EmailListManager.hasPendingOperation(target.id),
      };
    });

    if (result.skipped) {
      test.skip(true, 'No emails available in this build to exercise markAsRead');
      return;
    }

    expect(result.stillPending).toBe(false);
    expect(result.stillInMap).toBe(false);
  });

  test('templates: rejects empty name and body with 400', async ({ page }) => {
    const statuses = await page.evaluate(async () => {
      const noName = await fetch('/api/templates', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name: '', body: 'x' }),
      });
      const noBody = await fetch('/api/templates', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name: 'x', body: '' }),
      });
      return [noName.status, noBody.status];
    });

    for (const s of statuses) {
      expect(s).toBe(400);
    }
  });
});
