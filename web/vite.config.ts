// SPDX-License-Identifier: GPL-3.0-or-later

import { writeFileSync } from 'node:fs';
import { defineConfig } from 'vitest/config';
import { svelte } from '@sveltejs/vite-plugin-svelte';

const outDir = '../internal/ui/dist';

export default defineConfig({
  plugins: [
    svelte(),
    {
      name: 'keep-gitkeep',
      // go:embed needs the directory to exist even before the first build.
      closeBundle() {
        writeFileSync(`${outDir}/.gitkeep`, '');
      },
    },
  ],
  build: {
    outDir,
    emptyOutDir: true,
    assetsInlineLimit: 0,
  },
  server: {
    proxy: { '/api': 'http://localhost:8080', '/healthz': 'http://localhost:8080' },
  },
  test: {
    include: ['src/**/*.test.ts', 'scripts/**/*.test.mjs'],
  },
});
