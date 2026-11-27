// SPDX-License-Identifier: GPL-3.0-or-later

import { api, ApiError } from './api';
import { folderFromPath, folderPath, type Address } from './format';
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
});

let seq = 0;

function fail(e: unknown): string {
  if (e instanceof ApiError) {
    if (e.status === 401) session.expire();
    return e.code;
  }
  return 'unknown';
}

function readLocation(): { folder: string; page: number } {
  const folder = folderFromPath(location.pathname) ?? 'INBOX';
  const page = Math.max(1, Number(new URLSearchParams(location.search).get('page')) || 1);
  return { folder, page };
}

function writeLocation(folder: string, page: number, replace: boolean) {
  const url = folderPath(folder) + (page > 1 ? `?page=${page}` : '');
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

  start() {
    const { folder, page } = readLocation();
    this.loadFolders();
    this.open(folder, page, { replace: true });
    const onPop = () => {
      const loc = readLocation();
      this.open(loc.folder, loc.page, { history: false });
    };
    addEventListener('popstate', onPop);
    return () => removeEventListener('popstate', onPop);
  },
};
