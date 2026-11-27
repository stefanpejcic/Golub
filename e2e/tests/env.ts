// SPDX-License-Identifier: GPL-3.0-or-later

import { createHmac, randomBytes } from 'node:crypto';
import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';

const raw = readFileSync(fileURLToPath(new URL('../test.env', import.meta.url)), 'utf8');
export const env: Record<string, string> = Object.fromEntries(
  raw
    .split('\n')
    .filter((l) => l && !l.startsWith('#') && l.includes('='))
    .map((l) => [l.slice(0, l.indexOf('=')), l.slice(l.indexOf('=') + 1)]),
);

export const ALICE = `alice@${env.E2E_DOMAIN}`;
export const PASSWORD = env.E2E_PASSWORD;

const b64url = (b: Buffer | string) => Buffer.from(b).toString('base64url');

// Mirrors what a host app does: HMAC-SHA256 over email|exp|nonce|lang|theme.
export function ssoToken(email: string, opts: { lang?: string; theme?: string; ttl?: number } = {}): string {
  const exp = Math.floor(Date.now() / 1000) + (opts.ttl ?? 30);
  const payload = [email, exp, randomBytes(16).toString('hex'), opts.lang ?? '', opts.theme ?? ''].join('|');
  const mac = createHmac('sha256', env.GOLUB_SSO_SECRET).update(payload).digest();
  return `${b64url(payload)}.${b64url(mac)}`;
}
