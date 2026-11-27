// SPDX-License-Identifier: GPL-3.0-or-later
import { describe, expect, it } from 'vitest';
import { formatAddress, suggest, type Book } from './contacts.svelte';

const book: Book = {
  groups: ['Work'],
  contacts: [
    { id: '1', name: 'Željko Đurić', emails: ['zeljko@golub.test'], groups: ['Work'] },
    { id: '2', name: 'Doe, John', emails: ['john@golub.test', 'jd@golub.test'] },
  ],
  collected: [
    { email: 'rare@golub.test', count: 1, last: '' },
    { name: 'Zoran', email: 'zoran@golub.test', count: 5, last: '' },
  ],
};

describe('suggest', () => {
  it('matches names without diacritics, word starts and addresses', () => {
    expect(suggest(book, 'zelj').map((s) => s.detail)).toEqual(['zeljko@golub.test']);
    expect(suggest(book, 'duric')[0].detail).toBe('zeljko@golub.test');
    expect(suggest(book, 'jd')[0].values).toEqual(['"Doe, John" <jd@golub.test>']);
  });
  it('offers groups with all members and ranks collected by use', () => {
    const work = suggest(book, 'wor')[0];
    expect(work.kind).toBe('group');
    expect(work.values).toEqual(['Željko Đurić <zeljko@golub.test>']);
    expect(suggest(book, 'z').map((s) => s.detail)).toEqual(['zeljko@golub.test', 'zoran@golub.test']);
  });
  it('skips addresses already added', () => {
    expect(suggest(book, 'john', ['JOHN@golub.test']).map((s) => s.detail)).toEqual(['jd@golub.test']);
  });
});

describe('formatAddress', () => {
  it('quotes names with separators', () => {
    expect(formatAddress('Ana', 'a@x.test')).toBe('Ana <a@x.test>');
    expect(formatAddress('Doe, "J"', 'a@x.test')).toBe('"Doe, \\"J\\"" <a@x.test>');
    expect(formatAddress('', 'a@x.test')).toBe('a@x.test');
  });
});
