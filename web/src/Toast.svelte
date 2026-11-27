<!-- SPDX-License-Identifier: GPL-3.0-or-later -->
<script lang="ts">
  import { toast } from './lib/toast.svelte';
</script>

<div class="region" role="status" aria-live="polite">
  {#if toast.text}
    <div class="toast" class:error={toast.error}>
      <span data-testid="toast">{toast.text}</span>
      {#if toast.action}
        {@const action = toast.action}
        <button
          type="button"
          onclick={() => {
            // Take the callback first: {@const} is reactive and becomes null once the toast is dismissed.
            const run = action.run;
            toast.dismiss();
            run();
          }}
          data-testid="toast-action">{action.label}</button
        >
      {/if}
    </div>
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
    display: flex;
    align-items: center;
    gap: var(--space-3);
    pointer-events: auto;
    padding-block: var(--space-2);
    padding-inline: var(--space-3);
    color: var(--color-primary-text);
    background: var(--color-primary);
    border-radius: var(--radius);
    box-shadow: var(--shadow);
    font-weight: 500;
  }
  button {
    padding: 0;
    font: inherit;
    font-weight: 600;
    color: inherit;
    text-decoration: underline;
    background: none;
    border: none;
    cursor: pointer;
  }
  .error {
    color: #fff;
    background: var(--color-danger);
  }
</style>
