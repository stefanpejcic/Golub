<!-- SPDX-License-Identifier: GPL-3.0-or-later -->
<script lang="ts">
  import { MediaQuery } from 'svelte/reactivity';
  import { i18n } from './lib/locale.svelte';
  import { theme } from './lib/theme.svelte';
  import { session } from './lib/session.svelte';
  import { settings } from './lib/settings.svelte';
  import { mail } from './lib/mail.svelte';
  import { api } from './lib/api';

  const t = $derived(i18n.t);
  const systemDark = new MediaQuery('(prefers-color-scheme: dark)');
  const effective = $derived(theme.value === 'system' ? (systemDark.current ? 'dark' : 'light') : theme.value);
  const next = $derived(effective === 'dark' ? 'light' : 'dark');

  async function emptyTrash() {
    if (!settings.value.emptyTrashOnLogout) return;
    const name = settings.value.specialFolders?.trash || mail.folders.find((f) => f.special === 'trash')?.name;
    if (name) await api('POST', '/api/folders/empty', { name });
  }

  function toggle() {
    // Read once: `next` is derived from the theme and flips as soon as it is set.
    const value = next;
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
  <button
    type="button"
    class="btn btn-icon theme"
    onclick={toggle}
    aria-label={t(`theme.to_${next}`)}
    title={t(`theme.to_${next}`)}
    data-testid="theme-toggle"
    data-next={next}
  >
    <svg class="icon" viewBox="0 0 16 16" aria-hidden="true">
      {#if next === 'dark'}
        <path d="M13.5 9.5A5.5 5.5 0 0 1 6.5 2.5a5.5 5.5 0 1 0 7 7Z" />
      {:else}
        <circle cx="8" cy="8" r="3" /><path
          d="M8 1.5v1.5M8 13v1.5M1.5 8H3M13 8h1.5M3.4 3.4l1 1M11.6 11.6l1 1M3.4 12.6l1-1M11.6 4.4l1-1"
        />
      {/if}
    </svg>
  </button>
</div>

<style>
  .prefs {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-3);
    padding: var(--space-3);
    border-block-start: 1px solid var(--color-border);
  }
  .theme {
    margin-inline-start: auto;
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
