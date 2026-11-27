// SPDX-License-Identifier: GPL-3.0-or-later

const state = $state({ name: 'Golub', logo: false });

// Picks black or white text by WCAG relative luminance so buttons stay readable on any accent.
export function textOn(hex: string): string {
  const lin = (i: number) => {
    const c = parseInt(hex.slice(i, i + 2), 16) / 255;
    return c <= 0.03928 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4;
  };
  const l = 0.2126 * lin(1) + 0.7152 * lin(3) + 0.0722 * lin(5);
  return l > 0.179 ? '#0d0d0d' : '#ffffff';
}

function apply(accent: string) {
  const root = document.documentElement.style;
  const text = textOn(accent);
  for (const name of ['accent', 'primary']) {
    root.setProperty(`--color-${name}`, accent);
    root.setProperty(`--color-${name}-text`, text);
  }
  root.setProperty('--color-primary-hover', `color-mix(in srgb, ${accent} 85%, ${text})`);
}

export const brand = {
  get name() {
    return state.name;
  },
  get logo() {
    return state.logo;
  },
  async load() {
    try {
      const res = await fetch('/api/brand');
      if (!res.ok) return;
      const b = (await res.json()) as { name: string; accent: string; logo: boolean };
      state.name = b.name || 'Golub';
      state.logo = b.logo;
      document.title = state.name;
      if (b.accent) apply(b.accent);
    } catch {
      // the default brand is fine when the request fails
    }
  },
};
