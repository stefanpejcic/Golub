<!-- SPDX-License-Identifier: GPL-3.0-or-later -->
<script lang="ts">
  import { i18n } from './lib/locale.svelte';
  import { mail } from './lib/mail.svelte';
  import FolderList from './FolderList.svelte';
  import MessageList from './MessageList.svelte';
  import MessageView from './MessageView.svelte';
  import Prefs from './Prefs.svelte';
  import Toast from './Toast.svelte';
  import Logo from './Logo.svelte';
  import { session } from './lib/session.svelte';

  const t = $derived(i18n.t);

  $effect(() => mail.start());
</script>

<div class="mailbox" class:has-message={mail.uid !== null}>
  <aside>
    <div class="brand">
      <Logo size={28} />
      <span>{t('app.name')}</span>
    </div>
    <div class="account">
      <svg class="icon" viewBox="0 0 16 16" aria-hidden="true"><circle cx="8" cy="5.5" r="2.5" /><path d="M3 13.5a5 5 0 0 1 10 0" /></svg>
      <span data-testid="user-email" title={session.info?.email}>{session.info?.email}</span>
    </div>
    <div class="folders">
      <FolderList />
    </div>
    <Prefs />
  </aside>
  <main class="list-pane">
    <MessageList />
  </main>
  <section class="preview-pane">
    {#if mail.uid !== null}
      <MessageView />
    {:else}
      <div class="placeholder" data-testid="no-message">
        <Logo size={56} />
        <p>{t('message.none_selected')}</p>
      </div>
    {/if}
  </section>
</div>

<Toast />

<style>
  .mailbox {
    display: grid;
    grid-template-columns: var(--sidebar-width) minmax(20rem, 26rem) minmax(0, 1fr);
    block-size: 100vh;
  }
  aside {
    display: flex;
    flex-direction: column;
    min-block-size: 0;
    background: var(--color-surface);
    border-inline-end: 1px solid var(--color-border);
  }
  .brand {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    flex: none;
    block-size: var(--header-height);
    padding-inline: var(--space-3);
    font-size: 1.05rem;
    font-weight: 600;
    border-block-end: 1px solid var(--color-border);
  }
  .account {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    padding-block: var(--space-3) var(--space-1);
    padding-inline: var(--space-3);
    color: var(--color-text-muted);
    font-weight: 500;
  }
  .account span {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .folders {
    flex: 1;
    min-block-size: 0;
    overflow-y: auto;
    padding: var(--space-2);
  }
  .list-pane {
    min-block-size: 0;
    border-inline-end: 1px solid var(--color-border);
  }
  .preview-pane {
    display: flex;
    flex-direction: column;
    min-block-size: 0;
    overflow-y: auto;
  }
  .placeholder {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: var(--space-3);
    color: var(--color-text-muted);
    opacity: 0.8;
  }
  .placeholder :global(.logo) {
    color: var(--color-border);
  }
  /* Too narrow for three columns: show either the list or the open message. */
  @media (max-width: 64rem) {
    .mailbox {
      grid-template-columns: var(--sidebar-width) minmax(0, 1fr);
    }
    .list-pane {
      border-inline-end: none;
    }
    .mailbox:not(.has-message) .preview-pane,
    .mailbox.has-message .list-pane {
      display: none;
    }
  }
  @media (max-width: 48rem) {
    .mailbox {
      grid-template-columns: 1fr;
      grid-template-rows: auto 1fr;
      block-size: auto;
      min-block-size: 100vh;
    }
    aside {
      border-inline-end: none;
      border-block-end: 1px solid var(--color-border);
    }
    .folders {
      max-block-size: 40vh;
    }
  }
</style>
