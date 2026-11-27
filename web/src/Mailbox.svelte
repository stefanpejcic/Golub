<!-- SPDX-License-Identifier: GPL-3.0-or-later -->
<script lang="ts">
  import '@fontsource-variable/inter';
  import { i18n } from './lib/locale.svelte';
  import { mail } from './lib/mail.svelte';
  import FolderList from './FolderList.svelte';
  import MessageList from './MessageList.svelte';
  import MessageView from './MessageView.svelte';
  import Prefs from './Prefs.svelte';
  import Toast from './Toast.svelte';
  import Logo from './Logo.svelte';
  import { session } from './lib/session.svelte';
  import { settings } from './lib/settings.svelte';
  import { live } from './lib/live.svelte';
  import { route } from './lib/route.svelte';
  import { folderPath } from './lib/format';
  import { ui } from './lib/ui.svelte';
  import FolderSelect from './FolderSelect.svelte';
  import { folderLabel } from './lib/actions';
  import SettingsNav from './settings/SettingsNav.svelte';
  import SettingsPage from './settings/SettingsPage.svelte';
  import Compose from './Compose.svelte';
  import Resizer from './Resizer.svelte';
  import { clamp, LIMITS, loadColumns, saveColumns, type Columns } from './lib/resize';

  const t = $derived(i18n.t);

  let root: HTMLDivElement;
  let aside = $state<HTMLElement>();
  let listPane = $state<HTMLElement>();
  let columns = $state<Columns | null>(loadColumns());

  // Set through CSSOM, which the CSP allows, instead of a style attribute.
  $effect(() => {
    if (columns) root.style.setProperty('--columns', `${columns.sidebar}px ${columns.list}px minmax(0, 1fr)`);
    else root.style.removeProperty('--columns');
  });

  function current(): Columns {
    return {
      sidebar: columns?.sidebar ?? aside?.getBoundingClientRect().width ?? 256,
      list: columns?.list ?? listPane?.getBoundingClientRect().width ?? 384,
    };
  }

  function resize(which: keyof Columns, width: number, done: boolean) {
    columns = { ...current(), [which]: clamp(which, width) };
    if (done) saveColumns(columns);
  }

  const mobileTitle = $derived(
    route.settingsSection ? t('settings.title') : route.compose ? t('compose.title.new') : folderLabel(mail.folder),
  );

  // Anything that navigates closes the drawer, so it never stays over the page that was opened.
  $effect(() => {
    void route.path;
    void mail.folder;
    ui.close();
  });

  function reset() {
    columns = null;
    saveColumns(null);
  }

  $effect(() => {
    if (settings.loaded) live.watch(mail.folder);
  });
  $effect(() => () => live.stop());

  $effect(() => {
    let stop: (() => void) | undefined;
    let alive = true;
    settings.load().then(() => {
      if (alive) stop = mail.start();
    });
    return () => {
      alive = false;
      stop?.();
    };
  });
</script>

<svelte:window onkeydown={(e) => e.key === 'Escape' && ui.drawer && ui.close()} />

<div bind:this={root} class="mailbox" class:drawer-open={ui.drawer} class:custom={columns !== null} class:has-message={mail.uid !== null} class:in-settings={route.settingsSection !== null}>
  <div class="mobile-bar">
    <button
      type="button"
      class="btn btn-icon"
      aria-label={t('layout.menu')}
      aria-expanded={ui.drawer}
      onclick={() => ui.toggle()}
      data-testid="drawer-toggle"
    >
      <svg class="icon" viewBox="0 0 16 16" aria-hidden="true"><path d="M2.5 4h11M2.5 8h11M2.5 12h11" /></svg>
    </button>
    {#if route.settingsSection || route.compose}
      <span class="mobile-title">{mobileTitle}</span>
    {:else}
      <FolderSelect />
    {/if}
  </div>
  {#if ui.drawer}
    <button type="button" class="backdrop" aria-label={t('layout.close_menu')} onclick={() => ui.close()} data-testid="drawer-backdrop"></button>
  {/if}
  <aside bind:this={aside} data-testid="sidebar">
    <Resizer
      label={t('layout.resize_sidebar')}
      testid="resize-sidebar"
      min={LIMITS.sidebar[0]}
      max={LIMITS.sidebar[1]}
      start={() => current().sidebar}
      onresize={(w, done) => resize('sidebar', w, done)}
      onreset={reset}
    />
    <div class="account">
      <span class="email" data-testid="user-email" title={session.info?.email}>{session.info?.email}</span>
      <a
        class="cog"
        href="/settings/general"
        aria-label={t('settings.title')}
        title={t('settings.title')}
        aria-current={route.settingsSection ? 'page' : undefined}
        onclick={(e) => {
          if (e.metaKey || e.ctrlKey || e.shiftKey || e.button !== 0) return;
          e.preventDefault();
          if (route.settingsSection) {
            route.go(folderPath(mail.folder));
            mail.open(mail.folder, 1, { replace: true });
          } else {
            route.go('/settings/general');
          }
        }}
        data-testid="settings-open"
      >
        <svg class="gear" viewBox="0 0 24 24" aria-hidden="true">
          <path
            class="teeth"
            d="M19 12h2.6M16.95 16.95l1.84 1.84M12 19v2.6M7.05 16.95l-1.84 1.84M5 12H2.4M7.05 7.05 5.21 5.21M12 5V2.4M16.95 7.05l1.84-1.84"
          />
          <circle cx="12" cy="12" r="6.2" />
          <circle cx="12" cy="12" r="2.2" />
        </svg>
      </a>
    </div>
    {#if !route.settingsSection}
      <div class="compose-btn">
        <a
          class="btn btn-primary"
          href="/compose"
          onclick={(e) => {
            if (e.metaKey || e.ctrlKey || e.shiftKey || e.button !== 0) return;
            e.preventDefault();
            route.go('/compose');
          }}
          data-testid="compose-open"
        >
          <svg class="icon" viewBox="0 0 16 16" aria-hidden="true"><path d="M10.5 2.5 13.5 5.5 6 13H3v-3Z" /><path d="m9 4 3 3" /></svg>
          {t('compose.open')}
        </a>
      </div>
    {/if}
    <div class="folders" class:mobile-hidden={!route.settingsSection}>
      {#if route.settingsSection}
        <SettingsNav />
      {:else}
        <FolderList />
      {/if}
    </div>
    <Prefs />
  </aside>
  {#if route.settingsSection}
    <main class="settings-pane">
      <SettingsPage />
    </main>
  {:else if route.compose}
    <main class="settings-pane">
      {#key route.search}
        <Compose params={route.compose} />
      {/key}
    </main>
  {:else}
  <main class="list-pane" bind:this={listPane}>
    <MessageList />
    <Resizer
      label={t('layout.resize_list')}
      testid="resize-list"
      min={LIMITS.list[0]}
      max={LIMITS.list[1]}
      start={() => current().list}
      onresize={(w, done) => resize('list', w, done)}
      onreset={reset}
    />
  </main>
  <section class="preview-pane">
    {#if mail.selected.length > 1}
      <div class="placeholder" data-testid="multi-selected">
        <Logo size={120} variant="mascot" />
        <p>{t('list.selected', { count: mail.selected.length })}</p>
      </div>
    {:else if mail.uid !== null}
      <MessageView />
    {:else}
      <div class="placeholder" data-testid="no-message">
        <Logo size={120} variant="mascot" />
        <p>{t('message.none_selected')}</p>
      </div>
    {/if}
  </section>
  {/if}
</div>

<Toast />

<style>
  .mailbox {
    display: grid;
    grid-template-columns: var(--columns, fit-content(28rem) minmax(20rem, 26rem) minmax(0, 1fr));
    block-size: 100vh;
  }
  aside {
    position: relative;
    display: flex;
    flex-direction: column;
    min-block-size: 0;
    min-inline-size: 0;
    background: var(--color-surface);
  }
  /* The default width fits the address; once the user drags, their width wins and content shrinks. */
  .mailbox:not(.custom) aside {
    min-inline-size: var(--sidebar-width);
  }
  .account {
    margin-block-start: var(--space-3);
    display: flex;
    align-items: stretch;
    margin-inline: var(--space-2);
    color: var(--color-text);
    font-weight: 600;
    background: var(--color-hover);
    border-radius: var(--radius);
  }
  .email {
    flex: 1;
    min-inline-size: 0;
    padding: 0.6rem var(--space-3);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .cog {
    display: grid;
    place-items: center;
    flex: none;
    inline-size: 2.75rem;
    color: var(--color-text);
    border-inline-start: 1px solid var(--color-border);
    border-start-end-radius: var(--radius);
    border-end-end-radius: var(--radius);
  }
  .cog:hover,
  .cog[aria-current='page'] {
    color: var(--color-accent);
    background: var(--color-hover);
  }
  .gear {
    inline-size: 1.25rem;
    block-size: 1.25rem;
    fill: none;
    stroke: currentColor;
    stroke-width: 2.2;
  }
  .gear .teeth {
    stroke-width: 3.2;
  }
  .compose-btn {
    padding: var(--space-3) var(--space-2) 0;
  }
  .compose-btn .btn {
    inline-size: 100%;
    min-inline-size: 0;
    padding-block: 0.5rem;
    overflow: hidden;
    text-decoration: none;
  }
  .folders {
    flex: 1;
    min-block-size: 0;
    overflow-y: auto;
    padding: var(--space-2);
  }
  .list-pane {
    position: relative;
    min-block-size: 0;
    border-inline-end: 1px solid var(--color-border);
    background: var(--color-bg);
  }
  .settings-pane {
    grid-column: 2 / -1;
    min-block-size: 0;
    min-inline-size: 0;
  }
  .preview-pane {
    display: flex;
    flex-direction: column;
    min-block-size: 0;
    overflow-y: auto;
  }
  .placeholder {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: var(--space-3);
    color: var(--color-text-muted);
    opacity: 0.8;
  }
  .placeholder :global(.logo) {
    opacity: 0.35;
    filter: grayscale(0.4);
  }
  /* Too narrow for three columns: show either the list or the open message. */
  @media (max-width: 64rem) {
    .mailbox {
      grid-template-columns: fit-content(28rem) minmax(0, 1fr);
    }
    .list-pane {
      border-inline-end: none;
    }
    .mailbox:not(.has-message) .preview-pane,
    .mailbox.has-message .list-pane {
      display: none;
    }
  }
  .mobile-bar,
  .backdrop {
    display: none;
  }
  /* Phones: the sidebar becomes a drawer and the content gets the whole width. */
  @media (max-width: 48rem) {
    .mailbox {
      grid-template-columns: minmax(0, 1fr);
      grid-template-rows: auto minmax(0, 1fr);
      block-size: 100dvh;
    }
    .mobile-bar {
      display: flex;
      align-items: center;
      gap: var(--space-2);
      padding: var(--space-2) var(--space-3);
      background: var(--color-surface);
    }
    .mobile-title {
      overflow: hidden;
      font-weight: 600;
      text-overflow: ellipsis;
      white-space: nowrap;
    }
    aside {
      position: fixed;
      z-index: 40;
      inset-block: 0;
      inset-inline-start: 0;
      inline-size: min(85vw, 20rem);
      transform: translateX(-100%);
      transition: transform 0.2s ease;
      box-shadow: var(--shadow);
    }
    .mailbox:not(.custom) aside {
      min-inline-size: 0;
    }
    .drawer-open aside {
      transform: none;
    }
    .mobile-hidden {
      visibility: hidden;
    }
    .backdrop {
      display: block;
      position: fixed;
      z-index: 35;
      inset: 0;
      padding: 0;
      background: rgb(0 0 0 / 0.4);
      border: none;
    }
    .list-pane,
    .preview-pane,
    .settings-pane {
      grid-row: 2;
      grid-column: 1;
      border-inline-end: none;
    }
  }
  @media (max-width: 48rem) and (prefers-reduced-motion: reduce) {
    aside {
      transition: none;
    }
  }
</style>
