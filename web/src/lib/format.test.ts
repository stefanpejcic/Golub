// SPDX-License-Identifier: GPL-3.0-or-later

import { describe, expect, it } from 'vitest';
import { formatListDate as raw, displayName, folderPath, messagePath, parseMailPath, formatSize } from './format';

const now = new Date('2026-11-27T15:00:00');

// ICU versions differ in which space they put before AM/PM.
const formatListDate = (l: string, iso: string, n: Date) => raw(l, iso, n).replace(/\s/g, ' ');

describe('formatListDate', () => {
  it('shows time for today', () => {
    expect(formatListDate('en', '2026-11-27T09:05:00', now)).toBe('9:05 AM');
    expect(formatListDate('sr-Latn', '2026-11-27T09:05:00', now)).toBe('09:05');
  });
  it('shows day and month this year', () => {
    expect(formatListDate('en', '2026-03-02T09:05:00', now)).toBe('Mar 2');
  });
  it('shows full date for older years', () => {
    expect(formatListDate('en', '2024-03-02T09:05:00', now)).toBe('3/2/24');
  });
  it('tolerates bad input', () => {
    expect(formatListDate('en', 'nope', now)).toBe('');
  });
});

describe('displayName', () => {
  it('prefers the name', () => {
    expect(displayName({ name: 'Bob', address: 'bob@x.test' })).toBe('Bob');
    expect(displayName({ address: 'bob@x.test' })).toBe('bob@x.test');
    expect(displayName(undefined)).toBe('');
  });
});

describe('mail paths', () => {
  it('round-trips folder names with slashes and unicode', () => {
    for (const name of ['INBOX', 'Archive/2025', 'Posao & Lično', '100%']) {
      expect(parseMailPath(folderPath(name))).toEqual({ folder: name, uid: null });
      expect(parseMailPath(messagePath(name, 42))).toEqual({ folder: name, uid: 42 });
    }
  });
  it('returns null outside /mail/', () => {
    expect(parseMailPath('/')).toBeNull();
    expect(parseMailPath('/mail/')).toBeNull();
    expect(parseMailPath('/mail/INBOX/abc')).toBeNull();
  });
});

describe('formatSize', () => {
  const f = (l: string, b: number) => formatSize(l, b).replace(/\s/g, ' ');
  it('picks a sensible unit', () => {
    expect(f('en', 512)).toBe('512 byte');
    expect(f('en', 1536)).toBe('1.5 kB');
    expect(f('en', 25 * 1024 * 1024)).toBe('25 MB');
    expect(f('sr-Latn', 1536)).toBe('1,5 kB');
  });
});
