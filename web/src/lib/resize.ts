// SPDX-License-Identifier: GPL-3.0-or-later

// Column widths live in localStorage only; the server never sees them.
const KEY = 'golub.columns';

export interface Columns {
  sidebar: number;
  list: number;
}

export const LIMITS = { sidebar: [208, 480], list: [280, 720] } as const;

export function clamp(which: keyof Columns, px: number): number {
  const [min, max] = LIMITS[which];
  return Math.round(Math.min(max, Math.max(min, px)));
}

export function loadColumns(): Columns | null {
  try {
    const v = JSON.parse(localStorage.getItem(KEY) ?? 'null');
    if (v && typeof v.sidebar === 'number' && typeof v.list === 'number') {
      return { sidebar: clamp('sidebar', v.sidebar), list: clamp('list', v.list) };
    }
  } catch {
    // corrupt or blocked storage falls back to the default layout
  }
  return null;
}

export function saveColumns(c: Columns | null) {
  try {
    if (c) localStorage.setItem(KEY, JSON.stringify(c));
    else localStorage.removeItem(KEY);
  } catch {
    // storage may be disabled; resizing still works for this page view
  }
}
