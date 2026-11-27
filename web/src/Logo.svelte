<!-- SPDX-License-Identifier: GPL-3.0-or-later -->
<script lang="ts">
  // The mark is the pigeon head used as the favicon; the mascot is the full carrier pigeon.
  let { size = 32, variant = 'mark' }: { size?: number; variant?: 'mark' | 'mascot' } = $props();
</script>

{#if variant === 'mark'}
  <img class="logo" src="/favicon.svg" width={size} height={size} alt="" aria-hidden="true" draggable="false" />
{:else}
  <img class="logo" src="/golub.svg" width={size} height={size} alt="" aria-hidden="true" draggable="false" />
{/if}

<style>
  .logo {
    flex: none;
    display: block;
  }
</style>
