// SPDX-License-Identifier: GPL-3.0-or-later

import en from '../locales/en.json';
import srLatn from '../locales/sr-Latn.json';
import { createTranslator, resolveLocale, type Catalog } from './i18n';
import { LANG_COOKIE, readCookie, writePref } from './prefs';

const catalogs: Record<string, Catalog> = { en, 'sr-Latn': srLatn };
export const LOCALES = Object.keys(catalogs);

const initial = resolveLocale(
  [readCookie(document.cookie, LANG_COOKIE), ...navigator.languages],
  LOCALES,
);

const state = $state({ locale: initial });

export const i18n = {
  get locale() {
    return state.locale;
  },
  get t() {
    return createTranslator(state.locale, catalogs);
  },
  set(locale: string) {
    state.locale = resolveLocale([locale], LOCALES);
    document.documentElement.lang = state.locale;
    writePref(LANG_COOKIE, state.locale);
  },
};

document.documentElement.lang = initial;
