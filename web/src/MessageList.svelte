<!-- SPDX-License-Identifier: GPL-3.0-or-later -->
<script lang="ts">
  import { i18n } from './lib/locale.svelte';
  import { mail } from './lib/mail.svelte';
  import { displayName, formatListDate, formatFullDate, messagePath } from './lib/format';
  import { errorMessage } from './lib/errors';

  const t = $derived(i18n.t);
  const showRecipients = $derived(mail.current?.special === 'sent' || mail.current?.special === 'drafts');
  const title = $derived(
    mail.current?.special ? t(`folder.${mail.current.special}`) : (mail.current?.name ?? mail.folder),
  );
  const first = $derived(mail.total === 0 ? 0 : (mail.page - 1) * mail.pageSize + 1);
  const last = $derived(Math.min(mail.page * mail.pageSize, mail.total));
  const hasPrev = $derived(mail.page > 1);
  const hasNext = $derived(last < mail.total);
  const numbers = $derived(new Intl.NumberFormat(i18n.locale));

  function open(e: MouseEvent, uid: number) {
    if (e.metaKey || e.ctrlKey || e.shiftKey || e.button !== 0) return;
    e.preventDefault();
    mail.openMessage(uid);
  }
</script>

<section class="list" aria-busy={mail.loading}>
  <header class="page-header">
    <h1 class="page-title" data-testid="folder-title">{title}</h1>
    <div class="pager">
      {#if mail.total > 0}
        <span data-testid="range">
          {t('list.range', { from: numbers.format(first), to: numbers.format(last), total: numbers.format(mail.total) })}
        </span>
      {/if}
      <button
        type="button"
        class="btn btn-icon"
        disabled={!hasPrev || mail.loading}
        onclick={() => mail.open(mail.folder, mail.page - 1)}
        aria-label={t('list.prev')}
        data-testid="prev"
      >
        <svg class="icon" viewBox="0 0 16 16" aria-hidden="true"><path d="M10 3 5 8l5 5" /></svg>
      </button>
      <button
        type="button"
        class="btn btn-icon"
        disabled={!hasNext || mail.loading}
        onclick={() => mail.open(mail.folder, mail.page + 1)}
        aria-label={t('list.next')}
        data-testid="next"
      >
        <svg class="icon" viewBox="0 0 16 16" aria-hidden="true"><path d="m6 3 5 5-5 5" /></svg>
      </button>
    </div>
  </header>

  {#if mail.error}
    <p class="state error" role="alert">{errorMessage(t, mail.error)}</p>
  {:else if !mail.loading && mail.messages.length === 0}
    <p class="state" data-testid="empty">{t('list.empty')}</p>
  {:else}
    <ul data-testid="messages">
      {#each mail.messages as m (m.uid)}
        {@const who = showRecipients ? m.to.map(displayName).join(', ') : displayName(m.from[0])}
        <li class:unread={!m.seen} data-testid="message" data-uid={m.uid}>
          <a class="row" href={messagePath(mail.folder, m.uid)} onclick={(e) => open(e, m.uid)}>
          <span class="who">{who}</span>
          <span class="subject">
            {#if m.flagged}
              <svg class="icon flag" viewBox="0 0 16 16" role="img" aria-label={t('list.flagged')}>
                <path d="M4 14V2h8l-2 3 2 3H4" />
              </svg>
            {/if}
            {m.subject || t('list.no_subject')}
          </span>
          <span class="meta">
            {#if m.attachments}
              <svg class="icon clip" viewBox="0 0 16 16" role="img" aria-label={t('list.attachment')}>
                <path d="M11 5 6 10a1.5 1.5 0 0 0 2 2l5-5a3 3 0 0 0-4-4L4 8a4.5 4.5 0 0 0 6 6l4-4" />
              </svg>
            {/if}
            <time datetime={m.date} title={formatFullDate(i18n.locale, m.date)}>
              {formatListDate(i18n.locale, m.date)}
            </time>
          </span>
          </a>
        </li>
      {/each}
    </ul>
  {/if}
</section>

<style>
  .list {
    display: flex;
    flex-direction: column;
    min-inline-size: 0;
  }
  .pager {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    color: var(--color-text-muted);
    font-size: 0.9rem;
  }
  ul {
    list-style: none;
    margin: 0;
    padding: 0;
  }
  li {
    border-block-end: 1px solid var(--color-border);
    color: var(--color-text-muted);
  }
  .row {
    display: grid;
    grid-template-columns: minmax(8rem, 14rem) 1fr auto;
    gap: var(--space-3);
    align-items: center;
    padding-block: 0.625rem;
    padding-inline: var(--space-4);
    color: inherit;
    text-decoration: none;
  }
  .row:hover {
    background: var(--color-hover);
  }
  li.unread {
    color: var(--color-unread);
    font-weight: 600;
  }
  .who,
  .subject {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .subject {
    color: var(--color-text);
  }
  .meta {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    font-size: 0.85rem;
    white-space: nowrap;
  }
  .flag {
    color: var(--color-danger);
    fill: currentColor;
    vertical-align: -2px;
  }
  .subject {
    display: flex;
    align-items: center;
    gap: var(--space-1);
  }
  .state {
    padding: var(--space-4);
    color: var(--color-text-muted);
  }
  .error {
    color: var(--color-danger);
  }
  @media (max-width: 40rem) {
    .row {
      grid-template-columns: 1fr auto;
    }
    .subject {
      grid-column: 1 / -1;
      grid-row: 2;
    }
  }
</style>
