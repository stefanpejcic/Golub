// SPDX-License-Identifier: GPL-3.0-or-later

export interface ToastAction {
  label: string;
  run: () => void;
}

const state = $state({ text: '', error: false, id: 0, action: null as ToastAction | null });
let timer: ReturnType<typeof setTimeout> | undefined;

export const toast = {
  get text() {
    return state.text;
  },
  get error() {
    return state.error;
  },
  get action() {
    return state.action;
  },
  // A toast with an action (such as Undo) stays a little longer so there is time to use it.
  show(text: string, error = false, action: ToastAction | null = null) {
    state.text = text;
    state.error = error;
    state.action = action;
    state.id++;
    clearTimeout(timer);
    timer = setTimeout(() => {
      state.text = '';
      state.action = null;
    }, action ? 6000 : 4000);
  },
  dismiss() {
    clearTimeout(timer);
    state.text = '';
    state.action = null;
  },
};
