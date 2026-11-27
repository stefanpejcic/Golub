<!-- SPDX-License-Identifier: GPL-3.0-or-later -->
<script lang="ts">
  import { i18n } from './lib/locale.svelte';
  import { mail, type Folder } from './lib/mail.svelte';
  import { route } from './lib/route.svelte';
  import { folderPath } from './lib/format';

  const t = $derived(i18n.t);
  const folders = $derived(mail.folders.filter((f) => f.selectable && (f.subscribed || f.special === 'inbox')));

  function label(f: Folder) {
    const parts = f.delim ? f.name.split(f.delim) : [f.name];
    const text = f.special ? t(`folder.${f.special}`) : '  '.repeat(Math.min(parts.length - 1, 4)) + parts.at(-1);
    return f.unread > 0 ? `${text} (${f.unread})` : text;
  }

  async function change(e: Event) {
    const name = (e.currentTarget as HTMLSelectElement).value;
    if (route.compose || route.settingsSection) {
      if (await route.go(folderPath(name))) mail.open(name, 1, { replace: true });
      return;
    }
    mail.open(name);
  }
</script>

<select class="input folder-select" value={mail.folder} onchange={change} aria-label={t('layout.folders')} data-testid="folder-select">
  {#each folders as f (f.name)}
    <option value={f.name}>{label(f)}</option>
  {/each}
</select>

<style>
  .folder-select {
    flex: 1;
    min-inline-size: 0;
    font-weight: 600;
  }
</style>
