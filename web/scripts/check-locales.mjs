// SPDX-License-Identifier: GPL-3.0-or-later
// Fails CI on locale drift, unused keys, incomplete plurals and raw strings in templates.

import { readFileSync, readdirSync, statSync } from 'node:fs';
import { join, relative, basename } from 'node:path';
import { fileURLToPath } from 'node:url';
import { parse } from 'svelte/compiler';

const SOURCE = 'en';
const TEXT_ATTRS = new Set(['placeholder', 'title', 'alt', 'aria-label', 'label']);

export function compareKeys(catalogs) {
  const base = Object.keys(catalogs[SOURCE]);
  const problems = [];
  for (const [locale, cat] of Object.entries(catalogs)) {
    if (locale === SOURCE) continue;
    for (const k of base) if (!(k in cat)) problems.push(`${locale}: missing key "${k}"`);
    for (const k of Object.keys(cat)) if (!base.includes(k)) problems.push(`${locale}: unknown key "${k}"`);
  }
  return problems;
}

export function findUnused(keys, sources) {
  const text = sources.join('\n');
  const prefixes = [...text.matchAll(/`([\w.-]+\.)\$\{/g)].map((m) => m[1]);
  return keys.filter(
    (k) => !new RegExp(`['"\`]${k.replace(/[.-]/g, '\\$&')}['"\`]`).test(text) && !prefixes.some((p) => k.startsWith(p)),
  );
}

export function checkPlurals(catalogs) {
  const problems = [];
  for (const [locale, cat] of Object.entries(catalogs)) {
    const needed = new Intl.PluralRules(locale).resolvedOptions().pluralCategories;
    for (const [k, v] of Object.entries(cat)) {
      if (typeof v !== 'object') continue;
      for (const form of needed) if (!(form in v)) problems.push(`${locale}: "${k}" lacks plural form "${form}"`);
    }
  }
  return problems;
}

export function findRawStrings(source, file) {
  const ast = parse(source, { modern: true });
  const lineOf = (pos) => source.slice(0, pos).split('\n').length;
  const problems = [];
  const hasWords = (s) => /\p{L}/u.test(s);
  const walk = (node) => {
    if (!node || typeof node !== 'object') return;
    if (Array.isArray(node)) return node.forEach(walk);
    if (node.type === 'Text' && hasWords(node.data)) {
      problems.push(`${file}:${lineOf(node.start)} raw text "${node.data.trim()}"`);
    }
    if (node.type === 'Attribute' && TEXT_ATTRS.has(node.name) && Array.isArray(node.value)) {
      for (const part of node.value) {
        if (part.type === 'Text' && hasWords(part.data)) {
          problems.push(`${file}:${lineOf(node.start)} raw ${node.name} "${part.data.trim()}"`);
        }
      }
    }
    for (const [key, child] of Object.entries(node)) {
      if (key !== 'parent' && key !== 'value' && child && typeof child === 'object') walk(child);
    }
  };
  walk(ast.fragment);
  return problems;
}

function files(dir, ext) {
  return readdirSync(dir).flatMap((name) => {
    const p = join(dir, name);
    if (statSync(p).isDirectory()) return files(p, ext);
    return ext.some((e) => p.endsWith(e)) && !p.endsWith('.test.ts') ? [p] : [];
  });
}

function main() {
  const root = fileURLToPath(new URL('..', import.meta.url));
  const src = join(root, 'src');
  const localeDir = join(src, 'locales');
  const catalogs = Object.fromEntries(
    readdirSync(localeDir)
      .filter((f) => f.endsWith('.json'))
      .map((f) => [basename(f, '.json'), JSON.parse(readFileSync(join(localeDir, f), 'utf8'))]),
  );
  const codeFiles = files(src, ['.ts', '.svelte']);
  const sources = codeFiles.map((f) => readFileSync(f, 'utf8'));

  const problems = [
    ...compareKeys(catalogs),
    ...checkPlurals(catalogs),
    ...findUnused(Object.keys(catalogs[SOURCE]), sources).map((k) => `unused key "${k}"`),
    ...codeFiles.flatMap((f, i) => (f.endsWith('.svelte') ? findRawStrings(sources[i], relative(root, f)) : [])),
  ];
  if (problems.length) {
    console.error(problems.join('\n'));
    process.exit(1);
  }
  console.log(`locales ok: ${Object.keys(catalogs).join(', ')}`);
}

if (process.argv[1] === fileURLToPath(import.meta.url)) main();
