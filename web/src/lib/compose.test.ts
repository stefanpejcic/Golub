// SPDX-License-Identifier: GPL-3.0-or-later

import { describe, expect, it } from 'vitest';
import { createTranslator } from './i18n';
import { splitRecipients, isValidRecipient, quotePlain, quoteHtml, forwardPlain, withSignature, signatureBlock, escapeHtml } from './compose';

const t = createTranslator('en', {
  en: {
    'compose.wrote': 'On {date}, {name} wrote:',
    'compose.forwarded': '---------- Forwarded message ----------',
    'message.from': 'From',
    'message.date': 'Date',
    'message.to': 'To',
    'compose.subject': 'Subject',
  },
});

const quote = { from: { name: 'Bob', address: 'bob@x.test' }, date: '2026-11-02T09:00:00Z', text: 'line one\nline two', html: '<p>hi <b>there</b></p>' };

describe('recipients', () => {
  it('splits on commas, semicolons and new lines but not inside quotes', () => {
    expect(splitRecipients('a@x.test, "Doe, John" <j@x.test>;b@x.test\nc@x.test')).toEqual([
      'a@x.test',
      '"Doe, John" <j@x.test>',
      'b@x.test',
      'c@x.test',
    ]);
  });
  it('validates', () => {
    expect(isValidRecipient('Bob <bob@x.test>')).toBe(true);
    expect(isValidRecipient('bob@x.test')).toBe(true);
    expect(isValidRecipient('bob@')).toBe(false);
    expect(isValidRecipient('a b@x.test')).toBe(false);
  });
});

describe('quoting', () => {
  it('quotes plain text with a wrote line', () => {
    const q = quotePlain(t, 'en', quote);
    expect(q.startsWith('On ')).toBe(true);
    expect(q).toContain('Bob wrote:');
    expect(q).toContain('> line one\n> line two');
  });
  it('quotes html in a blockquote and escapes the name', () => {
    const q = quoteHtml(t, 'en', { ...quote, from: { name: '<Eve>', address: 'e@x.test' } });
    expect(q).toContain('<blockquote');
    expect(q).toContain('<b>there</b>');
    expect(q).toContain('&lt;Eve&gt;');
  });
  it('falls back to escaped text when the original has no html', () => {
    expect(quoteHtml(t, 'en', { ...quote, html: '', text: 'a < b' })).toContain('a &lt; b');
  });
  it('builds a forward header', () => {
    const f = forwardPlain(t, 'en', quote, 'Hello', ['carol@x.test']);
    expect(f).toContain('---------- Forwarded message ----------');
    expect(f).toContain('From: Bob <bob@x.test>');
    expect(f).toContain('Subject: Hello');
    expect(f).toContain('line two');
  });
});

describe('signature', () => {
  it('places the signature above or below the quote', () => {
    expect(withSignature('', 'Sig', 'QUOTE', 'above')).toBe('\n\n-- \nSig\n\nQUOTE');
    expect(withSignature('', 'Sig', 'QUOTE', 'below')).toBe('QUOTE\n\n\n\n-- \nSig');
    expect(withSignature('', '', '', 'above')).toBe('');
  });
  it('does not double an existing separator', () => {
    expect(signatureBlock('--\nAlice')).toBe('--\nAlice');
    expect(signatureBlock('-- \nAlice')).toBe('-- \nAlice');
    expect(signatureBlock('Alice')).toBe('-- \nAlice');
    expect(signatureBlock('  ')).toBe('');
  });
});

describe('html helpers', () => {
  it('escapes', () => {
    expect(escapeHtml('<a & "b">')).toBe('&lt;a &amp; &quot;b&quot;&gt;');
  });
});
