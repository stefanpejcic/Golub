<!-- SPDX-License-Identifier: GPL-3.0-or-later -->
<script lang="ts">
  import { i18n, LOCALES } from './lib/locale.svelte';
  import { theme } from './lib/theme.svelte';
  import { session } from './lib/session.svelte';
  import { THEMES, type Theme } from './lib/prefs';
  import Login from './Login.svelte';
  import Mailbox from './Mailbox.svelte';

  const t = $derived(i18n.t);

  session.load();

  function setTheme(v: Theme) {
    theme.set(v);
    session.savePrefs({ theme: v });
  }

  function setLocale(v: string) {
    i18n.set(v);
    session.savePrefs({ locale: i18n.locale });
  }
</script>

<header class="bar">
  <span class="brand">{t('app.name')}</span>
  <div class="controls">
    <label>
      <span>{t('settings.theme')}</span>
      <select
        value={theme.value}
        onchange={(e) => setTheme(e.currentTarget.value as Theme)}
        data-testid="theme-select"
      >
        {#each THEMES as v (v)}
          <option value={v}>{t(`settings.theme.${v}`)}</option>
        {/each}
      </select>
    </label>
    <label>
      <span>{t('settings.language')}</span>
      <select value={i18n.locale} onchange={(e) => setLocale(e.currentTarget.value)} data-testid="lang-select">
        {#each LOCALES as l (l)}
          <option value={l}>{t(`language.${l}`)}</option>
        {/each}
      </select>
    </label>
    {#if session.status === 'authenticated' && session.info}
      <span class="user" data-testid="user-email">{session.info.email}</span>
      <button type="button" class="link" onclick={() => session.logout()} data-testid="logout">
        {t('auth.logout')}
      </button>
    {/if}
  </div>
</header>

{#if session.status === 'loading'}
  <main class="empty" aria-busy="true"></main>
{:else if session.status === 'anonymous'}
  <main>
    <Login />
  </main>
{:else}
  <main>
    <Mailbox />
  </main>
{/if}

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
    align-items: center;
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
  .user {
    color: var(--color-text-muted);
  }
  .link {
    background: none;
    border: none;
    color: var(--color-accent);
    font: inherit;
    cursor: pointer;
    padding: 0;
  }
  .empty {
    padding: var(--space-4);
    color: var(--color-text-muted);
  }
</style>
