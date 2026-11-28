<!-- SPDX-License-Identifier: GPL-3.0-or-later -->
<script lang="ts">
  import { i18n } from './lib/locale.svelte';
  import { mail, type Filters } from './lib/mail.svelte';
  import Menu from './Menu.svelte';

  const t = $derived(i18n.t);
  // Edits stay local until Search is pressed, and reset whenever the applied filters change.
  let draft = $derived<Filters>({ ...mail.filters });
  const active = $derived(Object.values(mail.filters).filter(Boolean).length);

  const checks = ['body', 'all', 'unread', 'flagged', 'attachments'] as const;
</script>

<Menu testid="search-filters">
  {#snippet label()}
    <svg class="icon" viewBox="0 0 16 16" aria-hidden="true"><path d="M2.5 3.5h11l-4.5 5v4l-2 1v-5z" /></svg>
    <span class="sr-only">{t('search.filters')}</span>
    {#if active}<span class="badge" data-testid="search-filters-count">{active}</span>{/if}
  {/snippet}
  {#snippet children(close)}
    <form
      class="filters"
      onsubmit={(e) => {
        e.preventDefault();
        close();
        mail.setFilters(draft);
      }}
    >
      {#each checks as k (k)}
        <label class="check"><input type="checkbox" bind:checked={draft[k]} data-testid={`filter-${k}`} />{t(`search.filter.${k}`)}</label>
      {/each}
      <div class="dates">
        <label><span>{t('search.filter.since')}</span><input class="input" type="date" bind:value={draft.since} data-testid="filter-since" /></label>
        <label><span>{t('search.filter.before')}</span><input class="input" type="date" bind:value={draft.before} data-testid="filter-before" /></label>
      </div>
      <div class="buttons">
        <button
          type="button"
          class="btn"
          onclick={() => {
            close();
            mail.clearFilters();
          }}
          data-testid="filter-clear">{t('search.filter.clear')}</button
        >
        <button type="submit" class="btn btn-primary" data-testid="filter-apply">{t('search.filter.apply')}</button>
      </div>
    </form>
  {/snippet}
</Menu>

<style>
  .filters {
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
    inline-size: 17rem;
    padding: var(--space-2);
  }
  .check {
    display: flex;
    align-items: center;
    gap: var(--space-2);
  }
  /* one date per row, since a date field needs more than half the panel in many locales */
  .dates {
    display: grid;
    grid-template-columns: auto minmax(0, 1fr);
    align-items: center;
    gap: var(--space-2);
  }
  .dates label {
    display: contents;
    font-size: 0.85rem;
    color: var(--color-text-muted);
  }
  .dates input {
    inline-size: 100%;
    min-inline-size: 0;
  }
  .buttons {
    display: flex;
    justify-content: space-between;
    gap: var(--space-2);
  }
  .badge {
    min-inline-size: 1.1rem;
    padding: 0 0.3rem;
    font-size: 0.7rem;
    color: var(--color-primary-text);
    background: var(--color-primary);
    border-radius: 999px;
  }
  .sr-only {
    position: absolute;
    inline-size: 1px;
    block-size: 1px;
    overflow: hidden;
    clip-path: inset(50%);
  }
</style>
