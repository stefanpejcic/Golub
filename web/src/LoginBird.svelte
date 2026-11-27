<!-- SPDX-License-Identifier: GPL-3.0-or-later -->
<script lang="ts">
  // The login mascot peeks over its wing at an empty password field and covers its eye while you type.
  let { mode = 'open', size = 96 }: { mode?: 'open' | 'peek' | 'cover'; size?: number } = $props();
</script>

<svg class="bird" class:covering={mode === 'cover'} class:peeking={mode === 'peek'} width={size} height={size} viewBox="0 0 128 128" aria-hidden="true" data-testid="login-bird">
  <g stroke="#00ADD8" stroke-width="5" stroke-linecap="round">
  <path d="M4 62h16M10 74h14M4 86h18" />
  </g>
  <path d="M36 82 16 72l4 24Z" fill="#007D9C" stroke="#007D9C" stroke-width="5" stroke-linejoin="round" />
  <path d="M30 84c0-22 18-36 40-36 10 0 16-4 18-12 8-12 30-10 32 6 1 8-3 14-8 18 6 6 8 14 8 22 0 22-20 34-46 34-26 0-44-12-44-32Z" fill="#00ADD8" />
  <path d="M78 58c10 8 26 6 34-4" fill="none" stroke="#34D399" stroke-width="6" stroke-linecap="round" />
  <path d="M80 67c10 6 24 4 32-6" fill="none" stroke="#A78BFA" stroke-width="5" stroke-linecap="round" />
  <path d="M38 78c16-10 40-8 52 6-8 14-34 18-50 10-8-4-8-12-2-16Z" fill="#007D9C" />
  <path d="M46 90l10 2M50 96l8 1" stroke="#00ADD8" stroke-width="3" stroke-linecap="round" />
  <circle cx="102" cy="38" r="7.5" fill="#fff" />
  <circle cx="103.5" cy="38.5" r="4" fill="#0b1220" />
  <circle cx="105" cy="37" r="1.3" fill="#fff" />
  <ellipse cx="113" cy="40" rx="3.2" ry="2.4" fill="#fff" />
  <path d="M115 42.5l9 3.5-9 2.5Z" fill="#334155" stroke="#334155" stroke-width="2.5" stroke-linejoin="round" />
  <g transform="rotate(-10 102 96)">
  <rect x="89" y="87" width="26" height="18" rx="2.5" fill="#fff" stroke="#0b1220" stroke-width="2.5" />
  <path d="M90.5 89 102 97l11.5-8" fill="none" stroke="#0b1220" stroke-width="2.5" stroke-linejoin="round" />
  </g>
  <path d="M62 116v6M76 116v6M58 122h8M72 122h8" stroke="#F97316" stroke-width="4" stroke-linecap="round" />
  <g class="cover" data-testid="login-bird-wing">
    <path d="M86 52c1-15 13-26 27-25 9 1 12 8 7 15-6 8-19 13-34 10Z" fill="#007D9C" />
    <path
      d="M95 46c5-6 12-10 19-11M93 41c5-5 10-8 16-9"
      fill="none"
      stroke="#00ADD8"
      stroke-width="2.5"
      stroke-linecap="round"
    />
  </g>
</svg>

<style>
  .bird {
    display: block;
    flex: none;
  }
  .cover {
    transform-box: view-box;
    transform-origin: 103px 40px;
    transform: translate(-40px, 46px) rotate(12deg);
    opacity: 0;
    transition:
      transform 0.35s cubic-bezier(0.3, 1.3, 0.5, 1),
      opacity 0.2s;
  }
  .covering .cover {
    transform: none;
    opacity: 1;
  }
  .peeking .cover {
    transform: translate(2px, 11px) rotate(22deg);
    opacity: 1;
  }
  @media (prefers-reduced-motion: reduce) {
    .cover {
      transition: none;
    }
  }
</style>
