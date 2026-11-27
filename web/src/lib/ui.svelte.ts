// SPDX-License-Identifier: GPL-3.0-or-later

const state = $state({ drawer: false });

export const ui = {
  get drawer() {
    return state.drawer;
  },
  toggle() {
    state.drawer = !state.drawer;
  },
  close() {
    state.drawer = false;
  },
};
