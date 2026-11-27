<!-- SPDX-License-Identifier: GPL-3.0-or-later -->
<script lang="ts">
  import { onDestroy, tick, untrack } from 'svelte';
  import { i18n } from './lib/locale.svelte';
  import { settings, type Identity } from './lib/settings.svelte';
  import { session } from './lib/session.svelte';
  import { mail, type Attachment } from './lib/mail.svelte';
  import { route } from './lib/route.svelte';
  import { dialog } from './lib/dialog.svelte';
  import { toast } from './lib/toast.svelte';
  import { api, ApiError } from './lib/api';
  import { errorMessage } from './lib/errors';
  import { folderPath, formatSize } from './lib/format';
  import {
    escapeHtml,
    forwardHtml,
    forwardPlain,
    isValidRecipient,
    quoteHtml,
    quotePlain,
    withSignature,
    signatureBlock,
    type Quote,
  } from './lib/compose';
  import RecipientInput from './RecipientInput.svelte';
  import Menu from './Menu.svelte';

  let { params }: { params: URLSearchParams } = $props();
  const t = $derived(i18n.t);

  interface Source {
    subject: string;
    to: string[];
    cc: string[];
    bcc: string[];
    inReplyTo?: string;
    references?: string[];
    quote?: Quote;
    text?: string;
    html?: string;
    attachments: Attachment[];
    identity?: string;
  }
  interface Ref {
    folder: string;
    uid: number;
    part: string;
    filename: string;
    size: number;
  }

  const MAX = 25 * 1024 * 1024;
  // The component is keyed on the URL, so reading the parameters once is enough.
  const initial = untrack(() => params);
  const mode = initial.get('mode') ?? 'new';
  const srcFolder = initial.get('folder') ?? '';
  const srcUid = Number(initial.get('uid')) || 0;
  const fromDraft = initial.get('draft') === '1';

  let identityId = $state('');
  let to = $state<string[]>([]);
  let cc = $state<string[]>([]);
  let bcc = $state<string[]>([]);
  let showCc = $state(false);
  let showBcc = $state(false);
  let subject = $state('');
  let html = $state(settings.value.composeHtml);
  let text = $state('');
  let editor = $state<HTMLDivElement>();
  let textarea = $state<HTMLTextAreaElement>();
  let files = $state<File[]>([]);
  let refs = $state<Ref[]>([]);
  let inReplyTo = '';
  let references: string[] = [];
  let draftUid = $state(fromDraft ? srcUid : 0);
  let priority = $state<'normal' | 'high' | 'low'>('normal');
  let receipt = $state(false);
  let dirty = $state(false);
  let busy = $state<'' | 'send' | 'draft'>('');
  let ready = $state(false);
  let savedAt = $state('');
  let fileInput: HTMLInputElement;

  const identities = $derived(settings.value.identities);
  const identity = $derived<Identity | undefined>(identities.find((i) => i.id === identityId));
  const totalSize = $derived(files.reduce((n, f) => n + f.size, 0) + refs.reduce((n, r) => n + r.size, 0));
  const title = $derived(
    { reply: 'compose.title.reply', replyAll: 'compose.title.reply', forward: 'compose.title.forward', edit: 'compose.title.edit' }[
      mode
    ] ?? 'compose.title.new',
  );

  function signatureHtml(id: Identity | undefined): string {
    if (!id?.signature?.trim()) return '';
    const body = id.signatureHtml ? id.signature : escapeHtml(signatureBlock(id.signature)).replace(/\n/g, '<br>');
    return `<div data-golub-signature="">${body}</div>`;
  }

  async function init() {
    const def = identities.find((i) => i.default) ?? identities[0];
    identityId = def?.id ?? '';
    let src: Source | null = null;
    if (mode !== 'new' && srcFolder && srcUid) {
      try {
        src = await api<Source>('GET', `/api/compose/source?mode=${mode}&folder=${encodeURIComponent(srcFolder)}&uid=${srcUid}`);
      } catch (e) {
        toast.show(errorMessage(t, e instanceof ApiError ? e.code : 'unknown'), true);
      }
    }
    if (src?.identity && identities.some((i) => i.id === src!.identity)) identityId = src.identity;
    const id = identities.find((i) => i.id === identityId);
    const sigOn = mode === 'new' || (settings.value.signatureOnReply && mode !== 'edit');
    const position = settings.value.replyPosition;

    if (!src) {
      const prefillTo = initial.get('to');
      const prefillSubject = initial.get('subject');
      if (prefillTo) to = [prefillTo];
      if (prefillSubject) subject = prefillSubject;
    }
    if (src) {
      subject = src.subject;
      to = src.to;
      cc = src.cc;
      bcc = src.bcc;
      showCc = cc.length > 0;
      showBcc = bcc.length > 0;
      inReplyTo = src.inReplyTo ?? '';
      references = src.references ?? [];
      refs = (src.attachments ?? []).map((a) => ({ folder: srcFolder, uid: srcUid, part: a.id, filename: a.filename, size: a.size }));
    }

    const locale = i18n.locale;
    if (mode === 'edit' && src) {
      html = !!src.html && settings.value.composeHtml;
      text = src.text ?? '';
      await tick();
      // eslint-disable-next-line svelte/no-dom-manipulating
      if (html && editor) editor.innerHTML = src.html ?? '';
    } else if (html) {
      let quote = '';
      if (src?.quote) quote = mode === 'forward' ? forwardHtml(t, locale, src.quote, src.subject.replace(/^Fwd: /i, ''), []) : quoteHtml(t, locale, src.quote);
      const sig = sigOn ? signatureHtml(id) : '';
      await tick();
      if (editor) {
        // eslint-disable-next-line svelte/no-dom-manipulating
        editor.innerHTML =
          position === 'below' && quote && mode !== 'forward'
            ? `${quote}<p><br></p>${sig}`
            : `<p><br></p>${sig}${quote ? `<br>${quote}` : ''}`;
      }
    } else {
      let quote = '';
      if (src?.quote) quote = mode === 'forward' ? forwardPlain(t, locale, src.quote, src.subject.replace(/^Fwd: /i, ''), []) : quotePlain(t, locale, src.quote);
      text = withSignature('', sigOn ? (id?.signature ?? '') : '', quote, mode === 'forward' ? 'above' : position);
    }
    ready = true;
    await tick();
    if (!to.length) document.querySelector<HTMLInputElement>('[data-testid=compose-to-input]')?.focus();
    else if (html) editor?.focus();
    else {
      textarea?.focus();
      textarea?.setSelectionRange(0, 0);
    }
  }

  $effect(() => {
    if (settings.loaded && !ready) init();
  });

  function changeIdentity(id: string) {
    const old = identities.find((i) => i.id === identityId);
    identityId = id;
    const next = identities.find((i) => i.id === id);
    if (html && editor) {
      const el = editor.querySelector('[data-golub-signature]');
      const sig = signatureHtml(next);
      if (el) {
        // The editor's content belongs to the user, not to Svelte, so direct edits are expected here.
        if (sig) el.outerHTML = sig;
        else el.remove();
      }
    } else if (old?.signature && text.includes(signatureBlock(old.signature))) {
      text = text.replace(signatureBlock(old.signature), signatureBlock(next?.signature ?? ''));
    }
    dirty = true;
  }

  let keptHtml = '';
  let keptPlain = '';

  function toggleFormat() {
    if (html) {
      keptHtml = editor?.innerHTML ?? '';
      text = editor?.innerText ?? '';
      keptPlain = text;
      html = false;
    } else {
      const body = keptHtml && text === keptPlain ? keptHtml : escapeHtml(text).replace(/\n/g, '<br>');
      keptHtml = '';
      html = true;
      tick().then(() => {
        // eslint-disable-next-line svelte/no-dom-manipulating
        if (editor) editor.innerHTML = body;
      });
    }
    dirty = true;
  }

  function exec(cmd: string, value?: string) {
    editor?.focus();
    document.execCommand(cmd, false, value);
    dirty = true;
  }

  async function addLink() {
    const sel = window.getSelection();
    const range = sel && sel.rangeCount ? sel.getRangeAt(0) : null;
    const url = await dialog.prompt({ title: t('compose.link_prompt'), confirm: t('compose.link_insert'), input: 'https://' });
    if (sel && range) {
      sel.removeAllRanges();
      sel.addRange(range);
    }
    if (url && /^(https?:|mailto:)/i.test(url)) exec('createLink', url);
  }

  function insertResponse(textToInsert: string) {
    if (html) {
      editor?.focus();
      document.execCommand('insertText', false, textToInsert);
    } else if (textarea) {
      const { selectionStart: a, selectionEnd: b } = textarea;
      text = text.slice(0, a) + textToInsert + text.slice(b);
      tick().then(() => textarea?.setSelectionRange(a + textToInsert.length, a + textToInsert.length));
    }
    dirty = true;
  }

  function addFiles(list: FileList | null) {
    if (!list) return;
    const next = [...files, ...Array.from(list)];
    if (next.reduce((n, f) => n + f.size, 0) + refs.reduce((n, r) => n + r.size, 0) > MAX) {
      toast.show(t('errors.compose.too_large'), true);
      return;
    }
    files = next;
    dirty = true;
  }

  function payload() {
    const bodyHtml = html ? (editor?.innerHTML ?? '') : '';
    const bodyText = html ? (editor?.innerText ?? '') : text;
    const meta = {
      identity: identityId,
      to,
      cc: showCc ? cc : [],
      bcc: showBcc ? bcc : [],
      subject,
      text: bodyText,
      html: bodyHtml,
      inReplyTo,
      references,
      attach: refs.map(({ folder, uid, part }) => ({ folder, uid, part })),
      answered: srcUid && (mode === 'reply' || mode === 'replyAll' || mode === 'forward') ? { folder: srcFolder, uid: srcUid, forwarded: mode === 'forward' } : undefined,
      draftUid: draftUid || undefined,
      priority: priority === 'normal' ? undefined : priority,
      receipt: receipt || undefined,
    };
    const form = new FormData();
    form.append('message', JSON.stringify(meta));
    for (const f of files) form.append('attachment', f, f.name);
    return form;
  }

  async function post<T>(path: string): Promise<T> {
    const res = await fetch(path, { method: 'POST', body: payload(), credentials: 'same-origin' });
    const data = await res.json().catch(() => undefined);
    if (!res.ok) {
      const err = new ApiError(data?.error ?? 'server.error', res.status);
      (err as ApiError & { address?: string }).address = data?.address;
      throw err;
    }
    return data as T;
  }

  function leave() {
    dirty = false;
    route.go(folderPath(mail.folder));
    mail.refresh();
  }

  async function send() {
    const all = [...to, ...(showCc ? cc : []), ...(showBcc ? bcc : [])];
    if (!all.length) {
      toast.show(t('errors.compose.no_recipients'), true);
      return;
    }
    const bad = all.find((r) => !isValidRecipient(r));
    if (bad) {
      toast.show(t('errors.compose.invalid_recipient', { address: bad }), true);
      return;
    }
    if (!subject.trim() && !(await dialog.confirm({ title: t('compose.no_subject_confirm'), confirm: t('compose.send_anyway') }))) return;
    busy = 'send';
    try {
      const res = await post<{ savedToSent: boolean }>('/api/compose/send');
      toast.show(res.savedToSent ? t('compose.sent') : t('compose.sent_not_saved'));
      leave();
    } catch (e) {
      const err = e as ApiError & { address?: string };
      toast.show(errorMessage(t, err.code ?? 'unknown').replace('{address}', err.address ?? ''), true);
    } finally {
      busy = '';
    }
  }

  async function saveDraft(silent = false) {
    if (busy) return;
    busy = 'draft';
    try {
      const res = await post<{ uid: number }>('/api/compose/draft');
      draftUid = res.uid;
      dirty = false;
      savedAt = new Intl.DateTimeFormat(i18n.locale, { timeStyle: 'short' }).format(new Date());
      if (!silent) toast.show(t('compose.draft_saved'));
    } catch (e) {
      if (!silent) toast.show(errorMessage(t, e instanceof ApiError ? e.code : 'unknown'), true);
    } finally {
      busy = '';
    }
  }

  let leaveDialog = $state<HTMLDialogElement>();
  let answer: ((v: 'save' | 'discard' | 'cancel') => void) | null = null;

  function askLeave(): Promise<'save' | 'discard' | 'cancel'> {
    leaveDialog?.showModal();
    return new Promise((resolve) => (answer = resolve));
  }

  function choose(v: 'save' | 'discard' | 'cancel') {
    leaveDialog?.close();
    answer?.(v);
    answer = null;
  }

  async function guard(): Promise<boolean> {
    if (!dirty || busy === 'send') return true;
    const v = await askLeave();
    if (v === 'cancel') return false;
    if (v === 'save') {
      await saveDraft(true);
      if (dirty) return false;
    }
    dirty = false;
    return true;
  }

  route.setGuard(guard);

  async function discard() {
    if (await guard()) leave();
  }

  const autosave = settings.value.draftAutosave;
  const timer = autosave > 0 ? setInterval(() => dirty && !busy && saveDraft(true), autosave * 1000) : undefined;
  const beforeUnload = (e: BeforeUnloadEvent) => {
    if (dirty) e.preventDefault();
  };
  addEventListener('beforeunload', beforeUnload);
  onDestroy(() => {
    clearInterval(timer);
    removeEventListener('beforeunload', beforeUnload);
    route.setGuard(null);
  });

  async function onKey(e: KeyboardEvent) {
    // Esc leaves like any other navigation, so unsaved changes still get the save prompt.
    if (e.key === 'Escape' && !e.defaultPrevented && !document.querySelector('dialog[open]')) {
      e.preventDefault();
      if (await route.go(folderPath(mail.folder))) mail.refresh();
      return;
    }
    if ((e.ctrlKey || e.metaKey) && e.key === 'Enter') {
      e.preventDefault();
      send();
    }
    if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 's') {
      e.preventDefault();
      saveDraft();
    }
  }
</script>

<svelte:window onkeydown={onKey} />

<div class="compose" data-testid="compose">
  <header class="page-header">
    <h1 class="page-title">{t(title)}</h1>
    <div class="actions">
      {#if savedAt}<span class="saved" data-testid="compose-saved">{t('compose.saved_at', { time: savedAt })}</span>{/if}
      <button class="btn" type="button" onclick={discard} data-testid="compose-discard">{t('compose.discard')}</button>
      <button class="btn" type="button" onclick={() => saveDraft()} disabled={!!busy} data-testid="compose-save">{t('compose.save')}</button>
      <button class="btn btn-primary" type="button" onclick={send} disabled={!!busy || !ready} data-testid="compose-send">
        <svg class="icon" viewBox="0 0 16 16" aria-hidden="true"><path d="M14 2 7 9M14 2l-4.5 12L7 9 2 6.5Z" /></svg>
        {busy === 'send' ? t('compose.sending') : t('compose.send')}
      </button>
    </div>
  </header>

  <div class="headers">
    <label class="row">
      <span class="label">{t('compose.from')}</span>
      <select class="input from" value={identityId} onchange={(e) => changeIdentity(e.currentTarget.value)} data-testid="compose-from">
        {#each identities as id (id.id)}
          <option value={id.id}>{id.name ? `${id.name} <${id.email}>` : id.email}</option>
        {/each}
      </select>
    </label>
    <div class="with-toggles">
      <RecipientInput bind:value={to} label={t('message.to')} testid="compose-to" />
      <div class="toggles">
        {#if !showCc}<button type="button" class="link" onclick={() => (showCc = true)} data-testid="compose-show-cc">{t('message.cc')}</button>{/if}
        {#if !showBcc}<button type="button" class="link" onclick={() => (showBcc = true)} data-testid="compose-show-bcc">{t('compose.bcc')}</button>{/if}
      </div>
    </div>
    {#if showCc}<RecipientInput bind:value={cc} label={t('message.cc')} testid="compose-cc" />{/if}
    {#if showBcc}<RecipientInput bind:value={bcc} label={t('compose.bcc')} testid="compose-bcc" />{/if}
    <label class="row">
      <span class="label">{t('compose.subject')}</span>
      <input class="plain" bind:value={subject} oninput={() => (dirty = true)} spellcheck={settings.value.spellCheck} data-testid="compose-subject" />
    </label>
  </div>

  <div class="toolbar" role="toolbar" aria-label={t('compose.formatting')}>
    {#if html}
      <button type="button" class="tool" title={t('compose.bold')} aria-label={t('compose.bold')} onclick={() => exec('bold')}>
        <svg class="icon" viewBox="0 0 16 16" aria-hidden="true"><path d="M4.5 2.5h4a2.75 2.75 0 0 1 0 5.5h-4ZM4.5 8h4.75a2.75 2.75 0 0 1 0 5.5H4.5Z" /></svg>
      </button>
      <button type="button" class="tool" title={t('compose.italic')} aria-label={t('compose.italic')} onclick={() => exec('italic')}>
        <svg class="icon" viewBox="0 0 16 16" aria-hidden="true"><path d="M7 2.5h5M4 13.5h5M9.5 2.5l-3 11" /></svg>
      </button>
      <button type="button" class="tool" title={t('compose.underline')} aria-label={t('compose.underline')} onclick={() => exec('underline')}>
        <svg class="icon" viewBox="0 0 16 16" aria-hidden="true"><path d="M4.5 2.5v5a3.5 3.5 0 0 0 7 0v-5M3.5 14h9" /></svg>
      </button>
      <span class="sep"></span>
      <button type="button" class="tool" title={t('compose.bullets')} aria-label={t('compose.bullets')} onclick={() => exec('insertUnorderedList')}>
        <svg class="icon" viewBox="0 0 16 16" aria-hidden="true"><path d="M6 4h8M6 8h8M6 12h8M2.5 4h.01M2.5 8h.01M2.5 12h.01" /></svg>
      </button>
      <button type="button" class="tool" title={t('compose.numbers')} aria-label={t('compose.numbers')} onclick={() => exec('insertOrderedList')}>
        <svg class="icon" viewBox="0 0 16 16" aria-hidden="true"><path d="M6 4h8M6 8h8M6 12h8M2 3l1-.5V5M2 7.5h1.5L2 9.5h1.5" /></svg>
      </button>
      <button type="button" class="tool" title={t('compose.quote')} aria-label={t('compose.quote')} onclick={() => exec('formatBlock', 'blockquote')}>
        <svg class="icon" viewBox="0 0 16 16" aria-hidden="true"><path d="M3 4v8M6 5h7M6 8h7M6 11h5" /></svg>
      </button>
      <button type="button" class="tool" title={t('compose.link')} aria-label={t('compose.link')} onclick={addLink}>
        <svg class="icon" viewBox="0 0 16 16" aria-hidden="true"><path d="M7 9a3 3 0 0 0 4.2 0l2-2a3 3 0 0 0-4.2-4.2l-.8.8M9 7a3 3 0 0 0-4.2 0l-2 2a3 3 0 0 0 4.2 4.2l.8-.8" /></svg>
      </button>
      <button type="button" class="tool" title={t('compose.clear_format')} aria-label={t('compose.clear_format')} onclick={() => exec('removeFormat')}>
        <svg class="icon" viewBox="0 0 16 16" aria-hidden="true"><path d="M4 3h8M8 3l-2 10M3 13l10-10" /></svg>
      </button>
      <span class="sep"></span>
    {/if}
    <button type="button" class="tool text" onclick={toggleFormat} data-testid="compose-format">
      {html ? t('compose.to_plain') : t('compose.to_html')}
    </button>
    {#if settings.value.responses.length}
      <Menu testid="compose-responses">
        {#snippet label()}{t('compose.responses')}{/snippet}
        {#snippet children(close)}
          {#each settings.value.responses as r (r.id)}
            <button class="item" role="menuitem" onclick={() => (insertResponse(r.text), close())}>{r.name}</button>
          {/each}
        {/snippet}
      </Menu>
    {/if}
    <select class="input prio" bind:value={priority} aria-label={t('compose.priority')} data-testid="compose-priority">
      <option value="normal">{t('compose.priority_normal')}</option>
      <option value="high">{t('message.priority_high')}</option>
      <option value="low">{t('message.priority_low')}</option>
    </select>
    <label class="receipt">
      <input type="checkbox" bind:checked={receipt} data-testid="compose-receipt" />
      <span>{t('compose.request_receipt')}</span>
    </label>
    <button type="button" class="tool text attach" onclick={() => fileInput.click()} data-testid="compose-attach">
      <svg class="icon" viewBox="0 0 16 16" aria-hidden="true"><path d="M11 5 6 10a1.5 1.5 0 0 0 2 2l5-5a3 3 0 0 0-4-4L4 8a4.5 4.5 0 0 0 6 6l4-4" /></svg>
      {t('compose.attach')}
    </button>
    <input
      bind:this={fileInput}
      type="file"
      multiple
      hidden
      onchange={(e) => {
        addFiles(e.currentTarget.files);
        e.currentTarget.value = '';
      }}
      data-testid="compose-file"
    />
  </div>

  {#if files.length || refs.length}
    <ul class="files" data-testid="compose-attachments">
      {#each refs as r, i (r.part)}
        <li>
          <span>{r.filename}</span><small>{formatSize(i18n.locale, r.size)}</small>
          <button type="button" aria-label={t('compose.remove', { name: r.filename })} onclick={() => ((refs = refs.filter((_, j) => j !== i)), (dirty = true))}>×</button>
        </li>
      {/each}
      {#each files as f, i (f.name + i)}
        <li>
          <span>{f.name}</span><small>{formatSize(i18n.locale, f.size)}</small>
          <button type="button" aria-label={t('compose.remove', { name: f.name })} onclick={() => ((files = files.filter((_, j) => j !== i)), (dirty = true))}>×</button>
        </li>
      {/each}
      <li class="total">{formatSize(i18n.locale, totalSize)} / {formatSize(i18n.locale, MAX)}</li>
    </ul>
  {/if}

  <div class="body">
    {#if html}
      <div
        bind:this={editor}
        class="editor"
        contenteditable="true"
        spellcheck={settings.value.spellCheck}
        role="textbox"
        aria-multiline="true"
        aria-label={t('compose.body')}
        tabindex="0"
        oninput={() => (dirty = true)}
        data-testid="compose-editor"
      ></div>
    {:else}
      <textarea
        spellcheck={settings.value.spellCheck}
        bind:this={textarea}
        class="editor plain-editor"
        bind:value={text}
        oninput={() => (dirty = true)}
        aria-label={t('compose.body')}
        data-testid="compose-text"
      ></textarea>
    {/if}
  </div>
  <dialog bind:this={leaveDialog} class="modal" aria-labelledby="leave-title" oncancel={(e) => (e.preventDefault(), choose('cancel'))} data-testid="leave-dialog">
    <h2 id="leave-title">{t('compose.leave_title')}</h2>
    <p>{t('compose.leave_text')}</p>
    <div class="modal-buttons">
      <button class="btn" type="button" onclick={() => choose('cancel')} data-testid="leave-cancel">{t('compose.leave_cancel')}</button>
      <button class="btn danger" type="button" onclick={() => choose('discard')} data-testid="leave-discard">{t('compose.discard')}</button>
      <button class="btn btn-primary" type="button" onclick={() => choose('save')} data-testid="leave-save">{t('compose.leave_save')}</button>
    </div>
  </dialog>

  {#if session.info && identity && identity.email.toLowerCase() !== session.info.email.toLowerCase()}
    <p class="hint">{t('compose.identity_hint')}</p>
  {/if}
</div>

<style>
  .compose {
    display: flex;
    flex-direction: column;
    block-size: 100%;
    min-block-size: 0;
  }
  .actions {
    display: flex;
    align-items: center;
    gap: var(--space-2);
  }
  .saved {
    color: var(--color-text-muted);
    font-size: 0.8rem;
  }
  .headers {
    padding-inline: var(--space-4);
  }
  .row {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    padding-block: var(--space-1);
    border-block-end: 1px solid var(--color-border);
  }
  .label {
    flex: none;
    inline-size: 4.5rem;
    color: var(--color-text-muted);
  }
  .from {
    inline-size: auto;
    max-inline-size: 100%;
    padding-block: 0.3rem;
    border-color: transparent;
    box-shadow: none;
    background: transparent;
  }
  .plain {
    flex: 1;
    padding-block: 0.45rem;
    font: inherit;
    color: var(--color-text);
    background: none;
    border: none;
    outline: none;
  }
  .with-toggles {
    position: relative;
  }
  .toggles {
    position: absolute;
    inset-block-start: 0.6rem;
    inset-inline-end: 0;
    display: flex;
    gap: var(--space-2);
  }
  .link {
    padding: 0;
    font: inherit;
    color: var(--color-link);
    background: none;
    border: none;
    cursor: pointer;
  }
  .toolbar {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 2px;
    padding-block: var(--space-1);
    padding-inline: var(--space-3);
    border-block-end: 1px solid var(--color-border);
  }
  .tool {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: var(--space-1);
    min-inline-size: 2rem;
    block-size: 2rem;
    padding-inline: var(--space-2);
    font: inherit;
    color: var(--color-text);
    background: none;
    border: none;
    border-radius: var(--radius);
    cursor: pointer;
  }
  .tool:hover {
    background: var(--color-hover);
  }
  .tool.text {
    font-weight: 500;
  }
  .attach {
    margin-inline-start: auto;
  }
  .prio {
    inline-size: auto;
    margin-inline-start: var(--space-2);
    padding-block: 0.3rem;
  }
  .receipt {
    display: inline-flex;
    align-items: center;
    gap: var(--space-1);
    margin-inline-start: var(--space-2);
    color: var(--color-text-muted);
  }
  .sep {
    inline-size: 1px;
    block-size: 1.25rem;
    margin-inline: var(--space-1);
    background: var(--color-border);
  }
  .files {
    display: flex;
    flex-wrap: wrap;
    gap: var(--space-2);
    margin: 0;
    padding: var(--space-2) var(--space-4);
    list-style: none;
    border-block-end: 1px solid var(--color-border);
  }
  .files li {
    display: inline-flex;
    align-items: center;
    gap: var(--space-2);
    padding-block: 2px;
    padding-inline: var(--space-2) 2px;
    border: 1px solid var(--color-input-border);
    border-radius: 999px;
  }
  .files small,
  .files .total {
    color: var(--color-text-muted);
  }
  .files .total {
    border: none;
    font-size: 0.8rem;
  }
  .files button {
    padding-inline: var(--space-1);
    font: inherit;
    color: var(--color-text-muted);
    background: none;
    border: none;
    cursor: pointer;
  }
  .body {
    flex: 1;
    display: flex;
    min-block-size: 0;
  }
  .editor {
    flex: 1;
    min-block-size: 16rem;
    overflow-y: auto;
    padding: var(--space-4);
    line-height: 1.55;
    color: var(--color-text);
    background: var(--color-bg);
    outline: none;
    overflow-wrap: anywhere;
  }
  .editor :global(blockquote) {
    margin-block: 0;
    margin-inline: var(--space-1) 0;
    padding-inline-start: var(--space-3);
    color: var(--color-text-muted);
    border-inline-start: 3px solid var(--color-border);
  }
  .editor :global(a) {
    color: var(--color-link);
  }
  .plain-editor {
    font: inherit;
    border: none;
    resize: none;
  }
  .danger {
    color: var(--color-danger);
  }
  .hint {
    margin: 0;
    padding: var(--space-2) var(--space-4);
    font-size: 0.8rem;
    color: var(--color-text-muted);
  }
</style>
