// SPDX-License-Identifier: GPL-3.0-or-later

import { THEME_COOKIE, parseTheme, readCookie, writePref, type Theme } from './prefs';

const media = matchMedia('(prefers-color-scheme: dark)');
const state = $state({ theme: parseTheme(readCookie(document.cookie, THEME_COOKIE)), systemDark: media.matches });
media.addEventListener('change', (e) => (state.systemDark = e.matches));

export const theme = {
  get value(): Theme {
    return state.theme;
  },
  // what the user sees right now, with "system" resolved through the OS setting
  get dark(): boolean {
    return state.theme === 'dark' || (state.theme === 'system' && state.systemDark);
  },
  set(v: Theme) {
    state.theme = v;
    document.documentElement.dataset.theme = v;
    writePref(THEME_COOKIE, v);
  },
};
