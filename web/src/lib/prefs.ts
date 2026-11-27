// SPDX-License-Identifier: GPL-3.0-or-later

export const THEMES = ['system', 'light', 'dark'] as const;
export type Theme = (typeof THEMES)[number];

export const THEME_COOKIE = 'golub_theme';
export const LANG_COOKIE = 'golub_lang';

export function readCookie(cookie: string, name: string): string | undefined {
  for (const part of cookie.split(';')) {
    const [k, ...v] = part.trim().split('=');
    if (k === name) return decodeURIComponent(v.join('='));
  }
  return undefined;
}

export function parseTheme(v: string | undefined): Theme {
  return (THEMES as readonly string[]).includes(v ?? '') ? (v as Theme) : 'system';
}

// Readable by the pre-paint script, so not HttpOnly; holds no secrets.
export function writePref(name: string, value: string): void {
  const secure = location.protocol === 'https:' ? '; Secure' : '';
  document.cookie = `${name}=${encodeURIComponent(value)}; Path=/; Max-Age=31536000; SameSite=Lax${secure}`;
}
