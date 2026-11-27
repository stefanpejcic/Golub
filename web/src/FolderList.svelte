<!-- SPDX-License-Identifier: GPL-3.0-or-later -->
<script lang="ts">
  import { i18n } from './lib/locale.svelte';
  import { mail, type Folder } from './lib/mail.svelte';
  import { folderPath } from './lib/format';

  const t = $derived(i18n.t);

  function label(f: Folder): string {
    if (f.special) return t(`folder.${f.special}`);
    return f.delim ? f.name.split(f.delim).at(-1)! : f.name;
  }

  function depth(f: Folder): number {
    return f.special || !f.delim ? 0 : f.name.split(f.delim).length - 1;
  }

  function go(e: MouseEvent, f: Folder) {
    if (e.metaKey || e.ctrlKey || e.shiftKey || e.button !== 0) return;
    e.preventDefault();
    mail.open(f.name);
  }
</script>

<nav aria-label={t('folders.label')}>
  <ul>
    {#each mail.folders as f (f.name)}
      <li data-depth={Math.min(depth(f), 4)}>
        {#if f.selectable}
          <a
            href={folderPath(f.name)}
            aria-current={f.name === mail.folder ? 'page' : undefined}
            onclick={(e) => go(e, f)}
            data-testid="folder"
            data-folder={f.name}
          >
            <span class="name">{label(f)}</span>
            {#if f.unread > 0}
              <span class="badge" aria-label={t('folders.unread', { count: f.unread })}>{f.unread}</span>
            {/if}
          </a>
        {:else}
          <span class="group">{label(f)}</span>
        {/if}
      </li>
    {/each}
  </ul>
</nav>

<style>
  nav {
    padding-block: var(--space-2);
  }
  ul {
    list-style: none;
    margin: 0;
    padding: 0;
  }
  li[data-depth='1'] {
    padding-inline-start: var(--space-3);
  }
  li[data-depth='2'] {
    padding-inline-start: calc(2 * var(--space-3));
  }
  li[data-depth='3'] {
    padding-inline-start: calc(3 * var(--space-3));
  }
  li[data-depth='4'] {
    padding-inline-start: calc(4 * var(--space-3));
  }
  a,
  .group {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-2);
    padding-block: var(--space-2);
    padding-inline: var(--space-3);
    color: var(--color-text);
    text-decoration: none;
    border-radius: var(--radius);
  }
  .group {
    color: var(--color-text-muted);
  }
  a:hover {
    background: var(--color-surface-raised);
  }
  a[aria-current='page'] {
    background: var(--color-selected);
    font-weight: 600;
  }
  .name {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .badge {
    min-inline-size: 1.5rem;
    padding-inline: var(--space-1);
    text-align: center;
    font-size: 0.8rem;
    font-weight: 600;
    color: var(--color-accent-text);
    background: var(--color-accent);
    border-radius: 999px;
  }
</style>
