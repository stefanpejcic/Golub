<!-- SPDX-License-Identifier: GPL-3.0-or-later -->
<script lang="ts">
  import { i18n } from '../lib/locale.svelte';
  import { dialog } from '../lib/dialog.svelte';
  import { mail, type Folder } from '../lib/mail.svelte';
  import { toast } from '../lib/toast.svelte';
  import { errorMessage } from '../lib/errors';
  import { ApiError } from '../lib/api';
  import { folderLabel } from '../lib/actions';
  import { formatSize } from '../lib/format';

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

  const hasSizes = $derived(mail.folders.some((f) => f.size !== undefined));
  const totalSize = $derived(mail.folders.reduce((n, f) => n + (f.size ?? 0), 0));
  // Bars show each folder's share of the quota, or of all mail when there is no quota.
  const scale = $derived(mail.quota?.limit ?? totalSize);
  const share = (f: Folder) => (scale > 0 ? Math.min(100, ((f.size ?? 0) / scale) * 100) : 0);
  const bucket = (pct: number) => (pct > 0 ? Math.max(2, Math.ceil(pct / 2) * 2) : 0);
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
      {#if hasSizes}
        <th class="size-col">{t('folders.size')}</th>
      {/if}
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
        {#if hasSizes}
          <td class="size-col" data-testid="folder-size">
            <div class="usage">
              <div class="bar"><div class="fill" data-pct={bucket(share(f))}></div></div>
              <span>{formatSize(i18n.locale, f.size ?? 0)}</span>
            </div>
          </td>
        {/if}
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
        <td><div class="actions">
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
              onclick={async () =>
                (await dialog.confirm({ title: t('folders.empty_confirm', { folder: folderLabel(f.name) }), confirm: t('folders.empty'), danger: true })) &&
                run(() => mail.folderOp('empty', f.name), t('folders.emptied'))}
              data-testid="folder-empty">{t('folders.empty')}</button
            >
          {/if}
          {#if !protectedFolder(f)}
            <button
              class="btn danger"
              type="button"
              onclick={async () =>
                (await dialog.confirm({ title: t('folders.delete_confirm', { folder: folderLabel(f.name) }), confirm: t('folders.delete'), danger: true })) &&
                run(() => mail.folderOp('delete', f.name), t('folders.deleted'))}
              data-testid="folder-delete">{t('folders.delete')}</button
            >
          {/if}
        </div></td>
      </tr>
    {/each}
  </tbody>
  {#if hasSizes}
    <tfoot>
      <tr>
        <td>{t('folders.total')}</td>
        <td class="num">{numbers.format(mail.folders.reduce((n, f) => n + f.total, 0))}</td>
        <td class="size-col">
          {formatSize(i18n.locale, totalSize)}
          {#if mail.quota}{t('folders.of_quota', { limit: formatSize(i18n.locale, mail.quota.limit) })}{/if}
        </td>
        <td colspan="2"></td>
      </tr>
    </tfoot>
  {/if}
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
  tbody tr:last-child td {
    border-block-end: none;
  }
  .num {
    text-align: end;
  }
  .size-col {
    inline-size: 1%;
    white-space: nowrap;
  }
  .usage {
    display: flex;
    align-items: center;
    gap: var(--space-2);
  }
  .usage span {
    min-inline-size: 4.5rem;
    text-align: end;
    font-size: 0.85rem;
    color: var(--color-text-muted);
    white-space: nowrap;
  }
  .bar {
    flex: none;
    inline-size: 6rem;
    block-size: 0.375rem;
    overflow: hidden;
    background: var(--color-border);
    border-radius: 999px;
  }
  .fill {
    block-size: 100%;
    background: var(--color-accent);
  }
  .fill[data-pct='0'] {
    inline-size: 0%;
  }
  .fill[data-pct='2'] {
    inline-size: 2%;
  }
  .fill[data-pct='4'] {
    inline-size: 4%;
  }
  .fill[data-pct='6'] {
    inline-size: 6%;
  }
  .fill[data-pct='8'] {
    inline-size: 8%;
  }
  .fill[data-pct='10'] {
    inline-size: 10%;
  }
  .fill[data-pct='12'] {
    inline-size: 12%;
  }
  .fill[data-pct='14'] {
    inline-size: 14%;
  }
  .fill[data-pct='16'] {
    inline-size: 16%;
  }
  .fill[data-pct='18'] {
    inline-size: 18%;
  }
  .fill[data-pct='20'] {
    inline-size: 20%;
  }
  .fill[data-pct='22'] {
    inline-size: 22%;
  }
  .fill[data-pct='24'] {
    inline-size: 24%;
  }
  .fill[data-pct='26'] {
    inline-size: 26%;
  }
  .fill[data-pct='28'] {
    inline-size: 28%;
  }
  .fill[data-pct='30'] {
    inline-size: 30%;
  }
  .fill[data-pct='32'] {
    inline-size: 32%;
  }
  .fill[data-pct='34'] {
    inline-size: 34%;
  }
  .fill[data-pct='36'] {
    inline-size: 36%;
  }
  .fill[data-pct='38'] {
    inline-size: 38%;
  }
  .fill[data-pct='40'] {
    inline-size: 40%;
  }
  .fill[data-pct='42'] {
    inline-size: 42%;
  }
  .fill[data-pct='44'] {
    inline-size: 44%;
  }
  .fill[data-pct='46'] {
    inline-size: 46%;
  }
  .fill[data-pct='48'] {
    inline-size: 48%;
  }
  .fill[data-pct='50'] {
    inline-size: 50%;
  }
  .fill[data-pct='52'] {
    inline-size: 52%;
  }
  .fill[data-pct='54'] {
    inline-size: 54%;
  }
  .fill[data-pct='56'] {
    inline-size: 56%;
  }
  .fill[data-pct='58'] {
    inline-size: 58%;
  }
  .fill[data-pct='60'] {
    inline-size: 60%;
  }
  .fill[data-pct='62'] {
    inline-size: 62%;
  }
  .fill[data-pct='64'] {
    inline-size: 64%;
  }
  .fill[data-pct='66'] {
    inline-size: 66%;
  }
  .fill[data-pct='68'] {
    inline-size: 68%;
  }
  .fill[data-pct='70'] {
    inline-size: 70%;
  }
  .fill[data-pct='72'] {
    inline-size: 72%;
  }
  .fill[data-pct='74'] {
    inline-size: 74%;
  }
  .fill[data-pct='76'] {
    inline-size: 76%;
  }
  .fill[data-pct='78'] {
    inline-size: 78%;
  }
  .fill[data-pct='80'] {
    inline-size: 80%;
  }
  .fill[data-pct='82'] {
    inline-size: 82%;
  }
  .fill[data-pct='84'] {
    inline-size: 84%;
  }
  .fill[data-pct='86'] {
    inline-size: 86%;
  }
  .fill[data-pct='88'] {
    inline-size: 88%;
  }
  .fill[data-pct='90'] {
    inline-size: 90%;
  }
  .fill[data-pct='92'] {
    inline-size: 92%;
  }
  .fill[data-pct='94'] {
    inline-size: 94%;
  }
  .fill[data-pct='96'] {
    inline-size: 96%;
  }
  .fill[data-pct='98'] {
    inline-size: 98%;
  }
  .fill[data-pct='100'] {
    inline-size: 100%;
  }
  tfoot td {
    font-weight: 600;
    border-block-end: none;
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
