// SPDX-License-Identifier: GPL-3.0-or-later
import { api, ApiError } from './api';

export interface Contact {
  id: string;
  name: string;
  emails: string[];
  phones?: string[];
  org?: string;
  notes?: string;
  groups?: string[];
}

export interface Collected {
  name?: string;
  email: string;
  count: number;
  last: string;
}

export interface Book {
  contacts: Contact[];
  groups: string[];
  collected: Collected[];
}

export interface Suggestion {
  kind: 'contact' | 'collected' | 'group';
  label: string;
  detail: string;
  // what goes into the recipient field; a group expands to all its members
  values: string[];
}

const empty = (): Book => ({ contacts: [], groups: [], collected: [] });
const state = $state<{ book: Book; loaded: boolean }>({ book: empty(), loaded: false });
let loading: Promise<void> | null = null;

// Quotes display names that would otherwise split the address list.
export function formatAddress(name: string | undefined, email: string): string {
  if (!name) return email;
  return /[",;<>@()]/.test(name) ? `"${name.replace(/["\\]/g, '\\$&')}" <${email}>` : `${name} <${email}>`;
}

function fold(s: string): string {
  return s.normalize('NFD').replace(/\p{M}/gu, '').toLowerCase().replace(/đ/g, 'd');
}

export function suggest(book: Book, query: string, exclude: string[] = [], limit = 8): Suggestion[] {
  const q = fold(query.trim());
  if (!q) return [];
  const lowered = exclude.map((e) => e.toLowerCase());
  const taken = { has: (e: string) => lowered.includes(e) };
  const matches = (...fields: (string | undefined)[]) =>
    fields.some((f) => !!f && (fold(f).startsWith(q) || fold(f).split(/[\s.,@_-]+/).some((w) => w.startsWith(q))));
  const out: Suggestion[] = [];
  for (const g of book.groups) {
    if (!matches(g)) continue;
    const values = book.contacts.filter((c) => c.groups?.includes(g) && c.emails.length).map((c) => formatAddress(c.name, c.emails[0]));
    if (values.length) out.push({ kind: 'group', label: g, detail: String(values.length), values });
  }
  for (const c of book.contacts) {
    for (const e of c.emails) {
      if (!taken.has(e.toLowerCase()) && matches(c.name, e)) out.push({ kind: 'contact', label: c.name || e, detail: e, values: [formatAddress(c.name, e)] });
    }
  }
  const collected = [...book.collected].sort((a, b) => b.count - a.count);
  for (const c of collected) {
    if (!taken.has(c.email.toLowerCase()) && matches(c.name, c.email)) out.push({ kind: 'collected', label: c.name || c.email, detail: c.email, values: [formatAddress(c.name, c.email)] });
  }
  return out.slice(0, limit);
}

export const contacts = {
  get book() {
    return state.book;
  },
  get loaded() {
    return state.loaded;
  },
  load(force = false): Promise<void> {
    if (state.loaded && !force) return Promise.resolve();
    loading ??= api<Book>('GET', '/api/contacts')
      .then((b) => {
        state.book = b;
        state.loaded = true;
      })
      .finally(() => (loading = null));
    return loading;
  },
  async save(next: Book): Promise<Book> {
    const saved = await api<Book>('PUT', '/api/contacts', next);
    state.book = saved;
    return saved;
  },
  async importFile(file: File): Promise<{ added: number; updated: number }> {
    const res = await fetch('/api/contacts/import', {
      method: 'POST',
      credentials: 'same-origin',
      headers: { 'Content-Type': 'text/vcard' },
      body: file,
    });
    const data = await res.json().catch(() => ({}));
    if (!res.ok) throw new ApiError(data?.error ?? 'unknown', res.status);
    state.book = data.book;
    return { added: data.added, updated: data.updated };
  },
  suggest(query: string, exclude: string[] = []) {
    return suggest(state.book, query, exclude);
  },
};
