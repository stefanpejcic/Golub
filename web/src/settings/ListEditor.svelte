<!-- SPDX-License-Identifier: GPL-3.0-or-later -->
<script lang="ts">
  import { i18n } from '../lib/locale.svelte';
  import { settings, type Identity, type Response } from '../lib/settings.svelte';
  import { session } from '../lib/session.svelte';
  import { toast } from '../lib/toast.svelte';
  import { errorMessage } from '../lib/errors';
  import { ApiError } from '../lib/api';

  let { kind }: { kind: 'identities' | 'responses' } = $props();
  const t = $derived(i18n.t);

  type Item = Identity | Response;
  const items = $derived<Item[]>(settings.value[kind]);
  let selectedId = $state<string | null>(null);
  let draft = $state<Item | null>(null);

  $effect(() => {
    if (selectedId === null && items.length) select(items[0]);
  });

  function select(item: Item) {
    selectedId = item.id;
    draft = { ...item };
  }

  const newId = () => Math.random().toString(36).slice(2, 10);

  function add() {
    const item: Item =
      kind === 'identities'
        ? { id: newId(), name: '', email: session.info?.email ?? '', default: false }
        : { id: newId(), name: t('responses.new_name'), text: '' };
    selectedId = item.id;
    draft = item;
  }

  async function persist(next: Item[], done: string) {
    try {
      await settings.patch({ [kind]: next });
      toast.show(done);
    } catch (e) {
      const field = e instanceof ApiError ? e.code : 'unknown';
      toast.show(errorMessage(t, field === 'settings.invalid' ? 'settings.invalid' : field), true);
    }
  }

  function save(e: SubmitEvent) {
    e.preventDefault();
    if (!draft) return;
    const current = $state.snapshot(draft) as Item;
    let next = items.some((i) => i.id === current.id)
      ? items.map((i) => (i.id === current.id ? current : i))
      : [...items, current];
    if (kind === 'identities' && (current as Identity).default) {
      next = (next as Identity[]).map((i) => ({ ...i, default: i.id === current.id }));
    }
    persist(next, t('settings.saved'));
  }

  function remove() {
    if (!draft || !confirm(t('settings.delete_confirm'))) return;
    const id = draft.id;
    let next = items.filter((i) => i.id !== id);
    if (kind === 'identities' && next.length && !(next as Identity[]).some((i) => i.default)) {
      next = (next as Identity[]).map((i, n) => ({ ...i, default: n === 0 }));
    }
    selectedId = null;
    draft = null;
    persist(next, t('settings.deleted'));
  }

  const label = (i: Item) => (kind === 'identities' ? (i as Identity).name || (i as Identity).email : (i as Response).name);
</script>

<div class="editor">
  <div class="list">
    <ul>
      {#each items as item (item.id)}
        <li>
          <button type="button" aria-current={selectedId === item.id ? 'true' : undefined} onclick={() => select(item)} data-testid={`${kind}-item`}>
            <span>{label(item)}</span>
            {#if kind === 'identities'}
              <small>{(item as Identity).email}</small>
            {/if}
          </button>
        </li>
      {/each}
    </ul>
    <button class="btn" type="button" onclick={add} data-testid={`${kind}-add`}>{t('settings.add')}</button>
  </div>

  {#if draft}
    <form class="form" onsubmit={save}>
      {#if kind === 'identities'}
        {@const d = draft as Identity}
        <label class="field"><span>{t('identity.name')}</span><input class="input" bind:value={d.name} data-testid="identity-name" /></label>
        <label class="field"><span>{t('identity.email')}</span><input class="input" type="email" required bind:value={d.email} data-testid="identity-email" /></label>
        <label class="field"><span>{t('identity.organization')}</span><input class="input" bind:value={d.organization} /></label>
        <label class="field"><span>{t('identity.replyTo')}</span><input class="input" type="email" bind:value={d.replyTo} /></label>
        <label class="field"><span>{t('identity.bcc')}</span><input class="input" type="email" bind:value={d.bcc} /></label>
        <label class="field col"><span>{t('identity.signature')}</span><textarea class="input" rows="6" bind:value={d.signature} data-testid="identity-signature"></textarea></label>
        <label class="field check"><input type="checkbox" bind:checked={d.signatureHtml} /><span>{t('identity.signatureHtml')}</span></label>
        <label class="field check"><input type="checkbox" bind:checked={d.default} data-testid="identity-default" /><span>{t('identity.default')}</span></label>
      {:else}
        {@const d = draft as Response}
        <label class="field"><span>{t('responses.name')}</span><input class="input" required bind:value={d.name} data-testid="response-name" /></label>
        <label class="field col"><span>{t('responses.text')}</span><textarea class="input" rows="8" bind:value={d.text} data-testid="response-text"></textarea></label>
      {/if}
      <div class="buttons">
        <button class="btn btn-primary" type="submit" data-testid={`${kind}-save`}>{t('folders.save')}</button>
        {#if items.some((i) => i.id === draft!.id) && (kind === 'responses' || items.length > 1)}
          <button class="btn danger" type="button" onclick={remove} data-testid={`${kind}-delete`}>{t('folders.delete')}</button>
        {/if}
      </div>
    </form>
  {/if}
</div>

<style>
  .editor {
    display: grid;
    grid-template-columns: minmax(0, 16rem) minmax(0, 36rem);
    gap: var(--space-4);
    align-items: start;
  }
  .list {
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
    min-inline-size: 0;
  }
  ul {
    list-style: none;
    margin: 0;
    padding: 0;
    border: 1px solid var(--color-border);
    border-radius: var(--radius-lg);
    overflow: hidden;
  }
  li + li {
    border-block-start: 1px solid var(--color-border);
  }
  li button {
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    inline-size: 100%;
    padding: var(--space-2) var(--space-3);
    font: inherit;
    text-align: start;
    color: var(--color-text);
    background: none;
    border: none;
    cursor: pointer;
  }
  li button[aria-current='true'] {
    color: var(--color-accent);
    background: var(--color-hover);
  }
  li button span,
  small {
    max-inline-size: 100%;
    overflow-wrap: anywhere;
  }
  small {
    color: var(--color-text-muted);
  }
  .form {
    display: flex;
    flex-direction: column;
    gap: var(--space-3);
  }
  .field {
    display: grid;
    grid-template-columns: 9rem minmax(0, 1fr);
    align-items: center;
    gap: var(--space-3);
    font-weight: 500;
  }
  .field.col {
    align-items: start;
  }
  .field.check {
    display: flex;
    gap: var(--space-2);
    font-weight: 400;
  }
  textarea {
    resize: vertical;
    font-family: inherit;
  }
  .buttons {
    display: flex;
    gap: var(--space-2);
  }
  .danger {
    color: var(--color-danger);
  }
  @media (max-width: 56rem) {
    .editor,
    .field {
      grid-template-columns: 1fr;
    }
  }
</style>
