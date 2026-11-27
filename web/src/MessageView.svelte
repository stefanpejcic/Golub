<!-- SPDX-License-Identifier: GPL-3.0-or-later -->
<script lang="ts">
  import { i18n } from './lib/locale.svelte';
  import { mail } from './lib/mail.svelte';
  import { formatFullDate, formatSize, type Address } from './lib/format';
  import { errorMessage } from './lib/errors';

  const t = $derived(i18n.t);
  const m = $derived(mail.message);

  let showImages = $state(false);
  $effect(() => {
    void mail.uid;
    showImages = false;
  });

  const query = $derived(m ? `folder=${encodeURIComponent(m.folder)}&uid=${m.uid}` : '');
  const bodyUrl = $derived(`/api/message/body?${query}${showImages ? '&images=1' : ''}`);

  function people(list: Address[]): string {
    return list.map((a) => (a.name ? `${a.name} <${a.address}>` : a.address)).join(', ');
  }
</script>

<article class="message" aria-busy={mail.messageLoading} data-testid="message-view">
  <div class="toolbar">
    <button type="button" onclick={() => mail.closeMessage()} data-testid="back">
      <svg viewBox="0 0 16 16" aria-hidden="true"><path d="M10 3 5 8l5 5" /></svg>
      {t('message.back')}
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
        <button type="button" onclick={() => (showImages = true)} data-testid="show-images">
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
                <svg viewBox="0 0 16 16" aria-hidden="true">
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
  .message {
    display: flex;
    flex-direction: column;
    min-inline-size: 0;
    min-block-size: 0;
  }
  .toolbar {
    display: flex;
    gap: var(--space-2);
    padding-block: var(--space-2);
    padding-inline: var(--space-3);
    border-block-end: 1px solid var(--color-border);
  }
  button {
    display: inline-flex;
    align-items: center;
    gap: var(--space-1);
    padding-block: var(--space-1);
    padding-inline: var(--space-2);
    background: none;
    color: var(--color-text);
    border: 1px solid var(--color-border);
    border-radius: var(--radius);
    font: inherit;
    cursor: pointer;
  }
  svg {
    inline-size: 1rem;
    block-size: 1rem;
    fill: none;
    stroke: currentColor;
    stroke-width: 1.5;
    stroke-linecap: round;
    stroke-linejoin: round;
  }
  header {
    padding-block: var(--space-3) var(--space-2);
    padding-inline: var(--space-3);
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
    margin-inline: var(--space-3);
    margin-block-end: var(--space-2);
    padding: var(--space-2) var(--space-3);
    background: var(--color-surface);
    border: 1px solid var(--color-border);
    border-radius: var(--radius);
    font-size: 0.9rem;
  }
  .attachments ul {
    display: flex;
    flex-wrap: wrap;
    gap: var(--space-2);
    list-style: none;
    margin: 0;
    padding-block: 0 var(--space-2);
    padding-inline: var(--space-3);
  }
  .attachments a {
    display: inline-flex;
    align-items: center;
    gap: var(--space-2);
    padding-block: var(--space-1);
    padding-inline: var(--space-2);
    color: var(--color-text);
    text-decoration: none;
    border: 1px solid var(--color-border);
    border-radius: var(--radius);
  }
  .attachments a:hover {
    background: var(--color-surface);
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
    padding: var(--space-3);
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
