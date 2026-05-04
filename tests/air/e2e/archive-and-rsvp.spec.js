// @ts-check
const { test, expect } = require('@playwright/test');

/**
 * Archive + RSVP E2E tests — pin two regressions:
 *
 * 1. Gmail Archive: must send `folders: []` (an explicit empty array)
 *    over the wire. Earlier code elided the empty array via omitempty,
 *    making "archive" a no-op upstream while the optimistic UI removed
 *    the row.
 *
 * 2. RSVP: clicking Yes/Maybe/No must POST to /api/emails/{id}/rsvp
 *    with the matching status, show the in-flight loading state, and
 *    settle on the "active" highlight when the server confirms.
 *
 * The in-page parts of these tests (spying on AirAPI / window.fetch,
 * reading EmailListManager state) live in `page.evaluate` because no
 * Playwright locator can intercept a same-origin fetch — but every
 * user-facing click goes through semantic locators (`getByRole`) and
 * every assertion is on the test side, NOT inside `page.evaluate`.
 *
 * Semantic locators only — per CLAUDE.md, never CSS/XPath. The app
 * shell exposes role="application" with aria-label "Nylas Air Email
 * Client"; each email row exposes role="option" via email-renderer.js.
 */
test.describe('Archive + RSVP Integrations', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await expect(page.getByRole('application', { name: /Nylas Air/i })).toBeVisible();
    await page.waitForLoadState('domcontentloaded');
    // Wait for at least one email to render — beats arbitrary timeouts
    // and surfaces "demo data didn't load" as an explicit failure.
    await expect(page.getByRole('option').first()).toBeVisible({ timeout: 5000 });
  });

  test('archive on Gmail (no typed Archive folder) sends explicit empty folders array', async ({ page }) => {
    // Sanity-check that the bundle is wired up. We deliberately avoid
    // `if (skipped) return` here — a missing EmailListManager means the
    // bundle is broken and the test SHOULD fail, not silently pass.
    await expect.poll(
      () => page.evaluate(() => typeof EmailListManager !== 'undefined' && Array.isArray(EmailListManager.emails)),
      { timeout: 5000, message: 'EmailListManager bundle did not initialise' }
    ).toBe(true);

    // Pin Strategy 2 (Gmail label removal) explicitly. Real Gmail
    // accounts surface "All Mail" as system_folder='all', NOT 'archive',
    // so computeArchiveFolders falls through to dropping the INBOX
    // label and the resulting payload is folders:[]. Seed a Gmail-style
    // folder list directly rather than relying on the demo dataset,
    // which intentionally exposes every system_folder type for UI
    // exercise — including a typed Archive that would (correctly)
    // make Strategy 1 fire.
    //
    // The previous bug here was on the Go side: omitempty elided the
    // empty array on the wire, so the backend never received the
    // archive change. That fix is pinned by the Go-side tests
    // TestHTTPClient_UpdateMessage_ForwardsEmptyFolders and friends;
    // this test pins the JS half — that the payload reaches the wire
    // as exactly [] for a Gmail-shaped account.
    const result = await page.evaluate(async () => {
      const fakeEmail = { id: 'fake-gmail-archive-1', folders: ['INBOX'], unread: false, starred: false, subject: 'x', from: [], to: [] };
      EmailListManager.emails = [fakeEmail, ...EmailListManager.emails];
      EmailListManager.folders = [
        { id: 'INBOX', system_folder: 'inbox', name: 'INBOX' },
        { id: 'all-mail', system_folder: 'all', name: 'All Mail' },
      ];

      let capturedPayload = null;
      const originalUpdate = AirAPI.updateEmail;
      AirAPI.updateEmail = async (id, payload) => {
        capturedPayload = payload;
        return { status: 'success' };
      };

      try {
        await EmailListManager.archiveEmail(fakeEmail.id);
        return {
          payload: capturedPayload,
          wasRemoved: !EmailListManager.emails.find(e => e.id === fakeEmail.id),
        };
      } finally {
        AirAPI.updateEmail = originalUpdate;
      }
    });

    expect(result.payload).toBeDefined();
    expect(result.payload).toHaveProperty('folders');
    expect(result.payload.folders, 'Gmail archive must send an explicit empty array, not omit the key').toEqual([]);
    expect(result.wasRemoved).toBe(true);
  });

  test('archive on Microsoft/IMAP/EWS uses the Archive folder ID (Strategy 2)', async ({ page }) => {
    // Pin computeArchiveFolders' Strategy 2 — when the email is NOT in
    // an INBOX label (Gmail-style) but the account has a typed Archive
    // folder, archive replaces the email's folders with that one ID.
    // Without coverage, a refactor that "simplifies" the helper could
    // silently break archive on every non-Gmail provider.
    const result = await page.evaluate(async () => {
      // Manufacture a minimal non-Gmail email + folder shape directly on
      // the manager. We avoid touching the demo dataset so the assertion
      // is independent of provider classification heuristics.
      const fakeEmail = { id: 'fake-ms-1', folders: ['some-other-folder-id'], unread: false, starred: false, subject: 'x', from: [], to: [] };
      EmailListManager.emails = [fakeEmail, ...EmailListManager.emails];
      EmailListManager.folders = [{ id: 'arch-id-1', system_folder: 'archive', name: 'Archive' }];

      let captured = null;
      const originalUpdate = AirAPI.updateEmail;
      AirAPI.updateEmail = async (id, payload) => { captured = payload; return { status: 'success' }; };
      try {
        await EmailListManager.archiveEmail(fakeEmail.id);
        return { payload: captured };
      } finally {
        AirAPI.updateEmail = originalUpdate;
      }
    });

    expect(result.payload).toBeDefined();
    expect(result.payload.folders, 'Strategy 2 must replace folders with the Archive folder id, not send empty')
      .toEqual(['arch-id-1']);
  });

  test('archive on IMAP/EWS with folders:["INBOX"] + typed Archive uses the Archive folder, not folders:[]', async ({ page }) => {
    // Regression: IMAP/EWS surfaces the inbox folder as the literal name
    // "INBOX". The old computeArchiveFolders stripped INBOX before
    // checking for a typed Archive folder, sending folders:[] upstream
    // and moving the message out of every folder instead of into Archive.
    // This pins the corrected order — typed Archive wins over label
    // removal whenever a system_folder='archive' exists.
    const result = await page.evaluate(async () => {
      const fakeEmail = { id: 'fake-imap-1', folders: ['INBOX'], unread: false, starred: false, subject: 'x', from: [], to: [] };
      EmailListManager.emails = [fakeEmail, ...EmailListManager.emails];
      EmailListManager.folders = [
        { id: 'inbox-id', system_folder: 'inbox', name: 'INBOX' },
        { id: 'arch-imap-1', system_folder: 'archive', name: 'Archive' },
      ];

      let captured = null;
      const originalUpdate = AirAPI.updateEmail;
      AirAPI.updateEmail = async (id, payload) => { captured = payload; return { status: 'success' }; };
      try {
        await EmailListManager.archiveEmail(fakeEmail.id);
        return { payload: captured };
      } finally {
        AirAPI.updateEmail = originalUpdate;
      }
    });

    expect(result.payload).toBeDefined();
    expect(result.payload.folders, 'IMAP/EWS archive must move into the typed Archive folder, not send folders:[]')
      .toEqual(['arch-imap-1']);
  });

  test('archive does NOT mistake a Gmail user label named "Archive" for the system destination', async ({ page }) => {
    // Pin the safety property of using system_folder match instead of
    // name fallback: a Gmail account with a user-created label literally
    // named "Archive" (system_folder unset) must still archive via
    // Strategy 2 (drop INBOX), not move into the user's vanity label.
    const result = await page.evaluate(async () => {
      const fakeEmail = { id: 'fake-gmail-vanity-1', folders: ['INBOX', 'IMPORTANT'], unread: false, starred: false, subject: 'x', from: [], to: [] };
      EmailListManager.emails = [fakeEmail, ...EmailListManager.emails];
      // Gmail-style: only labels with empty system_folder. The user
      // happens to have a label called "Archive" — must not be picked.
      EmailListManager.folders = [
        { id: 'label-inbox', system_folder: 'inbox', name: 'INBOX' },
        { id: 'label-archive-vanity', system_folder: '', name: 'Archive' },
      ];

      let captured = null;
      const originalUpdate = AirAPI.updateEmail;
      AirAPI.updateEmail = async (id, payload) => { captured = payload; return { status: 'success' }; };
      try {
        await EmailListManager.archiveEmail(fakeEmail.id);
        return { payload: captured };
      } finally {
        AirAPI.updateEmail = originalUpdate;
      }
    });

    expect(result.payload).toBeDefined();
    expect(result.payload.folders, 'Gmail label removal must drop only INBOX, leaving other labels intact')
      .toEqual(['IMPORTANT']);
  });

  test('archive surfaces "no archive target" when no INBOX and no Archive folder', async ({ page }) => {
    // Pin the null branch of computeArchiveFolders: if the email isn't
    // in INBOX (Strategy 1 doesn't apply) AND no system Archive folder
    // is known (Strategy 2 doesn't apply), archiveEmail must fail
    // closed — show an error toast and return false. Without this, a
    // misconfigured account would archive into the void.
    const result = await page.evaluate(async () => {
      const fakeEmail = { id: 'fake-no-target-1', folders: ['exotic-folder'], unread: false, starred: false, subject: 'x', from: [], to: [] };
      EmailListManager.emails = [fakeEmail, ...EmailListManager.emails];
      EmailListManager.folders = []; // no archive folder available

      let updateCalled = false;
      const originalUpdate = AirAPI.updateEmail;
      AirAPI.updateEmail = async () => { updateCalled = true; return { status: 'success' }; };
      try {
        const ok = await EmailListManager.archiveEmail(fakeEmail.id);
        return {
          ok,
          updateCalled,
          stillInList: !!EmailListManager.emails.find(e => e.id === fakeEmail.id),
        };
      } finally {
        AirAPI.updateEmail = originalUpdate;
      }
    });

    expect(result.ok, 'archiveEmail must return false when no archive target exists').toBe(false);
    expect(result.updateCalled, 'AirAPI.updateEmail must NOT be called — fail closed').toBe(false);
    expect(result.stillInList, 'email must remain in the list when archive is unavailable').toBe(true);
  });

  test('clicking RSVP Yes calls POST /rsvp with status=yes and highlights the button', async ({ page }) => {
    // Select the demo invite email through the API surface so the test
    // doesn't depend on the email list's exact ordering.
    await page.evaluate(async () => EmailListManager.selectEmail('demo-email-invite-001'));

    const inviteCard = page.getByRole('region', { name: 'Calendar invitation' });
    await expect(inviteCard, 'invite card must render after selecting the demo invite').toBeVisible({ timeout: 3000 });

    const yesButton = inviteCard.getByRole('button', { name: 'Yes' });
    await expect(yesButton).toBeVisible();
    await expect(yesButton).toBeEnabled();

    // Install fetch spy in the page context — Playwright's request
    // interception runs in a different process and can't intercept
    // same-origin fetches reliably for SPAs.
    await page.evaluate(() => {
      window.__rsvpCall = null;
      window.__originalFetch = window.fetch;
      window.fetch = async (url, init) => {
        if (typeof url === 'string' && url.includes('/rsvp') && init?.method === 'POST') {
          window.__rsvpCall = { url, body: JSON.parse(init.body) };
          return new Response(JSON.stringify({ status: 'yes', event_id: 'evt-1', calendar_id: 'cal-1' }), {
            status: 200,
            headers: { 'Content-Type': 'application/json' },
          });
        }
        return window.__originalFetch(url, init);
      };
    });

    try {
      await yesButton.click();

      // Wait for the active class to appear via Playwright's auto-retry.
      await expect(yesButton).toHaveClass(/active/);
      // The is-loading class is added during flight and removed when
      // the request settles. After click + auto-retry above, it should
      // be gone — pin both halves of the contract.
      await expect(yesButton).not.toHaveClass(/is-loading/);

      const rsvpCall = await page.evaluate(() => window.__rsvpCall);
      expect(rsvpCall, 'fetch was never invoked — click did not route through the RSVP handler').not.toBeNull();
      expect(rsvpCall.url).toContain('/api/emails/demo-email-invite-001/rsvp');
      expect(rsvpCall.body.status).toBe('yes');
    } finally {
      await page.evaluate(() => { window.fetch = window.__originalFetch; });
    }
  });

  test('RSVP button shows loading + disabled state while in flight', async ({ page }) => {
    await page.evaluate(async () => EmailListManager.selectEmail('demo-email-invite-001'));
    const inviteCard = page.getByRole('region', { name: 'Calendar invitation' });
    await expect(inviteCard).toBeVisible({ timeout: 3000 });

    // Install a hanging fetch spy so we can observe the in-flight UI
    // state without a race. Resume the fetch from the test side once
    // we've snapshotted the loading classes.
    await page.evaluate(() => {
      window.__resolveFetch = null;
      window.__originalFetch = window.fetch;
      window.fetch = (url, init) => {
        if (typeof url === 'string' && url.includes('/rsvp')) {
          return new Promise((resolve) => {
            window.__resolveFetch = () => resolve(new Response(
              JSON.stringify({ status: 'yes' }),
              { status: 200, headers: { 'Content-Type': 'application/json' } }
            ));
          });
        }
        return window.__originalFetch(url, init);
      };
    });

    try {
      const yesButton = inviteCard.getByRole('button', { name: 'Yes' });
      await yesButton.click();

      // While the fetch is hanging, the button must be disabled and
      // class-loaded. expect(...).toHaveClass auto-retries up to the
      // default timeout, so this catches the in-flight state without
      // fragile setTimeouts.
      await expect(yesButton).toBeDisabled();
      await expect(yesButton).toHaveClass(/is-loading/);

      // Resume the fetch and verify the loading state goes away.
      await page.evaluate(() => window.__resolveFetch && window.__resolveFetch());
      await expect(yesButton).not.toHaveClass(/is-loading/);
      await expect(yesButton).toBeEnabled();
    } finally {
      await page.evaluate(() => { window.fetch = window.__originalFetch; });
    }
  });

  test('RSVP failure restores prior selection and surfaces an error toast', async ({ page }) => {
    await page.evaluate(async () => EmailListManager.selectEmail('demo-email-invite-001'));
    const inviteCard = page.getByRole('region', { name: 'Calendar invitation' });
    await expect(inviteCard).toBeVisible({ timeout: 3000 });

    await page.evaluate(() => {
      window.__originalFetch = window.fetch;
      window.fetch = async (url) => {
        if (typeof url === 'string' && url.includes('/rsvp')) {
          return new Response(JSON.stringify({ error: 'event not imported' }), {
            status: 404,
            headers: { 'Content-Type': 'application/json' },
          });
        }
        return window.__originalFetch(url);
      };
    });

    try {
      const yesButton = inviteCard.getByRole('button', { name: 'Yes' });
      await yesButton.click();

      // On failure, the button should NOT be left "active" — that
      // would lie to the user about the server state.
      await expect(yesButton).not.toHaveClass(/active/);
      await expect(yesButton).not.toHaveClass(/is-loading/);
      await expect(yesButton).toBeEnabled();
    } finally {
      await page.evaluate(() => { window.fetch = window.__originalFetch; });
    }
  });

  test('RSVP network failure (fetch throws) restores prior selection without active class', async ({ page }) => {
    // Pin the catch-block path: fetch rejects with a network error
    // (vs returning !response.ok). The previous tests cover the
    // !resp.ok branch; this one targets the `catch (err)` block where
    // previouslyActive is re-applied. Without this test a future
    // refactor that reorders the catch could silently break the
    // "no connectivity → restore prior selection" UX.
    await page.evaluate(async () => EmailListManager.selectEmail('demo-email-invite-001'));
    const inviteCard = page.getByRole('region', { name: 'Calendar invitation' });
    await expect(inviteCard).toBeVisible({ timeout: 3000 });

    // Pre-mark Maybe as the prior selection so we can verify the
    // catch block restores it (rather than leaving the chosen Yes
    // button active in error).
    const maybeButton = inviteCard.getByRole('button', { name: 'Maybe' });
    await page.evaluate(() => {
      const slot = document.querySelector('[id^="inviteSlot-"]');
      const buttons = slot ? slot.querySelectorAll('.calendar-invite-btn') : [];
      buttons.forEach((b) => {
        b.classList.toggle('active', b.dataset.rsvp === 'maybe');
      });
    });
    await expect(maybeButton).toHaveClass(/active/);

    await page.evaluate(() => {
      window.__originalFetch = window.fetch;
      window.fetch = async (url) => {
        if (typeof url === 'string' && url.includes('/rsvp')) {
          // Reject with a TypeError, mirroring how browsers report
          // "fetch failed" on connectivity loss / DNS failure.
          throw new TypeError('Failed to fetch');
        }
        return window.__originalFetch(url);
      };
    });

    try {
      const yesButton = inviteCard.getByRole('button', { name: 'Yes' });
      await yesButton.click();

      // Yes must not be marked active after a network error.
      await expect(yesButton).not.toHaveClass(/active/);
      // The previously-active button (Maybe) must be restored.
      await expect(maybeButton).toHaveClass(/active/);
      // Both buttons should be re-enabled and not in the loading
      // state, regardless of which one was clicked.
      await expect(yesButton).toBeEnabled();
      await expect(yesButton).not.toHaveClass(/is-loading/);
      await expect(maybeButton).toBeEnabled();
    } finally {
      await page.evaluate(() => { window.fetch = window.__originalFetch; });
    }
  });

  test('RSVP navigated-away guard skips active-class toggle on the new email', async ({ page }) => {
    // Pin the `this.selectedEmailId === emailId` guard inside
    // rsvpToInvite. Sequence:
    //   1) Select invite email A, click RSVP Yes (fetch hangs)
    //   2) Mid-flight, switch to a different email B
    //   3) Resolve the fetch
    // The guard must prevent the active class from being applied to
    // the (no-longer-visible) invite card on email A. Without the
    // guard, returning to A later would show a stale "active" Yes.
    await page.evaluate(async () => EmailListManager.selectEmail('demo-email-invite-001'));
    const inviteCard = page.getByRole('region', { name: 'Calendar invitation' });
    await expect(inviteCard).toBeVisible({ timeout: 3000 });

    await page.evaluate(() => {
      window.__resolveFetch = null;
      window.__originalFetch = window.fetch;
      window.fetch = (url) => {
        if (typeof url === 'string' && url.includes('/rsvp')) {
          return new Promise((resolve) => {
            window.__resolveFetch = () => resolve(new Response(
              JSON.stringify({ status: 'yes', event_id: 'evt-1', calendar_id: 'cal-1' }),
              { status: 200, headers: { 'Content-Type': 'application/json' } }
            ));
          });
        }
        return window.__originalFetch(url);
      };
    });

    try {
      const yesButton = inviteCard.getByRole('button', { name: 'Yes' });
      // Don't await — we need the request to be in flight when we
      // navigate away.
      void yesButton.click();

      // Wait for the loading class to confirm the request is in flight.
      await expect(yesButton).toHaveClass(/is-loading/);

      // Navigate to a different email while the RSVP is hanging.
      const otherEmail = await page.evaluate(() => {
        const other = EmailListManager.emails.find((e) => e.id !== 'demo-email-invite-001');
        return other ? other.id : null;
      });
      expect(otherEmail, 'demo dataset must contain at least one non-invite email').not.toBeNull();
      await page.evaluate((id) => EmailListManager.selectEmail(id), otherEmail);

      // Now resolve the fetch. The guard must prevent the active
      // class from being applied to the original invite card.
      await page.evaluate(() => window.__resolveFetch && window.__resolveFetch());

      // Switch back to the invite email and verify Yes is NOT active.
      await page.evaluate(async () => EmailListManager.selectEmail('demo-email-invite-001'));
      const reselected = page.getByRole('region', { name: 'Calendar invitation' });
      await expect(reselected).toBeVisible({ timeout: 3000 });
      const reselectedYes = reselected.getByRole('button', { name: 'Yes' });
      await expect(reselectedYes, 'navigated-away guard must NOT mark Yes active on the original card').not.toHaveClass(/active/);
    } finally {
      await page.evaluate(() => { window.fetch = window.__originalFetch; });
    }
  });

  test('archive Undo: "Restore unavailable" toast when original had no folders', async ({ page }) => {
    // Pin the archive-Undo defensive branch: if the email started
    // with folders:[] (rare — drafts/sent paths), we cannot restore
    // it via PUT folders:[...] without a meaningful target. The Undo
    // callback must short-circuit with an error toast rather than
    // making a no-op or, worse, an empty-folders PUT that would land
    // the email unfiled.
    const result = await page.evaluate(async () => {
      // Manufacture a fake email with empty folders so we control the
      // archive-state shape without depending on the demo dataset.
      const fake = {
        id: 'fake-empty-folders-1',
        folders: [],
        unread: false,
        starred: false,
        subject: 'Empty folders',
        from: [],
        to: [],
      };
      EmailListManager.emails = [fake, ...EmailListManager.emails];
      // Provide a system Archive folder so Strategy 2 succeeds during
      // the initial archive (the regression we're testing is the
      // Undo path, not the archive path).
      EmailListManager.folders = [{ id: 'arch-id-1', system_folder: 'archive', name: 'Archive' }];

      const captured = { toasts: [], updateCalls: 0 };
      const originalUpdate = AirAPI.updateEmail;
      const originalToast = window.showToast;
      AirAPI.updateEmail = async () => {
        captured.updateCalls++;
        return { status: 'success' };
      };
      window.showToast = (type, title, msg, opts) => {
        captured.toasts.push({ type, title, msg });
        // Synchronously fire the Undo action so the test can observe
        // the resulting toast without timing dance. We deliberately
        // capture only the toast events; the action itself goes
        // through the real callback.
        if (opts && typeof opts.onAction === 'function' && title === 'Archived') {
          // Fire the undo action — this is what the user clicking
          // "Undo" would trigger.
          opts.onAction();
        }
      };
      try {
        await EmailListManager.archiveEmail(fake.id);
        // Wait one microtask tick so the Undo's async toast lands.
        await new Promise((r) => setTimeout(r, 0));
        return captured;
      } finally {
        AirAPI.updateEmail = originalUpdate;
        window.showToast = originalToast;
      }
    });

    // First call is the archive PUT; no Undo PUT should fire because
    // the original folders array was empty.
    expect(result.updateCalls, 'only the initial archive should call updateEmail; Undo must short-circuit').toBe(1);
    const restoreToast = result.toasts.find((t) => t.title === 'Restore unavailable');
    expect(restoreToast, 'Undo with empty original folders must surface "Restore unavailable"').toBeDefined();
    expect(restoreToast.type).toBe('error');
  });

  test('archive Undo: "Restore failed" toast when restore PUT throws', async ({ page }) => {
    // Pin the archive-Undo error path: the initial archive PUT
    // succeeds, but the user clicks Undo and the restore PUT fails.
    // Without surfacing "Restore failed", the user thinks the email
    // is back in the inbox while it is still archived on the server.
    const result = await page.evaluate(async () => {
      const inbox = EmailListManager.emails.find((e) =>
        (e.folders || []).includes('inbox') &&
        !(e.folders || []).includes('archive')
      );
      if (!inbox) return { error: 'no inbox-only demo email available' };

      const captured = { toasts: [], updateCalls: 0, restoreThrew: false };
      const originalUpdate = AirAPI.updateEmail;
      const originalToast = window.showToast;
      AirAPI.updateEmail = async (id, payload) => {
        captured.updateCalls++;
        // First call (archive) succeeds; second call (Undo restore)
        // must throw to exercise the error branch.
        if (captured.updateCalls === 1) return { status: 'success' };
        captured.restoreThrew = true;
        throw new Error('simulated 503 on restore');
      };
      window.showToast = (type, title, msg, opts) => {
        captured.toasts.push({ type, title, msg });
        if (opts && typeof opts.onAction === 'function' && title === 'Archived') {
          opts.onAction();
        }
      };
      try {
        await EmailListManager.archiveEmail(inbox.id);
        await new Promise((r) => setTimeout(r, 0));
        return captured;
      } finally {
        AirAPI.updateEmail = originalUpdate;
        window.showToast = originalToast;
      }
    });

    expect(result.error, `setup failed: ${result.error}`).toBeUndefined();
    expect(result.updateCalls, 'archive + restore = 2 PUTs').toBe(2);
    expect(result.restoreThrew, 'restore branch must have been invoked').toBe(true);
    const failed = result.toasts.find((t) => t.title === 'Restore failed');
    expect(failed, 'Undo restore failure must surface "Restore failed"').toBeDefined();
    expect(failed.type).toBe('error');
    // Crucially, the success toast for "Restored" must NOT fire on
    // a failed restore — that would lie to the user.
    const restoredOK = result.toasts.find((t) => t.title === 'Restored');
    expect(restoredOK, 'must NOT announce "Restored" when the PUT failed').toBeUndefined();
  });

  test('archive success: detail pane transitions to empty state when archived email was selected', async ({ page }) => {
    // Pin the success-path counterpart to "archive failure restores
    // selection AND repopulates the detail pane". When archiveEmail
    // succeeds AND the email being archived was the currently
    // selected one, the detail pane MUST flip to the empty state —
    // otherwise the user sees stale content for an email that no
    // longer exists in the list. The failure case has its own test
    // (line ~306); this is the missing positive.
    await expect.poll(
      () => page.evaluate(() => typeof EmailListManager !== 'undefined' && Array.isArray(EmailListManager.emails)),
      { timeout: 5000, message: 'EmailListManager bundle did not initialise' }
    ).toBe(true);

    // Select an inbox-only email and confirm the detail pane is
    // populated (i.e., NOT showing the empty state) before we
    // archive. Anchoring on the empty-state copy directly is more
    // robust than checking class names that change between layouts.
    const setup = await page.evaluate(async () => {
      const email = EmailListManager.emails.find((e) =>
        (e.folders || []).includes('inbox') &&
        !(e.folders || []).includes('archive')
      );
      if (!email) return { error: 'no inbox-only demo email available' };
      await EmailListManager.selectEmail(email.id);
      return { emailId: email.id };
    });
    expect(setup.error, `setup failed: ${setup.error}`).toBeUndefined();

    // Pre-condition: detail pane shows the email, NOT the empty state.
    await expect(
      page.getByText('Select an email to view'),
      'detail pane must be populated before we archive',
    ).not.toBeVisible();

    // Stub a successful archive so we exercise the success branch
    // deterministically (the demo backend would also succeed, but
    // we want to also assert the AirAPI payload along the way).
    const archive = await page.evaluate(async (emailId) => {
      const originalUpdate = AirAPI.updateEmail;
      let capturedPayload = null;
      AirAPI.updateEmail = async (_id, payload) => {
        capturedPayload = payload;
        return { status: 'success' };
      };
      try {
        const ok = await EmailListManager.archiveEmail(emailId);
        return {
          ok,
          payload: capturedPayload,
          stillInList: !!EmailListManager.emails.find((e) => e.id === emailId),
          // Read selectedEmailId AFTER archive — the success path
          // sets it to null when the archived email was selected.
          selectionCleared: EmailListManager.selectedEmailId === null,
        };
      } finally {
        AirAPI.updateEmail = originalUpdate;
      }
    }, setup.emailId);

    expect(archive.ok, 'archiveEmail success-path return value').toBe(true);
    expect(archive.payload, 'archive PUT must include the new folders array').toBeDefined();
    expect(archive.stillInList, 'archived email must be removed from the list').toBe(false);
    expect(archive.selectionCleared, 'selectedEmailId must be cleared so the pane transitions out').toBe(true);

    // Detail pane MUST flip to the empty state. Using getByText keeps
    // us in semantic-locator territory and benefits from Playwright's
    // auto-retry, replacing any class-based assertion that could
    // shift with a future layout refactor.
    await expect(
      page.getByText('Select an email to view'),
      'detail pane MUST show the empty state after archive succeeds on the selected email',
    ).toBeVisible();
  });

  test('loadEmails returns 3-state outcome (loaded / in-progress / failed)', async ({ page }) => {
    // Pin the contract change behind the pull-to-refresh "Refresh failed"
    // false-alarm fix: the previous boolean conflated "skipped because
    // already loading" with "fetch failed", so a second pull during an
    // in-flight load showed an error toast even though no fetch failed.
    await expect.poll(
      () => page.evaluate(() => typeof EmailListManager !== 'undefined' && typeof EmailListManager.loadEmails === 'function'),
      { timeout: 5000, message: 'EmailListManager bundle did not initialise' }
    ).toBe(true);

    const result = await page.evaluate(async () => {
      const outcomes = {};
      const originalGet = AirAPI.getEmails;

      // 1) in-progress: another load is already running.
      EmailListManager.isLoading = true;
      outcomes.duringConcurrent = await EmailListManager.loadEmails();
      EmailListManager.isLoading = false;

      // 2) failed: the API throws.
      AirAPI.getEmails = async () => { throw new Error('simulated network failure'); };
      outcomes.onError = await EmailListManager.loadEmails();

      // 3) loaded: the API returns data.
      AirAPI.getEmails = async () => ({ emails: [], next_cursor: null, has_more: false });
      outcomes.onSuccess = await EmailListManager.loadEmails();

      AirAPI.getEmails = originalGet;
      return outcomes;
    });

    expect(result.duringConcurrent, 'concurrent load must NOT be reported as a failure').toBe('in-progress');
    expect(result.onError).toBe('failed');
    expect(result.onSuccess).toBe('loaded');
  });

  test('archive failure restores selection AND repopulates the detail pane', async ({ page }) => {
    // Pin the regression where archive rollback restored `this.emails`
    // but left `selectedEmailId === null` and the detail pane showing
    // the empty state — list and preview drifted out of sync.
    await expect.poll(
      () => page.evaluate(() => typeof EmailListManager !== 'undefined' && Array.isArray(EmailListManager.emails)),
      { timeout: 5000 }
    ).toBe(true);

    const setup = await page.evaluate(async () => {
      const email = EmailListManager.emails.find(e =>
        (e.folders || []).includes('inbox') &&
        !(e.folders || []).includes('archive')
      );
      if (!email) return { error: 'no inbox-only demo email available' };

      // Select the email so the detail pane is populated.
      await EmailListManager.selectEmail(email.id);
      return { emailId: email.id };
    });
    expect(setup.error, `setup failed: ${setup.error}`).toBeUndefined();

    // Stub AirAPI.updateEmail to fail; trigger the rollback path. Use
    // evaluate so the spy lives in the page context where the click
    // handler runs. We return only application state from the spy and
    // assert on the user-visible UI via Playwright locators below.
    const rollback = await page.evaluate(async (emailId) => {
      const originalUpdate = AirAPI.updateEmail;
      AirAPI.updateEmail = async () => { throw new Error('simulated 503'); };
      try {
        await EmailListManager.archiveEmail(emailId);
        return {
            stillInList: !!EmailListManager.emails.find(e => e.id === emailId),
            isSelected: EmailListManager.selectedEmailId === emailId,
        };
      } finally {
        AirAPI.updateEmail = originalUpdate;
      }
    }, setup.emailId);

    expect(rollback.stillInList, 'rollback must put the email back in the list').toBe(true);
    expect(rollback.isSelected, 'selectedEmailId must be restored after rollback').toBe(true);

    // Detail pane must NOT show the "Select an email to view" empty
    // state. Using getByText keeps us in semantic-locator territory and
    // benefits from Playwright's auto-retry, replacing the
    // detail.querySelector('.empty-state') CSS lookup.
    await expect(
      page.getByText('Select an email to view'),
      'detail pane must NOT be left at the empty state',
    ).not.toBeVisible();
  });

  test('keyboard Delete announces "Email deleted" only on success, "Delete failed" on rollback', async ({ page }) => {
    // Pin the screen-reader regression: the previous code fired
    // announce('Email deleted') synchronously, even when the underlying
    // fetch failed and the email was rolled back into the list.
    await expect.poll(
      () => page.evaluate(() => typeof EmailListManager !== 'undefined' && Array.isArray(EmailListManager.emails)),
      { timeout: 5000 }
    ).toBe(true);

    // The screen-reader announcer is a single live region near the top
    // of base.gohtml with role="status" and id="announcer". We address
    // it via the role; falling back to the id-based locator only if a
    // future refactor moves it. Both forms are stable — class names are
    // not.
    // The announcer is a single live region near the top of base.gohtml.
    // Use getByTestId rather than role='status' because the toast
    // container also has role='status' (so screen readers pick toasts up
    // too) — testid keeps us pinned to the announcer specifically.
    const announcer = page.getByTestId('announcer');

    // dispatchKeydown fires a synthetic Delete on the first row's
    // containing list. We use evaluate for the actual dispatch because
    // KeyboardEvent isn't reachable through Playwright's keyboard API
    // when the focus target is inside an SPA-managed widget, but every
    // assertion that follows is on the test side.
    async function pressDeleteOnFirstEmail() {
      return page.evaluate(() => {
        const email = EmailListManager.emails[0];
        if (!email) return { error: 'no demo emails available' };
        const row = document.querySelector(`[data-email-id="${email.id}"]`);
        if (!row) return { error: 'no list row for first email' };
        row.classList.add('focused');
        row.focus();
        const list = row.closest('[data-testid="email-list"]') || row.parentElement;
        list.dispatchEvent(new KeyboardEvent('keydown', { key: 'Delete', bubbles: true }));
        return { ok: true };
      });
    }

    // ============== Success path ==============
    await page.evaluate(() => {
      window.__originalDel = AirAPI.deleteEmail;
      AirAPI.deleteEmail = async () => ({ success: true });
    });
    try {
      const success = await pressDeleteOnFirstEmail();
      expect(success.error).toBeUndefined();
      // expect.poll auto-retries up to the default timeout — replaces
      // the brittle 250 ms setTimeout while still waiting for both the
      // deleteEmail await AND the announce setTimeout to fire.
      await expect.poll(
        () => announcer.textContent(),
        { timeout: 2000, message: 'announcer must say "Email deleted" on success' }
      ).toContain('Email deleted');
    } finally {
      await page.evaluate(() => { AirAPI.deleteEmail = window.__originalDel; });
    }

    // Reload to reset list state cleanly between phases.
    await page.reload();
    await expect(page.getByRole('option').first()).toBeVisible({ timeout: 5000 });

    // ============== Failure path ==============
    await page.evaluate(() => {
      window.__originalDel = AirAPI.deleteEmail;
      AirAPI.deleteEmail = async () => { throw new Error('simulated 5xx'); };
    });
    try {
      const failure = await pressDeleteOnFirstEmail();
      expect(failure.error).toBeUndefined();
      const announcer2 = page.getByTestId('announcer');
      await expect.poll(
        () => announcer2.textContent(),
        { timeout: 2000, message: 'announcer must say "Delete failed" on rollback' }
      ).toContain('Delete failed');
      expect(await announcer2.textContent(), 'must NOT announce success on rollback')
        .not.toContain('Email deleted');
    } finally {
      await page.evaluate(() => { AirAPI.deleteEmail = window.__originalDel; });
    }
  });
});
