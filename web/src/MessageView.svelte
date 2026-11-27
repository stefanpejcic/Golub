<!-- SPDX-License-Identifier: GPL-3.0-or-later -->
<script lang="ts">
  import { i18n } from './lib/locale.svelte';
  import { mail } from './lib/mail.svelte';
  import { formatFullDate, formatSize, type Address } from './lib/format';
  import { errorMessage } from './lib/errors';
  import { settings } from './lib/settings.svelte';
  import { route } from './lib/route.svelte';
  import { archiveMessages, copyMessages, flagMessages, folderLabel, moveMessages, removeMessages } from './lib/actions';
  import Menu from './Menu.svelte';

  let { standalone = false, onloaded }: { standalone?: boolean; onloaded?: () => void } = $props();
  const t = $derived(i18n.t);
  let headers = $state('');
  let picker = $state<'' | 'move' | 'copy'>('');
  let showHeaders = $state(false);
  const m = $derived(mail.message);

  let showImages = $state(false);
  let plain = $state(false);
  $effect(() => {
    void mail.uid;
    showImages = settings.value.remoteImages === 'always';
    plain = settings.value.preferPlain;
    showHeaders = false;
    headers = '';
  });

  async function toggleHeaders() {
    showHeaders = !showHeaders;
    if (showHeaders && !headers && m) {
      const res = await fetch(`/api/message/raw?${query}&headers=1`, { credentials: 'same-origin' });
      headers = res.ok ? await res.text() : '';
    }
  }

  function openWindow(path: string) {
    window.open(path, '_blank', 'noopener');
  }
  const asHtml = $derived(!!m?.html && !(plain && m.text.trim() !== ''));

  const query = $derived(m ? `folder=${encodeURIComponent(m.folder)}&uid=${m.uid}` : '');
  const bodyUrl = $derived(`/api/message/body?${query}${showImages ? '&images=1' : ''}`);

  const targets = $derived(
    mail.folders.filter((f) => f.selectable && f.name !== mail.folder),
  );

  const inDrafts = $derived(mail.current?.special === 'drafts');

  function compose(mode: string) {
    if (!m) return;
    const draft = mode === 'edit' && inDrafts ? '&draft=1' : '';
    route.go(`/compose?mode=${mode}&folder=${encodeURIComponent(m.folder)}&uid=${m.uid}${draft}`);
  }

  function remove() {
    if (m) removeMessages([m.uid]);
  }

  function move(to: string) {
    if (m && to) moveMessages([m.uid], to);
  }

  function onKey(e: KeyboardEvent) {
    const target = e.target as HTMLElement;
    if (e.key === 'Delete' && m && mail.selected.length <= 1 && !['INPUT', 'TEXTAREA', 'SELECT'].includes(target.tagName)) {
      remove();
    }
  }

  function people(list: Address[]): string {
    const full = settings.value.showAddresses;
    return list.map((a) => (a.name ? (full ? `${a.name} <${a.address}>` : a.name) : a.address)).join(', ');
  }
</script>

<svelte:window onkeydown={onKey} />

<article class="message" aria-busy={mail.messageLoading} data-testid="message-view">
  <div class="page-header">
    {#if m}
      <div class="actions" role="toolbar" aria-label={t('actions.label')}>
        {#if inDrafts}
          <button type="button" class="btn btn-primary" onclick={() => compose('edit')} data-testid="action-edit-draft">
            <svg class="icon" viewBox="0 0 16 16" aria-hidden="true"><path d="M10.5 2.5 13.5 5.5 6 13H3v-3Z" /></svg>
            <span>{t('actions.edit_draft')}</span>
          </button>
        {:else}
          <button type="button" class="btn" onclick={() => compose('reply')} data-testid="action-reply">
            <svg class="icon" viewBox="0 0 16 16" aria-hidden="true"><path d="M6.5 4 2.5 8l4 4M2.5 8h7a4 4 0 0 1 4 4v1" /></svg>
            <span>{t('actions.reply')}</span>
          </button>
          <button type="button" class="btn" onclick={() => compose('replyAll')} data-testid="action-reply-all">
            <svg class="icon" viewBox="0 0 16 16" aria-hidden="true"><path d="M8 4 4 8l4 4M4.5 4.5 1 8l3.5 3.5M4 8h6a4 4 0 0 1 4 4v1" /></svg>
            <span>{t('actions.reply_all')}</span>
          </button>
          <button type="button" class="btn" onclick={() => compose('forward')} data-testid="action-forward">
            <svg class="icon" viewBox="0 0 16 16" aria-hidden="true"><path d="M9.5 4l4 4-4 4M13.5 8h-7a4 4 0 0 0-4 4v1" /></svg>
            <span>{t('actions.forward')}</span>
          </button>
        {/if}
        <button type="button" class="btn" onclick={remove} data-testid="action-delete">
          <svg class="icon" viewBox="0 0 16 16" aria-hidden="true">
            <path d="M2.5 4h11M6 4V2.5h4V4M4 4l.7 9.5h6.6L12 4M6.5 6.5v4.5M9.5 6.5v4.5" />
          </svg>
          <span>{t('actions.delete')}</span>
        </button>
        <button type="button" class="btn" onclick={() => archiveMessages([m!.uid])} data-testid="action-archive">
          <svg class="icon" viewBox="0 0 16 16" aria-hidden="true">
            <rect x="2" y="2.5" width="12" height="3" rx=".5" /><path d="M3 5.5V13h10V5.5M6.5 8.5h3" />
          </svg>
          <span>{t('actions.archive')}</span>
        </button>
        <button
          type="button"
          class="btn"
          onclick={() => flagMessages([m!.uid], 'seen', !m!.seen)}
          data-testid="action-seen"
        >
          <svg class="icon" viewBox="0 0 16 16" aria-hidden="true">
            {#if m.seen}
              <rect x="2" y="3.5" width="12" height="9" rx="1" /><path d="m2.5 4.5 5.5 4 5.5-4" /><circle
                cx="13"
                cy="4"
                r="2"
                fill="currentColor"
              />
            {:else}
              <path d="M2 7 8 3l6 4v6H2Z" /><path d="m2.5 7.5 5.5 3.5 5.5-3.5" />
            {/if}
          </svg>
          <span>{m.seen ? t('actions.mark_unread') : t('actions.mark_read')}</span>
        </button>
        <button
          type="button"
          class="btn"
          aria-pressed={m.flagged}
          onclick={() => flagMessages([m!.uid], 'flagged', !m!.flagged)}
          data-testid="action-flag"
        >
          <svg class="icon" class:flagged={m.flagged} viewBox="0 0 16 16" aria-hidden="true">
            <path d="M4 14V2h8l-2 3 2 3H4" />
          </svg>
          <span>{m.flagged ? t('actions.unflag') : t('actions.flag')}</span>
        </button>
        <Menu testid="action-more">
          {#snippet label()}{t('actions.more')}{/snippet}
          {#snippet children(close)}
            <button class="item" role="menuitem" onclick={() => (openWindow(`/message?${query}&print=1`), close())} data-testid="more-print">
              {t('actions.print')}
            </button>
            <a class="item" role="menuitem" href={`/api/message/raw?${query}&download=1`} download onclick={close} data-testid="more-download">
              {t('actions.download')}
            </a>
            <button class="item" role="menuitem" onclick={() => (openWindow(`/api/message/raw?${query}`), close())} data-testid="more-source">
              {t('actions.source')}
            </button>
            <button class="item" role="menuitem" onclick={() => (compose('edit'), close())} data-testid="more-edit">
              {t('actions.edit_as_new')}
            </button>
            <button class="item" role="menuitem" onclick={() => (openWindow(`/message?${query}`), close())} data-testid="more-window">
              {t('actions.new_window')}
            </button>
            <div class="sep"></div>
            <button
              class="item"
              role="menuitem"
              aria-expanded={picker === 'move'}
              onclick={() => (picker = picker === 'move' ? '' : 'move')}
              data-testid="action-move"
            >
              {t('actions.move_to')}
            </button>
            {#if picker === 'move'}
              <div class="folders" role="group" aria-label={t('actions.move_to')}>
                {#each targets as f (f.name)}
                  <button class="item sub" role="menuitem" onclick={() => (move(f.name), (picker = ''), close())} data-folder={f.name}>
                    {folderLabel(f.name)}
                  </button>
                {/each}
              </div>
            {/if}
            <button
              class="item"
              role="menuitem"
              aria-expanded={picker === 'copy'}
              onclick={() => (picker = picker === 'copy' ? '' : 'copy')}
              data-testid="action-copy"
            >
              {t('actions.copy_to')}
            </button>
            {#if picker === 'copy'}
              <div class="folders" role="group" aria-label={t('actions.copy_to')}>
                {#each targets as f (f.name)}
                  <button
                    class="item sub"
                    role="menuitem"
                    onclick={() => {
                      if (m) copyMessages([m.uid], f.name);
                      picker = '';
                      close();
                    }}
                    data-folder={f.name}
                  >
                    {folderLabel(f.name)}
                  </button>
                {/each}
              </div>
            {/if}
          {/snippet}
        </Menu>
      </div>
    {/if}
    {#if !standalone}
    <button
      type="button"
      class="btn back"
      onclick={() => mail.closeMessage()}
      aria-label={t('message.back')}
      title={t('message.back')}
      data-testid="back"
    >
      <svg class="icon narrow" viewBox="0 0 16 16" aria-hidden="true"><path d="M10 3 5 8l5 5" /></svg>
      <span class="narrow">{t('message.back')}</span>
      <svg class="icon wide" viewBox="0 0 16 16" aria-hidden="true"><path d="m4 4 8 8M12 4l-8 8" /></svg>
    </button>
    {/if}
  </div>

  {#if mail.messageError}
    <p class="state error" role="alert">{errorMessage(t, mail.messageError)}</p>
  {:else if m}
    <header>
      <div class="head-icons">
        {#if m.html && m.text.trim()}
          <button
            type="button"
            class="btn btn-icon"
            aria-pressed={!asHtml}
            title={asHtml ? t('message.view_plain') : t('message.view_html')}
            aria-label={asHtml ? t('message.view_plain') : t('message.view_html')}
            onclick={() => (plain = asHtml)}
            data-testid="toggle-plain"
          >
            <svg class="icon" viewBox="0 0 16 16" aria-hidden="true">
              {#if asHtml}<path d="M3 4h10M3 8h10M3 12h6" />{:else}<path d="M5.5 4 2 8l3.5 4M10.5 4 14 8l-3.5 4" />{/if}
            </svg>
          </button>
        {/if}
        <button
          type="button"
          class="btn btn-icon"
          aria-pressed={showHeaders}
          title={t('message.all_headers')}
          aria-label={t('message.all_headers')}
          onclick={toggleHeaders}
          data-testid="toggle-headers"
        >
          <svg class="icon" viewBox="0 0 16 16" aria-hidden="true"><rect x="2.5" y="2.5" width="11" height="11" rx="1.5" /><path d="M5 6h6M5 8.5h6M5 11h3" /></svg>
        </button>
      </div>
      <h1 data-testid="message-subject">{m.subject || t('list.no_subject')}</h1>
      <dl>
        <dt>{t('message.from')}</dt>
        <dd data-testid="message-from">{people(m.from)}</dd>
        {#if m.to.length}
          <dt>{t('message.to')}</dt>
          <dd>{people(m.to)}</dd>
        {/if}
        {#if m.cc.length}
          <dt>{t('message.cc')}</dt>
          <dd>{people(m.cc)}</dd>
        {/if}
        <dt>{t('message.date')}</dt>
        <dd><time datetime={m.date}>{formatFullDate(i18n.locale, m.date)}</time></dd>
      </dl>
      {#if showHeaders}
        <pre class="raw-headers" data-testid="raw-headers">{headers}</pre>
      {/if}
    </header>

    {#if asHtml && m.blockedImages > 0 && !showImages}
      <div class="notice" data-testid="images-blocked">
        <span>{t('message.images_blocked', { count: m.blockedImages })}</span>
        <button type="button" class="btn" onclick={() => (showImages = true)} data-testid="show-images">
          {t('message.show_images')}
        </button>
      </div>
    {/if}

    {#if m.attachments.length}
      <section class="attachments" aria-label={t('message.attachments', { count: m.attachments.length })}>
        <ul>
          {#each m.attachments as a (a.id)}
            <li>
              <a
                href={`/api/message/part?${query}&part=${encodeURIComponent(a.id)}`}
                download={a.filename}
                data-testid="attachment"
              >
                <svg class="icon" viewBox="0 0 16 16" aria-hidden="true">
                  <path d="M11 5 6 10a1.5 1.5 0 0 0 2 2l5-5a3 3 0 0 0-4-4L4 8a4.5 4.5 0 0 0 6 6l4-4" />
                </svg>
                <span class="name">{a.filename}</span>
                <span class="size">{formatSize(i18n.locale, a.size)}</span>
              </a>
            </li>
          {/each}
        </ul>
      </section>
    {/if}

    {#if asHtml}
      <iframe
        onload={() => onloaded?.()}
        title={t('message.body')}
        src={bodyUrl}
        sandbox="allow-popups allow-popups-to-escape-sandbox"
        referrerpolicy="no-referrer"
        data-testid="message-body"
      ></iframe>
    {:else}
      <div class="text" data-testid="message-text">{m.text}</div>
    {/if}
  {/if}
</article>

<style>
  .page-header {
    justify-content: space-between;
    padding-inline: var(--space-3);
  }
  .actions {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: var(--space-2);
  }
  .flagged {
    color: var(--color-danger);
    fill: currentColor;
  }
  @media (max-width: 90rem) {
    .actions .btn span {
      display: none;
    }
  }
  .back .narrow {
    display: none;
  }
  @media (max-width: 64rem) {
    .page-header {
      flex-direction: row-reverse;
    }
    .back .narrow {
      display: inline;
    }
    .back .wide {
      display: none;
    }
  }
  .message {
    flex: 1;
    display: flex;
    flex-direction: column;
    min-inline-size: 0;
    min-block-size: 0;
  }
  header {
    position: relative;
    padding-block: var(--space-4) var(--space-3);
    padding-inline: var(--space-4);
  }
  .head-icons {
    position: absolute;
    inset-block-start: var(--space-3);
    inset-inline-end: var(--space-4);
    display: flex;
    gap: var(--space-1);
  }
  .head-icons .btn[aria-pressed='true'] {
    color: var(--color-accent);
    background: var(--color-hover);
  }
  header h1 {
    padding-inline-end: 5rem;
  }
  .raw-headers {
    max-block-size: 16rem;
    overflow: auto;
    margin-block: var(--space-3) 0;
    padding: var(--space-3);
    font-size: 0.8rem;
    white-space: pre-wrap;
    overflow-wrap: anywhere;
    background: var(--color-surface);
    border: 1px solid var(--color-border);
    border-radius: var(--radius);
  }
  .folders {
    max-block-size: 14rem;
    overflow-y: auto;
  }
  .folders :global(.sub) {
    padding-inline-start: var(--space-4);
    color: var(--color-text-muted);
  }
  .folders :global(.sub:hover) {
    color: var(--color-text);
  }
  @media print {
    .page-header,
    .head-icons,
    .notice {
      display: none;
    }
  }
  h1 {
    margin-block: 0 var(--space-2);
    font-size: 1.3rem;
    overflow-wrap: anywhere;
  }
  dl {
    display: grid;
    grid-template-columns: auto 1fr;
    gap: var(--space-1) var(--space-3);
    margin: 0;
    font-size: 0.9rem;
  }
  dt {
    color: var(--color-text-muted);
  }
  dd {
    margin: 0;
    overflow-wrap: anywhere;
  }
  .notice {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-3);
    margin-inline: var(--space-4);
    margin-block-end: var(--space-3);
    padding: var(--space-2) var(--space-3);
    background: var(--color-notice);
    border: 1px solid var(--color-notice-border);
    border-radius: var(--radius);
    border-radius: var(--radius);
    font-size: 0.9rem;
  }
  .attachments ul {
    display: flex;
    flex-wrap: wrap;
    gap: var(--space-2);
    list-style: none;
    margin: 0;
    padding-block: 0 var(--space-3);
    padding-inline: var(--space-4);
  }
  .attachments a {
    display: inline-flex;
    align-items: center;
    gap: var(--space-2);
    padding-block: var(--space-1);
    padding-inline: var(--space-2);
    color: var(--color-text);
    text-decoration: none;
    border: 1px solid var(--color-input-border);
    border-radius: var(--radius);
    box-shadow: var(--shadow-xs);
  }
  .attachments a:hover {
    background: var(--color-hover);
  }
  .size {
    color: var(--color-text-muted);
    font-size: 0.85rem;
  }
  iframe {
    flex: 1;
    min-block-size: 60vh;
    margin: 0 var(--space-3) var(--space-3);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-lg);
    background: var(--color-message-bg);
  }
  .text {
    padding: var(--space-4);
    white-space: pre-wrap;
    overflow-wrap: anywhere;
    font-family: var(--font-sans);
    line-height: 1.5;
  }
  .state {
    padding: var(--space-4);
    color: var(--color-text-muted);
  }
  .error {
    color: var(--color-danger);
  }
</style>
