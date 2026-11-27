// SPDX-License-Identifier: GPL-3.0-or-later
import { describe, expect, it } from 'vitest';
import { previewKind } from './preview';

describe('previewKind', () => {
  it('previews only safe types', () => {
    expect(previewKind('image/PNG', 10)).toBe('image');
    expect(previewKind('application/pdf', 10)).toBe('pdf');
    expect(previewKind('text/plain; charset=windows-1250', 10)).toBe('text');
    expect(previewKind('text/plain', 5 << 20)).toBeNull();
    expect(previewKind('image/svg+xml', 10)).toBeNull();
    expect(previewKind('text/html', 10)).toBeNull();
    expect(previewKind('application/zip', 10)).toBeNull();
  });
});
