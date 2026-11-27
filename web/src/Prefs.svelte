<!-- SPDX-License-Identifier: GPL-3.0-or-later -->
<script lang="ts">
  import { i18n } from './lib/locale.svelte';
  import { theme } from './lib/theme.svelte';
  import { session } from './lib/session.svelte';
  import { settings } from './lib/settings.svelte';
  import { mail } from './lib/mail.svelte';
  import { api } from './lib/api';

  const t = $derived(i18n.t);
  import { THEMES, type Theme } from './lib/prefs';

  async function emptyTrash() {
    if (!settings.value.emptyTrashOnLogout) return;
    const name = settings.value.specialFolders?.trash || mail.folders.find((f) => f.special === 'trash')?.name;
    if (name) await api('POST', '/api/folders/empty', { name });
  }

  function setTheme(value: Theme) {
    theme.set(value);
    session.savePrefs({ theme: value });
  }
</script>

<div class="prefs">
  {#if session.status === 'authenticated'}
    <button type="button" class="link" onclick={() => session.logout(emptyTrash)} data-testid="logout">
      <svg class="icon" viewBox="0 0 16 16" aria-hidden="true"><path d="M6 2.5H3.5v11H6M10 5l3 3-3 3M13 8H6" /></svg>
      {t('auth.logout')}
    </button>
  {/if}
  <div class="themes" role="radiogroup" aria-label={t('settings.theme')} data-testid="theme-select">
    {#each THEMES as v (v)}
      <button
        type="button"
        role="radio"
        aria-checked={theme.value === v}
        aria-label={t(`settings.theme.${v}`)}
        title={t(`settings.theme.${v}`)}
        data-value={v}
        onclick={() => setTheme(v)}
      >
        <svg class="icon" viewBox="0 0 16 16" aria-hidden="true">
          {#if v === 'system'}
            <rect x="2" y="3" width="12" height="8" rx="1" /><path d="M6 14h4M8 11v3" />
          {:else if v === 'light'}
            <circle cx="8" cy="8" r="3" /><path
              d="M8 1.5v1.5M8 13v1.5M1.5 8H3M13 8h1.5M3.4 3.4l1 1M11.6 11.6l1 1M3.4 12.6l1-1M11.6 4.4l1-1"
            />
          {:else}
            <path d="M13.5 9.5A5.5 5.5 0 0 1 6.5 2.5a5.5 5.5 0 1 0 7 7Z" />
          {/if}
        </svg>
      </button>
    {/each}
  </div>
</div>

<style>
  .prefs {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-3);
    padding: var(--space-3);
  }
  .themes {
    display: flex;
    margin-inline-start: auto;
    padding: 2px;
    border-radius: 999px;
    background: var(--color-hover);
  }
  .themes button {
    display: grid;
    place-items: center;
    inline-size: 1.75rem;
    block-size: 1.6rem;
    padding: 0;
    color: var(--color-text-muted);
    background: none;
    border: none;
    border-radius: 999px;
    cursor: pointer;
  }
  .themes button[aria-checked='true'] {
    color: var(--color-text);
    background: var(--color-bg);
    box-shadow: 0 1px 2px rgb(0 0 0 / 0.12);
  }
  .link {
    text-decoration: none;
    display: inline-flex;
    align-items: center;
    gap: var(--space-1);
    padding: 0;
    font: inherit;
    color: var(--color-text-muted);
    background: none;
    border: none;
    cursor: pointer;
    white-space: nowrap;
  }
  .link:hover {
    color: var(--color-text);
  }
</style>
