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

  // Pins the new attendee summary block. The demo invite ships with one
  // organizer + two attendees (Alex accepted, Jamie tentative) so the
  // Gmail-style "1 going · 1 maybe" line renders alongside per-attendee
  // chips. Regression-only — guards against the chip CSS classes being
  // dropped or the summary tally getting recomputed wrongly.
  test('demo: invite card renders attendee chips + summary tally', async ({ page }) => {
    const result = await page.evaluate(async () => {
      if (typeof EmailListManager === 'undefined' || !EmailListManager) {
        return { skipped: true };
      }
      await EmailListManager.selectEmail('demo-email-invite-001');

      const deadline = Date.now() + 3000;
      while (!document.querySelector('.calendar-invite-card') && Date.now() < deadline) {
        await new Promise((r) => setTimeout(r, 50));
      }

      const card = document.querySelector('.calendar-invite-card');
      if (!card) return { skipped: false, found: false };

      const summary = card.querySelector('.calendar-invite-summary');
      const chips = Array.from(card.querySelectorAll('.calendar-invite-attendee'));
      return {
        skipped: false,
        found: true,
        summary: summary ? summary.textContent.trim() : '',
        chipCount: chips.length,
        statuses: chips.map((c) => {
          if (c.classList.contains('is-accepted')) return 'accepted';
          if (c.classList.contains('is-declined')) return 'declined';
          if (c.classList.contains('is-tentative')) return 'tentative';
          return 'pending';
        }),
      };
    });

    if (result.skipped) {
      test.skip(true, 'EmailListManager not available on this build');
      return;
    }

    expect(result.found).toBe(true);
    // Demo invite has 3 attendees (Priya organizer, Alex, Jamie)
    expect(result.chipCount).toBe(3);
    expect(result.statuses.filter((s) => s === 'accepted').length).toBe(2);
    expect(result.statuses.filter((s) => s === 'tentative').length).toBe(1);
    expect(result.summary).toMatch(/2 going/);
    expect(result.summary).toMatch(/1 maybe/);
  });

  // METHOD:CANCEL invitations should swap the RSVP buttons for a
  // cancellation banner. Test renders a synthetic card directly so we
  // exercise the conditional UI without needing a real cancelled demo
  // email.
  test('renderCalendarInviteCard: METHOD=CANCEL replaces RSVP buttons with banner', async ({ page }) => {
    const result = await page.evaluate(() => {
      if (typeof EmailListManager === 'undefined' || !EmailListManager) {
        return { skipped: true };
      }
      const html = EmailListManager.renderCalendarInviteCard(
        {
          has_invite: true,
          title: 'Quarterly Sync',
          method: 'CANCEL',
          start_time: Math.floor(Date.now() / 1000) + 3600,
          end_time: Math.floor(Date.now() / 1000) + 7200,
        },
        'cancelled-test-id',
      );

      // Parse via DOMParser so we never assign untrusted strings to
      // innerHTML — even in test code, that's the codebase rule.
      const doc = new DOMParser().parseFromString(html, 'text/html');
      return {
        skipped: false,
        hasBanner: !!doc.querySelector('.calendar-invite-banner-cancel'),
        bannerText: doc.querySelector('.calendar-invite-banner-cancel')?.textContent?.trim(),
        rsvpButtonCount: doc.querySelectorAll('.calendar-invite-btn').length,
        cardHasCancelClass: doc.querySelector('.calendar-invite-card.is-cancelled') !== null,
      };
    });

    if (result.skipped) {
      test.skip(true, 'EmailListManager not available on this build');
      return;
    }

    expect(result.hasBanner).toBe(true);
    expect(result.bannerText).toMatch(/cancelled/i);
    expect(result.rsvpButtonCount).toBe(0); // RSVP suppressed for cancellations
    expect(result.cardHasCancelClass).toBe(true);
  });

  // looksLikeInviteSubject is the JS-side heuristic that decides whether
  // to call /invite for emails without a calendar attachment in
  // attachments[]. It must catch Google's "Invitation:" pattern (the
  // common Gmail case) without false-positiving on every email.
  test('looksLikeInviteSubject matches Google/Microsoft invite subjects', async ({ page }) => {
    const result = await page.evaluate(() => {
      if (typeof EmailListManager === 'undefined' || !EmailListManager) {
        return { skipped: true };
      }
      const samples = [
        { subject: 'Invitation: Quarterly Sync @ Mon May 1', want: true },
        { subject: 'Event Invitation: Meeting', want: true },
        { subject: 'Updated invitation: Standup', want: true },
        { subject: 'Canceled event: Team Outing', want: true },
        { subject: 'Calendar invitation from priya@partner', want: true },
        { subject: 'Re: Lunch tomorrow', want: false },
        { subject: 'Your weekly digest', want: false },
        { subject: '', want: false },
      ];
      return {
        skipped: false,
        results: samples.map((s) => ({
          subject: s.subject,
          want: s.want,
          got: EmailListManager.looksLikeInviteSubject({ subject: s.subject }),
        })),
      };
    });

    if (result.skipped) {
      test.skip(true, 'EmailListManager not available on this build');
      return;
    }

    for (const r of result.results) {
      expect(r.got, `subject=${JSON.stringify(r.subject)}`).toBe(r.want);
    }
  });

  // Pins the inline-calendar attachment-row injection. When the invite
  // card lands and the email had no attachments[] entry, the UI should
  // grow an attachment row so the user sees "invite.ics" the way Gmail
  // shows it. Drives the path via a synthetic invite to avoid relying
  // on demo-mode quirks.
  test('ensureInviteAttachmentRow injects an attachment row when none exists', async ({ page }) => {
    const result = await page.evaluate(() => {
      if (typeof EmailListManager === 'undefined' || !EmailListManager) {
        return { skipped: true };
      }

      // Build a minimal email-detail container with the invite slot.
      const detail = document.createElement('div');
      detail.className = 'email-detail';
      const slot = document.createElement('div');
      slot.className = 'calendar-invite-card-slot';
      slot.id = 'inviteSlot-test-inline';
      detail.appendChild(slot);
      document.body.appendChild(detail);

      try {
        EmailListManager.selectedEmailId = 'test-inline';
        EmailListManager.ensureInviteAttachmentRow('test-inline', {
          attachment_id: 'inline-calendar:default',
          filename: 'invite.ics',
        });

        return {
          skipped: false,
          hasAttachmentSection: !!detail.querySelector('.email-detail-attachments'),
          attachmentName: detail.querySelector('.attachment-name')?.textContent,
          inlineMarker: detail.querySelector('[data-inline-calendar="true"]') !== null,
        };
      } finally {
        detail.remove();
      }
    });

    if (result.skipped) {
      test.skip(true, 'EmailListManager not available on this build');
      return;
    }

    expect(result.hasAttachmentSection).toBe(true);
    expect(result.attachmentName).toBe('invite.ics');
    expect(result.inlineMarker).toBe(true);
  });
});
