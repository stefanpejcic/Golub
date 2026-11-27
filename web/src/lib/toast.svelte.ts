// SPDX-License-Identifier: GPL-3.0-or-later

const state = $state({ text: '', error: false, id: 0 });
let timer: ReturnType<typeof setTimeout> | undefined;

export const toast = {
  get text() {
    return state.text;
  },
  get error() {
    return state.error;
  },
  show(text: string, error = false) {
    state.text = text;
    state.error = error;
    state.id++;
    clearTimeout(timer);
    timer = setTimeout(() => (state.text = ''), 4000);
  },
};
