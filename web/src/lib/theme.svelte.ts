// SPDX-License-Identifier: GPL-3.0-or-later

import { THEME_COOKIE, parseTheme, readCookie, writePref, type Theme } from './prefs';

const state = $state({ theme: parseTheme(readCookie(document.cookie, THEME_COOKIE)) });

export const theme = {
  get value(): Theme {
    return state.theme;
  },
  set(v: Theme) {
    state.theme = v;
    document.documentElement.dataset.theme = v;
    writePref(THEME_COOKIE, v);
  },
};
