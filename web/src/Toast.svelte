<!-- SPDX-License-Identifier: GPL-3.0-or-later -->
<script lang="ts">
  import { toast } from './lib/toast.svelte';
</script>

<div class="region" role="status" aria-live="polite">
  {#if toast.text}
    <div class="toast" class:error={toast.error} data-testid="toast">{toast.text}</div>
  {/if}
</div>

<style>
  .region {
    position: fixed;
    inset-block-end: var(--space-4);
    inset-inline: 0;
    display: flex;
    justify-content: center;
    pointer-events: none;
    z-index: 10;
  }
  .toast {
    padding-block: var(--space-2);
    padding-inline: var(--space-3);
    color: var(--color-primary-text);
    background: var(--color-primary);
    border-radius: var(--radius);
    box-shadow: var(--shadow);
    font-weight: 500;
  }
  .error {
    color: #fff;
    background: var(--color-danger);
  }
</style>
