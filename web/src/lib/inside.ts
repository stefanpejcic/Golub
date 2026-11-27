// SPDX-License-Identifier: GPL-3.0-or-later

let mailbox: Promise<typeof import('../Mailbox.svelte')> | null = null;
let standalone: Promise<typeof import('../Standalone.svelte')> | null = null;

export function loadMailbox() {
  mailbox ??= import('../Mailbox.svelte');
  return mailbox;
}

export function loadStandalone() {
  standalone ??= import('../Standalone.svelte');
  return standalone;
}
