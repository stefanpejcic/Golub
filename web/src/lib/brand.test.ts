// SPDX-License-Identifier: GPL-3.0-or-later
import { describe, expect, it } from 'vitest';
import { textOn } from './brand.svelte';

describe('textOn', () => {
  it('uses white on dark accents and black on light ones', () => {
    expect(textOn('#2563eb')).toBe('#ffffff');
    expect(textOn('#000000')).toBe('#ffffff');
    expect(textOn('#ffcc00')).toBe('#0d0d0d');
    expect(textOn('#ffffff')).toBe('#0d0d0d');
  });
});
