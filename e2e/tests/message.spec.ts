// SPDX-License-Identifier: GPL-3.0-or-later

import { expect, test } from '@playwright/test';
import { openBySubject, signIn } from './helpers';

test.use({ locale: 'en-US', timezoneId: 'Europe/Belgrade' });

test('plain text message opens, marks read and goes back', async ({ page }) => {
  await signIn(page);
  const badge = page.locator('[data-testid=folder][data-folder=INBOX] .badge');
  const before = Number(await badge.textContent());
  const wasUnread = (await page.getByTestId('message').filter({ hasText: 'Pozdrav šefče' }).getAttribute('class'))?.includes('unread');

  await openBySubject(page, 'Pozdrav šefče');
  await expect(page).toHaveURL(/\/mail\/INBOX\/\d+$/);
  await expect(page.getByTestId('message-text')).toContainText('Šta ima, čika Đorđe? Žuta ćurka.');
  await expect(page.getByTestId('message-from')).toContainText('Bob Sender <bob@golub.test>');
  await expect(badge).toHaveText(String(wasUnread ? before - 1 : before));

  await page.reload();
  await expect(page.getByTestId('message-text')).toContainText('Žuta ćurka');

  await page.getByTestId('back').click();
  await expect(page).toHaveURL(/\/mail\/INBOX$/);
  await expect(page.getByTestId('message').filter({ hasText: 'Pozdrav šefče' })).not.toHaveClass(/unread/);

  await page.goBack();
  await expect(page.getByTestId('message-subject')).toHaveText('Pozdrav šefče');
});

test('HTML message holds back remote images until asked', async ({ page }) => {
  const remote: string[] = [];
  page.on('request', (r) => new URL(r.url()).hostname.endsWith('example.com') && remote.push(r.url()));
  await signIn(page);
  await openBySubject(page, 'Simple HTML');

  const frame = page.frameLocator('[data-testid=message-body]');
  await expect(frame.locator('h1')).toHaveText('Newsletter');
  await expect(page.getByTestId('images-blocked')).toContainText('1 remote image was hidden');
  await expect(page.getByTestId('trackers')).toHaveText('1 tracking pixel blocked');
  await expect(frame.locator('img')).toHaveCount(0);
  await expect(frame.locator('a[href="https://example.com/"]')).toHaveAttribute('target', '_blank');

  await page.getByTestId('show-images').click();
  await expect(page.getByTestId('message-body')).toHaveAttribute('src', /images=1/);
  await expect(frame.locator('img')).toHaveAttribute('src', /^\/proxy\?u=https%3A%2F%2Fexample\.com/);
  expect(remote).toEqual([]);
});

test('inline cid images render from the message itself', async ({ page }) => {
  await signIn(page);
  await openBySubject(page, 'Inline image');
  const img = page.frameLocator('[data-testid=message-body]').locator('img');
  await expect(img).toHaveAttribute('src', /^data:image\/png;base64,/);
  expect(await img.evaluate((el: HTMLImageElement) => el.naturalWidth)).toBe(1);
  await expect(page.getByTestId('attachment')).toHaveCount(0);
});

test('attachments download with their real names', async ({ page }) => {
  await signIn(page);
  await openBySubject(page, 'With attachments');
  await expect(page.getByTestId('attachment')).toHaveCount(2);
  await page.getByTestId('attachment').filter({ hasText: 'članarina šema.txt' }).click();
  const preview = page.getByTestId('attachment-preview');
  await expect(preview).toContainText('članarina šema.txt');
  await expect(page.getByTestId('preview-text')).toContainText('RFC 2231 encoded filename.');
  const [download] = await Promise.all([page.waitForEvent('download'), page.getByTestId('preview-download').click()]);
  expect(download.suggestedFilename()).toBe('članarina šema.txt');
  await page.keyboard.press('Escape');
  await expect(preview).toBeHidden();
  const pdf = page.getByTestId('attachment').filter({ hasText: 'report.pdf' });
  await expect(pdf).toContainText('byte');
});

const xss = [
  'XSS script tag',
  'XSS svg onload',
  'XSS javascript link',
  'XSS meta refresh',
  'XSS base tag',
  'XSS CSS',
  'XSS form and iframe',
];

for (const subject of xss) {
  test(`${subject}: no script runs and nothing leaves the browser`, async ({ page }) => {
    const leaks: string[] = [];
    const dialogs: string[] = [];
    page.on('request', (r) => new URL(r.url()).hostname.endsWith('evil.test') && leaks.push(r.url()));
    page.on('dialog', (d) => {
      dialogs.push(d.message());
      void d.dismiss();
    });
    await signIn(page);
    await openBySubject(page, subject);
    const frame = page.frameLocator('[data-testid=message-body]');
    await expect(frame.locator('body')).toBeAttached();

    for (const sel of ['script', 'svg', 'form', 'iframe', 'object', 'meta[http-equiv]', 'base', 'style:not(head style)', 'input']) {
      await expect(frame.locator(sel)).toHaveCount(0);
    }
    for (const a of await frame.locator('a').all()) {
      const href = (await a.getAttribute('href')) ?? '';
      expect(href.toLowerCase()).not.toMatch(/^(javascript|data):/);
    }
    await page.getByTestId('show-images').click({ timeout: 1000 }).catch(() => undefined);
    await page.waitForTimeout(1500);
    expect(leaks).toEqual([]);
    expect(dialogs).toEqual([]);
    await expect(page).toHaveURL(/\/mail\/INBOX\/\d+$/);
  });
}
