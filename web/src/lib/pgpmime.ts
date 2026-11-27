// SPDX-License-Identifier: GPL-3.0-or-later
// MIME for PGP/MIME (RFC 3156): what the browser encrypts or signs, and reading what comes back out.


export interface EntityFile {
  name: string;
  type: string;
  data: Uint8Array;
}

export interface Decoded {
  text: string;
  html: string;
  attachments: { filename: string; mimeType: string; content: ArrayBuffer | Uint8Array | string }[];
}

type PostalAttachment = { filename: string | null; mimeType: string; content: ArrayBuffer | Uint8Array | string };

function base64(data: Uint8Array): string {
  let bin = '';
  for (let i = 0; i < data.length; i += 0x8000) bin += String.fromCharCode(...data.subarray(i, i + 0x8000));
  return btoa(bin).replace(/.{76}(?=.)/g, '$&\r\n');
}

// quoted file names would break on a quote or a line break, so those become underscores
const safeName = (s: string) => s.replace(/["\\\r\n]/g, '_');

function boundary() {
  const b = new Uint8Array(12);
  crypto.getRandomValues(b);
  return 'golub-' + [...b].map((x) => x.toString(16).padStart(2, '0')).join('');
}

// buildEntity uses base64 throughout, so nothing on the way can rewrap lines and break a signature
export function buildEntity(text: string, files: EntityFile[] = []): string {
  const textPart =
    'Content-Type: text/plain; charset=utf-8\r\nContent-Transfer-Encoding: base64\r\n\r\n' + base64(new TextEncoder().encode(text)) + '\r\n';
  if (!files.length) return textPart;
  const b = boundary();
  let out = `Content-Type: multipart/mixed; boundary="${b}"\r\n\r\n--${b}\r\n${textPart}`;
  for (const f of files) {
    out +=
      `--${b}\r\nContent-Type: ${f.type || 'application/octet-stream'}; name="${safeName(f.name)}"\r\n` +
      `Content-Disposition: attachment; filename="${safeName(f.name)}"\r\nContent-Transfer-Encoding: base64\r\n\r\n` +
      base64(f.data) +
      '\r\n';
  }
  return out + `--${b}--\r\n`;
}

export async function parseEntity(entity: string): Promise<Decoded> {
  // only needed once something was decrypted, so it loads then
  const { default: PostalMime } = await import('postal-mime');
  const mail = await PostalMime.parse(entity);
  const attachments = ((mail.attachments ?? []) as PostalAttachment[]).map((a, i) => ({
    filename: a.filename || `attachment-${i + 1}`,
    mimeType: a.mimeType,
    content: a.content,
  }));
  return { text: mail.text ?? '', html: mail.html ?? '', attachments };
}

// htmlToText shows decrypted HTML as text; it never reaches the server's sanitizer, so it is not rendered as HTML
export function htmlToText(html: string): string {
  const doc = new DOMParser().parseFromString(html, 'text/html');
  doc.querySelectorAll('br').forEach((el) => el.replaceWith('\n'));
  doc.querySelectorAll('p,div,li,tr,h1,h2,h3,h4').forEach((el) => el.append('\n'));
  return (doc.body.textContent ?? '').replace(/\n{3,}/g, '\n\n').trim();
}

function headerValue(head: string, name: string): string {
  const unfolded = head.replace(/\r?\n[ \t]+/g, ' ');
  const m = new RegExp(`^${name}:\\s*(.*)$`, 'im').exec(unfolded);
  return m ? m[1] : '';
}

const param = (value: string, name: string) => new RegExp(`${name}="?([^";]+)"?`, 'i').exec(value)?.[1] ?? '';

// splitSigned returns the signed part byte for byte and its signature from a raw multipart/signed message
export function splitSigned(raw: string): { entity: string; signature: string } | null {
  const crlf = raw.replace(/\r?\n/g, '\r\n');
  const headEnd = crlf.indexOf('\r\n\r\n');
  if (headEnd < 0) return null;
  const b = param(headerValue(crlf.slice(0, headEnd), 'Content-Type'), 'boundary');
  if (!b) return null;
  const body = crlf.slice(headEnd + 4);
  const open = `--${b}\r\n`;
  const first = body.indexOf(open);
  if (first < 0) return null;
  const start = first + open.length;
  const end = body.indexOf(`\r\n--${b}`, start);
  if (end < 0) return null;
  const entity = body.slice(start, end);
  const rest = body.slice(end);
  const sig = /-----BEGIN PGP SIGNATURE-----[\s\S]*?-----END PGP SIGNATURE-----/.exec(rest);
  return sig ? { entity, signature: sig[0] } : null;
}

// encryptedPart pulls the armored message out of a raw multipart/encrypted message
export function encryptedPart(raw: string): string | null {
  return /-----BEGIN PGP MESSAGE-----[\s\S]*?-----END PGP MESSAGE-----/.exec(raw)?.[0] ?? null;
}

// rawLatin1 and latin1Bytes keep every byte of a raw message as one character, so a signed part survives untouched
export async function rawLatin1(res: Response): Promise<string> {
  return new TextDecoder('latin1').decode(await res.arrayBuffer());
}

export function latin1Bytes(s: string): Uint8Array {
  const out = new Uint8Array(s.length);
  for (let i = 0; i < s.length; i++) out[i] = s.charCodeAt(i) & 0xff;
  return out;
}
