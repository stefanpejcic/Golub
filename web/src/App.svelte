<!-- SPDX-License-Identifier: GPL-3.0-or-later -->
<script lang="ts">
  import { i18n, LOCALES } from './lib/locale.svelte';
  import { theme } from './lib/theme.svelte';
  import { THEMES, type Theme } from './lib/prefs';

  const t = $derived(i18n.t);
</script>

<header class="bar">
  <span class="brand">{t('app.name')}</span>
  <div class="controls">
    <label>
      <span>{t('settings.theme')}</span>
      <select
        value={theme.value}
        onchange={(e) => theme.set(e.currentTarget.value as Theme)}
        data-testid="theme-select"
      >
        {#each THEMES as v (v)}
          <option value={v}>{t(`settings.theme.${v}`)}</option>
        {/each}
      </select>
    </label>
    <label>
      <span>{t('settings.language')}</span>
      <select
        value={i18n.locale}
        onchange={(e) => i18n.set(e.currentTarget.value)}
        data-testid="lang-select"
      >
        {#each LOCALES as l (l)}
          <option value={l}>{t(`language.${l}`)}</option>
        {/each}
      </select>
    </label>
  </div>
</header>

<main class="empty">
  <p>{t('shell.placeholder')}</p>
</main>

<style>
  .bar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-3);
    padding-block: var(--space-2);
    padding-inline: var(--space-3);
    background: var(--color-surface);
    border-block-end: 1px solid var(--color-border);
  }
  .brand {
    font-weight: 600;
  }
  .controls {
    display: flex;
    gap: var(--space-3);
  }
  label {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    color: var(--color-text-muted);
  }
  select {
    background: var(--color-surface-raised);
    color: var(--color-text);
    border: 1px solid var(--color-border);
    border-radius: var(--radius);
    padding-block: var(--space-1);
    padding-inline: var(--space-2);
  }
  .empty {
    padding: var(--space-4);
    color: var(--color-text-muted);
  }
</style>
