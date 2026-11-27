<!-- SPDX-License-Identifier: GPL-3.0-or-later -->
<script lang="ts">
  import { i18n } from '../lib/locale.svelte';
  import { settings } from '../lib/settings.svelte';
  import { toast } from '../lib/toast.svelte';
  import { dialog } from '../lib/dialog.svelte';
  import { keys } from '../lib/keys.svelte';
  import * as pgp from '../lib/pgp';

  const t = $derived(i18n.t);
  let identity = $state('');
  let pass1 = $state('');
  let pass2 = $state('');
  let pasted = $state('');
  let busy = $state(false);
  let fileInput: HTMLInputElement;

  $effect(() => {
    keys.load().catch(() => toast.show(t('errors.unknown'), true));
  });
  $effect(() => {
    if (!identity && settings.value.identities.length) identity = settings.value.identities[0].id;
  });

  // fingerprints read in groups of four, the way other PGP tools show them
  const groups = (fp: string) => fp.toUpperCase().replace(/(.{4})(?=.)/g, '$1 ');
  const created = (iso: string) => new Date(iso).toLocaleDateString(i18n.locale);

  async function create(e: SubmitEvent) {
    e.preventDefault();
    const id = settings.value.identities.find((i) => i.id === identity);
    if (!id) return;
    if (pass1.length < 8) return toast.show(t('pgp.passphrase_short'), true);
    if (pass1 !== pass2) return toast.show(t('pgp.passphrase_mismatch'), true);
    busy = true;
    try {
      await keys.add([await pgp.generate(id.name || id.email, id.email, pass1)]);
      pass1 = pass2 = '';
      toast.show(t('pgp.created'));
    } catch {
      toast.show(t('errors.unknown'), true);
    } finally {
      busy = false;
    }
  }

  async function importText(text: string) {
    busy = true;
    try {
      let { keys: found, unprotected } = await pgp.inspect(text);
      if (!found.length) throw new Error('empty');
      if (unprotected) {
        const pass = await dialog.prompt({ title: t('pgp.protect_title'), text: t('pgp.protect_text'), confirm: t('pgp.protect'), password: true, inputLabel: t('pgp.passphrase') });
        if (!pass || pass.length < 8) {
          if (pass !== null) toast.show(t('pgp.passphrase_short'), true);
          return;
        }
        found = await pgp.protect(text, pass);
      }
      await keys.add(found);
      pasted = '';
      toast.show(t('pgp.imported', { count: found.length }));
    } catch {
      toast.show(t('pgp.not_a_key'), true);
    } finally {
      busy = false;
    }
  }

  async function importFile() {
    const f = fileInput.files?.[0];
    fileInput.value = '';
    if (f) await importText(await f.text());
  }

  function download(name: string, text: string) {
    const a = document.createElement('a');
    a.href = URL.createObjectURL(new Blob([text], { type: 'application/pgp-keys' }));
    a.download = name;
    a.click();
    setTimeout(() => URL.revokeObjectURL(a.href), 1000);
  }

  async function exportPublic(k: pgp.KeyInfo) {
    download(`${k.emails[0] ?? k.fingerprint.slice(-16)}.asc`, k.private ? await pgp.publicPart(k.armored) : k.armored);
  }

  async function remove(k: pgp.KeyInfo) {
    const ok = await dialog.confirm({
      title: t(k.private ? 'pgp.delete_own_confirm' : 'pgp.delete_confirm', { key: k.userIds[0] ?? k.fingerprint }),
      text: k.private ? t('pgp.delete_own_text') : undefined,
      confirm: t('dialog.delete'),
      danger: true,
    });
    if (!ok) return;
    try {
      await keys.remove(k.fingerprint);
      toast.show(t('settings.deleted'));
    } catch {
      toast.show(t('errors.unknown'), true);
    }
  }
</script>

<div class="pgp" data-testid="pgp-settings">
  <p class="hint">{t('pgp.intro')}</p>

  <section>
    <h2>{t('pgp.own')}</h2>
    {#each keys.own as k (k.fingerprint)}
      <div class="key" data-testid="pgp-own-key">
        <div class="who">
          <strong dir="auto">{k.userIds[0] ?? k.emails[0]}</strong>
          <code>{groups(k.fingerprint)}</code>
          <span class="muted">{t('pgp.created_on', { date: created(k.created) })}</span>
        </div>
        <div class="actions">
          <button type="button" class="btn" onclick={() => exportPublic(k)} data-testid="pgp-export-public">{t('pgp.export_public')}</button>
          <button type="button" class="btn" onclick={() => download(`${k.emails[0] ?? 'key'}-private.asc`, k.armored)} data-testid="pgp-export-private">{t('pgp.export_private')}</button>
          <button type="button" class="btn danger" onclick={() => remove(k)} data-testid="pgp-delete">{t('dialog.delete')}</button>
        </div>
      </div>
    {:else}
      <p class="muted">{t('pgp.no_own')}</p>
    {/each}

    <form class="create" onsubmit={create}>
      <h3>{t('pgp.create')}</h3>
      <label class="field"><span>{t('compose.from')}</span>
        <select class="input" bind:value={identity} data-testid="pgp-identity">
          {#each settings.value.identities as id (id.id)}<option value={id.id}>{id.name ? `${id.name} <${id.email}>` : id.email}</option>{/each}
        </select>
      </label>
      <label class="field"><span>{t('pgp.passphrase')}</span><input class="input" type="password" autocomplete="new-password" bind:value={pass1} data-testid="pgp-pass1" /></label>
      <label class="field"><span>{t('pgp.passphrase_repeat')}</span><input class="input" type="password" autocomplete="new-password" bind:value={pass2} data-testid="pgp-pass2" /></label>
      <p class="hint">{t('pgp.passphrase_hint')}</p>
      <button type="submit" class="btn btn-primary" disabled={busy} data-testid="pgp-create">{busy ? t('common.loading') : t('pgp.create')}</button>
    </form>
  </section>

  <section>
    <h2>{t('pgp.others')}</h2>
    {#each keys.others as k (k.fingerprint)}
      <div class="key" data-testid="pgp-other-key">
        <div class="who">
          <strong dir="auto">{k.userIds[0] ?? k.emails[0]}</strong>
          <code>{groups(k.fingerprint)}</code>
        </div>
        <div class="actions">
          <button type="button" class="btn" onclick={() => exportPublic(k)}>{t('pgp.export_public')}</button>
          <button type="button" class="btn danger" onclick={() => remove(k)}>{t('dialog.delete')}</button>
        </div>
      </div>
    {:else}
      <p class="muted">{t('pgp.no_others')}</p>
    {/each}
  </section>

  <section>
    <h2>{t('pgp.import')}</h2>
    <p class="hint">{t('pgp.import_hint')}</p>
    <textarea class="input" rows="5" bind:value={pasted} placeholder={t('pgp.paste_placeholder')} aria-label={t('pgp.import')} data-testid="pgp-paste"></textarea>
    <div class="actions">
      <input bind:this={fileInput} type="file" accept=".asc,.gpg,.key,.pgp,application/pgp-keys" hidden onchange={importFile} />
      <button type="button" class="btn" disabled={busy} onclick={() => fileInput.click()}>{t('pgp.import_file')}</button>
      <button type="button" class="btn btn-primary" disabled={busy || !pasted.trim()} onclick={() => importText(pasted)} data-testid="pgp-import">{t('pgp.import')}</button>
    </div>
  </section>

  <section>
    <button type="button" class="btn" onclick={() => (pgp.lock(), toast.show(t('pgp.locked')))} data-testid="pgp-lock">{t('pgp.lock_now')}</button>
  </section>
</div>

<style>
  .pgp {
    display: flex;
    flex-direction: column;
    gap: var(--space-5, 1.5rem);
    max-inline-size: 48rem;
  }
  h2 {
    margin: 0 0 var(--space-2);
    font-size: 1rem;
  }
  h3 {
    margin: 0;
    font-size: 0.95rem;
  }
  .key {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-2);
    padding: var(--space-3);
    border: 1px solid var(--color-border);
    border-radius: var(--radius);
    margin-block-end: var(--space-2);
  }
  .who {
    display: flex;
    flex-direction: column;
    gap: 2px;
    min-inline-size: 0;
  }
  code {
    font-size: 0.8rem;
    color: var(--color-text-muted);
    overflow-wrap: anywhere;
  }
  .actions {
    display: flex;
    flex-wrap: wrap;
    gap: var(--space-2);
  }
  .create {
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
    max-inline-size: 28rem;
    margin-block-start: var(--space-3);
  }
  .field {
    display: flex;
    flex-direction: column;
    gap: var(--space-1);
  }
  .field > span {
    font-size: 0.85rem;
    color: var(--color-text-muted);
  }
  .hint,
  .muted {
    margin: 0;
    color: var(--color-text-muted);
    font-size: 0.9rem;
  }
  .danger {
    color: var(--color-danger);
  }
  textarea {
    font-family: ui-monospace, monospace;
    font-size: 0.8rem;
    margin-block-end: var(--space-2);
  }
</style>
