// SPDX-License-Identifier: GPL-3.0-or-later

import { api, ApiError } from './api';
import { folderPath, messagePath, parseMailPath, type Address } from './format';
import { session } from './session.svelte';
import { settings } from './settings.svelte';
import { route } from './route.svelte';

export interface Folder {
  name: string;
  delim: string;
  special?: string;
  unread: number;
  total: number;
  selectable: boolean;
  subscribed: boolean;
  size?: number;
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
  trackers: number;
  attachments: Attachment[];
}

export interface Moved {
  folder: string;
  uids: number[];
  trashed?: boolean;
}

interface MessagePage {
  total: number;
  page: number;
  size: number;
  messages: Summary[];
}


const state = $state({
  folders: [] as Folder[],
  folder: 'INBOX',
  page: 1,
  total: 0,
  messages: [] as Summary[],
  loading: true,
  error: '',
  uid: null as number | null,
  message: null as MessageView | null,
  messageLoading: false,
  messageError: '',
  query: '',
  quota: null as { used: number; limit: number } | null,
  selected: [] as number[],
  anchor: null as number | null,
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
// Bumped by every local change so a refresh started earlier cannot undo it with stale data.
let mutations = 0;

function sortParams(): string {
  return `&sort=${settings.value.sortBy}&order=${settings.value.sortDesc ? 'desc' : 'asc'}`;
}
let markTimer: ReturnType<typeof setTimeout> | undefined;

function readLocation(): { folder: string; page: number; uid: number | null; query: string } {
  const parsed = parseMailPath(location.pathname);
  const params = new URLSearchParams(location.search);
  const page = Math.max(1, Number(params.get('page')) || 1);
  return { folder: parsed?.folder ?? 'INBOX', uid: parsed?.uid ?? null, page, query: params.get('q') ?? '' };
}

function writeLocation(folder: string, page: number, replace: boolean, uid: number | null = null) {
  // Background mailbox loads must not pull the user out of settings.
  if (location.pathname.startsWith('/settings') || location.pathname === '/compose') return;
  const base = uid === null ? folderPath(folder) : messagePath(folder, uid);
  const params = [page > 1 ? `page=${page}` : '', state.query ? `q=${encodeURIComponent(state.query)}` : '']
    .filter(Boolean)
    .join('&');
  const url = base + (params ? `?${params}` : '');
  if (url === location.pathname + location.search) return;
  if (replace) history.replaceState(null, '', url);
  else history.pushState(null, '', url);
  route.sync();
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
    return settings.value.pageSize;
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
  get query() {
    return state.query;
  },
  search(query: string) {
    const q = query.trim();
    if (q === state.query) return;
    state.query = q;
    this.open(state.folder, 1);
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
  get selected() {
    return state.selected;
  },
  isSelected(uid: number) {
    return state.selected.includes(uid);
  },
  select(uid: number, mode: 'single' | 'toggle' | 'range') {
    if (mode === 'range' && state.anchor !== null) {
      const uids = state.messages.map((m) => m.uid);
      const [a, b] = [uids.indexOf(state.anchor), uids.indexOf(uid)].sort((x, y) => x - y);
      if (a >= 0) {
        state.selected = uids.slice(a, b + 1);
        return;
      }
    }
    if (mode === 'toggle') {
      state.selected = state.selected.includes(uid) ? state.selected.filter((u) => u !== uid) : [...state.selected, uid];
    } else {
      state.selected = [uid];
    }
    state.anchor = uid;
  },
  selectAll() {
    state.selected = state.messages.map((m) => m.uid);
  },
  selectWhere(pick: (m: Summary) => boolean) {
    state.selected = state.messages.filter(pick).map((m) => m.uid);
  },
  invertSelection() {
    state.selected = state.messages.filter((m) => !state.selected.includes(m.uid)).map((m) => m.uid);
  },
  clearSelection() {
    state.selected = state.uid !== null ? [state.uid] : [];
  },
  get current(): Folder | undefined {
    return state.folders.find((f) => f.name === state.folder);
  },

  get quota() {
    return state.quota;
  },

  async loadFolders() {
    try {
      state.folders = (await api<{ folders: Folder[] }>('GET', '/api/folders')).folders;
    } catch (e) {
      state.error = fail(e);
    }
    api<{ supported: boolean; used?: number; limit?: number }>('GET', '/api/quota')
      .then((q) => (state.quota = q.supported ? { used: q.used!, limit: q.limit! } : null))
      .catch(() => undefined);
  },

  async open(folder: string, page = 1, opts: { replace?: boolean; history?: boolean } = {}) {
    const mine = ++seq;
    if (folder !== state.folder) state.query = '';
    if (opts.history !== false) {
      messageSeq++;
      state.uid = null;
      state.message = null;
    }
    state.selected = [];
    state.anchor = null;
    state.folder = folder;
    state.page = page;
    state.loading = true;
    state.error = '';
    if (opts.history !== false) writeLocation(folder, page, opts.replace ?? false);
    try {
      const search = state.query ? `&q=${encodeURIComponent(state.query)}` : '';
      const q = `folder=${encodeURIComponent(folder)}&page=${page}&size=${settings.value.pageSize}${search}${sortParams()}`;
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
    state.selected = [uid];
    state.anchor = uid;
    state.message = null;
    state.messageError = '';
    state.messageLoading = true;
    if (opts.history !== false) writeLocation(folder, state.page, opts.replace ?? false, uid);
    try {
      const mode = settings.value.markRead;
      const seen = mode === 'immediately' ? '' : '&seen=0';
      const m = await api<MessageView>('GET', `/api/message?folder=${encodeURIComponent(folder)}&uid=${uid}${seen}`);
      if (mine !== messageSeq) return;
      state.message = m;
      const row = state.messages.find((r) => r.uid === uid);
      if (mode === 'immediately' && row && !row.seen) {
        row.seen = true;
        const f = state.folders.find((x) => x.name === folder);
        if (f && f.unread > 0) f.unread--;
      }
      if (mode === 'delay' && !m.seen) {
        clearTimeout(markTimer);
        markTimer = setTimeout(() => {
          if (state.uid === uid && state.folder === folder && state.message && !state.message.seen) {
            this.setFlag([uid], 'seen', true).catch(() => undefined);
          }
        }, settings.value.markReadDelay * 1000);
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
    state.selected = [];
    state.message = null;
    state.messageError = '';
    if (opts.history !== false) writeLocation(state.folder, state.page, false);
  },

  async setFlag(uids: number[], flag: 'seen' | 'flagged', value: boolean) {
    mutations++;
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

  async removeRows(uids: number[]) {
    mutations++;
    const idx = state.messages.findIndex((r) => uids.includes(r.uid));
    state.messages = state.messages.filter((r) => !uids.includes(r.uid));
    state.selected = state.selected.filter((u) => !uids.includes(u));
    state.total = Math.max(0, state.total - uids.length);
    if (state.uid !== null && uids.includes(state.uid)) {
      const next = state.messages[Math.min(idx, state.messages.length - 1)];
      if (next) this.openMessage(next.uid, { replace: true });
      else this.closeMessage();
    }
    await this.loadFolders();
  },

  async move(uids: number[], to: string): Promise<Moved> {
    const res = await api<Moved>('POST', '/api/messages/move', { folder: state.folder, uids, to });
    await this.removeRows(uids);
    return res;
  },

  async spam(uids: number[]): Promise<Moved> {
    const res = await api<Moved>('POST', '/api/messages/spam', { folder: state.folder, uids });
    await this.removeRows(uids);
    return res;
  },

  async undoMove(moved: Moved, back: string) {
    if (!moved.folder || !moved.uids.length) return;
    await api('POST', '/api/messages/move', { folder: moved.folder, uids: moved.uids, to: back });
    await this.refresh();
  },

  async copy(uids: number[], to: string) {
    await api('POST', '/api/messages/copy', { folder: state.folder, uids, to });
    await this.loadFolders();
  },

  async openStandalone(folder: string, uid: number) {
    state.folder = folder;
    await Promise.all([this.loadFolders(), this.openMessage(uid, { history: false })]);
  },

  async archive(uids: number[]): Promise<Moved> {
    const res = await api<Moved>('POST', '/api/messages/archive', { folder: state.folder, uids });
    await this.removeRows(uids);
    return res;
  },

  async remove(uids: number[]): Promise<Moved> {
    const res = await api<Moved>('POST', '/api/messages/delete', { folder: state.folder, uids });
    await this.removeRows(uids);
    return res;
  },

  async createFolder(name: string) {
    await api('POST', '/api/folders/create', { name });
    await this.loadFolders();
  },
  async folderOp(op: 'rename' | 'delete' | 'empty' | 'subscribe', name: string, extra: Record<string, unknown> = {}) {
    await api('POST', `/api/folders/${op}`, { name, ...extra });
    await this.loadFolders();
  },

  async refresh() {
    if (state.loading) return;
    const started = mutations;
    await this.loadFolders();
    const search = state.query ? `&q=${encodeURIComponent(state.query)}` : '';
    const q = `folder=${encodeURIComponent(state.folder)}&page=${state.page}&size=${settings.value.pageSize}${search}${sortParams()}`;
    try {
      const res = await api<MessagePage>('GET', `/api/messages?${q}`);
      if (started !== mutations) return;
      state.total = res.total;
      state.messages = res.messages;
    } catch (e) {
      fail(e);
    }
  },

  start() {
    const { folder, page, uid, query } = readLocation();
    state.folder = folder;
    state.query = query;
    this.loadFolders();
    this.open(folder, page, { replace: true, history: uid === null });
    if (uid !== null) this.openMessage(uid, { history: false });
    const onPop = () => {
      if (!location.pathname.startsWith('/mail')) return;
      const loc = readLocation();
      if (loc.folder !== state.folder || loc.page !== state.page || loc.query !== state.query) {
        state.folder = loc.folder;
        state.query = loc.query;
        this.open(loc.folder, loc.page, { history: false });
      }
      if (loc.uid === null) this.closeMessage({ history: false });
      else if (loc.uid !== state.uid) this.openMessage(loc.uid, { history: false });
    };
    addEventListener('popstate', onPop);
    const every = settings.value.refreshInterval;
    const timer = every > 0 ? setInterval(() => document.visibilityState === 'visible' && this.refresh(), every * 1000) : undefined;
    return () => {
      removeEventListener('popstate', onPop);
      clearInterval(timer);
    };
  },
};
