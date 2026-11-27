<!-- SPDX-License-Identifier: GPL-3.0-or-later -->
<script lang="ts">
  import '@fontsource-variable/inter';
  import { mail } from './lib/mail.svelte';
  import { settings } from './lib/settings.svelte';
  import MessageView from './MessageView.svelte';
  import Toast from './Toast.svelte';
  import Dialog from './Dialog.svelte';

  const params = new URLSearchParams(location.search);
  const folder = params.get('folder') ?? 'INBOX';
  const uid = Number(params.get('uid')) || 0;
  const print = params.get('print') === '1';
  let printed = false;

  $effect(() => {
    settings.load().then(() => mail.openStandalone(folder, uid));
  });

  function loaded() {
    if (print && !printed) {
      printed = true;
      setTimeout(() => window.print(), 300);
    }
  }

  $effect(() => {
    if (print && mail.message && !mail.message.html) loaded();
  });
</script>

<div class="standalone">
  <MessageView standalone onloaded={loaded} />
</div>
<Toast />
<Dialog />

<style>
  .standalone {
    display: flex;
    flex-direction: column;
    min-block-size: 100vh;
    max-inline-size: 72rem;
    margin-inline: auto;
  }
</style>
