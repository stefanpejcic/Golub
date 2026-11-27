<!-- SPDX-License-Identifier: GPL-3.0-or-later -->
<script lang="ts">
  import { brand } from './lib/brand.svelte';

  let { size = 32, variant = 'mark' }: { size?: number; variant?: 'mark' | 'mascot' } = $props();
</script>

{#if brand.logo}
  <img class="logo brand" src="/brand/logo" height={size} alt="" aria-hidden="true" draggable="false" />
{:else if variant === 'mark'}
  <img class="logo" src="/favicon.svg" width={size} height={size} alt="" aria-hidden="true" draggable="false" />
{:else}
  <img class="logo" src="/golub.svg" width={size} height={size} alt="" aria-hidden="true" draggable="false" />
{/if}

<style>
  .logo {
    flex: none;
    display: block;
  }
  .brand {
    inline-size: auto;
    max-inline-size: 16rem;
    object-fit: contain;
  }
</style>
