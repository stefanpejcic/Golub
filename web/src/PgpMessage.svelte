<!-- SPDX-License-Identifier: GPL-3.0-or-later -->
<script lang="ts">
  import { i18n } from './lib/locale.svelte';
  import { keys, unlockKey } from './lib/keys.svelte';
  import * as pgp from './lib/pgp';
  import { encryptedPart, htmlToText, latin1Bytes, parseEntity, rawLatin1, splitSigned } from './lib/pgpmime';
  import { formatSize } from './lib/format';

  let { folder, uid, kind, text }: { folder: string; uid: number; kind: string; text: string } = $props();
  const t = $derived(i18n.t);

  type Att = { name: string; url: string; size: number };
  let phase = $state<'working' | 'locked' | 'nokey' | 'done' | 'error'>('working');
  let plain = $state('');
  let atts = $state<Att[]>([]);
  let signature = $state<pgp.SignatureStatus | undefined>(undefined);
  let pending: { armored: string; key: pgp.KeyInfo } | null = null;

  const encrypted = $derived(kind === 'encrypted' || kind === 'inline-encrypted');
  const known = () => keys.all;
  const raw = async () => {
    const res = await fetch(`/api/message/raw?folder=${encodeURIComponent(folder)}&uid=${uid}`, { credentials: 'same-origin' });
    if (!res.ok) throw new Error(String(res.status));
    return rawLatin1(res);
  };

  async function decrypt() {
    if (!pending) return;
    phase = 'working';
    const out = await pgp.decrypt(pending.armored, pending.key, known());
    signature = out.signature;
    if (kind === 'encrypted') {
      const parsed = await parseEntity(out.data);
      plain = parsed.text || htmlToText(parsed.html);
      atts = parsed.attachments.map((a) => {
        const blob = new Blob([a.content as ArrayBuffer], { type: a.mimeType });
        return { name: a.filename, url: URL.createObjectURL(blob), size: blob.size };
      });
    } else {
      plain = out.data;
    }
    phase = 'done';
  }

  async function unlock() {
    if (pending && (await unlockKey(pending.key))) await decrypt().catch(() => (phase = 'error'));
  }

  async function run() {
    phase = 'working';
    plain = '';
    atts = [];
    signature = undefined;
    await keys.load();
    if (encrypted) {
      const armored = kind === 'encrypted' ? encryptedPart(await raw()) : /-----BEGIN PGP MESSAGE-----[\s\S]*?-----END PGP MESSAGE-----/.exec(text)?.[0];
      if (!armored) return (phase = 'error');
      const key = await pgp.neededKey(armored, keys.own);
      if (!key) return (phase = 'nokey');
      pending = { armored, key };
      if (pgp.isUnlocked(key.fingerprint)) return decrypt();
      phase = 'locked';
    } else if (kind === 'signed') {
      const split = splitSigned(await raw());
      signature = split ? await pgp.verifyDetached(latin1Bytes(split.entity), split.signature, known()) : { state: 'invalid' };
      phase = 'done';
    } else if (kind === 'inline-signed') {
      const block = /-----BEGIN PGP SIGNED MESSAGE-----[\s\S]*?-----END PGP SIGNATURE-----/.exec(text)?.[0];
      signature = block ? (await pgp.verifyCleartext(block, known())).signature : { state: 'invalid' };
      phase = 'done';
    }
  }

  $effect(() => {
    void uid;
    run().catch(() => (phase = 'error'));
    return () => atts.forEach((a) => URL.revokeObjectURL(a.url));
  });
</script>

<section class="pgp" class:encrypted data-testid="pgp-panel">
  <div class="badges">
    {#if encrypted}
      <span class="badge enc" data-testid="pgp-encrypted">
        <svg class="icon" viewBox="0 0 16 16" aria-hidden="true"><rect x="3" y="7" width="10" height="7" rx="1.5" /><path d="M5.5 7V5a2.5 2.5 0 0 1 5 0v2" /></svg>
        {t('pgp.badge_encrypted')}
      </span>
    {/if}
    {#if signature}
      <span class="badge sig {signature.state}" data-testid="pgp-signature" data-state={signature.state}>
        {#if signature.state === 'verified'}{t('pgp.sig_verified', { signer: signature.signer ?? '' })}
        {:else if signature.state === 'unknown'}{t('pgp.sig_unknown')}
        {:else}{t('pgp.sig_invalid')}{/if}
      </span>
    {/if}
  </div>

  {#if phase === 'locked'}
    <p>{t('pgp.locked_message')}</p>
    <button type="button" class="btn btn-primary" onclick={unlock} data-testid="pgp-unlock">{t('pgp.unlock')}</button>
  {:else if phase === 'nokey'}
    <p class="muted" data-testid="pgp-nokey">{t('pgp.no_key_for_message')}</p>
  {:else if phase === 'error'}
    <p class="muted" data-testid="pgp-error">{t('pgp.decrypt_failed')}</p>
  {:else if phase === 'working'}
    <p class="muted">{t('common.loading')}</p>
  {:else if encrypted}
    <div class="text" dir="auto" data-testid="pgp-text">{plain}</div>
    {#if atts.length}
      <ul class="atts">
        {#each atts as a (a.url)}
          <li><a href={a.url} download={a.name} data-testid="pgp-attachment">{a.name} <span class="muted">{formatSize(i18n.locale, a.size)}</span></a></li>
        {/each}
      </ul>
    {/if}
  {/if}
</section>

<style>
  .pgp {
    margin: 0 var(--space-4) var(--space-3);
  }
  .pgp.encrypted {
    padding: var(--space-3) var(--space-4);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-lg);
  }
  .badges {
    display: flex;
    flex-wrap: wrap;
    gap: var(--space-2);
    margin-block-end: var(--space-2);
  }
  .badge {
    display: inline-flex;
    align-items: center;
    gap: var(--space-1);
    padding: 0.15rem 0.6rem;
    font-size: 0.8rem;
    border-radius: 999px;
    background: var(--color-input-bg);
  }
  .badge .icon {
    inline-size: 0.9rem;
    block-size: 0.9rem;
  }
  .sig.verified {
    color: var(--color-success);
  }
  .sig.invalid {
    color: var(--color-danger);
  }
  .text {
    white-space: pre-wrap;
    overflow-wrap: anywhere;
  }
  .atts {
    display: flex;
    flex-wrap: wrap;
    gap: var(--space-2);
    padding: 0;
    margin: var(--space-3) 0 0;
    list-style: none;
  }
  .atts a {
    padding: var(--space-1) var(--space-2);
    color: var(--color-text);
    text-decoration: none;
    border: 1px solid var(--color-input-border);
    border-radius: var(--radius);
  }
  .muted {
    color: var(--color-text-muted);
  }
</style>
