<!-- SPDX-License-Identifier: GPL-3.0-or-later -->
<script lang="ts">
  import { i18n } from './lib/locale.svelte';
  import { dialog } from './lib/dialog.svelte';

  const t = $derived(i18n.t);
  let el = $state<HTMLDialogElement>();
  let value = $state('');
  const ask = $derived(dialog.current);

  $effect(() => {
    if (!el) return;
    if (ask && !el.open) {
      value = ask.input ?? '';
      el.showModal();
    } else if (!ask && el.open) {
      el.close();
    }
  });

  function submit(e: SubmitEvent) {
    e.preventDefault();
    dialog.answer(ask?.input !== undefined ? value : '');
  }
</script>

<dialog bind:this={el} class="modal" aria-labelledby="dialog-title" oncancel={(e) => (e.preventDefault(), dialog.answer(null))} data-testid="dialog">
  {#if ask}
    <form onsubmit={submit}>
      <h2 id="dialog-title">{ask.title}</h2>
      {#if ask.text}<p>{ask.text}</p>{/if}
      {#if ask.input !== undefined}
        <!-- svelte-ignore a11y_autofocus -->
        <input class="input" bind:value aria-label={ask.inputLabel ?? ask.title} autofocus data-testid="dialog-input" />
      {/if}
      <div class="modal-buttons">
        <button class="btn" type="button" onclick={() => dialog.answer(null)} data-testid="dialog-cancel">{t('dialog.cancel')}</button>
        <button class="btn {ask.danger ? 'btn-danger' : 'btn-primary'}" type="submit" data-testid="dialog-confirm">{ask.confirm}</button>
      </div>
    </form>
  {/if}
</dialog>

<style>
  .input {
    margin-block-end: var(--space-4);
  }
</style>
