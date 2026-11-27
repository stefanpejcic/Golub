<!-- SPDX-License-Identifier: GPL-3.0-or-later -->
<script lang="ts">
  import { i18n } from './lib/locale.svelte';
  import { mail } from './lib/mail.svelte';
  import { formatFullDate, formatSize, type Address } from './lib/format';
  import { errorMessage } from './lib/errors';
  import { ApiError } from './lib/api';
  import { toast } from './lib/toast.svelte';

  const t = $derived(i18n.t);
  const m = $derived(mail.message);

  let showImages = $state(false);
  $effect(() => {
    void mail.uid;
    showImages = false;
  });

  const query = $derived(m ? `folder=${encodeURIComponent(m.folder)}&uid=${m.uid}` : '');
  const bodyUrl = $derived(`/api/message/body?${query}${showImages ? '&images=1' : ''}`);

  const targets = $derived(
    mail.folders.filter((f) => f.selectable && f.name !== mail.folder),
  );

  function folderLabel(name: string): string {
    const f = mail.folders.find((x) => x.name === name);
    return f?.special ? t(`folder.${f.special}`) : name;
  }

  async function run(action: () => Promise<string | void>) {
    try {
      const msg = await action();
      if (msg) toast.show(msg);
    } catch (e) {
      toast.show(errorMessage(t, e instanceof ApiError ? e.code : 'unknown'), true);
    }
  }

  function remove() {
    if (!m) return;
    const uid = m.uid;
    run(async () => ((await mail.remove([uid])) ? t('toast.trashed') : t('toast.deleted')));
  }

  function move(to: string) {
    if (!m || !to) return;
    const uid = m.uid;
    run(async () => {
      await mail.move([uid], to);
      return t('toast.moved', { folder: folderLabel(to) });
    });
  }

  function onKey(e: KeyboardEvent) {
    const target = e.target as HTMLElement;
    if (e.key === 'Delete' && m && !['INPUT', 'TEXTAREA', 'SELECT'].includes(target.tagName)) remove();
  }

  function people(list: Address[]): string {
    return list.map((a) => (a.name ? `${a.name} <${a.address}>` : a.address)).join(', ');
  }
</script>

<svelte:window onkeydown={onKey} />

<article class="message" aria-busy={mail.messageLoading} data-testid="message-view">
  <div class="page-header">
    {#if m}
      <div class="actions" role="toolbar" aria-label={t('actions.label')}>
        <button type="button" class="btn" onclick={remove} data-testid="action-delete">
          <svg class="icon" viewBox="0 0 16 16" aria-hidden="true">
            <path d="M2.5 4h11M6 4V2.5h4V4M4 4l.7 9.5h6.6L12 4M6.5 6.5v4.5M9.5 6.5v4.5" />
          </svg>
          <span>{t('actions.delete')}</span>
        </button>
        <button
          type="button"
          class="btn"
          onclick={() => run(() => mail.setFlag([m!.uid], 'seen', !m!.seen))}
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
          onclick={() => run(() => mail.setFlag([m!.uid], 'flagged', !m!.flagged))}
          data-testid="action-flag"
        >
          <svg class="icon" class:flagged={m.flagged} viewBox="0 0 16 16" aria-hidden="true">
            <path d="M4 14V2h8l-2 3 2 3H4" />
          </svg>
          <span>{m.flagged ? t('actions.unflag') : t('actions.flag')}</span>
        </button>
        <select
          class="input move"
          aria-label={t('actions.move')}
          onchange={(e) => {
            move(e.currentTarget.value);
            e.currentTarget.value = '';
          }}
          data-testid="action-move"
        >
          <option value="">{t('actions.move_to')}</option>
          {#each targets as f (f.name)}
            <option value={f.name}>{folderLabel(f.name)}</option>
          {/each}
        </select>
      </div>
    {/if}
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
  </div>

  {#if mail.messageError}
    <p class="state error" role="alert">{errorMessage(t, mail.messageError)}</p>
  {:else if m}
    <header>
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
    </header>

    {#if m.html && m.blockedImages > 0 && !showImages}
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

    {#if m.html}
      <iframe
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
  .move {
    inline-size: auto;
    padding-block: 0.375rem;
    cursor: pointer;
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
    padding-block: var(--space-4) var(--space-3);
    padding-inline: var(--space-4);
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
    border: none;
    border-block-start: 1px solid var(--color-border);
    background: var(--color-message-bg);
  }
  .text {
    padding: var(--space-4);
    border-block-start: 1px solid var(--color-border);
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
