// SPDX-License-Identifier: GPL-3.0-or-later
import { describe, expect, it } from 'vitest';
import * as pgp from './pgp';
import { buildEntity, encryptedPart, parseEntity, splitSigned } from './pgpmime';

describe('pgp', () => {
  it('encrypts to a recipient, who decrypts and sees a verified signature', async () => {
    const ana = await pgp.generate('Ana', 'ana@golub.test', 'ana-pass');
    const bob = await pgp.generate('Bob', 'bob@golub.test', 'bob-pass');
    expect(ana.private).toBe(true);
    expect(ana.emails).toEqual(['ana@golub.test']);
    expect(ana.armored).toContain('PRIVATE KEY BLOCK');

    await expect(pgp.unlock(ana, 'wrong')).rejects.toThrow();
    await pgp.unlock(ana, 'ana-pass');
    const bobPublic = { ...bob, private: false, armored: await pgp.publicPart(bob.armored) };
    const entity = buildEntity('Šta ima?', [{ name: 'a.txt', type: 'text/plain', data: new TextEncoder().encode('file body') }]);
    const armored = await pgp.encrypt(entity, [bobPublic, ana], ana);
    expect(armored).toContain('-----BEGIN PGP MESSAGE-----');

    expect((await pgp.neededKey(armored, [ana, bob]))?.fingerprint).toBe(ana.fingerprint);
    await pgp.unlock(bob, 'bob-pass');
    const anaPublic = { ...ana, private: false, armored: await pgp.publicPart(ana.armored) };
    const out = await pgp.decrypt(armored, bob, [anaPublic]);
    expect(out.signature).toEqual({ state: 'verified', signer: 'Ana <ana@golub.test>' });
    const parsed = await parseEntity(out.data);
    expect(parsed.text.trim()).toBe('Šta ima?');
    expect(parsed.attachments[0].filename).toBe('a.txt');

    pgp.lock();
    await expect(pgp.decrypt(armored, bob, [])).rejects.toThrow('locked');
  });

  it('signs a part detached, and the split of the sent message verifies', async () => {
    const ana = await pgp.generate('Ana', 'ana@golub.test', 'p');
    await pgp.unlock(ana, 'p');
    const entity = buildEntity('signed text');
    const { signature, micalg } = await pgp.signDetached(entity, ana);
    expect(micalg).toMatch(/^pgp-sha(256|384|512)$/);
    const raw =
      'Subject: x\r\nContent-Type: multipart/signed; micalg=pgp-sha256; protocol="application/pgp-signature"; boundary="b1"\r\n\r\n' +
      `preamble\r\n--b1\r\n${entity}\r\n--b1\r\nContent-Type: application/pgp-signature\r\n\r\n${signature}\r\n--b1--\r\n`;
    const split = splitSigned(raw);
    expect(split?.entity).toBe(entity);
    const pub = { ...ana, private: false, armored: await pgp.publicPart(ana.armored) };
    expect((await pgp.verifyDetached(split!.entity, split!.signature, [pub])).state).toBe('verified');
    expect((await pgp.verifyDetached(split!.entity.replace('charset=utf-8', 'charset=UTF-8'), split!.signature, [pub])).state).toBe('invalid');
    expect((await pgp.verifyDetached(split!.entity, split!.signature, [])).state).toBe('unknown');
    pgp.lock();
  });

  it('notices private keys without a passphrase, and protects them', async () => {
    const ana = await pgp.generate('Ana', 'ana@golub.test', 'p');
    await pgp.unlock(ana, 'p');
    const { inspect, protect } = pgp;
    const plain = (await import('openpgp')).decryptKey;
    const openpgp = await import('openpgp');
    const naked = (await plain({ privateKey: await openpgp.readPrivateKey({ armoredKey: ana.armored }), passphrase: 'p' })).armor();
    expect((await inspect(naked)).unprotected).toBe(true);
    const [protectedKey] = await protect(naked, 'new-pass');
    expect((await inspect(protectedKey.armored)).unprotected).toBe(false);
    await pgp.unlock(protectedKey, 'new-pass');
    pgp.lock();
  });

  it('finds the armored message inside a PGP/MIME message', () => {
    expect(encryptedPart('x\r\n-----BEGIN PGP MESSAGE-----\r\n\r\nabc\r\n-----END PGP MESSAGE-----\r\ny')).toBe(
      '-----BEGIN PGP MESSAGE-----\r\n\r\nabc\r\n-----END PGP MESSAGE-----',
    );
  });
});
