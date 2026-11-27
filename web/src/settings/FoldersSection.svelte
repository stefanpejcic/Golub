<!-- SPDX-License-Identifier: GPL-3.0-or-later -->
<script lang="ts">
  import { i18n } from '../lib/locale.svelte';
  import { mail, type Folder } from '../lib/mail.svelte';
  import { toast } from '../lib/toast.svelte';
  import { errorMessage } from '../lib/errors';
  import { ApiError } from '../lib/api';
  import { folderLabel } from '../lib/actions';

  const t = $derived(i18n.t);
  const numbers = $derived(new Intl.NumberFormat(i18n.locale));
  let newName = $state('');
  let renaming = $state<string | null>(null);
  let renameTo = $state('');

  async function run(action: () => Promise<void>, done: string) {
    try {
      await action();
      toast.show(done);
    } catch (e) {
      toast.show(errorMessage(t, e instanceof ApiError ? e.code : 'unknown'), true);
    }
  }

  function create(e: SubmitEvent) {
    e.preventDefault();
    const name = newName.trim();
    if (!name) return;
    run(async () => {
      await mail.createFolder(name);
      newName = '';
    }, t('folders.created'));
  }

  function rename(f: Folder) {
    const to = renameTo.trim();
    if (!to || to === f.name) {
      renaming = null;
      return;
    }
    run(async () => {
      await mail.folderOp('rename', f.name, { to });
      renaming = null;
    }, t('folders.renamed'));
  }

  const protectedFolder = (f: Folder) => f.special === 'inbox';
</script>

<form class="create" onsubmit={create}>
  <input class="input" bind:value={newName} placeholder={t('folders.new_placeholder')} aria-label={t('folders.new')} data-testid="folder-new" />
  <button class="btn btn-primary" type="submit" data-testid="folder-create">{t('folders.create')}</button>
</form>

<table>
  <thead>
    <tr>
      <th>{t('folders.name')}</th>
      <th class="num">{t('folders.messages')}</th>
      <th>{t('folders.subscribed')}</th>
      <th><span class="sr">{t('actions.label')}</span></th>
    </tr>
  </thead>
  <tbody>
    {#each mail.folders as f (f.name)}
      <tr data-testid="folder-row" data-folder={f.name}>
        <td>
          {#if renaming === f.name}
            <input
              class="input"
              bind:value={renameTo}
              onkeydown={(e) => {
                if (e.key === 'Enter') rename(f);
                if (e.key === 'Escape') renaming = null;
              }}
              aria-label={t('folders.rename')}
              data-testid="folder-rename-input"
            />
          {:else}
            {folderLabel(f.name)}
          {/if}
        </td>
        <td class="num">{numbers.format(f.total)}</td>
        <td>
          <input
            type="checkbox"
            checked={f.subscribed}
            disabled={protectedFolder(f) || !f.selectable}
            onchange={(e) =>
              run(() => mail.folderOp('subscribe', f.name, { value: e.currentTarget.checked }), t('settings.saved'))}
            aria-label={t('folders.subscribed')}
            data-testid="folder-subscribe"
          />
        </td>
        <td class="actions">
          {#if renaming === f.name}
            <button class="btn" type="button" onclick={() => rename(f)}>{t('folders.save')}</button>
          {:else if !protectedFolder(f)}
            <button
              class="btn"
              type="button"
              onclick={() => {
                renaming = f.name;
                renameTo = f.name;
              }}
              data-testid="folder-rename">{t('folders.rename')}</button
            >
          {/if}
          {#if f.selectable && f.total > 0}
            <button
              class="btn"
              type="button"
              onclick={() =>
                confirm(t('folders.empty_confirm', { folder: folderLabel(f.name) })) &&
                run(() => mail.folderOp('empty', f.name), t('folders.emptied'))}
              data-testid="folder-empty">{t('folders.empty')}</button
            >
          {/if}
          {#if !protectedFolder(f)}
            <button
              class="btn danger"
              type="button"
              onclick={() =>
                confirm(t('folders.delete_confirm', { folder: folderLabel(f.name) })) &&
                run(() => mail.folderOp('delete', f.name), t('folders.deleted'))}
              data-testid="folder-delete">{t('folders.delete')}</button
            >
          {/if}
        </td>
      </tr>
    {/each}
  </tbody>
</table>

<style>
  .create {
    display: flex;
    gap: var(--space-2);
    max-inline-size: 30rem;
    margin-block-end: var(--space-4);
  }
  table {
    inline-size: 100%;
    border-collapse: collapse;
  }
  th {
    text-align: start;
    font-size: 0.8rem;
    font-weight: 600;
    color: var(--color-text-muted);
  }
  th,
  td {
    padding-block: var(--space-2);
    padding-inline: var(--space-2);
    border-block-end: 1px solid var(--color-border);
  }
  .num {
    text-align: end;
  }
  .actions {
    display: flex;
    justify-content: flex-end;
    gap: var(--space-1);
  }
  .actions .btn {
    padding-block: 0.25rem;
  }
  .danger {
    color: var(--color-danger);
  }
  .sr {
    position: absolute;
    inline-size: 1px;
    block-size: 1px;
    overflow: hidden;
    clip-path: inset(50%);
  }
</style>
