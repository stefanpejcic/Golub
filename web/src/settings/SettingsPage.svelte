<!-- SPDX-License-Identifier: GPL-3.0-or-later -->
<script lang="ts">
  import { i18n } from '../lib/locale.svelte';
  import { route } from '../lib/route.svelte';
  import { settings } from '../lib/settings.svelte';
  import PrefsSection from './PrefsSection.svelte';
  import FoldersSection from './FoldersSection.svelte';
  import ListEditor from './ListEditor.svelte';
  import About from '../About.svelte';

  const t = $derived(i18n.t);
  const section = $derived(route.settingsSection ?? 'general');
</script>

<div class="page" data-testid="settings">
  <header class="page-header">
    <h1 class="page-title" data-testid="settings-title">{t(`settings.section.${section}`)}</h1>
  </header>
  <div class="body" aria-busy={!settings.loaded}>
    {#if !settings.loaded}
      <span></span>
    {:else if section === 'folders'}
      <FoldersSection />
    {:else if section === 'identities' || section === 'responses'}
      {#key section}
        <ListEditor kind={section} />
      {/key}
    {:else if section === 'about'}
      <About />
    {:else}
      <PrefsSection {section} />
    {/if}
  </div>
</div>

<style>
  .page {
    display: flex;
    flex-direction: column;
    min-block-size: 0;
    block-size: 100%;
  }
  .body {
    flex: 1;
    overflow-y: auto;
    padding: var(--space-4);
  }
</style>
