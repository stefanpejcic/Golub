// SPDX-License-Identifier: GPL-3.0-or-later

import { api } from './api';

export interface Identity {
  id: string;
  name: string;
  email: string;
  organization?: string;
  replyTo?: string;
  bcc?: string;
  signature?: string;
  signatureHtml?: boolean;
  default?: boolean;
}

export interface Response {
  id: string;
  name: string;
  text: string;
}

export interface Settings {
  pageSize: number;
  sortBy: 'date' | 'arrival' | 'subject' | 'from' | 'size';
  sortDesc: boolean;
  prettyDates: boolean;
  refreshInterval: number;
  markRead: 'immediately' | 'delay' | 'never';
  markReadDelay: number;
  preferPlain: boolean;
  remoteImages: 'ask' | 'always';
  showAddresses: boolean;
  composeHtml: boolean;
  replyPosition: 'above' | 'below';
  forwardMode: 'inline' | 'attachment';
  signatureOnReply: boolean;
  spellCheck: boolean;
  collectRecipients: boolean;
  draftAutosave: number;
  deleteMode: 'trash' | 'permanent';
  emptyTrashOnLogout: boolean;
  specialFolders?: Record<string, string>;
  identities: Identity[];
  responses: Response[];
}

export const DEFAULTS: Settings = {
  pageSize: 50,
  sortBy: 'date',
  sortDesc: true,
  prettyDates: true,
  refreshInterval: 60,
  markRead: 'immediately',
  markReadDelay: 3,
  preferPlain: false,
  remoteImages: 'ask',
  showAddresses: true,
  composeHtml: true,
  replyPosition: 'above',
  forwardMode: 'inline',
  signatureOnReply: true,
  spellCheck: true,
  collectRecipients: true,
  draftAutosave: 60,
  deleteMode: 'trash',
  emptyTrashOnLogout: false,
  identities: [],
  responses: [],
};

const state = $state({ value: { ...DEFAULTS } as Settings, loaded: false });

export const settings = {
  get value(): Settings {
    return state.value;
  },
  get loaded() {
    return state.loaded;
  },
  async load() {
    try {
      state.value = await api<Settings>('GET', '/api/settings');
    } catch {
      state.value = { ...DEFAULTS };
    }
    state.loaded = true;
  },
  async save(next: Settings) {
    state.value = await api<Settings>('PUT', '/api/settings', next);
  },
  async patch(changes: Partial<Settings>) {
    await this.save({ ...$state.snapshot(state.value), ...changes } as Settings);
  },
};
