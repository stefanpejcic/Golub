<!-- SPDX-License-Identifier: GPL-3.0-or-later -->
<script lang="ts">
  import { i18n } from './lib/locale.svelte';
  import { route } from './lib/route.svelte';
  import { toast } from './lib/toast.svelte';
  import { dialog } from './lib/dialog.svelte';
  import { ApiError } from './lib/api';
  import { errorMessage } from './lib/errors';
  import { contacts, formatAddress, type Book, type Contact } from './lib/contacts.svelte';

  const t = $derived(i18n.t);
  type View = { kind: 'all' } | { kind: 'group'; name: string } | { kind: 'collected' };
  let view = $state<View>({ kind: 'all' });
  let query = $state('');
  let selectedId = $state<string | null>(null);
  let draft = $state<{ id: string; name: string; emails: string; phones: string; org: string; notes: string; groups: string[] } | null>(null);
  let busy = $state(false);
  let fileInput: HTMLInputElement;

  $effect(() => {
    contacts.load(true).catch(() => toast.show(t('errors.unknown'), true));
  });

  const book = $derived(contacts.book);
  const fold = (s: string) => s.normalize('NFD').replace(/\p{M}/gu, '').toLowerCase().replace(/đ/g, 'd');
  const shown = $derived.by(() => {
    const q = fold(query.trim());
    return book.contacts
      .filter((c) => view.kind !== 'group' || c.groups?.includes(view.name))
      .filter((c) => !q || [c.name, c.org, ...c.emails].some((f) => f && fold(f).includes(q)))
      .sort((a, b) => (a.name || a.emails[0] || '').localeCompare(b.name || b.emails[0] || '', i18n.locale));
  });
  const shownCollected = $derived.by(() => {
    const q = fold(query.trim());
    return book.collected.filter((c) => !q || [c.name, c.email].some((f) => f && fold(f).includes(q)));
  });

  function edit(c: Contact | null) {
    selectedId = c?.id ?? null;
    draft = {
      id: c?.id ?? '',
      name: c?.name ?? '',
      emails: (c?.emails ?? []).join('\n'),
      phones: (c?.phones ?? []).join('\n'),
      org: c?.org ?? '',
      notes: c?.notes ?? '',
      groups: c?.groups ? [...c.groups] : view.kind === 'group' ? [view.name] : [],
    };
  }

  const lines = (s: string) => s.split(/[\n,;]+/).map((x) => x.trim()).filter(Boolean);

  async function persist(next: Book, message: string) {
    busy = true;
    try {
      const saved = await contacts.save(next);
      toast.show(message);
      return saved;
    } catch (e) {
      toast.show(errorMessage(t, e instanceof ApiError ? e.code : 'unknown'), true);
      return null;
    } finally {
      busy = false;
    }
  }

  async function save(e: SubmitEvent) {
    e.preventDefault();
    if (!draft) return;
    const c: Contact = { id: draft.id, name: draft.name, emails: lines(draft.emails), phones: lines(draft.phones), org: draft.org, notes: draft.notes, groups: draft.groups };
    const list = draft.id ? book.contacts.map((x) => (x.id === draft!.id ? c : x)) : [...book.contacts, c];
    const saved = await persist({ ...book, contacts: list }, t('settings.saved'));
    if (saved) edit(draft.id ? (saved.contacts.find((x) => x.id === draft!.id) ?? null) : saved.contacts.at(-1)!);
  }

  async function remove() {
    if (!draft?.id || !(await dialog.confirm({ title: t('contacts.delete_confirm', { name: draft.name || lines(draft.emails)[0] || '' }), confirm: t('dialog.delete'), danger: true }))) return;
    const id = draft.id;
    if (await persist({ ...book, contacts: book.contacts.filter((x) => x.id !== id) }, t('contacts.deleted'))) {
      draft = null;
      selectedId = null;
    }
  }

  function write(values: string[]) {
    route.go(`/compose?${new URLSearchParams({ to: values.join(', ') })}`);
  }

  async function newGroup() {
    const name = (await dialog.prompt({ title: t('contacts.new_group'), confirm: t('contacts.create'), inputLabel: t('contacts.group_name') }))?.trim();
    if (!name) return;
    if (await persist({ ...book, groups: [...book.groups, name] }, t('contacts.group_created'))) view = { kind: 'group', name };
  }

  async function renameGroup(old: string) {
    const name = (await dialog.prompt({ title: t('contacts.rename_group'), confirm: t('folders.rename'), input: old, inputLabel: t('contacts.group_name') }))?.trim();
    if (!name || name === old) return;
    const next: Book = {
      ...book,
      groups: book.groups.map((g) => (g === old ? name : g)),
      contacts: book.contacts.map((c) => ({ ...c, groups: c.groups?.map((g) => (g === old ? name : g)) })),
    };
    if (await persist(next, t('settings.saved'))) view = { kind: 'group', name };
  }

  async function deleteGroup(name: string) {
    if (!(await dialog.confirm({ title: t('contacts.delete_group_confirm', { name }), text: t('contacts.delete_group_text'), confirm: t('dialog.delete'), danger: true }))) return;
    const next: Book = { ...book, groups: book.groups.filter((g) => g !== name), contacts: book.contacts.map((c) => ({ ...c, groups: c.groups?.filter((g) => g !== name) })) };
    if (await persist(next, t('contacts.group_deleted'))) view = { kind: 'all' };
  }

  async function keepCollected(email: string, name?: string) {
    const next: Book = { ...book, contacts: [...book.contacts, { id: '', name: name ?? '', emails: [email] }], collected: book.collected.filter((c) => c.email !== email) };
    const saved = await persist(next, t('contacts.added'));
    if (saved) {
      view = { kind: 'all' };
      edit(saved.contacts.at(-1)!);
    }
  }

  async function forgetCollected(email: string) {
    await persist({ ...book, collected: book.collected.filter((c) => c.email !== email) }, t('contacts.forgotten'));
  }

  async function importFile() {
    const file = fileInput.files?.[0];
    fileInput.value = '';
    if (!file) return;
    busy = true;
    try {
      const r = await contacts.importFile(file);
      toast.show(t('contacts.imported', { added: r.added, updated: r.updated }));
    } catch (e) {
      toast.show(errorMessage(t, e instanceof ApiError ? e.code : 'unknown'), true);
    } finally {
      busy = false;
    }
  }

  const exportUrl = $derived(view.kind === 'group' ? `/api/contacts/export?group=${encodeURIComponent(view.name)}` : '/api/contacts/export');
  const count = (g: string) => book.contacts.filter((c) => c.groups?.includes(g)).length;
</script>

<div class="page" data-testid="contacts">
  <header class="page-header">
    <h1 class="page-title">{t('contacts.title')}</h1>
    <div class="actions">
      <input bind:this={fileInput} type="file" accept=".vcf,text/vcard,text/x-vcard" hidden onchange={importFile} data-testid="contacts-import-file" />
      <button type="button" class="btn" disabled={busy} onclick={() => fileInput.click()} data-testid="contacts-import">{t('contacts.import')}</button>
      <a class="btn" href={exportUrl} download data-testid="contacts-export">{t('contacts.export')}</a>
      <button type="button" class="btn btn-primary" onclick={() => ((view = view.kind === 'collected' ? { kind: 'all' } : view), edit(null))} data-testid="contact-new">{t('contacts.new')}</button>
    </div>
  </header>

  <div class="cols">
    <nav class="groups" aria-label={t('contacts.groups')}>
      <button type="button" class:current={view.kind === 'all'} onclick={() => (view = { kind: 'all' })} data-testid="group-all">
        <span>{t('contacts.all')}</span><span class="n">{book.contacts.length}</span>
      </button>
      <h2>{t('contacts.groups')}</h2>
      {#each book.groups as g (g)}
        <div class="group-row" class:current={view.kind === 'group' && view.name === g}>
          <button type="button" class="name" onclick={() => (view = { kind: 'group', name: g })} data-testid="group" data-group={g}>
            <span>{g}</span><span class="n">{count(g)}</span>
          </button>
          <button type="button" class="tool" aria-label={t('contacts.rename_group')} title={t('contacts.rename_group')} onclick={() => renameGroup(g)} data-testid="group-rename">
            <svg class="icon" viewBox="0 0 16 16" aria-hidden="true"><path d="M10.5 2.5 13.5 5.5 6 13H3v-3Z" /></svg>
          </button>
          <button type="button" class="tool" aria-label={t('contacts.delete_group')} title={t('contacts.delete_group')} onclick={() => deleteGroup(g)} data-testid="group-delete">
            <svg class="icon" viewBox="0 0 16 16" aria-hidden="true"><path d="M3 4.5h10M6.5 4.5V3h3v1.5M4.5 4.5l.5 8.5h6l.5-8.5" /></svg>
          </button>
        </div>
      {/each}
      <button type="button" class="add" onclick={newGroup} data-testid="group-new">+ {t('contacts.new_group')}</button>
      <h2>{t('contacts.other')}</h2>
      <button type="button" class:current={view.kind === 'collected'} onclick={() => ((view = { kind: 'collected' }), (draft = null))} data-testid="group-collected">
        <span>{t('contacts.collected')}</span><span class="n">{book.collected.length}</span>
      </button>
    </nav>

    <section class="list">
      <input class="input" type="search" bind:value={query} placeholder={t('contacts.search')} aria-label={t('contacts.search')} data-testid="contacts-search" />
      {#if view.kind === 'collected'}
        <p class="hint">{t('contacts.collected_hint')}</p>
        <ul>
          {#each shownCollected as c (c.email)}
            <li class="collected" data-testid="collected">
              <div class="who"><span class="name">{c.name || c.email}</span>{#if c.name}<span class="email">{c.email}</span>{/if}</div>
              <button type="button" class="btn" onclick={() => keepCollected(c.email, c.name)} data-testid="collected-keep">{t('contacts.add')}</button>
              <button type="button" class="tool" aria-label={t('contacts.forget')} title={t('contacts.forget')} onclick={() => forgetCollected(c.email)} data-testid="collected-forget">
                <svg class="icon" viewBox="0 0 16 16" aria-hidden="true"><path d="m4 4 8 8M12 4l-8 8" /></svg>
              </button>
            </li>
          {:else}
            <li class="empty">{t('contacts.none')}</li>
          {/each}
        </ul>
      {:else}
        <ul>
          {#each shown as c (c.id)}
            <li>
              <button type="button" class="contact" class:current={selectedId === c.id} onclick={() => edit(c)} data-testid="contact">
                <span class="name">{c.name || c.emails[0] || c.phones?.[0]}</span>
                {#if c.name && c.emails[0]}<span class="email">{c.emails[0]}</span>{/if}
              </button>
            </li>
          {:else}
            <li class="empty">{contacts.loaded ? t('contacts.none') : t('common.loading')}</li>
          {/each}
        </ul>
      {/if}
    </section>

    <section class="editor">
      {#if draft}
        <form onsubmit={save} data-testid="contact-form">
          <label class="field"><span>{t('contacts.name')}</span><input class="input" bind:value={draft.name} data-testid="contact-name" /></label>
          <label class="field"><span>{t('contacts.emails')}</span><textarea class="input" rows="3" bind:value={draft.emails} placeholder={t('contacts.one_per_line')} data-testid="contact-emails"></textarea></label>
          <label class="field"><span>{t('contacts.phones')}</span><textarea class="input" rows="2" bind:value={draft.phones} placeholder={t('contacts.one_per_line')} data-testid="contact-phones"></textarea></label>
          <label class="field"><span>{t('contacts.org')}</span><input class="input" bind:value={draft.org} data-testid="contact-org" /></label>
          <label class="field"><span>{t('contacts.notes')}</span><textarea class="input" rows="4" bind:value={draft.notes} data-testid="contact-notes"></textarea></label>
          {#if book.groups.length}
            <fieldset class="field">
              <legend>{t('contacts.groups')}</legend>
              <div class="checks">
                {#each book.groups as g (g)}
                  <label class="check"><input type="checkbox" value={g} bind:group={draft.groups} data-testid="contact-group" /><span>{g}</span></label>
                {/each}
              </div>
            </fieldset>
          {/if}
          <div class="buttons">
            {#if draft.id}
              <button type="button" class="btn danger" disabled={busy} onclick={remove} data-testid="contact-delete">{t('dialog.delete')}</button>
            {/if}
            {#if lines(draft.emails).length}
              <button type="button" class="btn" onclick={() => write(lines(draft!.emails).slice(0, 1).map((e) => formatAddress(draft!.name, e)))} data-testid="contact-write">{t('contacts.write')}</button>
            {/if}
            <button type="submit" class="btn btn-primary" disabled={busy} data-testid="contact-save">{t('folders.save')}</button>
          </div>
        </form>
      {:else if view.kind === 'group' && count(view.name) > 0}
        <div class="placeholder">
          <button type="button" class="btn" onclick={() => write(shown.filter((c) => c.emails.length).map((c) => formatAddress(c.name, c.emails[0])))} data-testid="group-write">{t('contacts.write_group', { name: view.kind === 'group' ? view.name : '' })}</button>
        </div>
      {:else}
        <p class="placeholder muted">{t('contacts.pick')}</p>
      {/if}
    </section>
  </div>
</div>

<style>
  .page {
    display: flex;
    flex-direction: column;
    block-size: 100%;
    min-block-size: 0;
  }
  .actions {
    display: flex;
    flex-wrap: wrap;
    gap: var(--space-2);
    margin-inline-start: auto;
  }
  .cols {
    display: grid;
    flex: 1;
    grid-template-columns: 13rem minmax(14rem, 22rem) minmax(0, 1fr);
    min-block-size: 0;
    border-block-start: 1px solid var(--color-border);
  }
  .groups,
  .list,
  .editor {
    min-block-size: 0;
    overflow-y: auto;
    padding: var(--space-3);
  }
  .groups,
  .list {
    border-inline-end: 1px solid var(--color-border);
  }
  .groups h2 {
    margin: var(--space-4) 0 var(--space-1);
    padding-inline: var(--space-2);
    color: var(--color-text-muted);
    font-size: 0.75rem;
    font-weight: 600;
    text-transform: uppercase;
  }
  .groups button,
  .contact {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    inline-size: 100%;
    padding: var(--space-2);
    font: inherit;
    color: var(--color-text);
    text-align: start;
    background: none;
    border: none;
    border-radius: var(--radius);
    cursor: pointer;
  }
  .groups button:hover,
  .contact:hover {
    background: var(--color-hover);
  }
  .groups .current,
  .groups button.current,
  .contact.current {
    background: var(--color-selected);
  }
  .n {
    margin-inline-start: auto;
    color: var(--color-text-muted);
    font-size: 0.85rem;
  }
  .group-row {
    display: flex;
    align-items: center;
    border-radius: var(--radius);
  }
  .group-row .name {
    flex: 1;
    min-inline-size: 0;
  }
  .group-row .tool {
    display: none;
    inline-size: auto;
    padding: var(--space-1);
  }
  .group-row:hover .tool,
  .group-row:focus-within .tool {
    display: grid;
  }
  .add {
    color: var(--color-text-muted) !important;
  }
  .list ul {
    margin: var(--space-2) 0 0;
    padding: 0;
    list-style: none;
  }
  .contact {
    flex-direction: column;
    align-items: flex-start;
    gap: 0;
  }
  .name {
    font-weight: 500;
  }
  .email {
    color: var(--color-text-muted);
    font-size: 0.85rem;
  }
  .collected {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    padding: var(--space-2);
  }
  .who {
    display: flex;
    flex: 1;
    flex-direction: column;
    min-inline-size: 0;
  }
  .tool {
    display: grid;
    place-items: center;
    padding: var(--space-1);
    color: var(--color-text-muted);
    background: none;
    border: none;
    border-radius: var(--radius);
    cursor: pointer;
  }
  .tool:hover {
    color: var(--color-text);
    background: var(--color-hover);
  }
  .hint,
  .empty,
  .muted {
    color: var(--color-text-muted);
    font-size: 0.9rem;
  }
  .empty {
    padding: var(--space-2);
  }
  form {
    display: flex;
    flex-direction: column;
    gap: var(--space-3);
    max-inline-size: 36rem;
  }
  .field {
    display: flex;
    flex-direction: column;
    gap: var(--space-1);
    margin: 0;
    padding: 0;
    border: none;
  }
  .field > span,
  legend {
    color: var(--color-text-muted);
    font-size: 0.85rem;
  }
  .checks {
    display: flex;
    flex-wrap: wrap;
    gap: var(--space-2) var(--space-4);
  }
  .check {
    display: inline-flex;
    align-items: center;
    gap: var(--space-1);
  }
  .buttons {
    display: flex;
    flex-wrap: wrap;
    justify-content: flex-end;
    gap: var(--space-2);
  }
  .buttons .danger {
    margin-inline-end: auto;
    color: var(--color-danger);
  }
  .placeholder {
    display: grid;
    place-items: center;
    block-size: 100%;
    margin: 0;
  }
  @media (max-width: 60rem) {
    .cols {
      grid-template-columns: minmax(0, 1fr);
      overflow-y: auto;
    }
    .groups,
    .list {
      overflow: visible;
      border-inline-end: none;
      border-block-end: 1px solid var(--color-border);
    }
    .editor {
      overflow: visible;
    }
  }
</style>
