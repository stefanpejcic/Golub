<!-- SPDX-License-Identifier: GPL-3.0-or-later -->
<script lang="ts">
  // The mark is the pigeon's head for small sizes; the mascot is the full carrier pigeon.
  let { size = 32, variant = 'mark' }: { size?: number; variant?: 'mark' | 'mascot' } = $props();
</script>

<img
  class="logo"
  src={variant === 'mark' ? '/favicon.svg' : '/golub.svg'}
  width={size}
  height={size}
  alt=""
  aria-hidden="true"
  draggable="false"
/>

<style>
  .logo {
    flex: none;
    display: block;
  }
</style>
