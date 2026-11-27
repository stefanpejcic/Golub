// SPDX-License-Identifier: GPL-3.0-or-later
// The PGP keyring lives in a hidden folder of the user's own mailbox; private keys in it are always passphrase protected.

import { api } from './api';
import { dialog } from './dialog.svelte';
import { i18n } from './locale.svelte';
import * as pgp from './pgp';

const state = $state<{ keys: pgp.KeyInfo[]; loaded: boolean }>({ keys: [], loaded: false });
let loading: Promise<void> | null = null;

export const keys = {
  get all() {
    return state.keys;
  },
  get loaded() {
    return state.loaded;
  },
  get own() {
    return state.keys.filter((k) => k.private);
  },
  get others() {
    return state.keys.filter((k) => !k.private);
  },
  load(force = false): Promise<void> {
    if (state.loaded && !force) return Promise.resolve();
    loading ??= api<{ keys: pgp.KeyInfo[] }>('GET', '/api/pgp')
      .then((r) => {
        state.keys = r.keys;
        state.loaded = true;
      })
      .finally(() => (loading = null));
    return loading;
  },
  async save(next: pgp.KeyInfo[]) {
    const r = await api<{ keys: pgp.KeyInfo[] }>('PUT', '/api/pgp', { keys: next });
    state.keys = r.keys;
  },
  // add merges new keys in; a private key replaces the public copy of the same key
  async add(list: pgp.KeyInfo[]) {
    const byFp: Record<string, pgp.KeyInfo> = Object.fromEntries(state.keys.map((k) => [k.fingerprint, k]));
    for (const k of list) if (!byFp[k.fingerprint]?.private || k.private) byFp[k.fingerprint] = k;
    await this.save(Object.values(byFp));
  },
  async remove(fingerprint: string) {
    await this.save(state.keys.filter((k) => k.fingerprint !== fingerprint));
  },
  // forEmail finds the key to encrypt to: someone else's public key, or our own key for our own address
  forEmail(email: string): pgp.KeyInfo | undefined {
    const e = email.toLowerCase();
    return state.keys.find((k) => !k.private && k.emails.includes(e)) ?? state.keys.find((k) => k.private && k.emails.includes(e));
  },
};

// unlockKey asks for the passphrase until it is right or the user gives up
export async function unlockKey(key: pgp.KeyInfo): Promise<boolean> {
  if (pgp.isUnlocked(key.fingerprint)) return true;
  const t = i18n.t;
  let error: string | undefined;
  for (;;) {
    const pass = await dialog.prompt({
      title: t('pgp.unlock_title'),
      text: t('pgp.unlock_text', { key: key.userIds[0] ?? key.emails[0] ?? key.fingerprint }),
      confirm: t('pgp.unlock'),
      inputLabel: t('pgp.passphrase'),
      password: true,
      error,
    });
    if (pass === null) return false;
    try {
      await pgp.unlock(key, pass);
      return true;
    } catch {
      error = t('pgp.wrong_passphrase');
    }
  }
}
