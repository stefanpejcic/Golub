<!-- SPDX-License-Identifier: GPL-3.0-or-later -->
<script lang="ts">
  import { i18n, LOCALES } from '../lib/locale.svelte';
  import { theme } from '../lib/theme.svelte';
  import { session } from '../lib/session.svelte';
  import { settings, type Settings } from '../lib/settings.svelte';
  import { mail } from '../lib/mail.svelte';
  import { toast } from '../lib/toast.svelte';
  import { errorMessage } from '../lib/errors';
  import { ApiError } from '../lib/api';
  import { THEMES, type Theme } from '../lib/prefs';

  let { section }: { section: string } = $props();
  const t = $derived(i18n.t);
  const s = $derived(settings.value);

  type Field =
    | { key: keyof Settings; kind: 'toggle' }
    | { key: keyof Settings; kind: 'select'; options: (string | number)[]; label?: (v: string | number) => string };

  const seconds = (v: string | number) => (Number(v) === 0 ? t('settings.never') : t('settings.seconds', { count: Number(v) }));

  const fields = $derived<Record<string, Field[]>>({
    general: [
      { key: 'prettyDates', kind: 'toggle' },
      { key: 'refreshInterval', kind: 'select', options: [0, 30, 60, 300, 600], label: seconds },
    ],
    mailbox: [
      { key: 'pageSize', kind: 'select', options: [25, 50, 100, 200] },
      { key: 'markRead', kind: 'select', options: ['immediately', 'delay', 'never'], label: (v) => t(`settings.markRead.${v}`) },
      ...(s.markRead === 'delay'
        ? [{ key: 'markReadDelay', kind: 'select', options: [1, 3, 5, 10, 30], label: seconds } as Field]
        : []),
      { key: 'deleteMode', kind: 'select', options: ['trash', 'permanent'], label: (v) => t(`settings.deleteMode.${v}`) },
      { key: 'emptyTrashOnLogout', kind: 'toggle' },
    ],
    display: [
      { key: 'preferPlain', kind: 'toggle' },
      { key: 'remoteImages', kind: 'select', options: ['ask', 'always'], label: (v) => t(`settings.remoteImages.${v}`) },
      { key: 'showAddresses', kind: 'toggle' },
    ],
    compose: [
      { key: 'composeHtml', kind: 'toggle' },
      { key: 'replyPosition', kind: 'select', options: ['above', 'below'], label: (v) => t(`settings.replyPosition.${v}`) },
      { key: 'forwardMode', kind: 'select', options: ['inline', 'attachment'], label: (v) => t(`settings.forwardMode.${v}`) },
      { key: 'signatureOnReply', kind: 'toggle' },
      { key: 'spellCheck', kind: 'toggle' },
      { key: 'collectRecipients', kind: 'toggle' },
      { key: 'draftAutosave', kind: 'select', options: [0, 30, 60, 180, 300], label: seconds },
    ],
  });

  async function save(changes: Partial<Settings>) {
    try {
      await settings.patch(changes);
      toast.show(t('settings.saved'));
    } catch (e) {
      toast.show(errorMessage(t, e instanceof ApiError ? e.code : 'unknown'), true);
    }
  }

  function setTheme(v: Theme) {
    theme.set(v);
    session.savePrefs({ theme: v });
  }

  function setLocale(v: string) {
    i18n.set(v);
    session.savePrefs({ locale: i18n.locale });
  }

  const specialRoles = ['drafts', 'sent', 'junk', 'trash', 'archive'];
  const selectable = $derived(mail.folders.filter((f) => f.selectable));

  function setSpecial(role: string, name: string) {
    const next = { ...(s.specialFolders ?? {}) };
    if (name) next[role] = name;
    else delete next[role];
    save({ specialFolders: next });
  }
</script>

<div class="form">
  {#if section === 'general'}
    <label class="field">
      <span>{t('settings.language')}</span>
      <select class="input" value={i18n.locale} onchange={(e) => setLocale(e.currentTarget.value)} data-testid="pref-language">
        {#each LOCALES as l (l)}
          <option value={l}>{t(`language.${l}`)}</option>
        {/each}
      </select>
    </label>
    <label class="field">
      <span>{t('settings.theme')}</span>
      <select class="input" value={theme.value} onchange={(e) => setTheme(e.currentTarget.value as Theme)} data-testid="pref-theme">
        {#each THEMES as v (v)}
          <option value={v}>{t(`settings.theme.${v}`)}</option>
        {/each}
      </select>
    </label>
  {/if}

  {#if section === 'special'}
    <p class="hint">{t('settings.special.hint')}</p>
    {#each specialRoles as role (role)}
      <label class="field">
        <span>{t(`folder.${role}`)}</span>
        <select
          class="input"
          value={s.specialFolders?.[role] ?? ''}
          onchange={(e) => setSpecial(role, e.currentTarget.value)}
          data-testid={`pref-special-${role}`}
        >
          <option value="">{t('settings.special.auto')}</option>
          {#each selectable as f (f.name)}
            <option value={f.name}>{f.name}</option>
          {/each}
        </select>
      </label>
    {/each}
  {/if}

  {#each fields[section] ?? [] as f (f.key)}
    {#if f.kind === 'toggle'}
      <label class="field toggle">
        <span>{t(`settings.${f.key}`)}</span>
        <input
          type="checkbox"
          checked={Boolean(s[f.key])}
          onchange={(e) => save({ [f.key]: e.currentTarget.checked })}
          data-testid={`pref-${f.key}`}
        />
      </label>
    {:else}
      <label class="field">
        <span>{t(`settings.${f.key}`)}</span>
        <select
          class="input"
          value={String(s[f.key])}
          onchange={(e) => {
            const raw = e.currentTarget.value;
            save({ [f.key]: typeof s[f.key] === 'number' ? Number(raw) : raw });
          }}
          data-testid={`pref-${f.key}`}
        >
          {#each f.options as o (o)}
            <option value={String(o)}>{f.label ? f.label(o) : o}</option>
          {/each}
        </select>
      </label>
    {/if}
  {/each}
</div>

<style>
  .form {
    display: flex;
    flex-direction: column;
    gap: var(--space-3);
    max-inline-size: 36rem;
  }
  .field {
    display: grid;
    grid-template-columns: minmax(10rem, 1fr) minmax(0, 16rem);
    align-items: center;
    gap: var(--space-3);
    font-weight: 500;
  }
  .toggle input {
    justify-self: start;
    inline-size: 1.1rem;
    block-size: 1.1rem;
    accent-color: var(--color-accent);
  }
  .hint {
    margin: 0;
    color: var(--color-text-muted);
  }
  @media (max-width: 40rem) {
    .field {
      grid-template-columns: 1fr;
    }
  }
</style>
