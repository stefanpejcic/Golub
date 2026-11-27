// SPDX-License-Identifier: GPL-3.0-or-later
// One-time setup of the Afterlogic WebMail Lite container from bench/compose.yml, done through its admin panel.
// Usage: node bench/setup-afterlogic.mjs http://localhost:18086
import { chromium } from '@playwright/test';

const base = process.argv[2] ?? 'http://localhost:18086';
const browser = await chromium.launch();
const page = await (await browser.newContext({ viewport: { width: 1300, height: 1100 } })).newPage();

await page.goto(`${base}/adminpanel/`);
// a fresh install has superadmin with an empty password
await page.locator('[data-test-id=admin-login-name]').fill('superadmin');
await page.keyboard.press('Enter');

await page.getByText('Database settings', { exact: true }).click();
await page.getByRole('button', { name: 'Test connection' }).waitFor();
await page.locator('input[type=password]:visible').fill('bench-afterlogic');
await page.getByRole('button', { name: 'Save' }).click();
await page.waitForTimeout(1500);
await page.getByRole('button', { name: 'Create/Update tables' }).click();
await page.waitForTimeout(6000);

await page.getByText('Mail servers', { exact: true }).click();
const add = page.getByRole('button', { name: 'Add New Server' });
await add.waitFor();
if (!(await page.getByText('Golub test').count())) {
  await add.click();
  const inputs = page.locator('input[type=text]:visible');
  await inputs.nth(0).fill('Golub test');
  await inputs.nth(1).fill('mailserver');
  await inputs.nth(2).fill('143');
  await inputs.nth(3).fill('mailserver');
  await inputs.nth(4).fill('587');
  await page.getByRole('button', { name: /^(Create|Save)$/ }).last().click();
  await page.waitForTimeout(2500);
}
console.log((await page.locator('.q-notification').allInnerTexts()).map((s) => s.replace(/\s*close$/, '')).join('\n'));
await browser.close();
