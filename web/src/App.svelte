<!-- SPDX-License-Identifier: GPL-3.0-or-later -->
<script lang="ts">
  import { session } from './lib/session.svelte';
  import Login from './Login.svelte';
  import Mailbox from './Mailbox.svelte';
  import Standalone from './Standalone.svelte';

  session.load();
</script>

{#if session.status === 'anonymous'}
  <Login />
{:else if session.status === 'authenticated'}
  {#if location.pathname === '/message'}
    <Standalone />
  {:else}
    <Mailbox />
  {/if}
{/if}
