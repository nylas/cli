// @ts-check
const { test, expect } = require('@playwright/test');
const selectors = require('../../shared/helpers/air-selectors');

/**
 * Regression tests for XSS issues found during the in-depth Air review.
 *
 * Each test loads the live page so all globals (CalendarManager, NotetakerModule,
 * sanitizeHtml, isSafeUrl, escapeHtml, stripHtml-style helpers) are present, then
 * exercises the renderer with hostile inputs and asserts that:
 *   - script-equivalent payloads (onerror, onload, srcdoc, etc.) never run
 *   - javascript: / vbscript: / data:text/html URLs never reach href/src
 */
test.describe('XSS regressions — Air', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await expect(page.locator(selectors.general.app)).toBeVisible();
    await page.waitForLoadState('domcontentloaded');
  });

  test('CalendarManager.stripHtml does not execute markup payloads', async ({ page }) => {
    const result = await page.evaluate(() => {
      // Sentinel that any onerror/onload/onfocus would flip.
      window.__xssExecuted = false;
      const payloads = [
        '<img src="x" onerror="window.__xssExecuted = true">',
        '<svg onload="window.__xssExecuted = true"></svg>',
        '<iframe srcdoc="&lt;script&gt;parent.__xssExecuted = true&lt;/script&gt;"></iframe>',
        '<body onload="window.__xssExecuted = true">hi</body>',
      ];
      const stripped = payloads.map((p) => CalendarManager.stripHtml(p));
      return { executed: window.__xssExecuted, stripped };
    });

    expect(result.executed).toBe(false);
    // stripHtml should still produce text — at minimum no markup tags.
    for (const text of result.stripped) {
      expect(text).not.toContain('<');
      expect(text).not.toContain('>');
    }
  });

  test('renderEventCard sanitizes javascript: conferencing URLs', async ({ page }) => {
    const html = await page.evaluate(() => {
      const malicious = {
        id: 'evt-malicious',
        title: 'Innocent looking meeting',
        description: '<img src=x onerror="window.__xssExecuted = true">Catch up',
        start_time: Math.floor(Date.now() / 1000),
        end_time: Math.floor(Date.now() / 1000) + 1800,
        is_all_day: false,
        participants: [],
        conferencing: { url: 'javascript:window.__xssExecuted=true' },
      };
      window.__xssExecuted = false;
      return CalendarManager.renderEventCard(malicious);
    });

    // The hostile URL must not be present as a clickable href anywhere.
    expect(html).not.toMatch(/href\s*=\s*["']?\s*javascript:/i);
    expect(html).not.toMatch(/href\s*=\s*["']?\s*vbscript:/i);
    // No raw <img onerror> from the description should survive into output.
    expect(html.toLowerCase()).not.toContain('onerror');

    const executed = await page.evaluate(() => window.__xssExecuted === true);
    expect(executed).toBe(false);
  });

  test('renderEventCard preserves safe https conferencing URLs', async ({ page }) => {
    const html = await page.evaluate(() => {
      return CalendarManager.renderEventCard({
        id: 'evt-good',
        title: 'Sales sync',
        description: 'Legit description',
        start_time: Math.floor(Date.now() / 1000),
        end_time: Math.floor(Date.now() / 1000) + 1800,
        is_all_day: false,
        participants: [],
        conferencing: { url: 'https://zoom.us/j/123456' },
      });
    });

    expect(html).toContain('href="https://zoom.us/j/123456"');
    // target=_blank should be paired with rel=noopener.
    expect(html).toMatch(/rel="[^"]*noopener[^"]*"/);
  });

  test('NotetakerModule scheduled link is safe for hostile meetingLink', async ({ page }) => {
    // NotetakerModule is exposed once the notetaker view module has loaded.
    // If the global is unavailable the test simply succeeds — there is nothing
    // to regress against on this build.
    const status = await page.evaluate(() => {
      if (typeof NotetakerModule === 'undefined') return { skipped: true };
      const renderer =
        NotetakerModule.renderPendingContent ||
        (NotetakerModule.actions && NotetakerModule.actions.renderPendingContent);
      if (typeof renderer !== 'function') return { skipped: true };

      const out = renderer.call(NotetakerModule, {
        state: 'scheduled',
        meetingLink: 'javascript:window.__xssExecuted=true',
      });
      return {
        skipped: false,
        html: out,
      };
    });

    if (status.skipped) {
      test.skip(true, 'NotetakerModule.renderPendingContent not available on this build');
      return;
    }

    expect(status.html).not.toMatch(/href\s*=\s*["']?\s*javascript:/i);
  });

  test('utils.isSafeUrl rejects dangerous schemes', async ({ page }) => {
    const verdicts = await page.evaluate(() => ({
      js: isSafeUrl('javascript:alert(1)'),
      vb: isSafeUrl('vbscript:msgbox(1)'),
      dataHtml: isSafeUrl('data:text/html,<script>alert(1)</script>'),
      file: isSafeUrl('file:///etc/passwd'),
      https: isSafeUrl('https://example.com'),
      mailto: isSafeUrl('mailto:user@example.com'),
      relative: isSafeUrl('/path/to/page'),
      anchor: isSafeUrl('#section'),
      dataPng: isSafeUrl('data:image/png;base64,abc'),
    }));

    expect(verdicts.js).toBe(false);
    expect(verdicts.vb).toBe(false);
    expect(verdicts.dataHtml).toBe(false);
    expect(verdicts.file).toBe(false);
    expect(verdicts.https).toBe(true);
    expect(verdicts.mailto).toBe(true);
    expect(verdicts.relative).toBe(true);
    expect(verdicts.anchor).toBe(true);
    expect(verdicts.dataPng).toBe(true);
  });

  test('escapeHtml escapes attribute-context delimiters', async ({ page }) => {
    const result = await page.evaluate(() => ({
      quote: escapeHtml('"'),
      apos: escapeHtml("'"),
      lt: escapeHtml('<'),
      gt: escapeHtml('>'),
      amp: escapeHtml('&'),
      mixed: escapeHtml(`<img src="x" onerror='alert(1)'>`),
      nullish: escapeHtml(null),
    }));

    expect(result.quote).toBe('&quot;');
    expect(result.apos).toBe('&#39;');
    expect(result.lt).toBe('&lt;');
    expect(result.gt).toBe('&gt;');
    expect(result.amp).toBe('&amp;');
    expect(result.mixed).not.toContain('<');
    expect(result.mixed).not.toContain('"');
    expect(result.mixed).not.toContain("'");
    expect(result.nullish).toBe('');
  });

  test('NotetakerModule.getMedia URL-encodes notetaker IDs with special chars', async ({ page }) => {
    // We don't care about the response, only that the URL is properly encoded.
    let capturedUrl = '';
    await page.route('**/api/notetakers/media**', async (route) => {
      capturedUrl = route.request().url();
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({}),
      });
    });

    const status = await page.evaluate(async () => {
      if (typeof NotetakerModule === 'undefined' || typeof NotetakerModule.getMedia !== 'function') {
        return { skipped: true };
      }
      try {
        await NotetakerModule.getMedia('id with spaces & ?weird#chars');
      } catch {
        // The mocked fulfill returns {} which may throw — ignore.
      }
      return { skipped: false };
    });

    if (status.skipped) {
      test.skip(true, 'NotetakerModule.getMedia not available on this build');
      return;
    }

    // The raw special characters must NOT appear unencoded in the URL.
    expect(capturedUrl).not.toContain(' ');
    expect(capturedUrl).not.toContain('#');
    // Spaces should become %20, ? should be %3F when in the value, & should be %26.
    expect(capturedUrl).toMatch(/id=[^&#?]*%20[^&#?]*/);
  });

  test('touch-gestures performAction URL-encodes hostile email IDs', async ({ page }) => {
    let capturedUrl = '';
    await page.route('**/api/emails/**', async (route) => {
      capturedUrl = route.request().url();
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({}),
      });
    });

    const status = await page.evaluate(async () => {
      if (typeof TouchGestures === 'undefined' || typeof TouchGestures.performAction !== 'function') {
        return { skipped: true };
      }
      await TouchGestures.performAction('a/b?c=1#x', 'archive');
      return { skipped: false };
    });

    if (status.skipped) {
      test.skip(true, 'TouchGestures.performAction not available on this build');
      return;
    }

    expect(capturedUrl).not.toContain('?c=1');
    expect(capturedUrl).not.toContain('#x');
    // The slash, ?, # should all be percent-encoded inside the path segment.
    expect(capturedUrl).toMatch(/\/api\/emails\/[^/?#]+/);
  });

  test('gradientFor returns a stable CSS var across calls', async ({ page }) => {
    const result = await page.evaluate(() => {
      const a1 = gradientFor('alice@example.com');
      const a2 = gradientFor('alice@example.com');
      const b = gradientFor('bob@example.com');
      const empty1 = gradientFor('');
      const empty2 = gradientFor(null);
      return { a1, a2, b, empty1, empty2 };
    });

    expect(result.a1).toBe(result.a2);
    expect(result.a1).toMatch(/^var\(--gradient-[1-5]\)$/);
    expect(result.b).toMatch(/^var\(--gradient-[1-5]\)$/);
    expect(result.empty1).toBe(result.empty2);
  });

  // ---- Notetaker detail rendering ---------------------------------------
  // Regression for the Round-4 finding: nt.summary, nt.meetingTitle, and
  // nt.attendees flowed unsanitised into innerHTML inside renderDetail() and
  // stripEmbeddedStyles() only stripped <style>, leaving <script>, <img
  // onerror>, etc. Now stripEmbeddedStyles routes through sanitizeHtml and
  // the title/attendees are escaped before interpolation.

  test('NotetakerModule.stripEmbeddedStyles strips dangerous tags AND inline styles', async ({
    page,
  }) => {
    const status = await page.evaluate(() => {
      if (
        typeof NotetakerModule === 'undefined' ||
        typeof NotetakerModule.stripEmbeddedStyles !== 'function'
      ) {
        return { skipped: true };
      }
      window.__xssExecuted = false;
      const hostile =
        '<style>body{background:red}</style>' +
        '<p style="color:red">styled</p>' +
        '<img src="x" onerror="window.__xssExecuted = true">' +
        '<svg onload="window.__xssExecuted = true"></svg>' +
        '<iframe src="javascript:window.__xssExecuted = true"></iframe>' +
        '<script>window.__xssExecuted = true</script>' +
        '<a href="javascript:window.__xssExecuted = true">click</a>';
      const cleaned = NotetakerModule.stripEmbeddedStyles(hostile);

      // Force the cleaned output into the live DOM so any latent payload
      // would actually run; insertAdjacentHTML matches how the real renderer
      // commits sanitized markup.
      const sandbox = document.createElement('div');
      document.body.appendChild(sandbox);
      sandbox.insertAdjacentHTML('beforeend', cleaned);

      return new Promise((resolve) =>
        setTimeout(() => {
          const xss = window.__xssExecuted === true;
          sandbox.remove();
          resolve({ skipped: false, cleaned, xss });
        }, 50),
      );
    });

    if (status.skipped) {
      test.skip(true, 'NotetakerModule.stripEmbeddedStyles not available on this build');
      return;
    }

    expect(status.xss).toBe(false);
    expect(status.cleaned.toLowerCase()).not.toContain('<script');
    expect(status.cleaned.toLowerCase()).not.toContain('<iframe');
    expect(status.cleaned.toLowerCase()).not.toContain('onerror');
    expect(status.cleaned.toLowerCase()).not.toContain('onload');
    // Dangerous href stripped — the `<a>` tag may survive but without href.
    expect(status.cleaned).not.toMatch(/href\s*=\s*["']?\s*javascript:/i);
    // <style> and inline style="" must be removed.
    expect(status.cleaned.toLowerCase()).not.toContain('<style');
    expect(status.cleaned).not.toMatch(/\sstyle="/i);
  });

  test('NotetakerModule.renderDetail escapes hostile title and attendees', async ({ page }) => {
    const status = await page.evaluate(() => {
      if (
        typeof NotetakerModule === 'undefined' ||
        typeof NotetakerModule.renderDetail !== 'function'
      ) {
        return { skipped: true };
      }

      // Provide the detail container the renderer expects.
      let detail = document.getElementById('notetakerDetail');
      if (!detail) {
        detail = document.createElement('div');
        detail.id = 'notetakerDetail';
        document.body.appendChild(detail);
      }

      window.__xssExecuted = false;

      NotetakerModule.selectedNotetaker = {
        id: 'nt-hostile',
        state: 'completed',
        provider: '<img src=x onerror="window.__xssExecuted=true">Provider',
        meetingTitle: '<img src=x onerror="window.__xssExecuted=true">Title',
        attendees: '<svg/onload="window.__xssExecuted=true">attendee@example.com',
        createdAt: new Date().toISOString(),
      };

      NotetakerModule.renderDetail();

      return new Promise((resolve) =>
        setTimeout(() => {
          const xss = window.__xssExecuted === true;
          const titleEl = detail.querySelector('h2');
          const attendeesEl = detail.querySelector('.notetaker-detail-attendees');
          resolve({
            skipped: false,
            xss,
            titleMarkup: titleEl ? titleEl.outerHTML : '',
            titleText: titleEl ? titleEl.textContent : '',
            attendeesMarkup: attendeesEl ? attendeesEl.outerHTML : '',
            attendeesText: attendeesEl ? attendeesEl.textContent : '',
          });
        }, 50),
      );
    });

    if (status.skipped) {
      test.skip(true, 'NotetakerModule.renderDetail not available on this build');
      return;
    }

    expect(status.xss).toBe(false);
    // Hostile <img/onerror> / <svg/onload> must have been escaped before
    // interpolation, so neither the literal tag nor an event-handler attr
    // survives in the DOM, but the visible text retains the readable label.
    expect(status.titleMarkup.toLowerCase()).not.toMatch(/<img[^>]*onerror/);
    expect(status.titleText).toContain('Title');
    expect(status.attendeesMarkup.toLowerCase()).not.toMatch(/<svg[^>]*onload/);
    expect(status.attendeesText).toContain('attendee@example.com');
  });
});
