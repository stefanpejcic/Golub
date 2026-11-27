<!-- SPDX-License-Identifier: GPL-3.0-or-later -->
<script lang="ts">
  import { i18n } from './lib/locale.svelte';

  let {
    label,
    testid,
    start,
    onresize,
    onreset,
    min = 0,
    max = 0,
  }: {
    label: string;
    testid: string;
    start: () => number;
    onresize: (width: number, done: boolean) => void;
    onreset: () => void;
    min?: number;
    max?: number;
  } = $props();
  const t = $derived(i18n.t);

  let origin = 0;
  let base = 0;
  let dragging = $state(false);
  // Logical properties keep this right in RTL layouts later.
  const dir = () => (document.dir === 'rtl' ? -1 : 1);

  function down(e: PointerEvent) {
    if (e.button !== 0) return;
    e.preventDefault();
    origin = e.clientX;
    base = start();
    dragging = true;
    (e.currentTarget as HTMLElement).setPointerCapture(e.pointerId);
  }
  function move(e: PointerEvent) {
    if (dragging) onresize(base + (e.clientX - origin) * dir(), false);
  }
  function up(e: PointerEvent) {
    if (!dragging) return;
    dragging = false;
    onresize(base + (e.clientX - origin) * dir(), true);
  }
  function key(e: KeyboardEvent) {
    const step = e.shiftKey ? 64 : 16;
    if (e.key === 'ArrowLeft' || e.key === 'ArrowRight') {
      e.preventDefault();
      onresize(start() + (e.key === 'ArrowRight' ? step : -step) * dir(), true);
    }
  }
</script>

<!-- A focusable separator is the ARIA window splitter pattern, which Svelte does not recognise. -->
<!-- svelte-ignore a11y_no_noninteractive_tabindex, a11y_no_noninteractive_element_interactions -->
<div
  class="resizer"
  class:dragging
  role="separator"
  aria-orientation="vertical"
  aria-label={label}
  aria-valuemin={min}
  aria-valuemax={max}
  aria-valuenow={Math.round(start())}
  title={t('layout.resize_hint')}
  tabindex="0"
  onpointerdown={down}
  onpointermove={move}
  onpointerup={up}
  onpointercancel={up}
  ondblclick={onreset}
  onkeydown={key}
  data-testid={testid}
></div>

<style>
  .resizer {
    position: absolute;
    z-index: 5;
    inset-block: 0;
    inset-inline-end: -4px;
    inline-size: 8px;
    cursor: col-resize;
    touch-action: none;
  }
  .resizer::after {
    content: '';
    position: absolute;
    inset-block: 0;
    inset-inline-start: 3px;
    inline-size: 2px;
    background: transparent;
    transition: background-color 0.1s;
  }
  .resizer:hover::after,
  .resizer:focus-visible::after,
  .dragging::after {
    background: var(--color-accent);
  }
  .resizer:focus-visible {
    outline: none;
  }
  @media (max-width: 64rem) {
    .resizer {
      display: none;
    }
  }
</style>
