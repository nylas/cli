// @ts-check
const { test, expect } = require('@playwright/test');

/**
 * Agent Studio smoke tests — board rendering, drawer, create flows, and the
 * matrix-constrained rule builder. Read-only against the live board state;
 * create flows are exercised up to (not including) submission so no test
 * resources are left behind.
 */

test.describe('Agent Studio', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    // Board render is the page-ready signal.
    await expect(page.locator('#totals')).not.toHaveText('Loading…');
  });

  test('board renders palette and workspace cards from live state', async ({ page }) => {
    await expect(page.locator('.brand')).toHaveText('Agent Studio');
    await expect(page.locator('#totals')).toContainText('accounts');
    await expect(page.locator('.ws-card').first()).toBeVisible();
    await expect(page.locator('.palette .palette-label').first()).toContainText('Policies');
  });

  test('workspace without a policy shows the plan-maximums fallback', async ({ page }) => {
    // No policy is special: a workspace with no policy attached (or a stale
    // reference) must explain that plan maximums apply.
    const noPolicy = page.locator('.ws-card').filter({ hasNot: page.locator('.slot-policy') });
    test.skip(await noPolicy.count() === 0, 'every workspace has a policy attached');
    await expect(noPolicy.first().getByText('plan maximums apply')).toBeVisible();
  });

  test('clicking an account chip opens the inspector drawer', async ({ page }) => {
    await page.locator('.acct-chip').first().click();
    const drawer = page.locator('#drawer');
    await expect(drawer).toHaveClass(/open/);
    await expect(drawer.getByText('Grant ID')).toBeVisible();
    await expect(drawer.getByRole('button', { name: /Send test email/ })).toBeVisible();
    await drawer.locator('.drawer-close').click();
    await expect(drawer).not.toHaveClass(/open/);
  });

  test('every policy drawer offers edit and delete', async ({ page }) => {
    // No policy is a read-only plan ceiling: the ceiling is the billing plan,
    // enforced by the API, so every policy is editable and deletable.
    // (Missing-policy chips carry a .chip-tag and render without edit/delete,
    // so exclude them by class, not by name text.)
    const chips = page.locator('.chip-policy').filter({ hasNot: page.locator('.chip-tag') });
    test.skip(await chips.count() === 0, 'no policies in live board state');
    await chips.first().click();
    const drawer = page.locator('#drawer');
    await expect(drawer).toHaveClass(/open/);
    await expect(drawer.getByRole('button', { name: /Edit policy/ })).toBeVisible();
    await expect(drawer.getByRole('button', { name: /Delete policy/ })).toBeVisible();
  });

  test('new menu lists all create flows and recipes', async ({ page }) => {
    await page.locator('#newBtn').click();
    const menu = page.locator('#newMenu');
    await expect(menu).toHaveClass(/open/);
    for (const item of ['Agent account', 'Workspace', 'Policy', 'Rule', 'List']) {
      await expect(menu.getByText(item, { exact: true })).toBeVisible();
    }
    await expect(menu.getByText(/Recipe:/).first()).toBeVisible();
  });

  test('rule builder constrains fields by trigger', async ({ page }) => {
    await page.locator('#newBtn').click();
    await page.locator('#newMenu').getByText('Rule', { exact: true }).click();
    const modal = page.locator('#modal');
    await expect(modal.getByText('New rule')).toBeVisible();

    // Inbound: no recipient fields.
    const fieldSelect = modal.locator('.condition-row select').first();
    await expect(fieldSelect.locator('option')).toHaveCount(3);

    // Outbound: recipient.* and outbound.type appear.
    await modal.locator('.field select').first().selectOption('outbound');
    const outboundFields = modal.locator('.condition-row select').first();
    await expect(outboundFields.locator('option')).toHaveCount(7);

    await modal.getByRole('button', { name: 'Cancel' }).click();
  });

  test('account form generates a valid app password', async ({ page }) => {
    await page.locator('#newBtn').click();
    await page.locator('#newMenu').getByText('Agent account', { exact: true }).click();
    const modal = page.locator('#modal');
    await modal.getByRole('button', { name: /Generate/ }).click();
    const value = await modal.locator('.field-row input').inputValue();
    expect(value.length).toBeGreaterThanOrEqual(18);
    expect(value).toMatch(/[A-Z]/);
    expect(value).toMatch(/[a-z]/);
    expect(value).toMatch(/[0-9]/);
    await modal.getByRole('button', { name: 'Cancel' }).click();
  });

  test('policy chips are draggable', async ({ page }) => {
    // Any policy may be attached to any workspace, including the default one
    // — there is no locked plan-ceiling chip. (Missing-policy chips carry a
    // .chip-tag and are the one non-draggable case.)
    const chips = page.locator('.chip-policy').filter({ hasNot: page.locator('.chip-tag') });
    test.skip(await chips.count() === 0, 'no policies in live board state');
    await expect(chips.first()).toHaveAttribute('draggable', 'true');
  });

  test('view tabs switch between board and accounts', async ({ page }) => {
    await expect(page.locator('#board')).toBeVisible();
    await page.locator('.tab[data-view="accounts"]').click();
    await expect(page.locator('#accountsView')).toBeVisible();
    await expect(page.locator('#board')).toBeHidden();
    await expect(page).toHaveURL(/#accounts$/);
    await page.locator('.tab[data-view="board"]').click();
    await expect(page.locator('#board')).toBeVisible();
    await expect(page.locator('#accountsView')).toBeHidden();
  });

  test('accounts view lists every account with status, workspace, and actions', async ({ page }) => {
    await page.locator('.tab[data-view="accounts"]').click();
    const rows = page.locator('.acct-card');
    await expect(rows.first()).toBeVisible();
    const first = rows.first();
    await expect(first.locator('.acct-dot')).toBeVisible();
    await expect(first.locator('.acct-email')).toContainText('@');
    await expect(first.locator('.acct-meta')).toContainText('·');
    for (const action of ['Test', 'Rotate', 'Move', 'Delete']) {
      await expect(first.getByRole('button', { name: new RegExp(action) })).toBeVisible();
    }
  });

  test('accounts search filters rows by substring', async ({ page }) => {
    await page.locator('.tab[data-view="accounts"]').click();
    const rows = page.locator('.acct-card');
    const total = await rows.count();
    expect(total).toBeGreaterThan(0);
    const firstEmail = await rows.first().locator('.acct-email').textContent();
    await page.locator('.accounts-search').fill(firstEmail.trim());
    await expect(rows).toHaveCount(1);
    await page.locator('.accounts-search').fill('no-account-matches-this');
    await expect(page.locator('#accountRows .empty')).toContainText('No accounts match');
    await page.locator('.accounts-search').fill('');
    await expect(rows).toHaveCount(total);
  });

  test('clicking the accounts total jumps to the accounts view', async ({ page }) => {
    await page.locator('.total-link', { hasText: 'accounts' }).click();
    await expect(page.locator('#accountsView')).toBeVisible();
    await expect(page.locator('#board')).toBeHidden();
  });

  test('move modal offers only other workspaces and cancels cleanly', async ({ page }) => {
    // Exercised up to (not including) submission so no live move happens.
    await page.locator('.tab[data-view="accounts"]').click();
    const first = page.locator('.acct-card').first();
    const workspace = await first.locator('.acct-meta span').first().textContent();
    await first.getByRole('button', { name: /Move/ }).click();
    const modal = page.locator('#modal');
    await expect(modal.locator('.modal-title')).toContainText('Move ');
    const options = modal.locator('select option');
    const count = await options.count();
    for (let i = 0; i < count; i++) {
      const label = (await options.nth(i).textContent()).trim();
      expect(workspace).not.toContain(label);
    }
    await modal.getByRole('button', { name: 'Cancel' }).click();
    await expect(page.locator('#modalBackdrop')).not.toHaveClass(/open/);
  });

  test('account chips on the board are draggable with status dots', async ({ page }) => {
    const chip = page.locator('.ws-card .acct-chip').first();
    await expect(chip).toHaveAttribute('draggable', 'true');
    await expect(chip.locator('.acct-dot')).toBeVisible();
  });
});
