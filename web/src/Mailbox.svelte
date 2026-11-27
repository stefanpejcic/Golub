<!-- SPDX-License-Identifier: GPL-3.0-or-later -->
<script lang="ts">
  import { mail } from './lib/mail.svelte';
  import FolderList from './FolderList.svelte';
  import MessageList from './MessageList.svelte';
  import MessageView from './MessageView.svelte';

  $effect(() => mail.start());
</script>

<div class="mailbox">
  <aside>
    <FolderList />
  </aside>
  {#if mail.uid !== null}
    <MessageView />
  {:else}
    <MessageList />
  {/if}
</div>

<style>
  .mailbox {
    display: grid;
    grid-template-columns: 15rem 1fr;
    min-block-size: calc(100vh - 3.5rem);
  }
  aside {
    background: var(--color-surface);
    border-inline-end: 1px solid var(--color-border);
    padding-inline: var(--space-2);
  }
  @media (max-width: 40rem) {
    .mailbox {
      grid-template-columns: 1fr;
    }
    aside {
      border-inline-end: none;
      border-block-end: 1px solid var(--color-border);
    }
  }
</style>
