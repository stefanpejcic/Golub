// SPDX-License-Identifier: GPL-3.0-or-later

import { api, ApiError } from './api';
import { i18n } from './locale.svelte';
import { theme } from './theme.svelte';
import { LANG_COOKIE, THEME_COOKIE, parseTheme, readCookie } from './prefs';

export interface SessionInfo {
  email: string;
  mode: 'password' | 'sso';
  locale: string;
  theme: string;
}

type Status = 'loading' | 'anonymous' | 'authenticated';

const state = $state<{ status: Status; info: SessionInfo | null }>({ status: 'loading', info: null });

// Preference cookies change instantly, so they win over a session that may still lag behind.
function adopt(info: SessionInfo) {
  state.info = info;
  state.status = 'authenticated';
  i18n.set(readCookie(document.cookie, LANG_COOKIE) || info.locale);
  theme.set(parseTheme(readCookie(document.cookie, THEME_COOKIE) || info.theme));
}

export const session = {
  get status() {
    return state.status;
  },
  get info() {
    return state.info;
  },
  async load() {
    try {
      adopt(await api<SessionInfo>('GET', '/api/session'));
    } catch (e) {
      if (!(e instanceof ApiError) || e.status !== 401) console.error(e);
      state.status = 'anonymous';
    }
  },
  async login(email: string, password: string) {
    adopt(await api<SessionInfo>('POST', '/api/login', { email, password, locale: i18n.locale, theme: theme.value }));
  },
  async logout(beforeLogout?: () => Promise<void>) {
    await beforeLogout?.().catch(() => undefined);
    await api('POST', '/api/logout').catch(() => undefined);
    state.info = null;
    state.status = 'anonymous';
    history.replaceState(null, '', '/');
  },
  expire() {
    state.info = null;
    state.status = 'anonymous';
  },
  async savePrefs(prefs: { locale?: string; theme?: string }) {
    if (state.status !== 'authenticated') return;
    await api('POST', '/api/prefs', prefs).catch(() => undefined);
  },
};
