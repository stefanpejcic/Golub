<!-- SPDX-License-Identifier: GPL-3.0-or-later -->
<script lang="ts">
  import { i18n } from './lib/locale.svelte';
  import type { PreviewKind } from './lib/preview';

  let {
    url,
    filename,
    kind,
    onclose,
  }: { url: string; filename: string; kind: Exclude<PreviewKind, null>; onclose: () => void } = $props();

  const t = $derived(i18n.t);
  let el = $state<HTMLDialogElement>();
  let text = $state<string | null>(null);
  let failed = $state(false);

  $effect(() => {
    el?.showModal();
  });

  $effect(() => {
    if (kind !== 'text') return;
    const ctl = new AbortController();
    fetch(`${url}&inline=1`, { signal: ctl.signal })
      .then((r) => (r.ok ? r.text() : Promise.reject(new Error(String(r.status)))))
      .then((body) => (text = body))
      .catch(() => !ctl.signal.aborted && (failed = true));
    return () => ctl.abort();
  });
</script>

<dialog
  bind:this={el}
  class="modal preview"
  aria-labelledby="preview-title"
  oncancel={(e) => (e.preventDefault(), onclose())}
  onclick={(e) => e.target === el && onclose()}
  data-testid="attachment-preview"
>
  <header>
    <h2 id="preview-title">{filename}</h2>
    <a class="btn" href={url} download={filename} data-testid="preview-download">{t('message.download')}</a>
    <button type="button" class="btn btn-icon" aria-label={t('common.close')} onclick={onclose} data-testid="preview-close">
      <svg class="icon" viewBox="0 0 16 16" aria-hidden="true"><path d="M4 4l8 8M12 4l-8 8" /></svg>
    </button>
  </header>
  <div class="body">
    {#if kind === 'image'}
      <img src={`${url}&inline=1`} alt={filename} data-testid="preview-image" />
    {:else if kind === 'pdf'}
      <iframe src={`${url}&inline=1`} title={filename} data-testid="preview-pdf"></iframe>
    {:else if failed}
      <p class="muted">{t('errors.unknown')}</p>
    {:else if text === null}
      <p class="muted">{t('common.loading')}</p>
    {:else}
      <pre data-testid="preview-text">{text}</pre>
    {/if}
  </div>
</dialog>

<style>
  .preview {
    display: flex;
    flex-direction: column;
    inline-size: min(100% - 2rem, 64rem);
    block-size: min(100% - 2rem, 52rem);
    padding: 0;
  }
  .preview:not([open]) {
    display: none;
  }
  header {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    padding: var(--space-3) var(--space-4);
    border-block-end: 1px solid var(--color-border);
  }
  h2 {
    flex: 1;
    min-inline-size: 0;
    margin: 0;
    overflow: hidden;
    font-size: 1rem;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .body {
    display: flex;
    flex: 1;
    min-block-size: 0;
    overflow: auto;
    background: var(--color-surface);
  }
  img {
    max-inline-size: 100%;
    max-block-size: 100%;
    margin: auto;
    object-fit: contain;
  }
  iframe {
    flex: 1;
    border: none;
  }
  pre {
    flex: 1;
    margin: 0;
    padding: var(--space-4);
    font-size: 0.85rem;
    white-space: pre-wrap;
    overflow-wrap: anywhere;
  }
  .muted {
    margin: auto;
    color: var(--color-text-muted);
  }
</style>
