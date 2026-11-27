// SPDX-License-Identifier: GPL-3.0-or-later
// Fills bob's test mailbox with made-up mail and captures the README screenshots.
// Usage: GOLUB_URL=http://localhost:8080 E2E_PASSWORD=... node screenshots/shots.mjs
import { chromium, devices, request } from '@playwright/test';
import { mkdirSync } from 'node:fs';

const base = process.env.GOLUB_URL ?? 'http://localhost:8080';
const password = process.env.E2E_PASSWORD;
const user = 'bob@golub.test';
const out = new URL('../../assets/screenshots/', import.meta.url).pathname;
if (!password) throw new Error('set E2E_PASSWORD');
mkdirSync(out, { recursive: true });

const hoursAgo = (h) => new Date(Date.now() - h * 3600e3);
const rfc = (d) => d.toUTCString().replace('GMT', '+0000');

// Every name, company and address here is invented for the screenshots.
const mail = [
  { folder: 'INBOX', h: 0.4, from: 'Ana Petrović <ana@golub.test>', subject: 'Design review moved to Thursday', unread: true, flagged: true,
    html: `<h2 style="font-family:sans-serif;margin:0 0 12px">Design review</h2><p style="font-family:sans-serif">Hi Bob,</p><p style="font-family:sans-serif">The review moved to <b>Thursday at 10:00</b>. I attached the latest mockups, and the agenda is below.</p><ul style="font-family:sans-serif"><li>New onboarding flow</li><li>Dark mode polish</li><li>Mobile navigation</li></ul><p style="font-family:sans-serif">Let me know if the time does not work for you.</p><p style="font-family:sans-serif">Ana</p>`,
    attach: 'mockups.pdf' },
  { folder: 'INBOX', h: 1.5, from: 'Marko Ilić <marko@golub.test>', subject: 'Re: Server migration plan', unread: true, id: 'mig-2', reply: 'mig-1',
    text: 'Sounds good. I will take the database part and send you the checklist tonight.\n\nMarko' },
  { folder: 'INBOX', h: 3, from: 'Northwind Travel <hello@northwind.golub.test>', subject: 'Your trip to Lisbon is confirmed', list: true,
    html: `<div style="font-family:sans-serif;max-width:520px"><h2 style="color:#0f766e">Your trip is confirmed</h2><p>Flight NW 482, Belgrade to Lisbon, departs <b>12 June at 09:40</b>.</p><p>Booking reference: <b>QX7PLM</b></p></div>` },
  { folder: 'INBOX', h: 5, from: 'Jelena Stanković <jelena@golub.test>', subject: 'Quarterly numbers', priority: 1, unread: true,
    text: 'Bob, can you check the Q2 report before Friday? The totals in section 3 look off.\n\nThanks,\nJelena', attach: 'q2-report.pdf' },
  { folder: 'INBOX', h: 9, from: 'Stefan Marić <stefan.m@golub.test>', subject: 'Lunch tomorrow?', text: 'The new place near the office, 13:00?' },
  { folder: 'INBOX', h: 26, from: 'Open Source Weekly <news@osw.golub.test>', subject: 'This week: faster builds and smaller images', list: true,
    text: 'Ten links worth reading this week.' },
  { folder: 'INBOX', h: 30, from: 'Ivana Kovač <ivana@golub.test>', subject: 'Photos from the team trip', text: 'Here are the best ones!', attach: 'photos.zip' },
  { folder: 'INBOX', h: 50, from: 'Marko Ilić <marko@golub.test>', subject: 'Server migration plan', id: 'mig-1',
    text: 'Draft plan for moving the mail servers next month. Can you review?' },
  { folder: 'INBOX', h: 75, from: 'Nikola Đorđević <nikola@golub.test>', subject: 'Invoice 2026-118', text: 'Invoice for May is attached.', attach: 'invoice-2026-118.pdf' },
  { folder: 'INBOX', h: 100, from: 'Maja Jovanović <maja@golub.test>', subject: 'Welcome to the team!', text: 'Great to have you on board, Bob.' },
  { folder: 'Work', h: 120, from: 'Ana Petrović <ana@golub.test>', subject: 'Roadmap for the next quarter', text: 'First draft attached.' },
  { folder: 'Work', h: 140, from: 'Jelena Stanković <jelena@golub.test>', subject: 'Budget approved', text: 'Good news, the budget is approved.' },
  { folder: 'Travel', h: 200, from: 'Northwind Travel <hello@northwind.golub.test>', subject: 'Check-in is open', text: 'Check in now and pick your seat.' },
];

function build(m) {
  const id = `<${m.id ?? Math.random().toString(36).slice(2)}@golub.test>`;
  const boundary = 'b-' + Math.random().toString(36).slice(2);
  const head = [
    `From: ${m.from}`, `To: Bob Novak <${user}>`, `Subject: ${m.subject}`, `Date: ${rfc(hoursAgo(m.h))}`, `Message-ID: ${id}`, 'MIME-Version: 1.0',
    ...(m.reply ? [`In-Reply-To: <${m.reply}@golub.test>`, `References: <${m.reply}@golub.test>`] : []),
    ...(m.priority ? ['X-Priority: 1 (Highest)', 'Importance: high'] : []),
    ...(m.list ? ['List-Unsubscribe: <mailto:unsubscribe@golub.test>'] : []),
  ];
  const bodyType = m.html ? 'text/html' : 'text/plain';
  const body = m.html ?? m.text;
  if (!m.attach) return [...head, `Content-Type: ${bodyType}; charset=utf-8`, '', body, ''].join('\r\n');
  return [
    ...head, `Content-Type: multipart/mixed; boundary="${boundary}"`, '',
    `--${boundary}`, `Content-Type: ${bodyType}; charset=utf-8`, '', body,
    `--${boundary}`, `Content-Type: application/pdf; name="${m.attach}"`, `Content-Disposition: attachment; filename="${m.attach}"`, 'Content-Transfer-Encoding: base64', '',
    Buffer.from('%PDF-1.4\n% demo\n'.repeat(400)).toString('base64').replace(/.{76}/g, '$&\r\n'),
    `--${boundary}--`, '',
  ].join('\r\n');
}

// mbox separators use the asctime date format, for example "Fri Nov 27 16:31:00 2026".
const asctime = (d) => {
  const [wd, day, mon, year, time] = d.toUTCString().replace(',', '').split(' ');
  return `${wd} ${mon} ${day.padStart(2, ' ')} ${time} ${year}`;
};
const mbox = (list) => list.map((m) => `From ${user} ${asctime(hoursAgo(m.h))}\n${build(m).replace(/\r\n/g, '\n').replace(/^(>*From )/gm, '>$1')}\n`).join('');

async function seed() {
  const api = await request.newContext({ baseURL: base });
  const ok = async (p) => {
    const r = await p;
    if (!r.ok()) throw new Error(`${r.url()} ${r.status()} ${await r.text()}`);
    return r.status() === 204 ? null : r.json().catch(() => null);
  };
  await ok(api.post('/api/login', { data: { email: user, password } }));
  const { folders } = await ok(api.get('/api/folders'));
  for (const f of folders) {
    if (!f.selectable) continue;
    if (f.special || f.name === 'INBOX') await ok(api.post('/api/folders/empty', { data: { name: f.name } }));
    else await ok(api.post('/api/folders/delete', { data: { name: f.name } }));
  }
  for (const name of ['Work', 'Travel', 'Receipts']) await ok(api.post('/api/folders/create', { data: { name } }));
  for (const folder of ['INBOX', 'Work', 'Travel']) {
    const list = mail.filter((m) => m.folder === folder);
    await ok(api.post(`/api/folders/import?folder=${folder}`, { data: Buffer.from(mbox(list)), headers: { 'Content-Type': 'application/octet-stream' } }));
  }
  const page = await ok(api.get('/api/messages?folder=INBOX&size=50'));
  const bySubject = Object.fromEntries(page.messages.map((m) => [m.subject, m.uid]));
  const pick = (f) => mail.filter((m) => m.folder === 'INBOX' && f(m)).map((m) => bySubject[m.subject]);
  await ok(api.post('/api/messages/flag', { data: { folder: 'INBOX', uids: pick((m) => m.unread), flag: 'seen', value: false } }));
  await ok(api.post('/api/messages/flag', { data: { folder: 'INBOX', uids: pick((m) => m.flagged), flag: 'flagged', value: true } }));

  const st = await ok(api.get('/api/settings'));
  st.identities = st.identities.map((i) => ({ ...i, name: 'Bob Novak', signature: '--\nBob Novak' }));
  st.threads = false;
  await ok(api.put('/api/settings', { data: st }));
  const people = [['Ana Petrović', 'ana'], ['Marko Ilić', 'marko'], ['Jelena Stanković', 'jelena'], ['Ivana Kovač', 'ivana'], ['Maja Jovanović', 'maja'], ['Nikola Đorđević', 'nikola']];
  await ok(api.put('/api/contacts', { data: {
    groups: ['Team', 'Family'],
    contacts: people.map(([name, local], i) => ({ id: '', name, emails: [`${local}@golub.test`], groups: i < 4 ? ['Team'] : [] })),
    collected: [],
  } }));
  await api.dispose();
}

async function login(page) {
  await page.goto(base);
  await page.locator('input[type=email]').fill(user);
  await page.locator('input[type=password]').fill(password);
  await page.locator('form button[type=submit]').click();
  await page.getByTestId('message').first().waitFor();
}

async function shoot() {
  const browser = await chromium.launch();
  const desktop = { viewport: { width: 1440, height: 900 }, deviceScaleFactor: 1, locale: 'en-US', timezoneId: 'Europe/Belgrade' };

  for (const scheme of ['light', 'dark']) {
    const ctx = await browser.newContext({ ...desktop, colorScheme: scheme });
    const page = await ctx.newPage();
    await login(page);
    await page.getByTestId('message').filter({ hasText: 'Design review moved' }).getByRole('link').click();
    await page.getByTestId('message-subject').waitFor();
    await page.mouse.move(1000, 600);
    await page.waitForTimeout(800);
    await page.screenshot({ path: `${out}/inbox-${scheme}.png` });
    if (scheme === 'light') {
      await page.getByTestId('compose-open').click();
      await page.getByTestId('compose-to-input').pressSequentially('team', { delay: 40 });
      await page.waitForTimeout(400);
      await page.screenshot({ path: `${out}/compose.png` });
      await page.keyboard.press('Escape');
      await page.goto(`${base}/contacts`);
      await page.getByTestId('contact').first().click();
      await page.waitForTimeout(400);
      await page.screenshot({ path: `${out}/contacts.png` });
    }
    await ctx.close();
  }

  const login0 = await browser.newContext({ ...desktop, colorScheme: 'light' });
  const lp = await login0.newPage();
  await lp.goto(base);
  await lp.locator('input[type=email]').fill(user);
  await lp.waitForTimeout(400);
  await lp.screenshot({ path: `${out}/login.png` });
  await login0.close();

  const { defaultBrowserType: _, ...iphone } = devices['iPhone 13'];
  const phone = await browser.newContext({ ...iphone, locale: 'en-US', timezoneId: 'Europe/Belgrade', colorScheme: 'light' });
  const mp = await phone.newPage();
  await login(mp);
  await mp.waitForTimeout(500);
  await mp.screenshot({ path: `${out}/mobile-list.png` });
  await mp.getByTestId('message').filter({ hasText: 'Design review moved' }).getByRole('link').tap();
  await mp.getByTestId('message-subject').waitFor();
  await mp.waitForTimeout(800);
  await mp.evaluate(() => document.querySelectorAll('*').forEach((el) => (el.scrollTop = 0)));
  await mp.screenshot({ path: `${out}/mobile-message.png` });
  await social(browser);
  await browser.close();
}

// GitHub's social preview: 1280x640, mascot and name on the left, the app on the right.
async function social(browser) {
  const { readFileSync } = await import('node:fs');
  const asset = (f, type) => `data:${type};base64,${readFileSync(new URL(f, new URL('../../assets/', import.meta.url))).toString('base64')}`;
  const page = await browser.newPage({ viewport: { width: 1280, height: 640 } });
  await page.setContent(`<!doctype html><html><body style="margin:0">
    <div style="width:1280px;height:640px;display:flex;align-items:center;overflow:hidden;font-family:Inter,system-ui,sans-serif;
                background:radial-gradient(circle at 20% 30%,#e0f7fa,#f8fafc 60%)">
      <div style="flex:none;width:470px;padding-left:72px">
        <img src="${asset('golub-mascot.svg', 'image/svg+xml')}" width="120" height="120">
        <div style="font-size:76px;font-weight:800;letter-spacing:-2px;color:#0d0d0d;margin-top:8px">Golub</div>
        <div style="font-size:30px;line-height:1.3;color:#334155;margin-top:10px">Privacy-first webmail<br>in a single Go binary</div>
        <div style="font-size:20px;color:#64748b;margin-top:22px">No database · No tracking · Self-hosted</div>
      </div>
      <img src="${asset('screenshots/inbox-light.png', 'image/png')}"
           style="width:980px;margin-left:10px;border-radius:14px;box-shadow:0 20px 60px rgba(15,23,42,.25);border:1px solid #e2e8f0">
    </div></body></html>`);
  await page.waitForTimeout(300);
  await page.screenshot({ path: `${out}/../social-preview.png` });
  await page.close();
}

await seed();
await shoot();
console.log('screenshots written to', out);
