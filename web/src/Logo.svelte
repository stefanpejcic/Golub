<!-- SPDX-License-Identifier: GPL-3.0-or-later -->
<script lang="ts">
  // The mark is a one-colour pigeon head that follows the theme; the mascot is the full carrier pigeon.
  let { size = 32, variant = 'mark' }: { size?: number; variant?: 'mark' | 'mascot' } = $props();
  const id = $props.id();
</script>

{#if variant === 'mark'}
  <svg class="logo mark" width={size} height={size} viewBox="0 0 64 64" aria-hidden="true">
    <mask id="golub-mark-{id}" maskUnits="userSpaceOnUse" x="0" y="0" width="64" height="64">
      <rect width="64" height="64" fill="#fff" />
      <circle cx="36" cy="25" r="7.5" fill="#000" />
      <circle cx="37.5" cy="25.5" r="3.6" fill="#fff" />
      <path
        d="M17 45c8 6 24 6 33-2M18 53c8 5 23 5 31-2"
        fill="none"
        stroke="#000"
        stroke-width="3.6"
        stroke-linecap="round"
      />
    </mask>
    <g class="shape" mask="url(#golub-mark-{id})">
      <path d="M8 62c0-15 7-24 17-28-5-7-4-17 4-22 10-7 25-3 27 10 1 8-3 14-8 17 8 4 12 12 12 23Z" />
      <path class="beak" d="M54 24.5l9 3.8-9 2.7Z" />
    </g>
  </svg>
{:else}
  <img class="logo" src="/golub.svg" width={size} height={size} alt="" aria-hidden="true" draggable="false" />
{/if}

<style>
  .logo {
    flex: none;
    display: block;
  }
  .mark {
    color: var(--color-logo);
  }
  .shape {
    fill: currentColor;
  }
  .beak {
    stroke: currentColor;
    stroke-width: 2.5;
    stroke-linejoin: round;
  }
</style>
