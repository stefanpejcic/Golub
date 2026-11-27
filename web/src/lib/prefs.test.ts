// SPDX-License-Identifier: GPL-3.0-or-later

import { describe, expect, it } from 'vitest';
import { readCookie, parseTheme } from './prefs';

describe('readCookie', () => {
  it('finds a value among others', () => {
    expect(readCookie('a=1; golub_theme=dark; b=2', 'golub_theme')).toBe('dark');
  });
  it('decodes values', () => {
    expect(readCookie('golub_lang=sr-Latn', 'golub_lang')).toBe('sr-Latn');
  });
  it('returns undefined when missing or prefix only', () => {
    expect(readCookie('xgolub_theme=dark', 'golub_theme')).toBeUndefined();
    expect(readCookie('', 'golub_theme')).toBeUndefined();
  });
});

describe('parseTheme', () => {
  it('accepts known values', () => {
    expect(parseTheme('dark')).toBe('dark');
    expect(parseTheme('light')).toBe('light');
  });
  it('defaults to system', () => {
    expect(parseTheme('blue')).toBe('system');
    expect(parseTheme(undefined)).toBe('system');
  });
});
