// SPDX-License-Identifier: GPL-3.0-or-later

const state = $state({ path: location.pathname, search: location.search });

addEventListener('popstate', () => {
  state.path = location.pathname;
  state.search = location.search;
});

export const route = {
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
  get settingsSection(): string | null {
    const m = /^\/settings(?:\/([\w-]+))?\/?$/.exec(state.path);
    return m ? (m[1] ?? 'general') : null;
  },
  go(path: string) {
    if (path !== location.pathname + location.search) history.pushState(null, '', path);
    state.path = location.pathname;
    state.search = location.search;
  },
  // Called after code that edits history directly.
  sync() {
    state.path = location.pathname;
    state.search = location.search;
  },
};
