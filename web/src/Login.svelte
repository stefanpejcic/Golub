<!-- SPDX-License-Identifier: GPL-3.0-or-later -->
<script lang="ts">
  import { i18n } from './lib/locale.svelte';
  import { session } from './lib/session.svelte';
  import { ApiError } from './lib/api';
  import { errorMessage } from './lib/errors';

  const t = $derived(i18n.t);

  let email = $state('');
  let password = $state('');
  let busy = $state(false);
  let error = $state(takeUrlError());

  // SSO failures arrive as ?error=code; show once, then clean the URL.
  function takeUrlError(): string {
    const url = new URL(location.href);
    const code = url.searchParams.get('error') ?? '';
    if (code) {
      url.searchParams.delete('error');
      history.replaceState(null, '', url.pathname + url.search + url.hash);
    }
    return code;
  }

  async function submit(e: SubmitEvent) {
    e.preventDefault();
    busy = true;
    error = '';
    try {
      await session.login(email.trim(), password);
      password = '';
    } catch (err) {
      error = err instanceof ApiError ? err.code : 'unknown';
    } finally {
      busy = false;
    }
  }
</script>

<form class="card" onsubmit={submit} data-testid="login-form">
  <h1>{t('login.title')}</h1>
  {#if error}
    <p class="error" role="alert" data-testid="login-error">{errorMessage(t, error)}</p>
  {/if}
  <label>
    <span>{t('login.email')}</span>
    <input type="email" name="email" autocomplete="username" required bind:value={email} />
  </label>
  <label>
    <span>{t('login.password')}</span>
    <input type="password" name="password" autocomplete="current-password" required bind:value={password} />
  </label>
  <button type="submit" disabled={busy}>{busy ? t('login.submitting') : t('login.submit')}</button>
</form>

<style>
  .card {
    inline-size: min(100%, 22rem);
    margin-block: 10vh 0;
    margin-inline: auto;
    padding: var(--space-4);
    display: flex;
    flex-direction: column;
    gap: var(--space-3);
    background: var(--color-surface-raised);
    border: 1px solid var(--color-border);
    border-radius: var(--radius);
  }
  h1 {
    margin: 0;
    font-size: 1.25rem;
  }
  label {
    display: flex;
    flex-direction: column;
    gap: var(--space-1);
    color: var(--color-text-muted);
  }
  input {
    padding: var(--space-2);
    background: var(--color-bg);
    color: var(--color-text);
    border: 1px solid var(--color-border);
    border-radius: var(--radius);
    font: inherit;
  }
  button {
    padding: var(--space-2);
    background: var(--color-accent);
    color: var(--color-accent-text);
    border: none;
    border-radius: var(--radius);
    font: inherit;
    cursor: pointer;
  }
  button:disabled {
    opacity: 0.7;
    cursor: progress;
  }
  .error {
    margin: 0;
    color: var(--color-danger);
  }
</style>
