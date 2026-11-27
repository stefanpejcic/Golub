<!-- SPDX-License-Identifier: GPL-3.0-or-later -->
<script lang="ts">
  import { i18n } from '../lib/locale.svelte';

  const t = $derived(i18n.t);
  const mac = typeof navigator !== 'undefined' && /Mac|iPhone|iPad/.test(navigator.platform);
  const mod = mac ? '⌘' : 'Ctrl';

  const groups = $derived([
    {
      title: 'shortcuts.group.list',
      items: [
        { keys: ['↓'], label: 'shortcuts.next' },
        { keys: ['↑'], label: 'shortcuts.previous' },
        { keys: [mod, t('shortcuts.click')], label: 'shortcuts.toggle_select' },
        { keys: ['Shift', t('shortcuts.click')], label: 'shortcuts.range_select' },
        { keys: [mod, 'A'], label: 'shortcuts.select_all' },
        { keys: ['Esc'], label: 'shortcuts.clear' },
      ],
    },
    {
      title: 'shortcuts.group.message',
      items: [
        { keys: ['Delete'], label: 'shortcuts.delete' },
        { keys: ['A'], label: 'shortcuts.archive' },
        { keys: ['S'], label: 'shortcuts.spam' },
      ],
    },
    {
      title: 'shortcuts.group.compose',
      items: [
        { keys: [mod, 'Enter'], label: 'shortcuts.send' },
        { keys: [mod, 'S'], label: 'shortcuts.save' },
        { keys: [mod, 'B'], label: 'compose.bold' },
        { keys: [mod, 'I'], label: 'compose.italic' },
        { keys: [mod, 'U'], label: 'compose.underline' },
      ],
    },
  ]);
</script>

<div class="shortcuts" data-testid="shortcuts">
  {#each groups as g (g.title)}
    <section>
      <h2>{t(g.title)}</h2>
      <dl>
        {#each g.items as item (item.label)}
          <dt>
            {#each item.keys as k, i (i)}
              {#if i > 0}<span class="plus">+</span>{/if}<kbd>{k}</kbd>
            {/each}
          </dt>
          <dd>{t(item.label)}</dd>
        {/each}
      </dl>
    </section>
  {/each}
</div>

<style>
  .shortcuts {
    display: flex;
    flex-direction: column;
    gap: var(--space-4);
    max-inline-size: 36rem;
  }
  h2 {
    margin: 0 0 var(--space-2);
    font-size: 0.8rem;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.04em;
    color: var(--color-text-muted);
  }
  dl {
    display: grid;
    grid-template-columns: minmax(9rem, auto) 1fr;
    gap: var(--space-2) var(--space-4);
    margin: 0;
  }
  dt {
    display: flex;
    align-items: center;
    gap: var(--space-1);
  }
  dd {
    margin: 0;
    align-self: center;
  }
  kbd {
    min-inline-size: 1.75rem;
    padding: 0.15rem 0.45rem;
    font: inherit;
    font-size: 0.8rem;
    text-align: center;
    background: var(--color-input-bg);
    border: 1px solid var(--color-border);
    border-block-end-width: 2px;
    border-radius: 0.4rem;
  }
  .plus {
    color: var(--color-text-muted);
    font-size: 0.8rem;
  }
</style>
