// SPDX-License-Identifier: GPL-3.0-or-later

import { createTranslator, resolveLocale, textDirection, type Catalog } from './i18n';
import { LANG_COOKIE, readCookie, writePref } from './prefs';

// each language is its own chunk and only the one in use is downloaded; CI keeps every catalog complete, so no fallback is bundled
const loaders: Record<string, () => Promise<{ default: Catalog }>> = {
  en: () => import('../locales/en.json'),
  'sr-Latn': () => import('../locales/sr-Latn.json'),
  ar: () => import('../locales/ar.json'),
};
export const LOCALES = Object.keys(loaders);

const initial = resolveLocale(
  [readCookie(document.cookie, LANG_COOKIE), ...navigator.languages],
  LOCALES,
);

const state = $state({ locale: initial });
// raw, so the big catalogs are not wrapped in reactive proxies key by key
let catalogs = $state.raw<Record<string, Catalog>>({});

async function load(locale: string) {
  if (catalogs[locale]) return;
  const { default: catalog } = await loaders[locale]();
  catalogs = { ...catalogs, [locale]: catalog };
}

// ready settles once the first language is in, the app mounts after it so nothing renders as raw keys
export const ready = load(initial);

export const i18n = {
  get locale() {
    return state.locale;
  },
  get t() {
    return createTranslator(state.locale, catalogs);
  },
  // set switches only after the new language arrived, so the page never shows half of each
  async set(locale: string) {
    const next = resolveLocale([locale], LOCALES);
    writePref(LANG_COOKIE, next);
    await load(next);
    state.locale = next;
    applyLang(next);
  },
};

function applyLang(locale: string) {
  document.documentElement.lang = locale;
  document.documentElement.dir = textDirection(locale);
}

applyLang(initial);
