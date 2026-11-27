// SPDX-License-Identifier: GPL-3.0-or-later

import { describe, expect, it } from 'vitest';
import { createTranslator, resolveLocale, textDirection } from './i18n';

const catalogs = {
  en: {
    'app.name': 'Golub',
    'greet': 'Hello, {name}',
    'list.count': { one: '{count} message', other: '{count} messages' },
    'only.en': 'English only',
  },
  'sr-Latn': {
    'app.name': 'Golub',
    'greet': 'Zdravo, {name}',
    'list.count': { one: '{count} poruka', few: '{count} poruke', other: '{count} poruka' },
  },
};

describe('createTranslator', () => {
  it('interpolates', () => {
    expect(createTranslator('en', catalogs)('greet', { name: 'Ana' })).toBe('Hello, Ana');
  });

  it('uses Serbian plural forms', () => {
    const t = createTranslator('sr-Latn', catalogs);
    expect(t('list.count', { count: 1 })).toBe('1 poruka');
    expect(t('list.count', { count: 3 })).toBe('3 poruke');
    expect(t('list.count', { count: 5 })).toBe('5 poruka');
    expect(t('list.count', { count: 21 })).toBe('21 poruka');
    expect(t('list.count', { count: 22 })).toBe('22 poruke');
  });

  it('formats plural count with locale digits grouping', () => {
    const t = createTranslator('en', catalogs);
    expect(t('list.count', { count: 1200 })).toBe('1,200 messages');
  });

  it('falls back to English, then to the key', () => {
    const t = createTranslator('sr-Latn', catalogs);
    expect(t('only.en')).toBe('English only');
    expect(t('missing.key')).toBe('missing.key');
  });

  it('leaves unknown placeholders intact', () => {
    expect(createTranslator('en', catalogs)('greet')).toBe('Hello, {name}');
  });
});

describe('resolveLocale', () => {
  const supported = ['en', 'sr-Latn'];
  it('prefers the first explicit candidate', () => {
    expect(resolveLocale(['sr-Latn', 'en'], supported)).toBe('sr-Latn');
  });
  it('maps plain Serbian tags to sr-Latn', () => {
    expect(resolveLocale(['sr'], supported)).toBe('sr-Latn');
    expect(resolveLocale(['sr-RS'], supported)).toBe('sr-Latn');
    expect(resolveLocale(['sr-Latn-RS'], supported)).toBe('sr-Latn');
  });
  it('does not map Cyrillic Serbian', () => {
    expect(resolveLocale(['sr-Cyrl'], supported)).toBe('en');
  });
  it('skips empty and unknown candidates', () => {
    expect(resolveLocale([undefined, '', 'de-DE', 'en-GB'], supported)).toBe('en');
    expect(resolveLocale([], supported)).toBe('en');
  });
});

describe('textDirection', () => {
  it('mirrors the layout only for right-to-left languages', () => {
    expect(textDirection('ar')).toBe('rtl');
    expect(textDirection('he-IL')).toBe('rtl');
    expect(textDirection('fa')).toBe('rtl');
    expect(textDirection('en')).toBe('ltr');
    expect(textDirection('sr-Latn')).toBe('ltr');
  });
});
