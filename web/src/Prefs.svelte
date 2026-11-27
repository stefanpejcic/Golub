<!-- SPDX-License-Identifier: GPL-3.0-or-later -->
<script lang="ts">
  import { i18n } from './lib/locale.svelte';
  import { session } from './lib/session.svelte';
  import { settings } from './lib/settings.svelte';
  import { mail } from './lib/mail.svelte';
  import { api } from './lib/api';
  import ThemeSwitch from './ThemeSwitch.svelte';

  const t = $derived(i18n.t);

  async function emptyTrash() {
    if (!settings.value.emptyTrashOnLogout) return;
    const name = settings.value.specialFolders?.trash || mail.folders.find((f) => f.special === 'trash')?.name;
    if (name) await api('POST', '/api/folders/empty', { name });
  }

</script>

<div class="prefs">
  {#if session.status === 'authenticated'}
    <button type="button" class="link" onclick={() => session.logout(emptyTrash)} data-testid="logout">
      <svg class="icon" viewBox="0 0 16 16" aria-hidden="true"><path d="M6 2.5H3.5v11H6M10 5l3 3-3 3M13 8H6" /></svg>
      {t('auth.logout')}
    </button>
  {/if}
  <ThemeSwitch />
</div>

<style>
  .prefs {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-3);
    padding: var(--space-3);
  }
  .prefs > :global(.themes) {
    margin-inline-start: auto;
  }
  .link {
    text-decoration: none;
    display: inline-flex;
    align-items: center;
    gap: var(--space-1);
    padding: 0;
    font: inherit;
    color: var(--color-text-muted);
    background: none;
    border: none;
    cursor: pointer;
    white-space: nowrap;
  }
  .link:hover {
    color: var(--color-text);
  }
</style>
