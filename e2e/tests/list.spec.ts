// SPDX-License-Identifier: GPL-3.0-or-later

import { expect, test, type Page } from '@playwright/test';
import { signIn } from './helpers';

test.use({ locale: 'en-US' });

const row = (page: Page, subject: string) =>
  page.getByTestId('message').filter({ has: page.locator('.subject .text', { hasText: new RegExp(`^${subject}$`) }) });

async function openBulk(page: Page) {
  await signIn(page);
  await page.locator('[data-testid=folder][data-folder=Bulk]').click();
  await expect(page.getByTestId('message').first()).toBeVisible();
}

test('search filters server-side and survives reload', async ({ page }) => {
  await signIn(page);
  await page.getByTestId('search').fill('Привет');
  await expect(page.getByTestId('message')).toHaveCount(1);
  await expect(page).toHaveURL(/q=%D0%9F/);
  await page.reload();
  await expect(page.getByTestId('search')).toHaveValue('Привет');
  await expect(page.getByTestId('message')).toHaveCount(1);

  await page.getByTestId('search').fill('no such message anywhere');
  await expect(page.getByTestId('empty')).toHaveText('No messages match your search.');
  await page.getByTestId('search').press('Escape');
  await expect(page.getByTestId('message')).toHaveCount(19);
});

test('hover shows size, flag and delete', async ({ page }) => {
  await openBulk(page);
  const r = row(page, 'Bulk 8');
  await r.hover();
  const actions = r.getByTestId('row-actions');
  await expect(actions).toBeVisible();
  await expect(actions).toContainText('byte');
  await actions.getByTestId('row-flag').click();
  await expect(r.getByRole('img', { name: 'Flagged' })).toBeVisible();
  await r.hover();
  await actions.getByTestId('row-flag').click();
  await expect(r.getByRole('img', { name: 'Flagged' })).toHaveCount(0);
});

test('ctrl and shift select several messages for bulk actions', async ({ page }) => {
  await openBulk(page);
  await row(page, 'Bulk 8').getByRole('link').click({ modifiers: ['ControlOrMeta'] });
  await row(page, 'Bulk 6').getByRole('link').click({ modifiers: ['ControlOrMeta'] });
  await expect(page.getByTestId('bulk-count')).toHaveText('2 selected');
  await expect(page.getByTestId('multi-selected')).toBeVisible();

  await row(page, 'Bulk 5').getByRole('link').click({ modifiers: ['Shift'] });
  await expect(page.getByTestId('bulk-count')).toHaveText('2 selected');
  await row(page, 'Bulk 8').getByRole('link').click({ modifiers: ['ControlOrMeta'] });
  await row(page, 'Bulk 5').getByRole('link').click({ modifiers: ['Shift'] });
  await expect(page.getByTestId('bulk-count')).toHaveText('4 selected');

  await page.getByTestId('bulk-seen').click();
  for (const n of [8, 7, 6, 5]) await expect(row(page, `Bulk ${n}`)).not.toHaveClass(/unread/);
  await expect(row(page, 'Bulk 4')).toHaveClass(/unread/);

  await page.keyboard.press('Escape');
  await expect(page.getByTestId('bulk-bar')).toHaveCount(0);
});

test('bulk archive and bulk delete', async ({ page }) => {
  await openBulk(page);
  await row(page, 'Bulk 4').getByRole('link').click({ modifiers: ['ControlOrMeta'] });
  await row(page, 'Bulk 3').getByRole('link').click({ modifiers: ['ControlOrMeta'] });
  await page.getByTestId('bulk-archive').click();
  await expect(page.getByTestId('toast')).toHaveText('Archived to Archive.');
  await expect(row(page, 'Bulk 4')).toHaveCount(0);
  await expect(page.locator('[data-testid=folder][data-folder=Archive]')).toBeVisible();

  await row(page, 'Bulk 2').getByRole('link').click({ modifiers: ['ControlOrMeta'] });
  await row(page, 'Bulk 1').getByRole('link').click({ modifiers: ['ControlOrMeta'] });
  await page.getByTestId('bulk-delete').click();
  await expect(page.getByTestId('toast')).toHaveText('2 messages moved to Trash.');
  await expect(row(page, 'Bulk 1')).toHaveCount(0);
});

test('touch swipe right archives, left deletes', async ({ page }) => {
  await openBulk(page);
  const swipe = async (subject: string, dir: 1 | -1) => {
    const slide = row(page, subject).locator('.slide');
    const box = (await slide.boundingBox())!;
    const y = box.y + box.height / 2;
    const x0 = box.x + box.width / 2;
    await slide.evaluate(
      (el, { x0, y, dir, w }) => {
        const fire = (type: string, x: number) =>
          el.dispatchEvent(new PointerEvent(type, { pointerType: 'touch', pointerId: 7, clientX: x, clientY: y, bubbles: true }));
        fire('pointerdown', x0);
        for (let i = 1; i <= 10; i++) fire('pointermove', x0 + dir * w * 0.05 * i);
        fire('pointerup', x0 + dir * w * 0.5);
      },
      { x0, y, dir, w: box.width },
    );
  };
  await swipe('Bulk 8', 1);
  await expect(page.getByTestId('toast')).toHaveText('Archived to Archive.');
  await expect(row(page, 'Bulk 8')).toHaveCount(0);
  await swipe('Bulk 7', -1);
  await expect(page.getByTestId('toast')).toHaveText('Moved to Trash.');
  await expect(row(page, 'Bulk 7')).toHaveCount(0);
});

test('toolbar: select menu, sort options and refresh', async ({ page }) => {
  await signIn(page);
  await page.locator('[data-testid=folder][data-folder=Paging]').click();
  await expect(page.getByTestId('message')).toHaveCount(50);
  await page.getByTestId('select').click();
  await page.getByTestId('select-all').click();
  await expect(page.getByTestId('bulk-count')).toHaveText('50 selected');
  await page.getByTestId('select').click();
  await page.getByTestId('select-invert').click();
  await expect(page.getByTestId('bulk-bar')).toHaveCount(0);
  await page.getByTestId('select').click();
  await page.getByTestId('select-unread').click();
  await expect(page.getByTestId('bulk-count')).toHaveText('50 selected');
  await page.getByTestId('select').click();
  await page.getByTestId('select-none').click();
  await expect(page.getByTestId('bulk-bar')).toHaveCount(0);

  await expect(page.getByTestId('message').first()).toContainText('Paging message 120');
  await page.getByTestId('list-options').click();
  await page.getByTestId('options-asc').check();
  await page.getByTestId('options-save').click();
  await expect(page.getByTestId('message').first()).toContainText('Paging message 001');
  await page.reload();
  await expect(page.getByTestId('message').first()).toContainText('Paging message 001');
  await page.getByTestId('list-options').click();
  await page.getByTestId('options-desc').check();
  await page.getByTestId('options-save').click();
  await expect(page.getByTestId('message').first()).toContainText('Paging message 120');

  await page.getByTestId('refresh').click();
  await expect(page.getByTestId('message').first()).toContainText('Paging message 120');
});

test('settings cog toggles between settings and mail', async ({ page }) => {
  await signIn(page);
  await page.getByTestId('settings-open').click();
  await expect(page).toHaveURL(/\/settings\/general$/);
  await page.getByTestId('settings-open').click();
  await expect(page).toHaveURL(/\/mail\/INBOX$/);
  await expect(page.getByTestId('message').first()).toBeVisible();
});
