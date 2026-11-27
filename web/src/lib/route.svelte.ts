// SPDX-License-Identifier: GPL-3.0-or-later

const state = $state({ path: location.pathname });

addEventListener('popstate', () => (state.path = location.pathname));

export const route = {
  get path() {
    return state.path;
  },
  get settingsSection(): string | null {
    const m = /^\/settings(?:\/([\w-]+))?\/?$/.exec(state.path);
    return m ? (m[1] ?? 'general') : null;
  },
  go(path: string) {
    if (path !== location.pathname + location.search) history.pushState(null, '', path);
    state.path = location.pathname;
  },
  // Called after code that edits history directly.
  sync() {
    state.path = location.pathname;
  },
};
