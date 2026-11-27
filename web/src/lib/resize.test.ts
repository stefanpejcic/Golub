// SPDX-License-Identifier: GPL-3.0-or-later

import { afterEach, describe, expect, it, vi } from 'vitest';
import { clamp, loadColumns, saveColumns } from './resize';

function fakeStorage() {
  const m = new Map<string, string>();
  vi.stubGlobal('localStorage', {
    getItem: (k: string) => m.get(k) ?? null,
    setItem: (k: string, v: string) => m.set(k, v),
    removeItem: (k: string) => m.delete(k),
  });
  return m;
}

afterEach(() => vi.unstubAllGlobals());

describe('columns', () => {
  it('clamps to limits', () => {
    expect(clamp('sidebar', 10)).toBe(208);
    expect(clamp('list', 9999)).toBe(720);
    expect(clamp('list', 400.4)).toBe(400);
  });
  it('round-trips and resets', () => {
    fakeStorage();
    expect(loadColumns()).toBeNull();
    saveColumns({ sidebar: 300, list: 500 });
    expect(loadColumns()).toEqual({ sidebar: 300, list: 500 });
    saveColumns(null);
    expect(loadColumns()).toBeNull();
  });
  it('ignores garbage', () => {
    const m = fakeStorage();
    m.set('golub.columns', '{bad');
    expect(loadColumns()).toBeNull();
    m.set('golub.columns', JSON.stringify({ sidebar: 5, list: 99999 }));
    expect(loadColumns()).toEqual({ sidebar: 208, list: 720 });
  });
});
