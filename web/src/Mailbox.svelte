<!-- SPDX-License-Identifier: GPL-3.0-or-later -->
<script lang="ts">
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
  import { route } from './lib/route.svelte';
  import { folderPath } from './lib/format';
  import SettingsNav from './settings/SettingsNav.svelte';
  import SettingsPage from './settings/SettingsPage.svelte';
  import Compose from './Compose.svelte';

  const t = $derived(i18n.t);

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

<div class="mailbox" class:has-message={mail.uid !== null} class:in-settings={route.settingsSection !== null}>
  <aside>
    <div class="brand">
      <Logo size={28} />
      <span>{t('app.name')}</span>
    </div>
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
        <svg class="icon" viewBox="0 0 24 24" aria-hidden="true">
          <path
            d="M10.3 2.6a1.7 1.7 0 0 1 3.4 0l.2 1.3a7.6 7.6 0 0 1 1.9 1.1l1.2-.5a1.7 1.7 0 0 1 2.1.8l.1.2a1.7 1.7 0 0 1-.4 2.1l-1 .8a7.7 7.7 0 0 1 0 2.2l1 .8a1.7 1.7 0 0 1 .4 2.1l-.1.2a1.7 1.7 0 0 1-2.1.8l-1.2-.5a7.6 7.6 0 0 1-1.9 1.1l-.2 1.3a1.7 1.7 0 0 1-3.4 0l-.2-1.3a7.6 7.6 0 0 1-1.9-1.1l-1.2.5a1.7 1.7 0 0 1-2.1-.8l-.1-.2a1.7 1.7 0 0 1 .4-2.1l1-.8a7.7 7.7 0 0 1 0-2.2l-1-.8a1.7 1.7 0 0 1-.4-2.1l.1-.2a1.7 1.7 0 0 1 2.1-.8l1.2.5a7.6 7.6 0 0 1 1.9-1.1Z"
          />
          <circle cx="12" cy="10" r="3" />
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
    <div class="folders">
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
  <main class="list-pane">
    <MessageList />
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
    grid-template-columns: fit-content(28rem) minmax(20rem, 26rem) minmax(0, 1fr);
    block-size: 100vh;
  }
  aside {
    display: flex;
    flex-direction: column;
    min-block-size: 0;
    min-inline-size: var(--sidebar-width);
    background: var(--color-surface);
    border-inline-end: 1px solid var(--color-border);
  }
  .brand {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    flex: none;
    block-size: var(--header-height);
    padding-inline: var(--space-3);
    font-size: 1.05rem;
    font-weight: 600;
    border-block-end: 1px solid var(--color-border);
  }
  .account {
    display: flex;
    align-items: stretch;
    color: var(--color-text);
    font-weight: 600;
    border-block-end: 1px solid var(--color-border);
  }
  .email {
    flex: 1;
    padding: var(--space-3);
    white-space: nowrap;
  }
  .cog {
    display: grid;
    place-items: center;
    flex: none;
    inline-size: 2.75rem;
    color: var(--color-text-muted);
    border-inline-start: 1px solid var(--color-border);
  }
  .cog:hover,
  .cog[aria-current='page'] {
    color: var(--color-accent);
    background: var(--color-hover);
  }
  .cog .icon {
    inline-size: 1.15rem;
    block-size: 1.15rem;
  }
  .compose-btn {
    padding: var(--space-2) var(--space-2) 0;
  }
  .compose-btn .btn {
    inline-size: 100%;
    padding-block: 0.5rem;
    text-decoration: none;
  }
  .folders {
    flex: 1;
    min-block-size: 0;
    overflow-y: auto;
    padding: var(--space-2);
  }
  .list-pane {
    min-block-size: 0;
    border-inline-end: 1px solid var(--color-border);
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
  @media (max-width: 48rem) {
    .mailbox {
      grid-template-columns: 1fr;
      grid-template-rows: auto 1fr;
      block-size: auto;
      min-block-size: 100vh;
    }
    aside {
      border-inline-end: none;
      border-block-end: 1px solid var(--color-border);
    }
    .folders {
      max-block-size: 40vh;
    }
  }
</style>
