// SPDX-License-Identifier: GPL-3.0-or-later

export type PreviewKind = 'image' | 'pdf' | 'text' | null;

const images = new Set(['image/png', 'image/jpeg', 'image/gif', 'image/webp', 'image/avif', 'image/bmp']);
const texts = new Set(['text/plain', 'text/csv', 'text/markdown']);

// Text previews above this size are slow to lay out and rarely useful.
export const MAX_TEXT_PREVIEW = 1 << 20;

export function previewKind(contentType: string, size: number): PreviewKind {
  const mt = contentType.split(';')[0].trim().toLowerCase();
  if (images.has(mt)) return 'image';
  if (mt === 'application/pdf') return 'pdf';
  if (texts.has(mt) && size <= MAX_TEXT_PREVIEW) return 'text';
  return null;
}
