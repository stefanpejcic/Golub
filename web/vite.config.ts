// SPDX-License-Identifier: GPL-3.0-or-later

import { readFileSync, readdirSync, statSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';
import { brotliCompressSync, constants, gzipSync } from 'node:zlib';
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
    {
      name: 'precompress',
      // The server sends these to browsers that accept them, so compression costs nothing per request.
      closeBundle() {
        const walk = (dir: string): string[] =>
          readdirSync(dir).flatMap((f) => {
            const p = join(dir, f);
            return statSync(p).isDirectory() ? walk(p) : [p];
          });
        for (const file of walk(outDir)) {
          if (!/\.(js|css|svg|html|json)$/.test(file)) continue;
          const data = readFileSync(file);
          if (data.length < 1024) continue;
          writeFileSync(`${file}.gz`, gzipSync(data, { level: 9 }));
          writeFileSync(`${file}.br`, brotliCompressSync(data, { params: { [constants.BROTLI_PARAM_QUALITY]: 11 } }));
        }
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
