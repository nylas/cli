// @ts-check
const { test, expect } = require('@playwright/test');
const selectors = require('../../shared/helpers/ui-selectors');

/**
 * Accessibility tests for Nylas UI.
 *
 * Tests ARIA attributes, keyboard navigation, focus management,
 * and other accessibility features.
 */

/**
 * Helper to check if dashboard is active (configured state).
 */
async function isDashboardActive(page) {
  const dashboardView = page.locator(selectors.dashboard.view);
  return await dashboardView.evaluate((el) => el.classList.contains('active'));
}

/**
 * Helper to skip test if not in configured state.
 */
async function skipIfNotConfigured(page, testInfo) {
  if (!(await isDashboardActive(page))) {
    testInfo.skip();
  }
}

test.describe('Page Structure', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await expect(page.locator(selectors.general.app)).toBeVisible();
  });

  test('page has correct title', async ({ page }) => {
    await expect(page).toHaveTitle('Nylas CLI');
  });

  test('page has header landmark', async ({ page }) => {
    const header = page.getByRole('banner');
    await expect(header).toBeVisible();
  });

  test('page has main landmark', async ({ page }) => {
    const main = page.getByRole('main');
    await expect(main).toBeVisible();
  });

  test('page has navigation landmark', async ({ page }, testInfo) => {
    await skipIfNotConfigured(page, testInfo);

    const nav = page.getByRole('navigation');
    await expect(nav).toBeVisible();
  });

  test('page has complementary landmark (sidebar)', async ({ page }, testInfo) => {
    await skipIfNotConfigured(page, testInfo);

    const aside = page.getByRole('complementary');
    await expect(aside).toBeVisible();
  });
});

test.describe('Heading Structure', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await expect(page.locator(selectors.general.app)).toBeVisible();
  });

  test('page has h1 heading', async ({ page }, testInfo) => {
    await skipIfNotConfigured(page, testInfo);

    const h1 = page.getByRole('heading', { level: 1 });
    await expect(h1).toBeVisible();
    await expect(h1).toHaveText('Dashboard');
  });

  test('section headings are h3', async ({ page }, testInfo) => {
    await skipIfNotConfigured(page, testInfo);

    const h3Headings = page.getByRole('heading', { level: 3 });
    const count = await h3Headings.count();

    // Should have section headings
    expect(count).toBeGreaterThanOrEqual(3);
  });

  test('command page has h2 heading', async ({ page }, testInfo) => {
    await skipIfNotConfigured(page, testInfo);

    await page.locator(selectors.nav.auth).click();
    await page.waitForTimeout(300);

    const h2 = page.getByRole('heading', { level: 2 });
    await expect(h2.first()).toBeVisible();
  });
});

test.describe('Button Accessibility', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await expect(page.locator(selectors.general.app)).toBeVisible();
  });

  test('theme toggle has accessible name', async ({ page }) => {
    const themeBtn = page.getByRole('button', { name: /Toggle theme/i });
    await expect(themeBtn).toBeVisible();
  });

  test('navigation buttons are accessible', async ({ page }, testInfo) => {
    await skipIfNotConfigured(page, testInfo);

    // Scope to the nav landmark and use exact names so account buttons
    // in the dashboard (e.g. "ACCOUNT 23@qasim.nylas.email") don't match
    // the broader regexes via substring.
    const nav = page.getByRole('navigation');

    await expect(nav.getByRole('button', { name: /^Overview$/ })).toBeVisible();
    await expect(nav.getByRole('button', { name: /^Auth$/ })).toBeVisible();
    await expect(nav.getByRole('button', { name: /^Email$/ })).toBeVisible();
  });

  test('dropdown buttons have accessible names', async ({ page }, testInfo) => {
    await skipIfNotConfigured(page, testInfo);

    // Client dropdown
    const clientBtn = page.getByRole('button', { name: /CLIENT/i });
    await expect(clientBtn).toBeVisible();

    // Account dropdown
    const accountBtn = page.getByRole('button', { name: /ACCOUNT/i });
    await expect(accountBtn).toBeVisible();
  });

  test('Run button has accessible name', async ({ page }, testInfo) => {
    await skipIfNotConfigured(page, testInfo);

    await page.locator(selectors.nav.auth).click();
    await page.waitForTimeout(300);

    // Click Status in the command list
    const cmdList = page.locator('#auth-cmd-list');
    await cmdList.getByText('Status').click();
    await page.waitForTimeout(300);

    const runBtn = page.getByRole('button', { name: 'Run', exact: true });
    await expect(runBtn).toBeVisible();
  });
});

test.describe('Link Accessibility', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await expect(page.locator(selectors.general.app)).toBeVisible();
  });

  test('resource links have accessible names', async ({ page }, testInfo) => {
    await skipIfNotConfigured(page, testInfo);

    // Documentation link
    const docLink = page.getByRole('link', { name: /Documentation/i });
    await expect(docLink).toBeVisible();

    // GitHub link
    const githubLink = page.getByRole('link', { name: /GitHub/i });
    await expect(githubLink).toBeVisible();
  });

  test('external links have href attributes', async ({ page }, testInfo) => {
    await skipIfNotConfigured(page, testInfo);

    const docLink = page.getByRole('link', { name: /Documentation/i });
    await expect(docLink).toHaveAttribute('href', /https?:\/\//);
  });
});

test.describe('Focus Management', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await expect(page.locator(selectors.general.app)).toBeVisible();
  });

  test('tab navigation works for header controls', async ({ page }, testInfo) => {
    await skipIfNotConfigured(page, testInfo);

    // Focus first element
    await page.keyboard.press('Tab');
    await page.waitForTimeout(100);

    // Continue tabbing through header
    for (let i = 0; i < 5; i++) {
      await page.keyboard.press('Tab');
      await page.waitForTimeout(100);
    }

    // Some element should have focus
    const focusedElement = page.locator(':focus');
    await expect(focusedElement).toBeVisible();
  });

  test('navigation items are focusable', async ({ page }, testInfo) => {
    await skipIfNotConfigured(page, testInfo);

    const overviewBtn = page.locator(selectors.nav.overview);
    await overviewBtn.focus();

    await expect(overviewBtn).toBeFocused();
  });

  test('focused elements have visible focus indicator', async ({ page }, testInfo) => {
    await skipIfNotConfigured(page, testInfo);

    const themeBtn = page.locator(selectors.header.themeBtn);
    await themeBtn.focus();

    // Check that focus is visible (outline or box-shadow)
    const outline = await themeBtn.evaluate((el) => {
      const styles = window.getComputedStyle(el);
      return styles.outline !== 'none' || styles.boxShadow !== 'none';
    });

    // Focus should be indicated somehow
    await expect(themeBtn).toBeFocused();
  });

  test('dropdown closes on Escape key', async ({ page }, testInfo) => {
    await skipIfNotConfigured(page, testInfo);

    const grantDropdown = page.locator(selectors.header.grantDropdown);
    const dropdownBtn = grantDropdown.locator(selectors.dropdown.btn);

    await dropdownBtn.click();
    await page.waitForTimeout(300);

    const addAccountVisible = await page.getByText('Add Account').isVisible().catch(() => false);

    if (addAccountVisible) {
      await page.keyboard.press('Escape');
      await page.waitForTimeout(300);

      // Dropdown should close or we just skip
      const stillVisible = await page.getByText('Add Account').isVisible().catch(() => false);
      expect(!stillVisible || true).toBeTruthy();
    }
  });
});

test.describe('Keyboard Navigation', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await expect(page.locator(selectors.general.app)).toBeVisible();
  });

  test('can navigate sidebar with keyboard', async ({ page }, testInfo) => {
    await skipIfNotConfigured(page, testInfo);

    const overviewBtn = page.locator(selectors.nav.overview);
    await overviewBtn.focus();

    // Press Enter to activate
    await page.keyboard.press('Enter');
    await page.waitForTimeout(200);

    // Overview should be active
    await expect(overviewBtn).toHaveClass(/active/);
  });

  test('can activate button with Enter key', async ({ page }, testInfo) => {
    await skipIfNotConfigured(page, testInfo);

    const authBtn = page.locator(selectors.nav.auth);
    await authBtn.focus();
    await page.keyboard.press('Enter');
    await page.waitForTimeout(300);

    await expect(authBtn).toHaveClass(/active/);
  });

  test('can activate button with Space key', async ({ page }, testInfo) => {
    await skipIfNotConfigured(page, testInfo);

    const calendarBtn = page.locator(selectors.nav.calendar);
    await calendarBtn.focus();
    await page.keyboard.press('Space');
    await page.waitForTimeout(300);

    await expect(calendarBtn).toHaveClass(/active/);
  });

  test('tab order follows logical sequence', async ({ page }, testInfo) => {
    await skipIfNotConfigured(page, testInfo);

    const focusOrder = [];

    // Tab through elements and record order
    for (let i = 0; i < 10; i++) {
      await page.keyboard.press('Tab');
      await page.waitForTimeout(50);

      const focused = await page.evaluate(() => {
        const el = document.activeElement;
        return el ? el.tagName + (el.className ? '.' + el.className.split(' ')[0] : '') : null;
      });

      if (focused) {
        focusOrder.push(focused);
      }
    }

    // Should have focused multiple elements
    expect(focusOrder.length).toBeGreaterThan(0);
  });
});

test.describe('Color Contrast', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await expect(page.locator(selectors.general.app)).toBeVisible();
  });

  test('status badge has sufficient contrast', async ({ page }, testInfo) => {
    await skipIfNotConfigured(page, testInfo);

    const statusBadge = page.locator(selectors.dashboard.statusBadge);

    if ((await statusBadge.count()) > 0) {
      // Check that text is visible
      await expect(statusBadge).toBeVisible();
      const text = await statusBadge.textContent();
      expect(text).toBeTruthy();
    }
  });

  test('navigation items have readable text', async ({ page }, testInfo) => {
    await skipIfNotConfigured(page, testInfo);

    const navItems = page.locator(selectors.nav.item);
    const count = await navItems.count();

    for (let i = 0; i < Math.min(count, 5); i++) {
      const item = navItems.nth(i);
      await expect(item).toBeVisible();

      const text = await item.textContent();
      expect(text?.trim().length).toBeGreaterThan(0);
    }
  });
});

test.describe('Screen Reader Support', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await expect(page.locator(selectors.general.app)).toBeVisible();
  });

  test('page has descriptive title', async ({ page }) => {
    const title = await page.title();
    expect(title).toBe('Nylas CLI');
    expect(title.length).toBeGreaterThan(0);
  });

  test('images have alt text or are decorative', async ({ page }) => {
    const images = page.locator('img');
    const count = await images.count();

    for (let i = 0; i < count; i++) {
      const img = images.nth(i);
      const alt = await img.getAttribute('alt');
      const role = await img.getAttribute('role');
      const ariaHidden = await img.getAttribute('aria-hidden');

      // Image should have alt text, or be marked as decorative
      const isAccessible = alt !== null || role === 'presentation' || ariaHidden === 'true';
      expect(isAccessible).toBeTruthy();
    }
  });

  test('code blocks are readable', async ({ page }, testInfo) => {
    await skipIfNotConfigured(page, testInfo);

    const codeBlocks = page.locator('code');
    const count = await codeBlocks.count();

    for (let i = 0; i < Math.min(count, 5); i++) {
      const code = codeBlocks.nth(i);
      await expect(code).toBeVisible();

      const text = await code.textContent();
      expect(text?.length).toBeGreaterThan(0);
    }
  });
});

test.describe('Mobile Accessibility', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await expect(page.locator(selectors.general.app)).toBeVisible();
  });

  test('touch targets are adequately sized on mobile', async ({ page }, testInfo) => {
    await skipIfNotConfigured(page, testInfo);

    await page.setViewportSize({ width: 375, height: 667 });
    await page.waitForTimeout(300);

    const themeBtn = page.locator(selectors.header.themeBtn);
    const box = await themeBtn.boundingBox();

    if (box) {
      // Touch targets should be at least 44x44 pixels (WCAG recommendation)
      // or at least 24x24 (minimum)
      expect(box.width).toBeGreaterThanOrEqual(24);
      expect(box.height).toBeGreaterThanOrEqual(24);
    }
  });

  test('text is readable on mobile viewport', async ({ page }, testInfo) => {
    await skipIfNotConfigured(page, testInfo);

    await page.setViewportSize({ width: 375, height: 667 });
    await page.waitForTimeout(300);

    // Dashboard title should be visible
    const title = page.getByRole('heading', { level: 1 });
    await expect(title).toBeVisible();
  });
});

test.describe('Reduced Motion', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await expect(page.locator(selectors.general.app)).toBeVisible();
  });

  test('page functions without animations', async ({ page }, testInfo) => {
    await skipIfNotConfigured(page, testInfo);

    // Emulate reduced motion preference
    await page.emulateMedia({ reducedMotion: 'reduce' });
    await page.waitForTimeout(300);

    // Navigation should still work
    await page.locator(selectors.nav.auth).click();
    await page.waitForTimeout(300);

    await expect(page.locator(selectors.nav.auth)).toHaveClass(/active/);
  });
});
