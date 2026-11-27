// SPDX-License-Identifier: GPL-3.0-or-later

import { ApiError } from './api';
import { errorMessage } from './errors';
import { i18n } from './locale.svelte';
import { mail, type Moved } from './mail.svelte';
import { toast } from './toast.svelte';


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

function fail(e: unknown) {
  toast.show(errorMessage(i18n.t, e instanceof ApiError ? e.code : 'unknown'), true);
}

async function moveWithUndo(work: () => Promise<Moved>, message: (m: Moved) => string) {
  const from = mail.folder;
  try {
    const m = await work();
    const undo = m.folder && m.uids.length ? { label: i18n.t('toast.undo'), run: () => mail.undoMove(m, from).catch(fail) } : null;
    toast.show(message(m), false, undo);
  } catch (e) {
    fail(e);
  }
}

export function removeMessages(uids: number[]) {
  uids = mail.withThreads(uids);
  return moveWithUndo(
    () => mail.remove(uids),
    (m) => {
      if (uids.length > 1) return i18n.t(m.trashed ? 'toast.trashed_many' : 'toast.deleted_many', { count: uids.length });
      return i18n.t(m.trashed ? 'toast.trashed' : 'toast.deleted');
    },
  );
}

export function archiveMessages(uids: number[]) {
  uids = mail.withThreads(uids);
  return moveWithUndo(
    () => mail.archive(uids),
    (m) => i18n.t('toast.archived', { folder: folderLabel(m.folder) }),
  );
}

export function spamMessages(uids: number[]) {
  uids = mail.withThreads(uids);
  return moveWithUndo(
    () => mail.spam(uids),
    (m) => i18n.t('toast.moved', { folder: folderLabel(m.folder) }),
  );
}

export function moveMessages(uids: number[], to: string) {
  uids = mail.withThreads(uids);
  return moveWithUndo(
    () => mail.move(uids, to),
    () => i18n.t('toast.moved', { folder: folderLabel(to) }),
  );
}

export function flagMessages(uids: number[], flag: 'seen' | 'flagged', value: boolean) {
  uids = mail.withThreads(uids);
  return run(() => mail.setFlag(uids, flag, value));
}

export function copyMessages(uids: number[], to: string) {
  uids = mail.withThreads(uids);
  return run(async () => {
    await mail.copy(uids, to);
    return i18n.t('toast.copied', { folder: folderLabel(to) });
  });
}
