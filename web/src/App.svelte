<!-- SPDX-License-Identifier: GPL-3.0-or-later -->
<script lang="ts">
  import { session } from './lib/session.svelte';
  import Login from './Login.svelte';
  import { loadMailbox, loadStandalone } from './lib/inside';

  session.load();
</script>

{#if session.status === 'anonymous'}
  <Login />
{:else if session.status === 'authenticated'}
  {#if location.pathname === '/message'}
    {#await loadStandalone() then mod}
      <mod.default />
    {/await}
  {:else}
    {#await loadMailbox() then mod}
      <mod.default />
    {/await}
  {/if}
{/if}
