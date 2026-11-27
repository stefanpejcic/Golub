<!-- SPDX-License-Identifier: GPL-3.0-or-later -->
<script lang="ts">
  import { i18n } from './lib/locale.svelte';
  import { mail } from './lib/mail.svelte';
  import { settings, type Settings } from './lib/settings.svelte';
  import { toast } from './lib/toast.svelte';
  import Menu from './Menu.svelte';

  const t = $derived(i18n.t);
  let refreshing = $state(false);
  let sortBy = $state<Settings['sortBy']>('date');
  let sortDesc = $state(true);

  $effect(() => {
    sortBy = settings.value.sortBy;
    sortDesc = settings.value.sortDesc;
  });

  async function refresh() {
    refreshing = true;
    try {
      await mail.refresh();
    } finally {
      refreshing = false;
    }
  }

  async function applySort(close: () => void) {
    close();
    try {
      await settings.patch({ sortBy, sortDesc });
      await mail.open(mail.folder, 1);
    } catch {
      toast.show(t('errors.unknown'), true);
    }
  }

  const keys: Settings['sortBy'][] = ['date', 'arrival', 'subject', 'from', 'size'];
</script>

<div class="toolbar" role="toolbar" aria-label={t('list.toolbar')}>
  <Menu testid="select">
    {#snippet label()}
      <svg class="icon" viewBox="0 0 16 16" aria-hidden="true"><rect x="2.5" y="2.5" width="11" height="11" rx="2" /><path d="m5.5 8 2 2 3-4" /></svg>
      {t('select.label')}
    {/snippet}
    {#snippet children(close)}
      <button class="item" role="menuitem" onclick={() => (mail.selectAll(), close())} data-testid="select-all">{t('select.all')}</button>
      <button class="item" role="menuitem" onclick={() => (mail.selectWhere((m) => !m.seen), close())} data-testid="select-unread">{t('select.unread')}</button>
      <button class="item" role="menuitem" onclick={() => (mail.selectWhere((m) => m.flagged), close())} data-testid="select-flagged">{t('select.flagged')}</button>
      <button class="item" role="menuitem" onclick={() => (mail.invertSelection(), close())} data-testid="select-invert">{t('select.invert')}</button>
      <div class="sep"></div>
      <button class="item" role="menuitem" onclick={() => (mail.selectWhere(() => false), close())} data-testid="select-none">{t('select.none')}</button>
    {/snippet}
  </Menu>

  <Menu testid="list-options">
    {#snippet label()}
      <svg class="icon" viewBox="0 0 16 16" aria-hidden="true"><path d="M2.5 4h11M4.5 8h7M6.5 12h3" /></svg>
      {t('options.label')}
    {/snippet}
    {#snippet children(close)}
      <form class="options" onsubmit={(e) => (e.preventDefault(), applySort(close))}>
        <label>
          <span>{t('options.sort_by')}</span>
          <select class="input" bind:value={sortBy} data-testid="options-sort">
            {#each keys as k (k)}
              <option value={k}>{t(`options.sort.${k}`)}</option>
            {/each}
          </select>
        </label>
        <fieldset>
          <legend>{t('options.order')}</legend>
          <label class="radio"><input type="radio" bind:group={sortDesc} value={true} data-testid="options-desc" />{t('options.desc')}</label>
          <label class="radio"><input type="radio" bind:group={sortDesc} value={false} data-testid="options-asc" />{t('options.asc')}</label>
        </fieldset>
        <button class="btn btn-primary" type="submit" data-testid="options-save">{t('folders.save')}</button>
      </form>
    {/snippet}
  </Menu>

  <button type="button" class="refresh" onclick={refresh} disabled={refreshing} data-testid="refresh">
    <svg class="icon" class:spin={refreshing} viewBox="0 0 16 16" aria-hidden="true">
      <path d="M13.5 8a5.5 5.5 0 1 1-1.6-3.9M13.5 2.5v3h-3" />
    </svg>
    {t('list.refresh')}
  </button>
</div>

<style>
  .toolbar {
    display: flex;
    align-items: center;
    gap: var(--space-1);
    flex: none;
    padding-block: var(--space-1);
    padding-inline: var(--space-2);
    border-block-end: 1px solid var(--color-border);
  }
  .refresh {
    display: inline-flex;
    align-items: center;
    gap: var(--space-1);
    margin-inline-start: auto;
    padding-block: 0.25rem;
    padding-inline: var(--space-2);
    font: inherit;
    font-weight: 500;
    color: var(--color-text);
    background: none;
    border: none;
    border-radius: var(--radius);
    cursor: pointer;
  }
  .refresh:hover {
    background: var(--color-hover);
  }
  .spin {
    animation: spin 0.8s linear infinite;
  }
  @keyframes spin {
    to {
      transform: rotate(360deg);
    }
  }
  .options {
    display: flex;
    flex-direction: column;
    gap: var(--space-3);
    padding: var(--space-2);
    min-inline-size: 14rem;
  }
  .options label {
    display: flex;
    flex-direction: column;
    gap: var(--space-1);
    font-weight: 500;
  }
  fieldset {
    margin: 0;
    padding: 0;
    border: none;
    display: flex;
    flex-direction: column;
    gap: var(--space-1);
  }
  legend {
    margin-block-end: var(--space-1);
    font-weight: 500;
  }
  .options .radio {
    flex-direction: row;
    align-items: center;
    gap: var(--space-2);
    font-weight: 400;
  }
</style>
