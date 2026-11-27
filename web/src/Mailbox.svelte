<!-- SPDX-License-Identifier: GPL-3.0-or-later -->
<script lang="ts">
  import { i18n } from './lib/locale.svelte';
  import { mail } from './lib/mail.svelte';
  import FolderList from './FolderList.svelte';
  import MessageList from './MessageList.svelte';
  import MessageView from './MessageView.svelte';
  import Prefs from './Prefs.svelte';
  import Logo from './Logo.svelte';

  const t = $derived(i18n.t);

  $effect(() => mail.start());
</script>

<div class="mailbox">
  <aside>
    <div class="brand">
      <Logo size={28} />
      <span>{t('app.name')}</span>
    </div>
    <div class="folders">
      <FolderList />
    </div>
    <Prefs />
  </aside>
  <main>
    {#if mail.uid !== null}
      <MessageView />
    {:else}
      <MessageList />
    {/if}
  </main>
</div>

<style>
  .mailbox {
    display: grid;
    grid-template-columns: var(--sidebar-width) minmax(0, 1fr);
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
  .folders {
    flex: 1;
    min-block-size: 0;
    overflow-y: auto;
    padding: var(--space-2);
  }
  main {
    display: flex;
    flex-direction: column;
    min-block-size: 0;
    overflow-y: auto;
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
