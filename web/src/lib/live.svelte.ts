// SPDX-License-Identifier: GPL-3.0-or-later

import { mail } from './mail.svelte';

const state = $state({ connected: false });
let source: EventSource | null = null;
let retry: ReturnType<typeof setTimeout> | undefined;
let current = '';

// Polling keeps working underneath; this only makes changes show up right away.
export const live = {
  get connected() {
    return state.connected;
  },
  watch(folder: string) {
    if (folder === current && source) return;
    this.stop();
    current = folder;
    if (typeof EventSource === 'undefined') return;
    const es = new EventSource(`/api/events?folder=${encodeURIComponent(folder)}`);
    source = es;
    const refresh = () => {
      if (folder === mail.folder) mail.refresh();
    };
    // ready catches up on mail that came in before IDLE started, but only when the folder really changed
    es.addEventListener('ready', async (e) => {
      state.connected = true;
      const { total, uidNext } = JSON.parse((e as MessageEvent).data) as { total?: number; uidNext?: number };
      if (total === undefined || uidNext === undefined || !(await mail.inSync(folder, total, uidNext))) refresh();
    });
    es.addEventListener('change', refresh);
    // A server-side "error" event means IDLE is unavailable; stay on polling and try again later.
    es.addEventListener('error', (e) => {
      if (e instanceof MessageEvent || es.readyState === EventSource.CLOSED) {
        state.connected = false;
        es.close();
        if (source === es) source = null;
        clearTimeout(retry);
        retry = setTimeout(() => current === folder && this.watch(folder), 60_000);
      }
    });
  },
  stop() {
    clearTimeout(retry);
    source?.close();
    source = null;
    current = '';
    state.connected = false;
  },
};
