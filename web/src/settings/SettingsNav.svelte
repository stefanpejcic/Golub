<!-- SPDX-License-Identifier: GPL-3.0-or-later -->
<script lang="ts">
  import { i18n } from '../lib/locale.svelte';
  import { route } from '../lib/route.svelte';
  import { mail } from '../lib/mail.svelte';
  import { folderPath } from '../lib/format';

  const t = $derived(i18n.t);
  const groups = [
    { title: 'settings.group.preferences', items: ['general', 'mailbox', 'display', 'compose', 'special'] },
    { title: 'settings.group.account', items: ['folders', 'identities', 'responses'] },
    { title: 'settings.group.golub', items: ['about'] },
  ];

  function back(e: MouseEvent) {
    if (e.metaKey || e.ctrlKey || e.shiftKey || e.button !== 0) return;
    e.preventDefault();
    route.go(folderPath(mail.folder));
    mail.open(mail.folder, 1, { replace: true });
  }

  function go(e: MouseEvent, path: string) {
    if (e.metaKey || e.ctrlKey || e.shiftKey || e.button !== 0) return;
    e.preventDefault();
    route.go(path);
  }
</script>

<nav aria-label={t('settings.title')}>
  <a class="back" href={folderPath(mail.folder)} onclick={back} data-testid="settings-back">
    <svg class="icon" viewBox="0 0 16 16" aria-hidden="true"><path d="M10 3 5 8l5 5" /></svg>
    {t('settings.back')}
  </a>
  {#each groups as g (g.title)}
    <h2>{t(g.title)}</h2>
    <ul>
      {#each g.items as item (item)}
        <li>
          <a
            href={`/settings/${item}`}
            aria-current={route.settingsSection === item ? 'page' : undefined}
            onclick={(e) => go(e, `/settings/${item}`)}
            data-testid="settings-nav"
            data-section={item}>{t(`settings.section.${item}`)}</a
          >
        </li>
      {/each}
    </ul>
  {/each}
</nav>

<style>
  nav {
    display: flex;
    flex-direction: column;
    gap: 1px;
  }
  h2 {
    margin-block: var(--space-3) var(--space-1);
    padding-inline: var(--space-2);
    font-size: 0.75rem;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.04em;
    color: var(--color-text-muted);
  }
  ul {
    list-style: none;
    margin: 0;
    padding: 0;
    display: flex;
    flex-direction: column;
    gap: 1px;
  }
  a {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    padding: var(--space-2);
    color: var(--color-text);
    text-decoration: none;
    border-radius: 4px;
  }
  a:hover {
    background: var(--color-hover);
  }
  a[aria-current='page'] {
    color: var(--color-accent);
    font-weight: 500;
    background: var(--color-selected);
    box-shadow: var(--shadow), 0 0 0 1px var(--color-selected-ring);
  }
  .back {
    color: var(--color-text-muted);
  }
</style>
