<!-- SPDX-License-Identifier: GPL-3.0-or-later -->
<script lang="ts">
  import { i18n } from './lib/locale.svelte';
  import { mail } from './lib/mail.svelte';
  import { displayName, formatListDate, formatFullDate, formatSize, messagePath } from './lib/format';
  import { errorMessage } from './lib/errors';
  import { settings } from './lib/settings.svelte';
  import { archiveMessages, flagMessages, folderLabel, moveMessages, removeMessages } from './lib/actions';

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

  const bulk = $derived(mail.selected.length > 1);
  const selectedRows = $derived(mail.messages.filter((m) => mail.selected.includes(m.uid)));
  const anyUnread = $derived(selectedRows.some((m) => !m.seen));
  const anyUnflagged = $derived(selectedRows.some((m) => !m.flagged));
  const targets = $derived(mail.folders.filter((f) => f.selectable && f.name !== mail.folder));
  let swiped = false;
  let searchTimer: ReturnType<typeof setTimeout> | undefined;
  let searchInput: HTMLInputElement;

  function onSearch(value: string, now = false) {
    clearTimeout(searchTimer);
    if (now) mail.search(value);
    else searchTimer = setTimeout(() => mail.search(value), 350);
  }

  function open(e: MouseEvent, uid: number) {
    if (e.button !== 0) return;
    if (swiped) {
      e.preventDefault();
      swiped = false;
      return;
    }
    if (e.ctrlKey || e.metaKey) {
      e.preventDefault();
      mail.select(uid, 'toggle');
      return;
    }
    if (e.shiftKey) {
      e.preventDefault();
      mail.select(uid, 'range');
      return;
    }
    e.preventDefault();
    mail.openMessage(uid);
  }

  function onKey(e: KeyboardEvent) {
    const tag = (e.target as HTMLElement).tagName;
    if (['INPUT', 'TEXTAREA', 'SELECT'].includes(tag)) return;
    if (e.key === 'Escape' && bulk) mail.clearSelection();
    if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'a' && mail.messages.length) {
      e.preventDefault();
      mail.selectAll();
    }
    if (e.key === 'Delete' && bulk) removeMessages([...mail.selected]);
  }

  // Touch swipe: right archives, left deletes. Only horizontal drags count, vertical ones scroll.
  const SWIPE_RATIO = 0.35;
  function swipe(node: HTMLElement, uid: number) {
    let startX = 0;
    let startY = 0;
    let dx = 0;
    let active = false;
    let decided = false;
    const item = node.parentElement!;

    function down(e: PointerEvent) {
      if (e.pointerType !== 'touch') return;
      startX = e.clientX;
      startY = e.clientY;
      dx = 0;
      active = true;
      decided = false;
    }
    function move(e: PointerEvent) {
      if (!active) return;
      const x = e.clientX - startX;
      const y = e.clientY - startY;
      if (!decided) {
        if (Math.abs(x) < 8 && Math.abs(y) < 8) return;
        decided = true;
        if (Math.abs(y) > Math.abs(x)) {
          active = false;
          return;
        }
        try {
          node.setPointerCapture(e.pointerId);
        } catch {
          // synthetic or already released pointers cannot be captured
        }
      }
      dx = x;
      node.style.transform = `translateX(${dx}px)`;
      item.dataset.swipe = dx > 0 ? 'archive' : 'delete';
    }
    function up() {
      if (!active) return;
      active = false;
      if (!decided) return;
      swiped = true;
      const width = node.offsetWidth;
      if (Math.abs(dx) > width * SWIPE_RATIO) {
        node.style.transition = 'transform 0.15s ease-out';
        node.style.transform = `translateX(${dx > 0 ? width : -width}px)`;
        setTimeout(() => (dx > 0 ? archiveMessages([uid]) : removeMessages([uid])), 150);
      } else {
        node.style.transition = 'transform 0.15s ease-out';
        node.style.transform = '';
        setTimeout(() => {
          node.style.transition = '';
          delete item.dataset.swipe;
        }, 150);
      }
    }
    node.addEventListener('pointerdown', down);
    node.addEventListener('pointermove', move);
    node.addEventListener('pointerup', up);
    node.addEventListener('pointercancel', up);
    return {
      destroy() {
        node.removeEventListener('pointerdown', down);
        node.removeEventListener('pointermove', move);
        node.removeEventListener('pointerup', up);
        node.removeEventListener('pointercancel', up);
      },
    };
  }
</script>

<svelte:window onkeydown={onKey} />

<section class="list" aria-busy={mail.loading}>
  {#if bulk}
    <div class="page-header bulk" role="toolbar" aria-label={t('actions.label')} data-testid="bulk-bar">
      <span class="count" data-testid="bulk-count">{t('list.selected', { count: mail.selected.length })}</span>
      <div class="bulk-actions">
        <button
          type="button"
          class="btn btn-icon"
          title={anyUnread ? t('actions.mark_read') : t('actions.mark_unread')}
          aria-label={anyUnread ? t('actions.mark_read') : t('actions.mark_unread')}
          onclick={() => flagMessages([...mail.selected], 'seen', anyUnread)}
          data-testid="bulk-seen"
        >
          <svg class="icon" viewBox="0 0 16 16" aria-hidden="true">
            <rect x="2" y="3.5" width="12" height="9" rx="1" /><path d="m2.5 4.5 5.5 4 5.5-4" />
          </svg>
        </button>
        <button
          type="button"
          class="btn btn-icon"
          title={anyUnflagged ? t('actions.flag') : t('actions.unflag')}
          aria-label={anyUnflagged ? t('actions.flag') : t('actions.unflag')}
          onclick={() => flagMessages([...mail.selected], 'flagged', anyUnflagged)}
          data-testid="bulk-flag"
        >
          <svg class="icon" viewBox="0 0 16 16" aria-hidden="true"><path d="M4 14V2h8l-2 3 2 3H4" /></svg>
        </button>
        <button
          type="button"
          class="btn btn-icon"
          title={t('actions.archive')}
          aria-label={t('actions.archive')}
          onclick={() => archiveMessages([...mail.selected])}
          data-testid="bulk-archive"
        >
          <svg class="icon" viewBox="0 0 16 16" aria-hidden="true">
            <rect x="2" y="2.5" width="12" height="3" rx=".5" /><path d="M3 5.5V13h10V5.5M6.5 8.5h3" />
          </svg>
        </button>
        <button
          type="button"
          class="btn btn-icon"
          title={t('actions.delete')}
          aria-label={t('actions.delete')}
          onclick={() => removeMessages([...mail.selected])}
          data-testid="bulk-delete"
        >
          <svg class="icon" viewBox="0 0 16 16" aria-hidden="true">
            <path d="M2.5 4h11M6 4V2.5h4V4M4 4l.7 9.5h6.6L12 4M6.5 6.5v4.5M9.5 6.5v4.5" />
          </svg>
        </button>
        <select
          class="input move"
          aria-label={t('actions.move')}
          onchange={(e) => {
            if (e.currentTarget.value) moveMessages([...mail.selected], e.currentTarget.value);
            e.currentTarget.value = '';
          }}
          data-testid="bulk-move"
        >
          <option value="">{t('actions.move_to')}</option>
          {#each targets as f (f.name)}
            <option value={f.name}>{folderLabel(f.name)}</option>
          {/each}
        </select>
        <button
          type="button"
          class="btn btn-icon"
          title={t('list.clear_selection')}
          aria-label={t('list.clear_selection')}
          onclick={() => mail.clearSelection()}
          data-testid="bulk-clear"
        >
          <svg class="icon" viewBox="0 0 16 16" aria-hidden="true"><path d="m4 4 8 8M12 4l-8 8" /></svg>
        </button>
      </div>
    </div>
  {/if}
  <form
    class="search"
    role="search"
    onsubmit={(e) => {
      e.preventDefault();
      onSearch(searchInput.value, true);
    }}
  >
    <svg class="icon" viewBox="0 0 16 16" aria-hidden="true"><circle cx="7" cy="7" r="4.5" /><path d="m10.5 10.5 3 3" /></svg>
    <input
      bind:this={searchInput}
      class="input"
      type="search"
      value={mail.query}
      placeholder={t('search.placeholder')}
      aria-label={t('search.label')}
      oninput={(e) => onSearch(e.currentTarget.value)}
      onkeydown={(e) => {
        if (e.key === 'Escape') {
          e.currentTarget.value = '';
          onSearch('', true);
        }
      }}
      data-testid="search"
    />
  </form>
  <header class="page-header" class:hidden={bulk}>
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
    <p class="state" data-testid="empty">{mail.query ? t('search.no_results') : t('list.empty')}</p>
  {:else}
    <ul data-testid="messages">
      {#each mail.messages as m (m.uid)}
        {@const who = showRecipients ? m.to.map(displayName).join(', ') : displayName(m.from[0])}
        <li
          class:unread={!m.seen}
          class:selected={mail.isSelected(m.uid)}
          data-testid="message"
          data-uid={m.uid}
          data-swipe={undefined}
        >
          <div class="swipe-bg" aria-hidden="true">
            <svg class="icon archive" viewBox="0 0 16 16">
              <rect x="2" y="2.5" width="12" height="3" rx=".5" /><path d="M3 5.5V13h10V5.5M6.5 8.5h3" />
            </svg>
            <svg class="icon trash" viewBox="0 0 16 16">
              <path d="M2.5 4h11M6 4V2.5h4V4M4 4l.7 9.5h6.6L12 4M6.5 6.5v4.5M9.5 6.5v4.5" />
            </svg>
          </div>
          <div class="slide" use:swipe={m.uid}>
            <a
              class="row"
              href={messagePath(mail.folder, m.uid)}
              aria-current={mail.uid === m.uid ? 'true' : undefined}
              onclick={(e) => open(e, m.uid)}
            >
              <span class="dot" aria-hidden="true"></span>
              <span class="who">{who}</span>
              <time datetime={m.date} title={formatFullDate(i18n.locale, m.date)}>
                {formatListDate(i18n.locale, m.date, undefined, settings.value.prettyDates)}
              </time>
              <span class="subject">
                {#if m.flagged}
                  <svg class="icon flag" viewBox="0 0 16 16" role="img" aria-label={t('list.flagged')}>
                    <path d="M4 14V2h8l-2 3 2 3H4" />
                  </svg>
                {/if}
                <span class="text">{m.subject || t('list.no_subject')}</span>
              </span>
              {#if m.attachments}
                <svg class="icon clip" viewBox="0 0 16 16" role="img" aria-label={t('list.attachment')}>
                  <path d="M11 5 6 10a1.5 1.5 0 0 0 2 2l5-5a3 3 0 0 0-4-4L4 8a4.5 4.5 0 0 0 6 6l4-4" />
                </svg>
              {/if}
            </a>
            <div class="hover" data-testid="row-actions">
              <span class="size">{formatSize(i18n.locale, m.size)}</span>
              <button
                type="button"
                class="btn btn-icon"
                aria-pressed={m.flagged}
                aria-label={m.flagged ? t('actions.unflag') : t('actions.flag')}
                title={m.flagged ? t('actions.unflag') : t('actions.flag')}
                onclick={() => flagMessages([m.uid], 'flagged', !m.flagged)}
                data-testid="row-flag"
              >
                <svg class="icon" class:flagged={m.flagged} viewBox="0 0 16 16" aria-hidden="true">
                  <path d="M4 14V2h8l-2 3 2 3H4" />
                </svg>
              </button>
              <button
                type="button"
                class="btn btn-icon"
                aria-label={t('actions.delete')}
                title={t('actions.delete')}
                onclick={() => removeMessages([m.uid])}
                data-testid="row-delete"
              >
                <svg class="icon" viewBox="0 0 16 16" aria-hidden="true">
                  <path d="M2.5 4h11M6 4V2.5h4V4M4 4l.7 9.5h6.6L12 4M6.5 6.5v4.5M9.5 6.5v4.5" />
                </svg>
              </button>
            </div>
          </div>
        </li>
      {/each}
    </ul>
  {/if}
  {#if mail.quota}
    {@const pct = Math.min(100, Math.round((mail.quota.used / mail.quota.limit) * 100))}
    <footer class="quota" data-testid="quota" title={t('quota.title', { percent: pct })}>
      <div
        class="bar"
        role="meter"
        aria-valuemin={0}
        aria-valuemax={100}
        aria-valuenow={pct}
        aria-label={t('quota.label')}
      >
        <div class="fill" class:warn={pct >= 80} class:full={pct >= 95} data-pct={Math.ceil(pct / 5) * 5}></div>
      </div>
      <span>
        {t('quota.usage', {
          used: formatSize(i18n.locale, mail.quota.used),
          limit: formatSize(i18n.locale, mail.quota.limit),
        })}
      </span>
    </footer>
  {/if}
</section>

<style>
  .list {
    display: flex;
    flex-direction: column;
    min-inline-size: 0;
    min-block-size: 0;
    block-size: 100%;
  }
  .page-header {
    flex: none;
    padding-inline: var(--space-3);
  }
  .page-title {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .pager {
    display: flex;
    align-items: center;
    gap: var(--space-1);
    flex: none;
    color: var(--color-text-muted);
    font-size: 0.85rem;
  }
  .pager span {
    margin-inline-end: var(--space-1);
  }
  ul {
    flex: 1;
    min-block-size: 0;
    overflow-y: auto;
    list-style: none;
    margin: 0;
    padding: 0;
  }
  li {
    position: relative;
    overflow: hidden;
    border-block-end: 1px solid var(--color-border);
  }
  .slide {
    position: relative;
    background: var(--color-bg);
    touch-action: pan-y;
  }
  .swipe-bg {
    position: absolute;
    inset: 0;
    display: none;
    align-items: center;
    justify-content: space-between;
    padding-inline: var(--space-4);
    color: #fff;
  }
  li[data-swipe] .swipe-bg {
    display: flex;
  }
  li[data-swipe='archive'] .swipe-bg {
    background: var(--color-accent);
  }
  li[data-swipe='delete'] .swipe-bg {
    background: var(--color-danger);
  }
  li[data-swipe='archive'] .trash,
  li[data-swipe='delete'] .archive {
    visibility: hidden;
  }
  .swipe-bg .icon {
    inline-size: 1.4rem;
    block-size: 1.4rem;
  }
  .hover {
    position: absolute;
    inset-block: 0;
    inset-inline-end: 0;
    display: none;
    align-items: center;
    gap: var(--space-1);
    padding-inline: var(--space-4) var(--space-2);
    background: linear-gradient(to right, transparent, var(--color-bg) var(--space-4));
  }
  .hover .btn-icon {
    inline-size: 1.75rem;
    block-size: 1.75rem;
  }
  .size {
    margin-inline-end: var(--space-1);
    color: var(--color-text-muted);
    font-size: 0.8rem;
    white-space: nowrap;
  }
  .flagged {
    color: var(--color-danger);
    fill: currentColor;
  }
  @media (hover: hover) {
    li:hover .hover,
    li:focus-within .hover {
      display: flex;
    }
  }
  li.selected .slide,
  li.selected .hover {
    background: var(--color-selected);
  }
  li.selected .row {
    border-inline-start-color: var(--color-accent);
  }
  .hidden {
    display: none;
  }
  .quota {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    flex: none;
    padding: var(--space-2) var(--space-3);
    font-size: 0.8rem;
    color: var(--color-text-muted);
    border-block-start: 1px solid var(--color-border);
  }
  .bar {
    flex: 1;
    block-size: 0.375rem;
    overflow: hidden;
    background: var(--color-border);
    border-radius: 999px;
  }
  .fill {
    block-size: 100%;
    background: var(--color-accent);
  }
  .fill.warn {
    background: #d97706;
  }
  .fill.full {
    background: var(--color-danger);
  }
  .fill[data-pct='0'] {
    inline-size: 0%;
  }
  .fill[data-pct='5'] {
    inline-size: 5%;
  }
  .fill[data-pct='10'] {
    inline-size: 10%;
  }
  .fill[data-pct='15'] {
    inline-size: 15%;
  }
  .fill[data-pct='20'] {
    inline-size: 20%;
  }
  .fill[data-pct='25'] {
    inline-size: 25%;
  }
  .fill[data-pct='30'] {
    inline-size: 30%;
  }
  .fill[data-pct='35'] {
    inline-size: 35%;
  }
  .fill[data-pct='40'] {
    inline-size: 40%;
  }
  .fill[data-pct='45'] {
    inline-size: 45%;
  }
  .fill[data-pct='50'] {
    inline-size: 50%;
  }
  .fill[data-pct='55'] {
    inline-size: 55%;
  }
  .fill[data-pct='60'] {
    inline-size: 60%;
  }
  .fill[data-pct='65'] {
    inline-size: 65%;
  }
  .fill[data-pct='70'] {
    inline-size: 70%;
  }
  .fill[data-pct='75'] {
    inline-size: 75%;
  }
  .fill[data-pct='80'] {
    inline-size: 80%;
  }
  .fill[data-pct='85'] {
    inline-size: 85%;
  }
  .fill[data-pct='90'] {
    inline-size: 90%;
  }
  .fill[data-pct='95'] {
    inline-size: 95%;
  }
  .fill[data-pct='100'] {
    inline-size: 100%;
  }
  .search {
    position: relative;
    flex: none;
    padding: var(--space-2) var(--space-3);
    border-block-end: 1px solid var(--color-border);
  }
  .search .icon {
    position: absolute;
    inset-block-start: 50%;
    inset-inline-start: calc(var(--space-3) + 0.625rem);
    transform: translateY(-50%);
    color: var(--color-text-muted);
    pointer-events: none;
  }
  .search .input {
    padding-inline-start: 2rem;
  }
  .bulk {
    background: var(--color-surface);
  }
  .count {
    font-weight: 600;
    white-space: nowrap;
  }
  .bulk-actions {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: var(--space-1);
  }
  .move {
    inline-size: 7rem;
    padding-block: 0.3rem;
  }
  .row {
    display: grid;
    grid-template-columns: 0.5rem minmax(0, 1fr) auto;
    grid-template-areas:
      'dot who date'
      '. subject clip';
    column-gap: var(--space-2);
    row-gap: 2px;
    align-items: center;
    padding-block: 0.625rem;
    padding-inline: var(--space-2) var(--space-3);
    color: var(--color-text-muted);
    text-decoration: none;
    border-inline-start: 3px solid transparent;
  }
  .row:hover {
    background: var(--color-hover);
  }
  .row[aria-current='true'] {
    background: var(--color-selected);
    border-inline-start-color: var(--color-accent);
    box-shadow: inset 0 0 0 1px var(--color-selected-ring);
  }
  .dot {
    grid-area: dot;
    inline-size: 0.5rem;
    block-size: 0.5rem;
    border-radius: 999px;
  }
  .unread .dot {
    background: var(--color-accent);
  }
  .who {
    grid-area: who;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    color: var(--color-text);
  }
  .unread .who {
    font-weight: 600;
    color: var(--color-unread);
  }
  time {
    grid-area: date;
    font-size: 0.8rem;
    white-space: nowrap;
  }
  .unread time {
    color: var(--color-accent);
    font-weight: 600;
  }
  .subject {
    grid-area: subject;
    display: flex;
    align-items: center;
    gap: var(--space-1);
    min-inline-size: 0;
  }
  .subject .text {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .unread .subject {
    color: var(--color-text);
    font-weight: 500;
  }
  .clip {
    grid-area: clip;
  }
  .flag {
    color: var(--color-danger);
    fill: currentColor;
  }
  .state {
    padding: var(--space-4);
    color: var(--color-text-muted);
  }
  .error {
    color: var(--color-danger);
  }
</style>
