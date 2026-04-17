// @ts-check
const { test, expect } = require('@playwright/test');
const selectors = require('../../shared/helpers/ui-selectors');

/**
 * Command List Tests for Nylas UI.
 *
 * Tests clicking "List" command on each command tab and verifying
 * the command detail panel shows up with correct elements.
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

/**
 * Command pages with their List command configurations.
 * Each entry specifies the nav selector, page id, cmd list id, and the section containing List.
 */
const commandPagesWithList = [
  {
    name: 'Admin',
    nav: 'admin',
    pageId: 'page-admin',
    cmdListId: 'admin-cmd-list',
    listSection: 'Grants', // Admin has List under Grants section
    expectedCommand: 'nylas admin grant list',
  },
  {
    name: 'Auth',
    nav: 'auth',
    pageId: 'page-auth',
    cmdListId: 'auth-cmd-list',
    listSection: null, // Auth doesn't have a traditional List command
    hasListCommand: false,
  },
  {
    name: 'Calendar',
    nav: 'calendar',
    pageId: 'page-calendar',
    cmdListId: 'calendar-cmd-list',
    listSection: 'Events',
    expectedCommand: 'nylas calendar events list',
  },
  {
    name: 'Contacts',
    nav: 'contacts',
    pageId: 'page-contacts',
    cmdListId: 'contacts-cmd-list',
    listSection: 'Contacts',
    expectedCommand: 'nylas contacts list',
  },
  {
    name: 'Email',
    nav: 'email',
    pageId: 'page-email',
    cmdListId: 'email-cmd-list',
    listSection: 'Messages',
    expectedCommand: 'nylas email list',
  },
  {
    name: 'Notetaker',
    nav: 'notetaker',
    pageId: 'page-notetaker',
    cmdListId: 'notetaker-cmd-list',
    listSection: 'Notetakers',
    expectedCommand: 'nylas notetaker list',
  },
  {
    name: 'OTP',
    nav: 'otp',
    pageId: 'page-otp',
    cmdListId: 'otp-cmd-list',
    listSection: null,
    hasListCommand: false,
  },
  {
    name: 'Scheduler',
    nav: 'scheduler',
    pageId: 'page-scheduler',
    cmdListId: 'scheduler-cmd-list',
    listSection: 'Configurations',
    expectedCommand: 'nylas scheduler configuration list',
  },
  {
    name: 'Timezone',
    nav: 'timezone',
    pageId: 'page-timezone',
    cmdListId: 'timezone-cmd-list',
    listSection: 'Zones',
    expectedCommand: 'nylas timezone list',
  },
  {
    name: 'Webhook',
    nav: 'webhook',
    pageId: 'page-webhook',
    cmdListId: 'webhook-cmd-list',
    listSection: 'Webhooks',
    expectedCommand: 'nylas webhook list',
  },
];

test.describe('Command List - Navigate and Click List', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await expect(page.locator(selectors.general.app)).toBeVisible();
  });

  // Generate tests for each command page with List command
  for (const cmdPage of commandPagesWithList) {
    if (cmdPage.hasListCommand === false) {
      // Skip pages without List command
      test(`${cmdPage.name} page - no List command (expected)`, async ({ page }, testInfo) => {
        await skipIfNotConfigured(page, testInfo);

        // Navigate to page
        await page.locator(selectors.nav[cmdPage.nav]).click();
        await page.waitForTimeout(300);

        // Page should be active
        await expect(page.locator(`#${cmdPage.pageId}`)).toHaveClass(/active/);

        // Just verify page is accessible
        await expect(page.locator(`#${cmdPage.pageId}`)).toBeVisible();
      });
      continue;
    }

    test(`${cmdPage.name} page - click List shows command detail`, async ({ page }, testInfo) => {
      await skipIfNotConfigured(page, testInfo);

      // Navigate to command page
      await page.locator(selectors.nav[cmdPage.nav]).click();
      await page.waitForTimeout(300);

      // Verify page is active
      await expect(page.locator(`#${cmdPage.pageId}`)).toHaveClass(/active/);

      // Find and click List command in the command list
      const cmdList = page.locator(`#${cmdPage.cmdListId}`);
      await expect(cmdList).toBeVisible();

      // Click on List command
      const listCommand = cmdList.getByText('List', { exact: true }).first();
      await listCommand.click();
      await page.waitForTimeout(300);

      // Verify command detail panel shows List heading
      const heading = page.locator('h2').filter({ hasText: 'List' });
      await expect(heading).toBeVisible();
    });

    test(`${cmdPage.name} page - List command shows code block`, async ({ page }, testInfo) => {
      await skipIfNotConfigured(page, testInfo);

      await page.locator(selectors.nav[cmdPage.nav]).click();
      await page.waitForTimeout(300);

      const cmdList = page.locator(`#${cmdPage.cmdListId}`);
      const listCommand = cmdList.getByText('List', { exact: true }).first();
      await listCommand.click();
      await page.waitForTimeout(300);

      // Verify code block exists and contains command
      const codeBlock = page.locator(`#${cmdPage.pageId} code`).first();
      await expect(codeBlock).toBeVisible();

      // Verify command contains expected text (partial match)
      const codeText = await codeBlock.textContent();
      expect(codeText).toContain('nylas');
    });

    test(`${cmdPage.name} page - List command has Run button`, async ({ page }, testInfo) => {
      await skipIfNotConfigured(page, testInfo);

      await page.locator(selectors.nav[cmdPage.nav]).click();
      await page.waitForTimeout(300);

      const cmdList = page.locator(`#${cmdPage.cmdListId}`);
      const listCommand = cmdList.getByText('List', { exact: true }).first();
      await listCommand.click();
      await page.waitForTimeout(300);

      // Run button should be visible
      const runBtn = page.getByRole('button', { name: 'Run', exact: true });
      await expect(runBtn).toBeVisible();
    });

    test(`${cmdPage.name} page - List command has output panel`, async ({ page }, testInfo) => {
      await skipIfNotConfigured(page, testInfo);

      await page.locator(selectors.nav[cmdPage.nav]).click();
      await page.waitForTimeout(300);

      const cmdList = page.locator(`#${cmdPage.cmdListId}`);
      const listCommand = cmdList.getByText('List', { exact: true }).first();
      await listCommand.click();
      await page.waitForTimeout(300);

      // Output section should exist
      const outputSection = page.locator(`#${cmdPage.pageId}`).getByText('Output');
      await expect(outputSection.first()).toBeVisible();
    });

    test(`${cmdPage.name} page - List shows placeholder before run`, async ({ page }, testInfo) => {
      await skipIfNotConfigured(page, testInfo);

      await page.locator(selectors.nav[cmdPage.nav]).click();
      await page.waitForTimeout(300);

      const cmdList = page.locator(`#${cmdPage.cmdListId}`);
      const listCommand = cmdList.getByText('List', { exact: true }).first();
      await listCommand.click();
      await page.waitForTimeout(300);

      // Placeholder text should be visible
      await expect(page.getByText('Click "Run" to execute command')).toBeVisible();
    });
  }
});

test.describe('Command List - Execute List Commands', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await expect(page.locator(selectors.general.app)).toBeVisible();
  });

  // Test running List on key pages
  const executeTestPages = [
    { name: 'Email', nav: 'email', cmdListId: 'email-cmd-list', pageId: 'page-email' },
    { name: 'Calendar', nav: 'calendar', cmdListId: 'calendar-cmd-list', pageId: 'page-calendar' },
    { name: 'Contacts', nav: 'contacts', cmdListId: 'contacts-cmd-list', pageId: 'page-contacts' },
    { name: 'Timezone', nav: 'timezone', cmdListId: 'timezone-cmd-list', pageId: 'page-timezone' },
  ];

  for (const cmdPage of executeTestPages) {
    test(`${cmdPage.name} - running List command executes successfully`, async ({
      page,
    }, testInfo) => {
      await skipIfNotConfigured(page, testInfo);

      // Navigate and click List
      await page.locator(selectors.nav[cmdPage.nav]).click();
      await page.waitForTimeout(300);

      const cmdList = page.locator(`#${cmdPage.cmdListId}`);
      const listCommand = cmdList.getByText('List', { exact: true }).first();
      await listCommand.click();
      await page.waitForTimeout(300);

      // Click Run button
      const runBtn = page.getByRole('button', { name: 'Run', exact: true });
      await runBtn.click();
      await page.waitForTimeout(3000);

      // Verify command completed - toast or output changed
      const completedToast = page.getByText('Command completed');
      const hasToast = (await completedToast.count()) > 0;

      // Or check that placeholder is gone
      const placeholder = page.getByText('Click "Run" to execute command');
      const placeholderVisible = await placeholder.isVisible().catch(() => false);

      // Either toast appeared or placeholder is gone
      expect(hasToast || !placeholderVisible).toBeTruthy();
    });

    test(`${cmdPage.name} - running List shows last run timestamp`, async ({ page }, testInfo) => {
      await skipIfNotConfigured(page, testInfo);

      await page.locator(selectors.nav[cmdPage.nav]).click();
      await page.waitForTimeout(300);

      const cmdList = page.locator(`#${cmdPage.cmdListId}`);
      const listCommand = cmdList.getByText('List', { exact: true }).first();
      await listCommand.click();
      await page.waitForTimeout(300);

      await page.getByRole('button', { name: 'Run', exact: true }).click();
      await page.waitForTimeout(2000);

      // Should show last run timestamp
      await expect(page.getByText('Last run:')).toBeVisible();
    });
  }
});

test.describe('Command List - Navigation Flow', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await expect(page.locator(selectors.general.app)).toBeVisible();
  });

  test('can navigate through all tabs and click List', async ({ page }, testInfo) => {
    await skipIfNotConfigured(page, testInfo);

    const tabsWithList = ['email', 'calendar', 'contacts', 'timezone', 'webhook'];

    for (const tab of tabsWithList) {
      // Navigate to tab
      await page.locator(selectors.nav[tab]).click();
      await page.waitForTimeout(300);

      // Verify page is active
      await expect(page.locator(`#page-${tab}`)).toHaveClass(/active/);

      // Find and click List
      const cmdList = page.locator(`#${tab}-cmd-list`);
      if ((await cmdList.count()) > 0) {
        const listCommand = cmdList.getByText('List', { exact: true }).first();
        if ((await listCommand.count()) > 0) {
          await listCommand.click();
          await page.waitForTimeout(200);

          // Verify List detail is shown - scope to active page
          const heading = page.locator(`#page-${tab} h2`).filter({ hasText: 'List' });
          await expect(heading).toBeVisible();
        }
      }
    }
  });

  test('switching tabs preserves UI state', async ({ page }, testInfo) => {
    await skipIfNotConfigured(page, testInfo);

    // Go to Email and click List
    await page.locator(selectors.nav.email).click();
    await page.waitForTimeout(300);

    const emailCmdList = page.locator('#email-cmd-list');
    await emailCmdList.getByText('List', { exact: true }).first().click();
    await page.waitForTimeout(300);

    // Switch to Calendar
    await page.locator(selectors.nav.calendar).click();
    await page.waitForTimeout(300);
    await expect(page.locator('#page-calendar')).toHaveClass(/active/);

    // Switch back to Email
    await page.locator(selectors.nav.email).click();
    await page.waitForTimeout(300);

    // Email page should still be functional
    await expect(page.locator('#page-email')).toHaveClass(/active/);
  });
});

test.describe('Command List - Email Specific Tests', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await expect(page.locator(selectors.general.app)).toBeVisible();
  });

  test('Email List has correct command sections', async ({ page }, testInfo) => {
    await skipIfNotConfigured(page, testInfo);

    await page.locator(selectors.nav.email).click();
    await page.waitForTimeout(300);

    // Email should have these sections
    const expectedSections = ['Messages', 'Folders', 'Drafts', 'Threads'];

    for (const section of expectedSections) {
      await expect(page.getByText(section, { exact: true }).first()).toBeVisible();
    }
  });

  test('Email Messages List has options panel', async ({ page }, testInfo) => {
    await skipIfNotConfigured(page, testInfo);

    await page.locator(selectors.nav.email).click();
    await page.waitForTimeout(300);

    const cmdList = page.locator('#email-cmd-list');
    await cmdList.getByText('List', { exact: true }).first().click();
    await page.waitForTimeout(300);

    // Should have Options panel with filters
    await expect(page.getByText('Options')).toBeVisible();
    await expect(page.getByText('Unread only')).toBeVisible();
    await expect(page.getByText('Starred only')).toBeVisible();
  });

  test('Email List can be run with filters', async ({ page }, testInfo) => {
    await skipIfNotConfigured(page, testInfo);

    await page.locator(selectors.nav.email).click();
    await page.waitForTimeout(300);

    const cmdList = page.locator('#email-cmd-list');
    await cmdList.getByText('List', { exact: true }).first().click();
    await page.waitForTimeout(300);

    // Click Run button
    await page.getByRole('button', { name: 'Run', exact: true }).click();
    await page.waitForTimeout(3000);

    // Verify output shows something (either messages or "No messages found")
    const placeholder = page.getByText('Click "Run" to execute command');
    const placeholderVisible = await placeholder.isVisible().catch(() => false);

    // Placeholder should be gone after running
    expect(placeholderVisible).toBeFalsy();
  });
});

test.describe('Command List - Calendar Specific Tests', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await expect(page.locator(selectors.general.app)).toBeVisible();
  });

  test('Calendar List has correct command sections', async ({ page }, testInfo) => {
    await skipIfNotConfigured(page, testInfo);

    await page.locator(selectors.nav.calendar).click();
    await page.waitForTimeout(300);

    // Calendar should have Events section
    await expect(page.getByText('Events', { exact: true }).first()).toBeVisible();
  });

  test('Calendar Events List shows command detail', async ({ page }, testInfo) => {
    await skipIfNotConfigured(page, testInfo);

    await page.locator(selectors.nav.calendar).click();
    await page.waitForTimeout(300);

    const cmdList = page.locator('#calendar-cmd-list');
    await cmdList.getByText('List', { exact: true }).first().click();
    await page.waitForTimeout(300);

    // Verify heading
    const heading = page.locator('h2').filter({ hasText: 'List' });
    await expect(heading).toBeVisible();

    // Verify code block
    const codeBlock = page.locator('#page-calendar code').first();
    const codeText = await codeBlock.textContent();
    expect(codeText).toContain('calendar');
  });
});

test.describe('Command List - Contacts Specific Tests', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await expect(page.locator(selectors.general.app)).toBeVisible();
  });

  test('Contacts List shows command detail', async ({ page }, testInfo) => {
    await skipIfNotConfigured(page, testInfo);

    await page.locator(selectors.nav.contacts).click();
    await page.waitForTimeout(300);

    const cmdList = page.locator('#contacts-cmd-list');
    await cmdList.getByText('List', { exact: true }).first().click();
    await page.waitForTimeout(300);

    // Verify heading
    const heading = page.locator('h2').filter({ hasText: 'List' });
    await expect(heading).toBeVisible();

    // Verify Run button
    const runBtn = page.getByRole('button', { name: 'Run', exact: true });
    await expect(runBtn).toBeVisible();
  });
});

test.describe('Command List - Webhook Specific Tests', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await expect(page.locator(selectors.general.app)).toBeVisible();
  });

  test('Webhook List shows command detail', async ({ page }, testInfo) => {
    await skipIfNotConfigured(page, testInfo);

    await page.locator(selectors.nav.webhook).click();
    await page.waitForTimeout(300);

    const cmdList = page.locator('#webhook-cmd-list');
    await cmdList.getByText('List', { exact: true }).first().click();
    await page.waitForTimeout(300);

    // Verify heading
    const heading = page.locator('h2').filter({ hasText: 'List' });
    await expect(heading).toBeVisible();
  });
});

test.describe('Command List - Admin Specific Tests', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await expect(page.locator(selectors.general.app)).toBeVisible();
  });

  test('Admin page has Grants section with List', async ({ page }, testInfo) => {
    await skipIfNotConfigured(page, testInfo);

    await page.locator(selectors.nav.admin).click();
    await page.waitForTimeout(300);

    // Admin should have Grants section
    await expect(page.getByText('Grants', { exact: true }).first()).toBeVisible();

    // Click List under Grants
    const cmdList = page.locator('#admin-cmd-list');
    const listCommand = cmdList.getByText('List', { exact: true }).first();
    await listCommand.click();
    await page.waitForTimeout(300);

    // Verify command detail
    const heading = page.locator('h2').filter({ hasText: 'List' });
    await expect(heading).toBeVisible();
  });
});

test.describe('Command List - Run Commands and Verify Output', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await expect(page.locator(selectors.general.app)).toBeVisible();
  });

  test('Email Folders List - run and verify output', async ({ page }, testInfo) => {
    await skipIfNotConfigured(page, testInfo);

    await page.locator(selectors.nav.email).click();
    await page.waitForTimeout(300);

    // Click on Folders List (not Messages List)
    const cmdList = page.locator('#email-cmd-list');
    const foldersSection = cmdList.getByText('Folders', { exact: true });
    await expect(foldersSection).toBeVisible();

    // Find List under Folders section
    const allListItems = cmdList.getByText('List', { exact: true });
    // Get the second List (under Folders)
    await allListItems.nth(1).click();
    await page.waitForTimeout(300);

    // Verify heading shows List
    const heading = page.locator('h2').filter({ hasText: 'List' });
    await expect(heading).toBeVisible();

    // Click Run
    await page.getByRole('button', { name: 'Run', exact: true }).click();
    await page.waitForTimeout(3000);

    // Verify output is shown (not placeholder)
    const placeholder = page.getByText('Click "Run" to execute command');
    const placeholderVisible = await placeholder.isVisible().catch(() => false);
    expect(placeholderVisible).toBeFalsy();

    // Verify last run timestamp
    await expect(page.getByText('Last run:')).toBeVisible();
  });

  test('Email List with All Folders option - run and verify output', async ({ page }, testInfo) => {
    await skipIfNotConfigured(page, testInfo);

    await page.locator(selectors.nav.email).click();
    await page.waitForTimeout(300);

    const cmdList = page.locator('#email-cmd-list');
    await cmdList.getByText('List', { exact: true }).first().click();
    await page.waitForTimeout(300);

    // Click "All folders" option
    const allFoldersOption = page.getByText('All folders');
    if ((await allFoldersOption.count()) > 0) {
      await allFoldersOption.click();
      await page.waitForTimeout(200);
    }

    // Click Run
    await page.getByRole('button', { name: 'Run', exact: true }).click();
    await page.waitForTimeout(3000);

    // Verify command completed
    const completedToast = page.getByText('Command completed');
    await expect(completedToast).toBeVisible({ timeout: 5000 });

    // Verify output is shown
    const placeholder = page.getByText('Click "Run" to execute command');
    const placeholderVisible = await placeholder.isVisible().catch(() => false);
    expect(placeholderVisible).toBeFalsy();
  });

  test('Calendar List - run and verify output', async ({ page }, testInfo) => {
    await skipIfNotConfigured(page, testInfo);

    await page.locator(selectors.nav.calendar).click();
    await page.waitForTimeout(300);

    const cmdList = page.locator('#calendar-cmd-list');
    await cmdList.getByText('List', { exact: true }).first().click();
    await page.waitForTimeout(300);

    // Click Run
    await page.getByRole('button', { name: 'Run', exact: true }).click();
    await page.waitForTimeout(3000);

    // Verify command completed
    const completedToast = page.getByText('Command completed');
    await expect(completedToast).toBeVisible({ timeout: 5000 });

    // Verify last run timestamp
    await expect(page.getByText('Last run:')).toBeVisible();
  });

  test('Contacts List - run and verify output', async ({ page }, testInfo) => {
    await skipIfNotConfigured(page, testInfo);

    await page.locator(selectors.nav.contacts).click();
    await page.waitForTimeout(300);

    const cmdList = page.locator('#contacts-cmd-list');
    await cmdList.getByText('List', { exact: true }).first().click();
    await page.waitForTimeout(300);

    // Click Run
    await page.getByRole('button', { name: 'Run', exact: true }).click();
    await page.waitForTimeout(3000);

    // Verify command completed
    const completedToast = page.getByText('Command completed');
    await expect(completedToast).toBeVisible({ timeout: 5000 });

    // Verify last run timestamp
    await expect(page.getByText('Last run:')).toBeVisible();
  });

  test('Timezone List - run and verify output', async ({ page }, testInfo) => {
    await skipIfNotConfigured(page, testInfo);

    await page.locator(selectors.nav.timezone).click();
    await page.waitForTimeout(300);

    const cmdList = page.locator('#timezone-cmd-list');
    await cmdList.getByText('List', { exact: true }).first().click();
    await page.waitForTimeout(300);

    // Click Run
    await page.getByRole('button', { name: 'Run', exact: true }).click();
    await page.waitForTimeout(3000);

    // Verify command completed
    const completedToast = page.getByText('Command completed');
    await expect(completedToast).toBeVisible({ timeout: 5000 });

    // Verify output shows timezone data (not placeholder)
    const placeholder = page.getByText('Click "Run" to execute command');
    const placeholderVisible = await placeholder.isVisible().catch(() => false);
    expect(placeholderVisible).toBeFalsy();
  });

  test('Webhook List - run and verify output', async ({ page }, testInfo) => {
    await skipIfNotConfigured(page, testInfo);

    await page.locator(selectors.nav.webhook).click();
    await page.waitForTimeout(300);

    const cmdList = page.locator('#webhook-cmd-list');
    await cmdList.getByText('List', { exact: true }).first().click();
    await page.waitForTimeout(300);

    // Click Run
    await page.getByRole('button', { name: 'Run', exact: true }).click();
    await page.waitForTimeout(3000);

    // Verify command completed
    const completedToast = page.getByText('Command completed');
    await expect(completedToast).toBeVisible({ timeout: 5000 });

    // Verify last run timestamp
    await expect(page.getByText('Last run:')).toBeVisible();
  });

  test('Admin Grant List - run and verify output', async ({ page }, testInfo) => {
    await skipIfNotConfigured(page, testInfo);

    await page.locator(selectors.nav.admin).click();
    await page.waitForTimeout(300);

    const cmdList = page.locator('#admin-cmd-list');
    await cmdList.getByText('List', { exact: true }).first().click();
    await page.waitForTimeout(300);

    // Click Run
    await page.getByRole('button', { name: 'Run', exact: true }).click();
    await page.waitForTimeout(3000);

    // Verify command completed
    const completedToast = page.getByText('Command completed');
    await expect(completedToast).toBeVisible({ timeout: 5000 });

    // Verify last run timestamp
    await expect(page.getByText('Last run:')).toBeVisible();
  });
});

test.describe('Command List - All Tabs Navigation and Run', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await expect(page.locator(selectors.general.app)).toBeVisible();
  });

  test('navigate all tabs, click List, and run commands', async ({ page }, testInfo) => {
    await skipIfNotConfigured(page, testInfo);

    const tabsWithList = [
      { nav: 'admin', cmdListId: 'admin-cmd-list', pageId: 'page-admin' },
      { nav: 'calendar', cmdListId: 'calendar-cmd-list', pageId: 'page-calendar' },
      { nav: 'contacts', cmdListId: 'contacts-cmd-list', pageId: 'page-contacts' },
      { nav: 'email', cmdListId: 'email-cmd-list', pageId: 'page-email' },
      { nav: 'timezone', cmdListId: 'timezone-cmd-list', pageId: 'page-timezone' },
      { nav: 'webhook', cmdListId: 'webhook-cmd-list', pageId: 'page-webhook' },
    ];

    for (const tab of tabsWithList) {
      // Navigate to tab
      await page.locator(selectors.nav[tab.nav]).click();
      await page.waitForTimeout(300);

      // Verify page is active
      await expect(page.locator(`#${tab.pageId}`)).toHaveClass(/active/);

      // Find and click List
      const cmdList = page.locator(`#${tab.cmdListId}`);
      const listCommand = cmdList.getByText('List', { exact: true }).first();
      await listCommand.click();
      await page.waitForTimeout(300);

      // Verify List detail is shown - scope to active page
      const heading = page.locator(`#${tab.pageId} h2`).filter({ hasText: 'List' });
      await expect(heading).toBeVisible();

      // Click Run
      const runBtn = page.getByRole('button', { name: 'Run', exact: true });
      await runBtn.click();
      await page.waitForTimeout(2000);

      // Verify command completed (toast appears)
      const completedToast = page.getByText('Command completed');
      await expect(completedToast).toBeVisible({ timeout: 5000 });

      // Wait for toast to disappear before next iteration
      await page.waitForTimeout(1000);
    }
  });
});
