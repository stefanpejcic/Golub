// SPDX-License-Identifier: GPL-3.0-or-later

export interface Ask {
  title: string;
  text?: string;
  confirm: string;
  danger?: boolean;
  // set for a prompt: the input starts with this value
  input?: string;
  inputLabel?: string;
  // a passphrase field, never autofilled or shown
  password?: boolean;
  // a line in the danger colour under the field, for a failed try
  error?: string;
}

interface Pending extends Ask {
  resolve: (value: string | null) => void;
}

const state = $state<{ current: Pending | null }>({ current: null });

function open(ask: Ask): Promise<string | null> {
  state.current?.resolve(null);
  return new Promise((resolve) => (state.current = { ...ask, resolve }));
}

export const dialog = {
  get current() {
    return state.current;
  },
  async confirm(ask: Omit<Ask, 'input' | 'inputLabel'>): Promise<boolean> {
    return (await open(ask)) !== null;
  },
  prompt(ask: Ask): Promise<string | null> {
    return open({ ...ask, input: ask.input ?? '' });
  },
  answer(value: string | null) {
    const c = state.current;
    state.current = null;
    c?.resolve(value);
  },
};
