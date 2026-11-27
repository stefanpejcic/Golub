<!-- SPDX-License-Identifier: GPL-3.0-or-later -->
<script lang="ts">
  import { i18n } from './lib/locale.svelte';
  import { isValidRecipient, splitRecipients } from './lib/compose';

  let { value = $bindable([]), label, testid }: { value: string[]; label: string; testid: string } = $props();
  const t = $derived(i18n.t);
  let draft = $state('');
  let input: HTMLInputElement;

  function commit() {
    const parts = splitRecipients(draft);
    if (parts.length) value = [...value, ...parts.filter((p) => !value.includes(p))];
    draft = '';
  }

  function onKey(e: KeyboardEvent) {
    if (e.key === 'Enter' || e.key === ',' || e.key === ';' || (e.key === 'Tab' && draft.trim())) {
      if (draft.trim()) {
        e.preventDefault();
        commit();
      }
    } else if (e.key === 'Backspace' && !draft && value.length) {
      value = value.slice(0, -1);
    }
  }

  function onPaste(e: ClipboardEvent) {
    const text = e.clipboardData?.getData('text') ?? '';
    if (/[,;\n]/.test(text)) {
      e.preventDefault();
      draft += text;
      commit();
    }
  }
</script>

<div class="field" data-testid={testid}>
  <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
  <span class="label" onclick={() => input.focus()}>{label}</span>
  <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
  <div class="box" onclick={() => input.focus()}>
    {#each value as r, i (r)}
      <span class="chip" class:invalid={!isValidRecipient(r)} title={isValidRecipient(r) ? r : t('compose.invalid_address')} data-testid="recipient">
        {r}
        <button type="button" aria-label={t('compose.remove', { name: r })} onclick={() => (value = value.filter((_, j) => j !== i))}>
          <svg class="icon" viewBox="0 0 16 16" aria-hidden="true"><path d="m5 5 6 6M11 5l-6 6" /></svg>
        </button>
      </span>
    {/each}
    <input
      bind:this={input}
      bind:value={draft}
      type="text"
      autocomplete="email"
      aria-label={label}
      onkeydown={onKey}
      onblur={commit}
      onpaste={onPaste}
      data-testid={`${testid}-input`}
    />
  </div>
</div>

<style>
  .field {
    display: flex;
    align-items: flex-start;
    gap: var(--space-2);
    padding-block: var(--space-1);
    border-block-end: 1px solid var(--color-border);
  }
  .label {
    flex: none;
    inline-size: 4.5rem;
    padding-block: 0.4rem;
    color: var(--color-text-muted);
    cursor: text;
  }
  .box {
    flex: 1;
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: var(--space-1);
    min-block-size: 2rem;
    cursor: text;
  }
  input {
    flex: 1;
    min-inline-size: 10rem;
    padding-block: 0.4rem;
    font: inherit;
    color: var(--color-text);
    background: none;
    border: none;
    outline: none;
  }
  .chip {
    display: inline-flex;
    align-items: center;
    gap: 2px;
    max-inline-size: 100%;
    padding-block: 2px;
    padding-inline: var(--space-2) 2px;
    background: var(--color-hover);
    border: 1px solid var(--color-border);
    border-radius: 999px;
    overflow-wrap: anywhere;
  }
  .chip.invalid {
    color: var(--color-danger);
    border-color: var(--color-danger);
  }
  .chip button {
    display: grid;
    place-items: center;
    padding: 2px;
    color: inherit;
    background: none;
    border: none;
    border-radius: 999px;
    cursor: pointer;
  }
  .chip .icon {
    inline-size: 0.8rem;
    block-size: 0.8rem;
  }
</style>
