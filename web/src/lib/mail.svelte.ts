// SPDX-License-Identifier: GPL-3.0-or-later

import { api, ApiError } from './api';
import { folderPath, messagePath, parseMailPath, type Address } from './format';
import { session } from './session.svelte';

export interface Folder {
  name: string;
  delim: string;
  special?: string;
  unread: number;
  total: number;
  selectable: boolean;
}

export interface Summary {
  uid: number;
  subject: string;
  from: Address[];
  to: Address[];
  date: string;
  seen: boolean;
  flagged: boolean;
  answered: boolean;
  attachments: boolean;
  size: number;
}

export interface Attachment {
  id: string;
  filename: string;
  contentType: string;
  size: number;
  inline: boolean;
}

export interface MessageView {
  uid: number;
  folder: string;
  subject: string;
  from: Address[];
  to: Address[];
  cc: Address[];
  replyTo: Address[];
  date: string;
  seen: boolean;
  flagged: boolean;
  text: string;
  html: boolean;
  blockedImages: number;
  attachments: Attachment[];
}

interface MessagePage {
  total: number;
  page: number;
  size: number;
  messages: Summary[];
}

const PAGE_SIZE = 50;

const state = $state({
  folders: [] as Folder[],
  folder: 'INBOX',
  page: 1,
  total: 0,
  messages: [] as Summary[],
  loading: false,
  error: '',
  uid: null as number | null,
  message: null as MessageView | null,
  messageLoading: false,
  messageError: '',
});

let seq = 0;

function fail(e: unknown): string {
  if (e instanceof ApiError) {
    if (e.status === 401) session.expire();
    return e.code;
  }
  return 'unknown';
}

let messageSeq = 0;

function readLocation(): { folder: string; page: number; uid: number | null } {
  const parsed = parseMailPath(location.pathname);
  const page = Math.max(1, Number(new URLSearchParams(location.search).get('page')) || 1);
  return { folder: parsed?.folder ?? 'INBOX', uid: parsed?.uid ?? null, page };
}

function writeLocation(folder: string, page: number, replace: boolean, uid: number | null = null) {
  const base = uid === null ? folderPath(folder) : messagePath(folder, uid);
  const url = base + (page > 1 ? `?page=${page}` : '');
  if (url === location.pathname + location.search) return;
  if (replace) history.replaceState(null, '', url);
  else history.pushState(null, '', url);
}

export const mail = {
  get folders() {
    return state.folders;
  },
  get folder() {
    return state.folder;
  },
  get page() {
    return state.page;
  },
  get pageSize() {
    return PAGE_SIZE;
  },
  get total() {
    return state.total;
  },
  get messages() {
    return state.messages;
  },
  get loading() {
    return state.loading;
  },
  get error() {
    return state.error;
  },
  get uid() {
    return state.uid;
  },
  get message() {
    return state.message;
  },
  get messageLoading() {
    return state.messageLoading;
  },
  get messageError() {
    return state.messageError;
  },
  get current(): Folder | undefined {
    return state.folders.find((f) => f.name === state.folder);
  },

  async loadFolders() {
    try {
      state.folders = (await api<{ folders: Folder[] }>('GET', '/api/folders')).folders;
    } catch (e) {
      state.error = fail(e);
    }
  },

  async open(folder: string, page = 1, opts: { replace?: boolean; history?: boolean } = {}) {
    const mine = ++seq;
    if (opts.history !== false) {
      messageSeq++;
      state.uid = null;
      state.message = null;
    }
    state.folder = folder;
    state.page = page;
    state.loading = true;
    state.error = '';
    if (opts.history !== false) writeLocation(folder, page, opts.replace ?? false);
    try {
      const q = `folder=${encodeURIComponent(folder)}&page=${page}&size=${PAGE_SIZE}`;
      const res = await api<MessagePage>('GET', `/api/messages?${q}`);
      if (mine !== seq) return;
      state.total = res.total;
      state.messages = res.messages;
    } catch (e) {
      if (mine !== seq) return;
      state.messages = [];
      state.total = 0;
      state.error = fail(e);
    } finally {
      if (mine === seq) state.loading = false;
    }
  },

  async openMessage(uid: number, opts: { history?: boolean; replace?: boolean } = {}) {
    const mine = ++messageSeq;
    const folder = state.folder;
    state.uid = uid;
    state.message = null;
    state.messageError = '';
    state.messageLoading = true;
    if (opts.history !== false) writeLocation(folder, state.page, opts.replace ?? false, uid);
    try {
      const m = await api<MessageView>('GET', `/api/message?folder=${encodeURIComponent(folder)}&uid=${uid}`);
      if (mine !== messageSeq) return;
      state.message = m;
      const row = state.messages.find((r) => r.uid === uid);
      if (row && !row.seen) {
        row.seen = true;
        const f = state.folders.find((x) => x.name === folder);
        if (f && f.unread > 0) f.unread--;
      }
    } catch (e) {
      if (mine === messageSeq) state.messageError = fail(e);
    } finally {
      if (mine === messageSeq) state.messageLoading = false;
    }
  },

  closeMessage(opts: { history?: boolean } = {}) {
    messageSeq++;
    state.uid = null;
    state.message = null;
    state.messageError = '';
    if (opts.history !== false) writeLocation(state.folder, state.page, false);
  },

  // Updates rows, the open message and unread counters before the server answers.
  async setFlag(uids: number[], flag: 'seen' | 'flagged', value: boolean) {
    const folder = state.folder;
    const f = state.folders.find((x) => x.name === folder);
    for (const row of state.messages) {
      if (!uids.includes(row.uid)) continue;
      if (flag === 'seen' && row.seen !== value && f) f.unread += value ? -1 : 1;
      row[flag] = value;
    }
    if (state.message && uids.includes(state.message.uid)) state.message[flag] = value;
    try {
      await api('POST', '/api/messages/flag', { folder, uids, flag, value });
    } catch (e) {
      fail(e);
      await this.loadFolders();
      throw e;
    }
  },

  // Removes rows from the list and opens the next message, like a desktop client.
  async removeRows(uids: number[]) {
    const idx = state.messages.findIndex((r) => uids.includes(r.uid));
    state.messages = state.messages.filter((r) => !uids.includes(r.uid));
    state.total = Math.max(0, state.total - uids.length);
    if (state.uid !== null && uids.includes(state.uid)) {
      const next = state.messages[Math.min(idx, state.messages.length - 1)];
      if (next) this.openMessage(next.uid, { replace: true });
      else this.closeMessage();
    }
    await this.loadFolders();
  },

  async move(uids: number[], to: string) {
    await api('POST', '/api/messages/move', { folder: state.folder, uids, to });
    await this.removeRows(uids);
  },

  async remove(uids: number[]): Promise<boolean> {
    const res = await api<{ trashed: boolean }>('POST', '/api/messages/delete', { folder: state.folder, uids });
    await this.removeRows(uids);
    return res.trashed;
  },

  start() {
    const { folder, page, uid } = readLocation();
    this.loadFolders();
    this.open(folder, page, { replace: true, history: uid === null });
    if (uid !== null) this.openMessage(uid, { history: false });
    const onPop = () => {
      const loc = readLocation();
      if (loc.folder !== state.folder || loc.page !== state.page) {
        this.open(loc.folder, loc.page, { history: false });
      }
      if (loc.uid === null) this.closeMessage({ history: false });
      else if (loc.uid !== state.uid) this.openMessage(loc.uid, { history: false });
    };
    addEventListener('popstate', onPop);
    return () => removeEventListener('popstate', onPop);
  },
};
