<!-- SPDX-License-Identifier: GPL-3.0-or-later -->
<script lang="ts">
  import type { Snippet } from 'svelte';

  let { label, testid, children }: { label: Snippet; testid: string; children: Snippet<[() => void]> } = $props();
  let open = $state(false);
  let root: HTMLElement;
  const close = () => (open = false);

  function outside(e: PointerEvent) {
    if (open && !root.contains(e.target as Node)) close();
  }
  function onKey(e: KeyboardEvent) {
    if (open && e.key === 'Escape') {
      e.stopPropagation();
      close();
    }
  }
</script>

<svelte:window onpointerdown={outside} onkeydown={onKey} />

<div class="menu" bind:this={root}>
  <button
    type="button"
    class="trigger"
    aria-haspopup="true"
    aria-expanded={open}
    onclick={() => (open = !open)}
    data-testid={testid}
  >
    {@render label()}
    <svg class="icon caret" viewBox="0 0 16 16" aria-hidden="true"><path d="m4 6 4 4 4-4" /></svg>
  </button>
  {#if open}
    <div class="panel" role="menu" data-testid={`${testid}-menu`}>
      {@render children(close)}
    </div>
  {/if}
</div>

<style>
  .menu {
    position: relative;
  }
  .trigger {
    display: inline-flex;
    align-items: center;
    gap: var(--space-1);
    padding-block: 0.25rem;
    padding-inline: var(--space-2);
    font: inherit;
    font-weight: 500;
    color: var(--color-text);
    background: none;
    border: none;
    border-radius: var(--radius);
    cursor: pointer;
  }
  .trigger:hover,
  .trigger[aria-expanded='true'] {
    background: var(--color-hover);
  }
  .caret {
    inline-size: 0.75rem;
    block-size: 0.75rem;
    color: var(--color-text-muted);
  }
  .panel {
    position: absolute;
    inset-block-start: calc(100% + 4px);
    inset-inline-start: 0;
    z-index: 20;
    min-inline-size: 12rem;
    padding: var(--space-1);
    background: var(--color-surface-raised);
    border: 1px solid var(--color-border);
    border-radius: var(--radius);
    box-shadow: 0 10px 15px -3px rgb(0 0 0 / 0.1), 0 4px 6px -4px rgb(0 0 0 / 0.1);
  }
  .panel :global(.item) {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    inline-size: 100%;
    padding: var(--space-2);
    font: inherit;
    text-align: start;
    color: var(--color-text);
    background: none;
    border: none;
    border-radius: 4px;
    cursor: pointer;
  }
  .panel :global(.item:hover) {
    background: var(--color-hover);
  }
  .panel :global(.sep) {
    block-size: 1px;
    margin-block: var(--space-1);
    background: var(--color-border);
  }
</style>
