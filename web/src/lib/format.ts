// SPDX-License-Identifier: GPL-3.0-or-later

export interface Address {
  name?: string;
  address: string;
}

const sameDay = (a: Date, b: Date) =>
  a.getFullYear() === b.getFullYear() && a.getMonth() === b.getMonth() && a.getDate() === b.getDate();

export function formatListDate(locale: string, iso: string, now = new Date()): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return '';
  if (sameDay(d, now)) return new Intl.DateTimeFormat(locale, { timeStyle: 'short' }).format(d);
  if (d.getFullYear() === now.getFullYear()) {
    return new Intl.DateTimeFormat(locale, { month: 'short', day: 'numeric' }).format(d);
  }
  return new Intl.DateTimeFormat(locale, { dateStyle: 'short' }).format(d);
}

export function formatFullDate(locale: string, iso: string): string {
  const d = new Date(iso);
  return Number.isNaN(d.getTime()) ? '' : new Intl.DateTimeFormat(locale, { dateStyle: 'full', timeStyle: 'short' }).format(d);
}

export function displayName(a: Address | undefined): string {
  if (!a) return '';
  return a.name?.trim() || a.address;
}

export function folderPath(name: string): string {
  return `/mail/${encodeURIComponent(name)}`;
}

export function folderFromPath(path: string): string | null {
  const m = /^\/mail\/(.+)$/.exec(path);
  if (!m) return null;
  try {
    return decodeURIComponent(m[1]);
  } catch {
    return null;
  }
}
