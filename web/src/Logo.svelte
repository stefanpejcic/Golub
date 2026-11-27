<!-- SPDX-License-Identifier: GPL-3.0-or-later -->
<script lang="ts">
  let { size = 32 }: { size?: number } = $props();
  const id = $props.id();
</script>

<svg class="logo" width={size} height={size} viewBox="0 0 64 64" aria-hidden="true">
  <mask id="golub-logo-{id}" maskUnits="userSpaceOnUse" x="0" y="0" width="64" height="64">
    <rect width="64" height="64" fill="#fff" />
    <rect x="17" y="21" width="30" height="22" rx="3.5" fill="none" stroke="#000" stroke-width="4.5" />
    <path
      d="M18.5 24.5 32 34.5l13.5-10"
      fill="none"
      stroke="#000"
      stroke-width="4.5"
      stroke-linecap="round"
      stroke-linejoin="round"
    />
  </mask>
  <path
    class="hex"
    d="M32 7 54 19.7v24.6L32 57 10 44.3V19.7Z"
    stroke-width="11"
    stroke-linejoin="round"
    mask="url(#golub-logo-{id})"
  />
</svg>

<style>
  .logo {
    flex: none;
    color: var(--color-logo);
  }
  .hex {
    fill: currentColor;
    stroke: currentColor;
  }
</style>
