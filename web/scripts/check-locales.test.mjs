// SPDX-License-Identifier: GPL-3.0-or-later

import { describe, expect, it } from 'vitest';
import { compareKeys, findUnused, findRawStrings, checkPlurals } from './check-locales.mjs';

describe('compareKeys', () => {
  it('reports missing and extra keys', () => {
    const problems = compareKeys({ en: { a: 'A', b: 'B' }, 'sr-Latn': { a: 'A', c: 'C' } });
    expect(problems).toEqual([
      'sr-Latn: missing key "b"',
      'sr-Latn: unknown key "c"',
    ]);
  });
});

describe('findUnused', () => {
  it('treats literals and dynamic prefixes as used', () => {
    const sources = ["t('a.b')", 't(`theme.${v}`)', 'x = "c.d"'];
    const unused = findUnused(['a.b', 'theme.dark', 'c.d', 'e.f'], sources);
    expect(unused).toEqual(['e.f']);
  });
});

describe('findRawStrings', () => {
  it('flags text nodes and text attributes', () => {
    const src = `<script>let x = 1;</script>
<p>Hello</p>
<p>{t('ok')}</p>
<input placeholder="Search" />
<img alt={t('x')} />
<span>  {x} · 123 </span>
<style>p { color: red; }</style>`;
    expect(findRawStrings(src, 'A.svelte')).toEqual([
      'A.svelte:2 raw text "Hello"',
      'A.svelte:4 raw placeholder "Search"',
    ]);
  });
});

describe('checkPlurals', () => {
  it('requires every plural category of the locale', () => {
    const problems = checkPlurals({
      en: { n: { one: '1', other: 'n' } },
      'sr-Latn': { n: { one: '1', other: 'n' } },
    });
    expect(problems).toEqual(['sr-Latn: "n" lacks plural form "few"']);
  });
});
