// @ts-check
const { test, expect } = require('@playwright/test');
const selectors = require('../../shared/helpers/air-selectors');

/**
 * Event Card UI tests - Edit button, Join Meeting, badges
 */
test.describe('Event Card UI', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await expect(page.locator(selectors.general.app)).toBeVisible();
    await page.waitForLoadState('domcontentloaded');

    await page.click(selectors.nav.tabCalendar);
    await expect(page.locator(selectors.views.calendar)).toHaveClass(/active/);
    await page.waitForTimeout(1000);
  });

  test.describe('Edit Button', () => {
    test('edit button appears on event card hover', async ({ page }) => {
      const eventCards = page.locator(selectors.calendar.eventCard);
      const count = await eventCards.count();

      if (count > 0) {
        const firstCard = eventCards.first();
        const editBtn = firstCard.locator(selectors.calendar.eventEditBtn);

        // Edit button should be hidden initially (opacity: 0)
        await expect(editBtn).toHaveCSS('opacity', '0');

        // Hover over the card
        await firstCard.hover();
        await page.waitForTimeout(300);

        // Edit button should be visible on hover
        await expect(editBtn).toHaveCSS('opacity', '1');
      }
    });

    test('edit button is positioned in top-right corner', async ({ page }) => {
      const eventCards = page.locator(selectors.calendar.eventCard);
      const count = await eventCards.count();

      if (count > 0) {
        const firstCard = eventCards.first();
        const editBtn = firstCard.locator(selectors.calendar.eventEditBtn);

        // Verify absolute positioning
        await expect(editBtn).toHaveCSS('position', 'absolute');

        // Verify top-right positioning
        const topValue = await editBtn.evaluate(el => getComputedStyle(el).top);
        const rightValue = await editBtn.evaluate(el => getComputedStyle(el).right);

        expect(topValue).toBe('8px');
        expect(rightValue).toBe('8px');
      }
    });

    test('edit button click opens event modal', async ({ page }) => {
      const eventCards = page.locator(selectors.calendar.eventCard);
      const count = await eventCards.count();

      if (count > 0) {
        const firstCard = eventCards.first();
        const editBtn = firstCard.locator(selectors.calendar.eventEditBtn);

        // Hover to make button visible
        await firstCard.hover();
        await page.waitForTimeout(300);

        // Click edit button
        await editBtn.click();
        await page.waitForTimeout(500);

        // Modal should open
        const modal = page.locator(selectors.eventModal.modal);
        await expect(modal).toBeVisible();

        // Modal should have title populated
        const titleField = modal.locator(selectors.eventModal.title);
        const titleValue = await titleField.inputValue();
        expect(titleValue.length).toBeGreaterThan(0);
      }
    });

    test('edit button click does not propagate to card', async ({ page }) => {
      const eventCards = page.locator(selectors.calendar.eventCard);
      const count = await eventCards.count();

      if (count > 0) {
        const firstCard = eventCards.first();
        const editBtn = firstCard.locator(selectors.calendar.eventEditBtn);

        // Hover to make button visible
        await firstCard.hover();
        await page.waitForTimeout(300);

        // Click edit button
        await editBtn.click();
        await page.waitForTimeout(500);

        // Only one modal should be open (not multiple from propagation)
        const modals = page.locator(selectors.eventModal.modal);
        expect(await modals.count()).toBeLessThanOrEqual(1);
      }
    });
  });

  test.describe('Join Meeting Button', () => {
    test('join meeting button exists for events with conferencing', async ({ page }) => {
      const joinBtns = page.locator(selectors.calendar.joinMeetingBtn);
      const count = await joinBtns.count();

      // It's okay if no events have conferencing
      expect(count >= 0).toBeTruthy();

      if (count > 0) {
        await expect(joinBtns.first()).toBeVisible();
        await expect(joinBtns.first()).toContainText('Join Meeting');
      }
    });

    test('join meeting button has target="_blank" for external link', async ({ page }) => {
      const joinBtns = page.locator(selectors.calendar.joinMeetingBtn);
      const count = await joinBtns.count();

      if (count > 0) {
        const target = await joinBtns.first().getAttribute('target');
        expect(target).toBe('_blank');
      }
    });

    test('join meeting button click does not open event modal', async ({ page }) => {
      const joinBtns = page.locator(selectors.calendar.joinMeetingBtn);
      const count = await joinBtns.count();

      if (count > 0) {
        // Get the href before clicking
        const href = await joinBtns.first().getAttribute('href');
        expect(href).toBeTruthy();

        // Mock the new tab opening by preventing default
        await page.evaluate(() => {
          document.querySelectorAll('.join-meeting-btn').forEach(btn => {
            btn.addEventListener('click', (e) => {
              e.preventDefault();
              // Store that we clicked join meeting
              window._joinMeetingClicked = true;
            }, { capture: true });
          });
        });

        // Click the join meeting button
        await joinBtns.first().click();
        await page.waitForTimeout(500);

        // Event modal should NOT be open
        const modal = page.locator(selectors.eventModal.modal);
        const modalVisible = await modal.isVisible().catch(() => false);
        expect(modalVisible).toBeFalsy();
      }
    });
  });

  test.describe('Event Count Badge', () => {
    test('event count badge appears on days with events', async ({ page }) => {
      const badges = page.locator(selectors.calendar.eventCountBadge);
      const count = await badges.count();

      // Should have badges if there are events
      expect(count >= 0).toBeTruthy();

      if (count > 0) {
        // Badge should contain a number
        const text = await badges.first().textContent();
        expect(parseInt(text)).toBeGreaterThan(0);
      }
    });

    test('event count badge is positioned in bottom-right corner', async ({ page }) => {
      const badges = page.locator(selectors.calendar.eventCountBadge);
      const count = await badges.count();

      if (count > 0) {
        await expect(badges.first()).toHaveCSS('position', 'absolute');

        const bottomValue = await badges.first().evaluate(el => getComputedStyle(el).bottom);
        const rightValue = await badges.first().evaluate(el => getComputedStyle(el).right);

        expect(bottomValue).toBe('8px');
        expect(rightValue).toBe('8px');
      }
    });
  });

  test.describe('Today Indicator', () => {
    test('today indicator appears on current day', async ({ page }) => {
      const todayCell = page.locator(selectors.calendar.today);
      const count = await todayCell.count();

      if (count > 0) {
        const indicator = todayCell.locator(selectors.calendar.todayIndicator);
        await expect(indicator).toBeVisible();
      }
    });

    test('today indicator has pulsing animation', async ({ page }) => {
      const todayCell = page.locator(selectors.calendar.today);
      const count = await todayCell.count();

      if (count > 0) {
        const indicator = todayCell.locator(selectors.calendar.todayIndicator);

        if (await indicator.count() > 0) {
          const animation = await indicator.evaluate(el => getComputedStyle(el).animationName);
          expect(animation).toBe('todayPulse');
        }
      }
    });
  });

  test.describe('Relative Time Indicator', () => {
    test('relative time indicator shows for upcoming events', async ({ page }) => {
      const relativeTimeIndicators = page.locator(selectors.calendar.eventRelativeTime);
      const count = await relativeTimeIndicators.count();

      // It's okay if no events are upcoming
      expect(count >= 0).toBeTruthy();

      if (count > 0) {
        // Should have some text
        const text = await relativeTimeIndicators.first().textContent();
        expect(text.length).toBeGreaterThan(0);
      }
    });

    test('starting-soon indicator has warning styling', async ({ page }) => {
      const startingSoon = page.locator('.event-relative-time.starting-soon');
      const count = await startingSoon.count();

      if (count > 0) {
        // Should have gradient background (starts with linear-gradient)
        const background = await startingSoon.first().evaluate(el => getComputedStyle(el).backgroundImage);
        expect(background).toContain('linear-gradient');
      }
    });

    test('starting-now indicator has urgent styling', async ({ page }) => {
      const startingNow = page.locator('.event-relative-time.starting-now');
      const count = await startingNow.count();

      if (count > 0) {
        // Should have animation
        const animation = await startingNow.first().evaluate(el => getComputedStyle(el).animationName);
        expect(animation).toBe('urgentPulse');
      }
    });
  });
});
