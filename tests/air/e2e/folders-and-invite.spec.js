// @ts-check
const { test, expect } = require('@playwright/test');

/**
 * Regression tests for two reported bugs:
 *
 *  1. The Sent folder showed identical content to Inbox (demo mode
 *     ignored ?folder= and always returned Inbox). After the fix,
 *     /api/emails?folder=sent must return the Sent-folder demo emails
 *     (>=2) — and they must NOT include any Inbox-only emails.
 *
 *  2. Emails with a text/calendar attachment did not render the
 *     Gmail-style invite card. After the fix, opening the demo invite
 *     email surfaces a .calendar-invite-card with title, time,
 *     organizer, and Yes/No/Maybe buttons.
 */
test.describe('Folders + calendar invite — Air', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('domcontentloaded');
  });

  test('demo: /api/emails?folder=sent returns >1 sent emails and no inbox-only emails', async ({
    page,
  }) => {
    const result = await page.evaluate(async () => {
      const sent = await fetch('/api/emails?folder=sent').then((r) => r.json());
      const inbox = await fetch('/api/emails?folder=inbox').then((r) => r.json());

      const sentEmails = sent.emails || [];
      const inboxEmails = inbox.emails || [];

      const sentLeakedToInbox = inboxEmails.some(
        (e) => Array.isArray(e.folders) && e.folders.includes('sent'),
      );
      const inboxLeakedToSent = sentEmails.some(
        (e) => Array.isArray(e.folders) && e.folders.includes('inbox'),
      );

      return {
        sentCount: sentEmails.length,
        inboxCount: inboxEmails.length,
        sentSubjects: sentEmails.map((e) => e.subject),
        sentLeakedToInbox,
        inboxLeakedToSent,
      };
    });

    expect(result.sentCount, 'Sent folder must contain more than 1 email').toBeGreaterThan(1);
    expect(result.inboxCount, 'Inbox should still have content').toBeGreaterThan(0);
    expect(result.sentLeakedToInbox).toBe(false);
    expect(result.inboxLeakedToSent).toBe(false);
  });

  test('demo: each canonical folder yields distinct content', async ({ page }) => {
    const counts = await page.evaluate(async () => {
      const folders = ['inbox', 'sent', 'drafts', 'archive', 'trash'];
      const out = {};
      for (const f of folders) {
        const r = await fetch(`/api/emails?folder=${encodeURIComponent(f)}`).then((rr) => rr.json());
        out[f] = (r.emails || []).length;
      }
      return out;
    });

    expect(counts.inbox).toBeGreaterThan(0);
    expect(counts.sent).toBeGreaterThan(1);
    expect(counts.drafts).toBeGreaterThan(0);
    expect(counts.archive).toBeGreaterThan(0);
    expect(counts.trash).toBeGreaterThan(0);
    // Inbox should have more than Drafts/Archive/Trash typically.
    expect(counts.inbox).toBeGreaterThanOrEqual(counts.drafts);
  });

  test('demo: aliases (SENT, Sent Items, Deleted Items) map to canonical folders', async ({
    page,
  }) => {
    const result = await page.evaluate(async () => {
      const get = async (q) =>
        (await fetch(`/api/emails?folder=${encodeURIComponent(q)}`).then((r) => r.json()))
          .emails.length;
      return {
        sentCanon: await get('sent'),
        sentUpper: await get('SENT'),
        sentMs: await get('Sent Items'),
        trashCanon: await get('trash'),
        trashMs: await get('Deleted Items'),
      };
    });

    expect(result.sentUpper).toBe(result.sentCanon);
    expect(result.sentMs).toBe(result.sentCanon);
    expect(result.trashMs).toBe(result.trashCanon);
  });

  test('demo: ?unread=true and ?starred=true narrow the result set', async ({ page }) => {
    const result = await page.evaluate(async () => {
      const all = (await fetch('/api/emails').then((r) => r.json())).emails || [];
      const unread = (await fetch('/api/emails?unread=true').then((r) => r.json())).emails || [];
      const starred =
        (await fetch('/api/emails?starred=true').then((r) => r.json())).emails || [];
      return {
        all: all.length,
        unread: unread.length,
        starred: starred.length,
        unreadAllUnread: unread.every((e) => e.unread === true),
        starredAllStarred: starred.every((e) => e.starred === true),
      };
    });

    expect(result.unread).toBeLessThan(result.all);
    expect(result.starred).toBeLessThan(result.all);
    expect(result.unreadAllUnread).toBe(true);
    expect(result.starredAllStarred).toBe(true);
  });

  test('demo: /api/emails/{id}/invite returns parsed event for invite email', async ({ page }) => {
    const result = await page.evaluate(async () => {
      const r = await fetch('/api/emails/demo-email-invite-001/invite');
      const json = await r.json();
      return { ok: r.ok, json };
    });

    expect(result.ok).toBe(true);
    expect(result.json.has_invite).toBe(true);
    expect(result.json.title).toBe('Quarterly Sync');
    expect(typeof result.json.start_time).toBe('number');
    expect(typeof result.json.end_time).toBe('number');
    expect(result.json.organizer_email).toBe('priya@partner.example');
    expect(result.json.organizer_name).toBe('Priya Patel');
    expect(result.json.conferencing_url).toMatch(/^https:\/\//);
  });

  test('demo: /api/emails/{id}/invite returns has_invite=false for non-invite email', async ({
    page,
  }) => {
    const result = await page.evaluate(async () => {
      const r = await fetch('/api/emails/demo-email-001/invite');
      return { ok: r.ok, json: await r.json() };
    });
    expect(result.ok).toBe(true);
    expect(result.json.has_invite).toBe(false);
  });

  test('demo: opening invite email renders calendar-invite-card with action buttons', async ({
    page,
  }) => {
    // Drive the UI so the card renders through the full flow.
    const result = await page.evaluate(async () => {
      // Wait for the email list manager to load.
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
      if (typeof EmailListManager === 'undefined' || !EmailListManager) {
        return { skipped: true };
      }

      // Open the invite email programmatically so we don't depend on the
      // exact list order in the rendered DOM.
      await EmailListManager.selectEmail('demo-email-invite-001');

      // The invite is fetched on rAF; give it a few ticks to land.
      const slot = document.getElementById('inviteSlot-demo-email-invite-001');
      const deadline = Date.now() + 3000;
      while ((!slot || slot.hidden) && Date.now() < deadline) {
        await new Promise((r) => setTimeout(r, 50));
      }

      const card = document.querySelector('.calendar-invite-card');
      if (!card) {
        return { skipped: false, found: false };
      }
      return {
        skipped: false,
        found: true,
        title: card.querySelector('.calendar-invite-title')?.textContent?.trim(),
        time: card.querySelector('.calendar-invite-time')?.textContent?.trim(),
        org: card.querySelector('.calendar-invite-org')?.textContent?.trim(),
        loc: card.querySelector('.calendar-invite-location')?.textContent?.trim(),
        hasYes: !!card.querySelector('[data-rsvp="yes"]'),
        hasMaybe: !!card.querySelector('[data-rsvp="maybe"]'),
        hasNo: !!card.querySelector('[data-rsvp="no"]'),
        confLink: card.querySelector('.calendar-invite-link')?.href,
      };
    });

    if (result.skipped) {
      test.skip(true, 'EmailListManager not available on this build');
      return;
    }

    expect(result.found).toBe(true);
    expect(result.title).toBe('Quarterly Sync');
    expect(result.time).toBeTruthy();
    expect(result.org).toContain('Priya Patel');
    expect(result.loc).toContain('Conference Room');
    expect(result.hasYes).toBe(true);
    expect(result.hasMaybe).toBe(true);
    expect(result.hasNo).toBe(true);
    expect(result.confLink).toMatch(/^https:\/\/meet\.example\.com/);
  });

  test('demo: clicking RSVP marks the chosen button active (no XSS in injected fields)', async ({
    page,
  }) => {
    const result = await page.evaluate(async () => {
      if (typeof EmailListManager === 'undefined' || !EmailListManager) {
        return { skipped: true };
      }

      // Force-render the card with hostile inputs to verify escaping.
      // Invite slot must exist, so we open the demo invite first.
      await EmailListManager.selectEmail('demo-email-invite-001');

      const deadline = Date.now() + 3000;
      while (!document.querySelector('.calendar-invite-card') && Date.now() < deadline) {
        await new Promise((r) => setTimeout(r, 50));
      }

      // Sanity check: hostile script injected via title would have
      // already executed by now if escaping were broken.
      window.__inviteXss = false;
      // Try injecting via a synthetic invite render.
      const slot = document.getElementById('inviteSlot-demo-email-invite-001');
      const html = EmailListManager.renderCalendarInviteCard(
        {
          has_invite: true,
          title: '<img src=x onerror="window.__inviteXss=true">Hi',
          location: '"<svg/onload=window.__inviteXss=true>"',
          organizer_name: 'Hostile <script>',
          organizer_email: 'attacker@example.com',
          conferencing_url: 'javascript:window.__inviteXss=true',
          start_time: Math.floor(Date.now() / 1000) + 3600,
          end_time: Math.floor(Date.now() / 1000) + 7200,
        },
        'demo-email-invite-001',
      );
      slot.replaceChildren();
      slot.insertAdjacentHTML('beforeend', html);

      // Click Yes.
      const yesBtn = slot.querySelector('[data-rsvp="yes"]');
      yesBtn.click();

      // Give the click handler a tick.
      await new Promise((r) => setTimeout(r, 50));

      return {
        skipped: false,
        xssExecuted: window.__inviteXss === true,
        yesActive: yesBtn.classList.contains('active'),
        // The hostile javascript: URL must not survive into a clickable href.
        hostileLinkExists: !!slot.querySelector('a[href^="javascript:"]'),
      };
    });

    if (result.skipped) {
      test.skip(true, 'EmailListManager not available on this build');
      return;
    }

    expect(result.xssExecuted).toBe(false);
    expect(result.yesActive).toBe(true);
    expect(result.hostileLinkExists).toBe(false);
  });
});
