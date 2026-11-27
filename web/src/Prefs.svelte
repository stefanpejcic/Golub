<!-- SPDX-License-Identifier: GPL-3.0-or-later -->
<script lang="ts">
  import { i18n, LOCALES } from './lib/locale.svelte';
  import { theme } from './lib/theme.svelte';
  import { session } from './lib/session.svelte';
  import { THEMES, type Theme } from './lib/prefs';
  import About from './About.svelte';

  const t = $derived(i18n.t);
  let about: About;

  function setTheme(v: Theme) {
    theme.set(v);
    session.savePrefs({ theme: v });
  }

  function setLocale(v: string) {
    i18n.set(v);
    session.savePrefs({ locale: i18n.locale });
  }
</script>

<div class="prefs">
  {#if session.status === 'authenticated' && session.info}
    <div class="account">
      <span class="email" data-testid="user-email" title={session.info.email}>{session.info.email}</span>
      <button type="button" class="link" onclick={() => session.logout()} data-testid="logout">
        {t('auth.logout')}
      </button>
    </div>
  {/if}
  <div class="row">
    <select
      class="input select"
      value={i18n.locale}
      onchange={(e) => setLocale(e.currentTarget.value)}
      aria-label={t('settings.language')}
      data-testid="lang-select"
    >
      {#each LOCALES as l (l)}
        <option value={l}>{t(`language.${l}`)}</option>
      {/each}
    </select>
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
  <button type="button" class="link about" onclick={() => about.open()} data-testid="about-open">
    <svg class="icon" viewBox="0 0 16 16" aria-hidden="true"><circle cx="8" cy="8" r="6.5" /><path d="M8 7.5V11M8 5v.01" /></svg>
    {t('about.open')}
  </button>
</div>

<About bind:this={about} />

<style>
  .prefs {
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
    padding: var(--space-3);
    border-block-start: 1px solid var(--color-border);
  }
  .account {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-2);
  }
  .email {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    font-weight: 500;
  }
  .row {
    display: flex;
    gap: var(--space-2);
  }
  .select {
    flex: 1;
    min-inline-size: 0;
    padding-block: 0.3rem;
  }
  .themes {
    display: flex;
    padding: 2px;
    border: 1px solid var(--color-input-border);
    border-radius: var(--radius);
    background: var(--color-surface-raised);
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
    border-radius: 4px;
    cursor: pointer;
  }
  .themes button[aria-checked='true'] {
    color: var(--color-text);
    background: var(--color-hover);
  }
  .link {
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
  .about {
    align-self: flex-start;
  }
</style>
