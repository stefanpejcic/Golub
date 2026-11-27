// SPDX-License-Identifier: GPL-3.0-or-later

export type PluralForms = Partial<Record<Intl.LDMLPluralRule, string>> & { other: string };
export type Message = string | PluralForms;
export type Catalog = Record<string, Message>;
export type Params = Record<string, string | number>;
export type Translate = (key: string, params?: Params) => string;

export const FALLBACK_LOCALE = 'en';

export function createTranslator(locale: string, catalogs: Record<string, Catalog>): Translate {
  const primary = catalogs[locale] ?? {};
  const fallback = catalogs[FALLBACK_LOCALE] ?? {};
  const plurals = new Intl.PluralRules(locale);
  const numbers = new Intl.NumberFormat(locale);

  return (key, params = {}) => {
    const msg = primary[key] ?? fallback[key];
    if (msg === undefined) return key;
    let text: string;
    if (typeof msg === 'string') {
      text = msg;
    } else {
      const n = Number(params.count ?? 0);
      text = msg[plurals.select(n)] ?? msg.other;
    }
    return text.replace(/\{(\w+)\}/g, (whole, name: string) => {
      const v = params[name];
      if (v === undefined) return whole;
      return typeof v === 'number' ? numbers.format(v) : v;
    });
  };
}

// Serbian always means Latin here; Cyrillic is not shipped.
function normalize(tag: string): string | undefined {
  let canonical: string;
  try {
    canonical = Intl.getCanonicalLocales(tag)[0];
  } catch {
    return undefined;
  }
  const loc = new Intl.Locale(canonical);
  if (loc.language === 'sr') return loc.script && loc.script !== 'Latn' ? undefined : 'sr-Latn';
  return loc.language;
}

export function resolveLocale(candidates: (string | undefined | null)[], supported: string[]): string {
  for (const c of candidates) {
    if (!c) continue;
    if (supported.includes(c)) return c;
    const n = normalize(c);
    if (n && supported.includes(n)) return n;
  }
  return FALLBACK_LOCALE;
}

// Arabic, Hebrew, Persian and Urdu mirror the whole layout; the CSS uses logical properties for that
const RTL = ['ar', 'he', 'fa', 'ur'];

export function textDirection(locale: string): 'rtl' | 'ltr' {
  return RTL.includes(locale.split('-')[0]) ? 'rtl' : 'ltr';
}
