// SPDX-License-Identifier: GPL-3.0-or-later
// All PGP work happens here in the browser; the server only ever sees armored keys and encrypted or signed blocks.

import type * as OpenPGP from 'openpgp';

export interface KeyInfo {
  fingerprint: string;
  userIds: string[];
  emails: string[];
  private: boolean;
  created: string;
  armored: string;
}

export interface SignatureStatus {
  state: 'verified' | 'invalid' | 'unknown';
  signer?: string;
}

// OpenPGP.js is a few hundred KB, so it only loads the first time someone uses PGP
let lib: Promise<typeof OpenPGP> | null = null;
const openpgp = () => (lib ??= import('openpgp'));

// unlocked keys live in this tab's memory only and lock again after a quiet while
const LOCK_AFTER = 15 * 60 * 1000;
const unlocked = new Map<string, OpenPGP.PrivateKey>();
let lockTimer: ReturnType<typeof setTimeout> | undefined;

function touch() {
  clearTimeout(lockTimer);
  lockTimer = setTimeout(lock, LOCK_AFTER);
}

export function lock() {
  unlocked.clear();
  clearTimeout(lockTimer);
}

export function isUnlocked(fingerprint: string) {
  return unlocked.has(fingerprint);
}

const emailOf = (userId: string) => (/<([^<>\s]+@[^<>\s]+)>/.exec(userId)?.[1] ?? (userId.includes('@') ? userId.trim() : '')).toLowerCase();

async function info(key: OpenPGP.Key): Promise<KeyInfo> {
  const userIds = key.getUserIDs();
  return {
    fingerprint: key.getFingerprint().toLowerCase(),
    userIds,
    emails: [...new Set(userIds.map(emailOf).filter(Boolean))],
    private: key.isPrivate(),
    created: key.getCreationTime().toISOString(),
    armored: key.armor(),
  };
}

export async function generate(name: string, email: string, passphrase: string): Promise<KeyInfo> {
  const pgp = await openpgp();
  const { privateKey } = await pgp.generateKey({ type: 'curve25519', userIDs: [{ name, email }], passphrase, format: 'object' });
  return info(privateKey);
}

// inspect reads every key in an armored block; private keys have to be passphrase protected before they are stored
export async function inspect(armored: string): Promise<{ keys: KeyInfo[]; unprotected: boolean }> {
  const pgp = await openpgp();
  const keys = await pgp.readKeys({ armoredKeys: armored.trim() });
  const unprotected = keys.some((k) => k.isPrivate() && (k as OpenPGP.PrivateKey).isDecrypted());
  return { keys: await Promise.all(keys.map(info)), unprotected };
}

export async function protect(armored: string, passphrase: string): Promise<KeyInfo[]> {
  const pgp = await openpgp();
  const keys = await pgp.readKeys({ armoredKeys: armored.trim() });
  return Promise.all(
    keys.map(async (k) => {
      if (!k.isPrivate()) return info(k);
      const priv = k as OpenPGP.PrivateKey;
      return info(priv.isDecrypted() ? await pgp.encryptKey({ privateKey: priv, passphrase }) : priv);
    }),
  );
}

export async function publicPart(armored: string): Promise<string> {
  const pgp = await openpgp();
  return (await pgp.readKey({ armoredKey: armored })).toPublic().armor();
}

// unlock throws on a wrong passphrase
export async function unlock(key: KeyInfo, passphrase: string): Promise<void> {
  const pgp = await openpgp();
  const privateKey = await pgp.decryptKey({ privateKey: await pgp.readPrivateKey({ armoredKey: key.armored }), passphrase });
  unlocked.set(key.fingerprint, privateKey);
  touch();
}

const readKeys = async (pgp: typeof OpenPGP, list: KeyInfo[]) => Promise.all(list.map((k) => pgp.readKey({ armoredKey: k.armored })));

// neededKey picks which of our private keys an encrypted message was made for
export async function neededKey(armoredMessage: string, own: KeyInfo[]): Promise<KeyInfo | undefined> {
  const pgp = await openpgp();
  const message = await pgp.readMessage({ armoredMessage });
  const ids = message.getEncryptionKeyIDs().map((id) => id.toHex());
  for (const k of own) {
    const key = await pgp.readKey({ armoredKey: k.armored });
    if (key.getKeys().some((sub) => ids.includes(sub.getKeyID().toHex()))) return k;
  }
  return undefined;
}

// OpenPGP.js does not export its verification result type
type Verification = { keyID: { toHex(): string }; verified: Promise<unknown> };

async function status(pgp: typeof OpenPGP, sigs: Verification[], known: KeyInfo[]): Promise<SignatureStatus | undefined> {
  if (!sigs.length) return undefined;
  const sig = sigs[0];
  const keyId = sig.keyID.toHex();
  let signer: string | undefined;
  for (const k of known) {
    const key = await pgp.readKey({ armoredKey: k.armored });
    if (key.getKeys().some((sub) => sub.getKeyID().toHex() === keyId)) signer = k.userIds[0] ?? k.emails[0];
  }
  if (!signer) return { state: 'unknown' };
  try {
    await sig.verified;
    return { state: 'verified', signer };
  } catch {
    return { state: 'invalid', signer };
  }
}

export async function decrypt(armoredMessage: string, key: KeyInfo, known: KeyInfo[]): Promise<{ data: string; signature?: SignatureStatus }> {
  const pgp = await openpgp();
  const privateKey = unlocked.get(key.fingerprint);
  if (!privateKey) throw new Error('locked');
  touch();
  const message = await pgp.readMessage({ armoredMessage });
  const { data, signatures } = await pgp.decrypt({ message, decryptionKeys: privateKey, verificationKeys: await readKeys(pgp, known) });
  return { data: data as string, signature: await status(pgp, signatures, known) };
}

export async function encrypt(entity: string, recipients: KeyInfo[], signer?: KeyInfo): Promise<string> {
  const pgp = await openpgp();
  const signingKeys = signer ? unlocked.get(signer.fingerprint) : undefined;
  if (signer && !signingKeys) throw new Error('locked');
  if (signer) touch();
  const message = await pgp.createMessage({ text: entity });
  return (await pgp.encrypt({ message, encryptionKeys: await readKeys(pgp, recipients), signingKeys })) as string;
}

// signDetached signs the MIME part in text mode, so CRLF line ends are what gets signed; micalg names the hash for multipart/signed
export async function signDetached(entity: string, signer: KeyInfo): Promise<{ signature: string; micalg: string }> {
  const pgp = await openpgp();
  const signingKeys = unlocked.get(signer.fingerprint);
  if (!signingKeys) throw new Error('locked');
  touch();
  const message = await pgp.createMessage({ text: entity });
  const signature = (await pgp.sign({ message, signingKeys, detached: true })) as string;
  const packet = (await pgp.readSignature({ armoredSignature: signature })).packets[0] as unknown as { hashAlgorithm: number };
  const hash = pgp.enums.read(pgp.enums.hash, packet.hashAlgorithm) as string;
  return { signature, micalg: 'pgp-' + hash.toLowerCase() };
}

export async function verifyDetached(entity: string, armoredSignature: string, known: KeyInfo[]): Promise<SignatureStatus> {
  const pgp = await openpgp();
  const message = await pgp.createMessage({ binary: new TextEncoder().encode(entity) });
  const signature = await pgp.readSignature({ armoredSignature });
  const { signatures } = await pgp.verify({ message, signature, verificationKeys: await readKeys(pgp, known) });
  return (await status(pgp, signatures, known)) ?? { state: 'invalid' };
}

export async function verifyCleartext(armored: string, known: KeyInfo[]): Promise<{ text: string; signature: SignatureStatus }> {
  const pgp = await openpgp();
  const message = await pgp.readCleartextMessage({ cleartextMessage: armored });
  const { data, signatures } = await pgp.verify({ message, verificationKeys: await readKeys(pgp, known) });
  return { text: data, signature: (await status(pgp, signatures, known)) ?? { state: 'invalid' } };
}
