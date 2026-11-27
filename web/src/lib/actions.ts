// SPDX-License-Identifier: GPL-3.0-or-later

import { ApiError } from './api';
import { errorMessage } from './errors';
import { i18n } from './locale.svelte';
import { mail } from './mail.svelte';
import { toast } from './toast.svelte';

// Shared by the toolbar, the bulk bar, row hover buttons and swipes so they all report the same way.

export function folderLabel(name: string): string {
  const f = mail.folders.find((x) => x.name === name);
  return f?.special ? i18n.t(`folder.${f.special}`) : name;
}

async function run(action: () => Promise<string | void>) {
  try {
    const msg = await action();
    if (msg) toast.show(msg);
  } catch (e) {
    toast.show(errorMessage(i18n.t, e instanceof ApiError ? e.code : 'unknown'), true);
  }
}

export function removeMessages(uids: number[]) {
  return run(async () => {
    const trashed = await mail.remove(uids);
    if (uids.length > 1) return i18n.t(trashed ? 'toast.trashed_many' : 'toast.deleted_many', { count: uids.length });
    return i18n.t(trashed ? 'toast.trashed' : 'toast.deleted');
  });
}

export function archiveMessages(uids: number[]) {
  return run(async () => {
    const folder = await mail.archive(uids);
    return i18n.t('toast.archived', { folder: folderLabel(folder) });
  });
}

export function moveMessages(uids: number[], to: string) {
  return run(async () => {
    await mail.move(uids, to);
    return i18n.t('toast.moved', { folder: folderLabel(to) });
  });
}

export function flagMessages(uids: number[], flag: 'seen' | 'flagged', value: boolean) {
  return run(() => mail.setFlag(uids, flag, value));
}
