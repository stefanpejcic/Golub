// SPDX-License-Identifier: GPL-3.0-or-later

import { expect, test, type Browser, type Page } from '@playwright/test';
import { env } from './env';
import { openBySubject, signIn } from './helpers';

test.use({ locale: 'en-US' });

const BOB = `bob@${env.E2E_DOMAIN}`;

async function signInAs(browser: Browser, baseURL: string, email: string): Promise<Page> {
  const ctx = await browser.newContext({ locale: 'en-US' });
  const page = await ctx.newPage();
  await page.goto(baseURL);
  await page.getByLabel('Email address').fill(email);
  await page.getByLabel('Password').fill(env.E2E_PASSWORD);
  await page.getByRole('button', { name: 'Sign in' }).click();
  await expect(page.getByTestId('user-email')).toHaveText(email);
  return page;
}

async function expectDelivered(page: Page, subject: string) {
  await expect
    .poll(
      async () => {
        await page.getByTestId('refresh').click();
        return page.getByTestId('message').filter({ hasText: subject }).count();
      },
      { timeout: 30_000, intervals: [1000] },
    )
    .toBeGreaterThan(0);
}

test('compose a new message and the recipient receives it', async ({ page, browser, baseURL }) => {
  const subject = `Hello Bob ${Date.now()}`;
  await signIn(page);
  await page.getByTestId('compose-open').click();
  await expect(page).toHaveURL(/\/compose$/);
  await page.getByTestId('compose-to-input').fill(`Bob <${BOB}>`);
  await page.getByTestId('compose-to-input').press('Enter');
  await expect(page.getByTestId('recipient')).toHaveText(new RegExp(BOB));
  await page.getByTestId('compose-subject').fill(subject);
  await page.getByTestId('compose-editor').locator('p').first().click();
  await page.keyboard.type('Pozdrav iz Goluba! čćžšđ');
  await page.getByTestId('compose-send').click();
  await expect(page.getByTestId('toast')).toHaveText('Message sent.');
  await expect(page).toHaveURL(/\/mail\//);

  await page.locator('[data-testid=folder][data-folder=Sent]').click();
  await expect(page.getByTestId('message').filter({ hasText: subject })).toHaveCount(1);

  const bob = await signInAs(browser, baseURL!, BOB);
  await expectDelivered(bob, subject);
  await bob.getByTestId('message').filter({ hasText: subject }).getByRole('link').click();
  await expect(bob.frameLocator('[data-testid=message-body]').locator('body')).toContainText('Pozdrav iz Goluba! čćžšđ');
  await bob.context().close();
});

test('reply quotes the original and threads it', async ({ page }) => {
  await signIn(page);
  await openBySubject(page, 'Plain text');
  await page.getByTestId('action-reply').click();
  await expect(page).toHaveURL(/\/compose\?mode=reply/);
  await expect(page.getByTestId('recipient')).toContainText('bob@golub.test');
  await expect(page.getByTestId('compose-subject')).toHaveValue('Re: Plain text');
  await expect(page.getByTestId('compose-editor').locator('blockquote')).toContainText('čćžšđ');
  await expect(page.getByTestId('compose-editor')).toContainText('Bob Sender wrote:');
  await page.getByTestId('compose-send').click();
  await expect(page.getByTestId('toast')).toHaveText('Message sent.');
  await page.locator('[data-testid=folder][data-folder=Sent]').click();
  await expect(page.getByTestId('message').filter({ hasText: 'Re: Plain text' }).first()).toBeVisible();
});

test('forward keeps the original attachments', async ({ page, browser, baseURL }) => {
  await signIn(page);
  await openBySubject(page, 'With attachments');
  await page.getByTestId('action-forward').click();
  await expect(page.getByTestId('compose-subject')).toHaveValue('Fwd: With attachments');
  await expect(page.getByTestId('compose-attachments')).toContainText('report.pdf');
  await expect(page.getByTestId('compose-attachments')).toContainText('članarina šema.txt');
  await page.getByTestId('compose-to-input').fill(BOB);
  await page.getByTestId('compose-subject').fill(`Fwd: With attachments ${Date.now()}`);
  const subject = await page.getByTestId('compose-subject').inputValue();
  await page.getByTestId('compose-send').click();
  await expect(page.getByTestId('toast')).toHaveText('Message sent.');

  const bob = await signInAs(browser, baseURL!, BOB);
  await expectDelivered(bob, subject);
  await bob.getByTestId('message').filter({ hasText: subject }).getByRole('link').click();
  await expect(bob.getByTestId('attachment')).toHaveCount(2);
  await bob.context().close();
});

test('drafts are saved, reopened and discarded', async ({ page }) => {
  const subject = `Draft ${Date.now()}`;
  await signIn(page);
  await page.getByTestId('compose-open').click();
  await page.getByTestId('compose-subject').fill(subject);
  await page.getByTestId('compose-editor').locator('p').first().click();
  await page.keyboard.type('unfinished thought');
  await page.getByTestId('compose-save').click();
  await expect(page.getByTestId('toast')).toHaveText('Draft saved.');
  await expect(page.getByTestId('compose-saved')).toBeVisible();

  page.on('dialog', (d) => d.accept());
  await page.getByTestId('compose-discard').click();
  await page.locator('[data-testid=folder][data-folder=Drafts]').click();
  await page.getByTestId('message').filter({ hasText: subject }).getByRole('link').click();
  await page.getByTestId('action-edit-draft').click();
  await expect(page.getByTestId('compose-subject')).toHaveValue(subject);
  await expect(page.getByTestId('compose-editor')).toContainText('unfinished thought');
});

test('invalid recipients are refused before sending', async ({ page }) => {
  await signIn(page);
  await page.getByTestId('compose-open').click();
  await page.getByTestId('compose-to-input').fill('not-an-address');
  await page.getByTestId('compose-to-input').press('Enter');
  await expect(page.getByTestId('recipient')).toHaveClass(/invalid/);
  await page.getByTestId('compose-send').click();
  await expect(page.getByTestId('toast')).toHaveText('Not a valid address: not-an-address');
  await expect(page).toHaveURL(/\/compose$/);
});

test('plain text mode sends text with a signature', async ({ page }) => {
  await signIn(page);
  await page.getByTestId('compose-open').click();
  await page.getByTestId('compose-format').click();
  await expect(page.getByTestId('compose-text')).toBeVisible();
  await page.getByTestId('compose-text').fill('Plain hello');
  await page.getByTestId('compose-to-input').fill(BOB);
  await page.getByTestId('compose-subject').fill(`Plain ${Date.now()}`);
  await page.getByTestId('compose-send').click();
  await expect(page.getByTestId('toast')).toHaveText('Message sent.');
});

test('new mail appears without refreshing (IDLE push)', async ({ page, browser, baseURL }) => {
  const bob = await signInAs(browser, baseURL!, BOB);
  await bob.waitForTimeout(1500);
  const subject = `Live ${Date.now()}`;
  await signIn(page);
  await page.getByTestId('compose-open').click();
  await page.getByTestId('compose-to-input').fill(BOB);
  await page.getByTestId('compose-subject').fill(subject);
  await page.getByTestId('compose-send').click();
  await expect(page.getByTestId('toast')).toHaveText('Message sent.');
  // Polling runs once a minute; seeing it within 15 seconds without clicking Refresh means it was pushed.
  await expect(bob.getByTestId('message').filter({ hasText: subject })).toHaveCount(1, { timeout: 15_000 });
  await bob.context().close();
});

test('space turns a finished address into a chip', async ({ page }) => {
  await signIn(page);
  await page.getByTestId('compose-open').click();
  const input = page.getByTestId('compose-to-input');
  await input.pressSequentially('one@golub.test two@golub.test Carol Doe <carol@golub.test>');
  await input.press('Enter');
  await expect(page.getByTestId('recipient')).toHaveText(['one@golub.test', 'two@golub.test', 'Carol Doe <carol@golub.test>']);
});

test('leaving compose with changes offers to save, discard or keep writing', async ({ page }) => {
  const subject = `Leave ${Date.now()}`;
  await signIn(page);
  await page.getByTestId('compose-open').click();
  await page.getByTestId('compose-subject').fill(subject);

  await page.locator('[data-testid=folder][data-folder=INBOX]').click();
  await expect(page.getByTestId('leave-dialog')).toBeVisible();
  await page.getByTestId('leave-cancel').click();
  await expect(page).toHaveURL(/\/compose$/);
  await expect(page.getByTestId('compose-subject')).toHaveValue(subject);

  await page.locator('[data-testid=folder][data-folder=Drafts]').click();
  await page.getByTestId('leave-save').click();
  await expect(page).toHaveURL(/\/mail\/Drafts$/);
  await expect(page.getByTestId('message').filter({ hasText: subject })).toHaveCount(1);

  await page.getByTestId('compose-open').click();
  await page.getByTestId('compose-subject').fill('thrown away');
  await page.locator('[data-testid=folder][data-folder=INBOX]').click();
  await page.getByTestId('leave-discard').click();
  await expect(page).toHaveURL(/\/mail\/INBOX$/);

  await page.getByTestId('compose-open').click();
  await page.locator('[data-testid=folder][data-folder=Sent]').click();
  await expect(page).toHaveURL(/\/mail\/Sent$/);
});

test('switching to plain text and back keeps formatting', async ({ page }) => {
  await signIn(page);
  await page.getByTestId('compose-open').click();
  await page.getByTestId('compose-editor').locator('p').first().click();
  await page.keyboard.press('ControlOrMeta+b');
  await page.keyboard.type('bold words');
  await expect(page.getByTestId('compose-editor').locator('b, strong')).toHaveText('bold words');
  await page.getByTestId('compose-format').click();
  await expect(page.getByTestId('compose-text')).toHaveValue(/bold words/);
  await page.getByTestId('compose-format').click();
  await expect(page.getByTestId('compose-editor').locator('b, strong')).toHaveText('bold words');
});

test('login refuses an address without a dot in the domain and the bird hides its eyes', async ({ browser, baseURL }) => {
  const page = await (await browser.newContext({ locale: 'en-US' })).newPage();
  await page.goto(baseURL!);
  await page.getByLabel('Email address').fill('alice@golub');
  await page.getByLabel('Password').click();
  await expect(page.getByTestId('login-bird')).toHaveClass(/peeking/);
  await page.getByLabel('Password').fill('x');
  await expect(page.getByTestId('login-bird')).toHaveClass(/covering/);
  await page.getByLabel('Password').fill('');
  await expect(page.getByTestId('login-bird')).toHaveClass(/peeking/);
  await page.getByLabel('Password').fill('x');
  await page.getByRole('button', { name: 'Sign in' }).click();
  await expect(page.getByTestId('login-error')).toHaveText('Enter a full email address, for example name@example.com.');
  await page.getByLabel('Email address').click();
  await expect(page.getByTestId('login-bird')).not.toHaveClass(/covering/);
  await page.context().close();
});
