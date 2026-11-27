<!-- SPDX-License-Identifier: GPL-3.0-or-later -->
<script lang="ts">
  import { i18n } from './lib/locale.svelte';
  import { api } from './lib/api';
  import Logo from './Logo.svelte';

  const t = $derived(i18n.t);
  let dialog: HTMLDialogElement;
  let version = $state('');

  const credits = [
    { name: 'Go', url: 'https://go.dev' },
    { name: 'Svelte', url: 'https://svelte.dev' },
    { name: 'go-imap', url: 'https://github.com/emersion/go-imap' },
    { name: 'enmime', url: 'https://github.com/jhillyerd/enmime' },
    { name: 'bluemonday', url: 'https://github.com/microcosm-cc/bluemonday' },
    { name: 'Inter', url: 'https://rsms.me/inter/' },
  ];

  export async function open() {
    dialog.showModal();
    if (!version) {
      version = (await api<{ version: string }>('GET', '/healthz').catch(() => ({ version: '' }))).version;
    }
  }
</script>

<dialog bind:this={dialog} aria-labelledby="about-title" data-testid="about">
  <div class="head">
    <Logo size={48} />
    <div>
      <h2 id="about-title">{t('app.name')}</h2>
      {#if version}
        <p class="muted" data-testid="about-version">{t('about.version', { version })}</p>
      {/if}
    </div>
  </div>
  <p>{t('about.description')}</p>
  <p class="muted">{t('about.copyright', { year: String(new Date().getFullYear()) })}</p>
  <p class="muted">
    {t('about.license')}
    <a href="https://github.com/stefanpejcic/Golub" target="_blank" rel="noopener noreferrer">{t('about.source')}</a>
  </p>
  <h3>{t('about.credits')}</h3>
  <ul>
    {#each credits as c (c.name)}
      <li><a href={c.url} target="_blank" rel="noopener noreferrer">{c.name}</a></li>
    {/each}
  </ul>
  <form method="dialog">
    <button class="btn btn-primary" data-testid="about-close">{t('about.close')}</button>
  </form>
</dialog>

<style>
  dialog {
    inline-size: min(100% - 2rem, 26rem);
    padding: var(--space-4);
    color: var(--color-text);
    background: var(--color-surface-raised);
    border: 1px solid var(--color-border);
    border-radius: 8px;
    box-shadow: var(--shadow);
  }
  dialog::backdrop {
    background: rgb(0 0 0 / 0.4);
  }
  .head {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    margin-block-end: var(--space-3);
  }
  h2 {
    margin: 0;
    font-size: 1.25rem;
  }
  h3 {
    margin-block: var(--space-3) var(--space-1);
    font-size: 0.9rem;
  }
  p {
    margin-block: 0 var(--space-2);
    line-height: 1.5;
  }
  .muted {
    color: var(--color-text-muted);
  }
  a {
    color: var(--color-link);
  }
  ul {
    display: flex;
    flex-wrap: wrap;
    gap: var(--space-1) var(--space-3);
    margin: 0;
    padding: 0;
    list-style: none;
  }
  form {
    display: flex;
    justify-content: flex-end;
    margin-block-start: var(--space-4);
  }
</style>
