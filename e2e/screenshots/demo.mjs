// SPDX-License-Identifier: GPL-3.0-or-later
// The invented demo mailbox behind the README and wiki screenshots: bob's test account, emptied and filled again on every run.
import { request } from '@playwright/test';

export const base = process.env.GOLUB_URL ?? 'http://localhost:8080';
export const password = process.env.E2E_PASSWORD;
export const user = 'bob@golub.test';
if (!password) throw new Error('set E2E_PASSWORD');

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
    html: `<div style="font-family:sans-serif;max-width:520px"><h2 style="color:#0f766e">Your trip is confirmed</h2><p>Flight NW 482, Belgrade to Lisbon, departs <b>12 June at 09:40</b>.</p><p>Booking reference: <b>QX7PLM</b></p><img src="https://images.example.com/northwind/map.png" width="480" alt="Route map"></div>` },
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

export async function seed() {
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
  st.identities = st.identities.filter((i) => i.default).map((i) => ({ ...i, name: 'Bob Novak', signature: '--\nBob Novak' }));
  st.identities.push({ id: 'support', name: 'Bob Novak', email: 'support@golub.test', organization: 'Golub Support', signature: '--\nBob Novak\nGolub Support' });
  st.responses = [
    { id: 'thanks', name: 'Thanks, will check', text: 'Thanks! I will check and get back to you today.' },
    { id: 'ooo', name: 'Out of office', text: 'I am out of the office until Monday and will reply when I am back.' },
  ];
  st.threads = false;
  await ok(api.put('/api/settings', { data: st }));
  await ok(api.put('/api/pgp', { data: { keys: [] } }));
  const people = [['Ana Petrović', 'ana'], ['Marko Ilić', 'marko'], ['Jelena Stanković', 'jelena'], ['Ivana Kovač', 'ivana'], ['Maja Jovanović', 'maja'], ['Nikola Đorđević', 'nikola']];
  await ok(api.put('/api/contacts', { data: {
    groups: ['Team', 'Family'],
    contacts: people.map(([name, local], i) => ({ id: '', name, emails: [`${local}@golub.test`], groups: i < 4 ? ['Team'] : [] })),
    collected: [],
  } }));
  await api.dispose();
}

export async function login(page) {
  await page.goto(base);
  await page.locator('input[type=email]').fill(user);
  await page.locator('input[type=password]').fill(password);
  await page.locator('form button[type=submit]').click();
  await page.getByTestId('message').first().waitFor();
}

