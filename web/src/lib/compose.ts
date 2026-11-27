// SPDX-License-Identifier: GPL-3.0-or-later

import type { Translate } from './i18n';
import type { Address } from './format';
import { formatFullDate } from './format';

export interface Quote {
  from: Address;
  date: string;
  text: string;
  html: string;
}

export function escapeHtml(s: string): string {
  return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
}

export function splitRecipients(input: string): string[] {
  const out: string[] = [];
  let cur = '';
  let quoted = false;
  let angle = false;
  for (const ch of input) {
    if (ch === '"') quoted = !quoted;
    if (ch === '<') angle = true;
    if (ch === '>') angle = false;
    if ((ch === ',' || ch === ';' || ch === '\n') && !quoted && !angle) {
      if (cur.trim()) out.push(cur.trim());
      cur = '';
      continue;
    }
    cur += ch;
  }
  if (cur.trim()) out.push(cur.trim());
  return out;
}

const addrRe = /^[^\s@<>",;]+@[^\s@<>",;]+\.[^\s@<>",;]+$/;

export function recipientAddress(r: string): string {
  const m = /<([^>]+)>\s*$/.exec(r);
  return (m ? m[1] : r).trim();
}

export function isBareAddress(s: string): boolean {
  return addrRe.test(s.trim());
}

export function isValidRecipient(r: string): boolean {
  return addrRe.test(recipientAddress(r));
}

function person(a: Address): string {
  return a.name ? `${a.name} <${a.address}>` : a.address;
}

function wroteLine(t: Translate, locale: string, q: Quote): string {
  return t('compose.wrote', { date: formatFullDate(locale, q.date), name: q.from.name || q.from.address });
}

export function quotePlain(t: Translate, locale: string, q: Quote): string {
  const body = q.text.replace(/\r\n/g, '\n').replace(/\n+$/, '');
  return `${wroteLine(t, locale, q)}\n${body
    .split('\n')
    .map((l) => (l.startsWith('>') ? `>${l}` : `> ${l}`))
    .join('\n')}`;
}

export function quoteHtml(t: Translate, locale: string, q: Quote): string {
  const inner = q.html || `<div>${escapeHtml(q.text).replace(/\n/g, '<br>')}</div>`;
  return `<p>${escapeHtml(wroteLine(t, locale, q))}</p><blockquote type="cite">${inner}</blockquote>`;
}

function forwardHeader(t: Translate, locale: string, q: Quote, subject: string, to: string[]): string[] {
  return [
    t('compose.forwarded'),
    `${t('message.from')}: ${person(q.from)}`,
    `${t('message.date')}: ${formatFullDate(locale, q.date)}`,
    `${t('compose.subject')}: ${subject}`,
    ...(to.length ? [`${t('message.to')}: ${to.join(', ')}`] : []),
  ];
}

export function forwardPlain(t: Translate, locale: string, q: Quote, subject: string, to: string[]): string {
  return `${forwardHeader(t, locale, q, subject, to).join('\n')}\n\n${q.text}`;
}

export function forwardHtml(t: Translate, locale: string, q: Quote, subject: string, to: string[]): string {
  const head = forwardHeader(t, locale, q, subject, to).map(escapeHtml).join('<br>');
  const inner = q.html || `<div>${escapeHtml(q.text).replace(/\n/g, '<br>')}</div>`;
  return `<p>${head}</p>${inner}`;
}

// Plain-text body: new text, signature and quote in the user's chosen order.
// Signatures that already start with a "--" separator keep it instead of getting a second one.
export function signatureBlock(signature: string): string {
  if (!signature.trim()) return '';
  return /^--\s*(\n|$)/.test(signature) ? signature : `-- \n${signature}`;
}

export function withSignature(body: string, signature: string, quote: string, position: 'above' | 'below'): string {
  const sig = signatureBlock(signature);
  if (position === 'below' && quote) return `${quote}\n\n${body}${sig ? `\n\n${sig}` : ''}`;
  if (!sig && !quote) return body;
  return `${body}\n\n${sig}${sig && quote ? '\n\n' : ''}${quote}`;
}
