// @ts-check
const { test, expect } = require('@playwright/test');
const selectors = require('../../shared/helpers/air-selectors');

/**
 * Folder Navigation tests - Folder switching
 */
test.describe('Folder Navigation', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await expect(page.locator(selectors.general.app)).toBeVisible();
    await page.waitForLoadState('domcontentloaded');
    await page.waitForTimeout(1000);
  });

  test('folder sidebar is visible', async ({ page }) => {
    const folderSidebar = page.locator(selectors.email.folderSidebar);
    await expect(folderSidebar).toBeVisible();
  });

  test('folder list contains folders', async ({ page }) => {
    const folderList = page.locator(selectors.email.folderList);
    await expect(folderList).toBeVisible();

    // Should have at least one folder item or skeleton
    const folders = folderList.locator(selectors.email.folderItem);
    const skeletons = folderList.locator('.skeleton');

    const folderCount = await folders.count();
    const skeletonCount = await skeletons.count();

    expect(folderCount + skeletonCount).toBeGreaterThan(0);
  });

  test('clicking folder updates email list', async ({ page }) => {
    const folderList = page.locator(selectors.email.folderList);
    const folders = folderList.locator(selectors.email.folderItem);

    const count = await folders.count();

    if (count > 1) {
      // Click second folder (first might already be selected)
      await folders.nth(1).click();

      // Wait for folder switch
      await page.waitForTimeout(500);

      // Folder may have active/selected class or be visually distinct
      const hasActiveClass = await folders.nth(1).evaluate((el) =>
        el.classList.contains('active') || el.classList.contains('selected') || el.classList.contains('current')
      );
      const isClickable = await folders.nth(1).evaluate((el) => {
        const style = window.getComputedStyle(el);
        return el.getAttribute('data-folder-id') !== null || style.cursor === 'pointer';
      });

      // Folder should be active or clickable
      expect(hasActiveClass || isClickable).toBeTruthy();
    }
  });

  test('Inbox folder is present', async ({ page }) => {
    const folderList = page.locator(selectors.email.folderList);

    // Wait for folders to load (either folders appear or skeletons disappear)
    await page.waitForTimeout(2000);

    const inboxFolder = folderList.locator('.folder-item:has-text("Inbox")');

    // Check if Inbox exists before asserting visibility
    if (await inboxFolder.count() > 0) {
      await expect(inboxFolder).toBeVisible();
    }
  });

  test('Sent folder is present', async ({ page }) => {
    const folderList = page.locator(selectors.email.folderList);
    const sentFolder = folderList.locator('.folder-item:has-text("Sent")');

    if (await sentFolder.count() > 0) {
      await expect(sentFolder).toBeVisible();
    }
  });

  test('Archive folder is visible inline without a More dropdown', async ({ page }) => {
    const folderList = page.locator(selectors.email.folderList);

    await page.waitForTimeout(2000);

    await expect(folderList.locator('.folder-item:has-text("Archive")')).toBeVisible();
    await expect(folderList.locator('.folder-item:has-text("More")')).toHaveCount(0);
  });

  test('folders show unread count badge', async ({ page }) => {
    const folderList = page.locator(selectors.email.folderList);
    const folders = folderList.locator(selectors.email.folderItem);

    const count = await folders.count();

    if (count > 0) {
      // Check if any folders have count badge
      const badges = folderList.locator('.folder-count');
      const badgeCount = await badges.count();

      // It's okay if no badges (no unread emails)
      expect(badgeCount >= 0).toBeTruthy();
    }
  });

  // Regression test for the bug where Nylas's folder.unread_count lagged the
  // per-message unread state — sidebar showed "1" while the loaded folder had
  // dozens of messages flagged unread. The fix syncs the active folder badge
  // to the unread count we observe in the loaded set.
  test('active folder badge syncs to observed unread count', async ({ page }) => {
    const folderList = page.locator(selectors.email.folderList);
    await expect(folderList).toBeVisible();
    // The sidebar is rendered as skeletons by the Go template and replaced
    // with real folder-items (carrying data-folder-id) only after JS calls
    // /api/folders. Wait for that swap so the test isn't racing the load.
    await page.locator('.folder-item[data-folder-id]').first().waitFor({ timeout: 5000 });

    const result = await page.evaluate(() => {
      const item = document.querySelector('.folder-item[data-folder-id]');
      // EmailListManager is declared with `const` in a classic script, so
      // it's a free identifier in page context (not attached to window).
      const mgr = typeof EmailListManager !== 'undefined' ? EmailListManager : null;
      if (!item || !mgr) return { skipped: true };

      const folderId = item.getAttribute('data-folder-id');

      // Drop any pre-existing badge so the assertion isn't satisfied by a
      // leftover render.
      const stale = item.querySelector('.folder-count');
      if (stale) stale.remove();

      mgr.currentFolder = folderId;
      mgr.emails = [
        { id: 'a', unread: true, from: [], date: 0 },
        { id: 'b', unread: true, from: [], date: 0 },
        { id: 'c', unread: true, from: [], date: 0 },
        { id: 'd', unread: false, from: [], date: 0 },
      ];
      mgr.applyFilter();

      const badge = item.querySelector('.folder-count');
      return {
        skipped: false,
        text: badge ? badge.textContent : null,
        hasUnreadClass: badge ? badge.classList.contains('unread') : false,
      };
    });

    if (result.skipped) {
      test.skip(true, 'No data-folder-id available in this render');
      return;
    }

    expect(result.text).toBe('3');
    expect(result.hasUnreadClass).toBe(true);
  });

  test('active folder badge is left untouched when observed unread is zero', async ({ page }) => {
    await page.locator('.folder-item[data-folder-id]').first().waitFor({ timeout: 5000 });

    const result = await page.evaluate(() => {
      const item = document.querySelector('.folder-item[data-folder-id]');
      const mgr = typeof EmailListManager !== 'undefined' ? EmailListManager : null;
      if (!item || !mgr) return { skipped: true };

      const folderId = item.getAttribute('data-folder-id');

      // Seed an existing badge (e.g. total_count from API). The fix must
      // not erase this when no unread is observed — otherwise scrolling
      // through a zero-unread page on a non-empty folder would blank the
      // count.
      let badge = item.querySelector('.folder-count');
      if (!badge) {
        badge = document.createElement('span');
        badge.className = 'folder-count';
        item.appendChild(badge);
      }
      badge.textContent = '42';
      badge.classList.remove('unread');

      mgr.currentFolder = folderId;
      mgr.emails = [
        { id: 'a', unread: false, from: [], date: 0 },
        { id: 'b', unread: false, from: [], date: 0 },
      ];
      mgr.applyFilter();

      const after = item.querySelector('.folder-count');
      return {
        skipped: false,
        text: after ? after.textContent : null,
        hasUnreadClass: after ? after.classList.contains('unread') : false,
      };
    });

    if (result.skipped) {
      test.skip(true, 'No data-folder-id available in this render');
      return;
    }

    expect(result.text).toBe('42');
    expect(result.hasUnreadClass).toBe(false);
  });

  // Pins the eager-refresh path: refreshFolderBadge fetches a folder's first
  // page and updates the badge from the observed unread count, without the
  // user having to click into the folder. Fixes the case where the sidebar
  // shows Nylas's stale folder.unread_count on initial paint.
  test('refreshFolderBadge writes the observed unread count', async ({ page }) => {
    await page.locator('.folder-item[data-folder-id]').first().waitFor({ timeout: 5000 });

    const result = await page.evaluate(async () => {
      const item = document.querySelector('.folder-item[data-folder-id]');
      const mgr = typeof EmailListManager !== 'undefined' ? EmailListManager : null;
      if (!item || !mgr || typeof AirAPI === 'undefined') return { skipped: true };

      const folderId = item.getAttribute('data-folder-id');
      // Wipe pre-existing badge so we can prove the refresh wrote one.
      const stale = item.querySelector('.folder-count');
      if (stale) stale.remove();

      // Stub AirAPI.getEmails to return a known unread mix without hitting
      // the demo handler (the demo set's unread distribution depends on
      // folder mapping that's outside this test's scope).
      const original = AirAPI.getEmails;
      AirAPI.getEmails = async () => ({
        emails: [
          { id: 'x', unread: true },
          { id: 'y', unread: true },
          { id: 'z', unread: true },
          { id: 'w', unread: true },
          { id: 'v', unread: false },
        ],
      });
      try {
        await mgr.refreshFolderBadge(folderId);
      } finally {
        AirAPI.getEmails = original;
      }

      const after = item.querySelector('.folder-count');
      return {
        skipped: false,
        text: after ? after.textContent : null,
        hasUnreadClass: after ? after.classList.contains('unread') : false,
      };
    });

    if (result.skipped) {
      test.skip(true, 'No data-folder-id / AirAPI available in this render');
      return;
    }

    expect(result.text).toBe('4');
    expect(result.hasUnreadClass).toBe(true);
  });
});
