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
    {#each mail.folders.filter((x) => x.subscribed || x.special === 'inbox') as f (f.name)}
      <li data-depth={Math.min(depth(f), 4)}>
        {#if f.selectable}
          <a
            href={folderPath(f.name)}
            aria-current={f.name === mail.folder ? 'page' : undefined}
            onclick={(e) => go(e, f)}
            data-testid="folder"
            data-folder={f.name}
          >
            <svg class="icon" viewBox="0 0 16 16" aria-hidden="true">
              {#if f.special === 'inbox'}
                <path d="M2 9.5 3.8 3.4a1 1 0 0 1 1-.9h6.4a1 1 0 0 1 1 .9L14 9.5V13a.5.5 0 0 1-.5.5h-11A.5.5 0 0 1 2 13Z" /><path
                  d="M2 9.5h3.5l1 1.5h3l1-1.5H14"
                />
              {:else if f.special === 'drafts'}
                <path d="M10.5 2.5 13.5 5.5 6 13H3v-3Z" /><path d="m9 4 3 3" />
              {:else if f.special === 'sent'}
                <path d="M14 2 7 9M14 2l-4.5 12L7 9 2 6.5Z" />
              {:else if f.special === 'junk'}
                <circle cx="8" cy="8" r="6" /><path d="m3.8 3.8 8.4 8.4" />
              {:else if f.special === 'trash'}
                <path d="M2.5 4h11M6 4V2.5h4V4M4 4l.7 9.5h6.6L12 4M6.5 6.5v4.5M9.5 6.5v4.5" />
              {:else if f.special === 'archive'}
                <rect x="2" y="2.5" width="12" height="3" rx=".5" /><path d="M3 5.5V13h10V5.5M6.5 8.5h3" />
              {:else}
                <path d="M2 4a1 1 0 0 1 1-1h3.2l1.5 1.5H13a1 1 0 0 1 1 1V12a1 1 0 0 1-1 1H3a1 1 0 0 1-1-1Z" />
              {/if}
            </svg>
            <span class="name">{label(f)}</span>
            {#if f.unread > 0}
              <span class="badge" aria-label={t('folders.unread', { count: f.unread })}>{f.unread}</span>
            {/if}
          </a>
        {:else}
          <span class="group">
            <svg class="icon" viewBox="0 0 16 16" aria-hidden="true">
              <path d="M2 4a1 1 0 0 1 1-1h3.2l1.5 1.5H13a1 1 0 0 1 1 1V12a1 1 0 0 1-1 1H3a1 1 0 0 1-1-1Z" />
            </svg>
            <span class="name">{label(f)}</span>
          </span>
        {/if}
      </li>
    {/each}
  </ul>
</nav>

<style>
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
  ul {
    display: flex;
    flex-direction: column;
    gap: 1px;
  }
  a,
  .group {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    padding: 0.45rem var(--space-2);
    color: var(--color-text);
    text-decoration: none;
    border-radius: var(--radius);
    transition: background-color 0.1s ease-in-out;
  }
  .group {
    color: var(--color-text-muted);
  }
  a:hover {
    background: var(--color-hover);
  }
  a[aria-current='page'] {
    font-weight: 600;
    background: var(--color-selected);
  }
  a .icon,
  .group .icon {
    color: var(--color-text-muted);
  }
  a[aria-current='page'] .icon {
    color: var(--color-text);
  }
  .name {
    flex: 1;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .badge {
    min-inline-size: 1.5rem;
    padding-inline: 0.4rem;
    text-align: center;
    font-size: 0.75rem;
    font-weight: 600;
    line-height: 1.25rem;
    color: var(--color-badge-text);
    background: var(--color-badge);
    border-radius: 999px;
  }
</style>
