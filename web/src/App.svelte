<!-- SPDX-License-Identifier: GPL-3.0-or-later -->
<script lang="ts">
  import { session } from './lib/session.svelte';
  import Login from './Login.svelte';
  import Mailbox from './Mailbox.svelte';

  session.load();
</script>

{#if session.status === 'anonymous'}
  <Login />
{:else if session.status === 'authenticated'}
  <Mailbox />
{/if}
