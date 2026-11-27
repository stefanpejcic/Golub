// SPDX-License-Identifier: GPL-3.0-or-later

const state = $state({ path: location.pathname, search: location.search });

addEventListener('popstate', () => {
  state.path = location.pathname;
  state.search = location.search;
});

let guard: (() => Promise<boolean>) | null = null;

export const route = {
  setGuard(fn: (() => Promise<boolean>) | null) {
    guard = fn;
  },
  get path() {
    return state.path;
  },
  get search() {
    return state.search;
  },
  get compose(): URLSearchParams | null {
    // A fresh, read-only snapshot per navigation; nothing mutates it.
    // eslint-disable-next-line svelte/prefer-svelte-reactivity
    return state.path === '/compose' ? new URLSearchParams(state.search) : null;
  },
  get contacts(): boolean {
    return state.path === '/contacts';
  },
  get settingsSection(): string | null {
    const m = /^\/settings(?:\/([\w-]+))?\/?$/.exec(state.path);
    return m ? (m[1] ?? 'general') : null;
  },
  async go(path: string): Promise<boolean> {
    const leavingCompose = state.path === '/compose' && !path.startsWith('/compose');
    if (leavingCompose && guard && !(await guard())) return false;
    if (path !== location.pathname + location.search) history.pushState(null, '', path);
    state.path = location.pathname;
    state.search = location.search;
    return true;
  },
  sync() {
    state.path = location.pathname;
    state.search = location.search;
  },
};
