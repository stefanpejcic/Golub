<!-- SPDX-License-Identifier: GPL-3.0-or-later -->
<script lang="ts">
  import { i18n } from './lib/locale.svelte';
  import { isBareAddress, isValidRecipient, recipientAddress, splitRecipients } from './lib/compose';
  import { contacts, type Suggestion } from './lib/contacts.svelte';

  let { value = $bindable([]), label, testid }: { value: string[]; label: string; testid: string } = $props();
  const t = $derived(i18n.t);
  let draft = $state('');
  let input: HTMLInputElement;
  let active = $state(0);
  let open = $state(false);
  const listId = $props.id();
  const suggestions = $derived(
    open && contacts.loaded ? contacts.suggest(draft, value.map((v) => recipientAddress(v))) : [],
  );

  $effect(() => {
    void draft;
    active = 0;
  });

  function pick(sg: Suggestion) {
    value = [...value, ...sg.values.filter((v) => !value.includes(v))];
    draft = '';
    input.focus();
  }

  function commit() {
    const parts = splitRecipients(draft);
    if (parts.length) value = [...value, ...parts.filter((p) => !value.includes(p))];
    draft = '';
  }

  function onKey(e: KeyboardEvent) {
    if (suggestions.length) {
      if (e.key === 'ArrowDown' || e.key === 'ArrowUp') {
        e.preventDefault();
        active = (active + (e.key === 'ArrowDown' ? 1 : suggestions.length - 1)) % suggestions.length;
        return;
      }
      if (e.key === 'Enter' || e.key === 'Tab') {
        e.preventDefault();
        pick(suggestions[active]);
        return;
      }
      if (e.key === 'Escape') {
        e.preventDefault();
        open = false;
        return;
      }
    }
    if (e.key === 'Enter' || e.key === ',' || e.key === ';' || (e.key === 'Tab' && draft.trim())) {
      if (draft.trim()) {
        e.preventDefault();
        commit();
      }
    } else if (e.key === ' ' && isBareAddress(draft)) {
      e.preventDefault();
      commit();
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
      role="combobox"
      aria-expanded={suggestions.length > 0}
      aria-controls={listId}
      aria-autocomplete="list"
      aria-activedescendant={suggestions.length ? `${listId}-${active}` : undefined}
      onkeydown={onKey}
      onfocus={() => contacts.load().catch(() => undefined)}
      oninput={() => (open = true)}
      onblur={() => {
        open = false;
        commit();
      }}
      onpaste={onPaste}
      data-testid={`${testid}-input`}
    />
    {#if suggestions.length}
      <ul class="suggest" id={listId} role="listbox" data-testid="suggestions">
        {#each suggestions as sg, i (sg.kind + sg.detail + sg.label)}
          <!-- mousedown keeps focus in the input so blur does not commit the half-typed text first -->
          <li
            id={`${listId}-${i}`}
            role="option"
            aria-selected={i === active}
            class:active={i === active}
            onmousedown={(e) => (e.preventDefault(), pick(sg))}
            data-testid="suggestion"
          >
            {#if sg.kind === 'group'}
              <svg class="icon" viewBox="0 0 16 16" aria-hidden="true"><circle cx="6" cy="6" r="2.5" /><circle cx="11" cy="7" r="2" /><path d="M1.5 13.5c.5-2.5 2.3-3.5 4.5-3.5s4 1 4.5 3.5M10.5 10.5c2 0 3.5.8 4 3" /></svg>
              <span class="label">{sg.label}</span>
              <span class="detail">{t('contacts.members', { count: sg.values.length })}</span>
            {:else}
              <span class="label">{sg.label}</span>
              {#if sg.label !== sg.detail}<span class="detail">{sg.detail}</span>{/if}
            {/if}
          </li>
        {/each}
      </ul>
    {/if}
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
    position: relative;
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
    background: var(--color-input-bg);
    border: 1px solid transparent;
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
  .suggest {
    position: absolute;
    z-index: 20;
    inset-block-start: 100%;
    inset-inline-start: 0;
    inline-size: min(100%, 30rem);
    max-block-size: 18rem;
    margin: var(--space-1) 0 0;
    padding: var(--space-1);
    overflow-y: auto;
    list-style: none;
    background: var(--color-surface-raised);
    border: 1px solid var(--color-border);
    border-radius: var(--radius);
    box-shadow: var(--shadow);
  }
  .suggest li {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    padding: var(--space-2);
    border-radius: var(--radius);
    cursor: pointer;
  }
  .suggest li.active {
    background: var(--color-hover);
  }
  .suggest .label {
    flex: none;
    max-inline-size: 60%;
    overflow: hidden;
    font-weight: 500;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .suggest .detail {
    flex: 1;
    min-inline-size: 0;
    overflow: hidden;
    color: var(--color-text-muted);
    font-size: 0.85rem;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .chip .icon {
    inline-size: 0.8rem;
    block-size: 0.8rem;
  }
</style>
