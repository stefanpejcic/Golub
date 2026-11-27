<!-- SPDX-License-Identifier: GPL-3.0-or-later -->
<script lang="ts">
  import { i18n, LOCALES } from './lib/locale.svelte';
  import { session } from './lib/session.svelte';
  import { ApiError } from './lib/api';
  import { errorMessage } from './lib/errors';
  import LoginBird from './LoginBird.svelte';
  import { isBareAddress } from './lib/compose';
  import Prefs from './Prefs.svelte';

  const t = $derived(i18n.t);

  let email = $state('');
  let password = $state('');
  let busy = $state(false);
  let hideEyes = $state(false);
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
    // The browser's own check accepts addresses without a dot in the domain, such as name@host.
    if (!isBareAddress(email)) {
      error = 'auth.invalid_email';
      return;
    }
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

<div class="screen">
  <main>
    <div class="brand" role="img" aria-label={t('app.name')}>
      <LoginBird mode={hideEyes ? (password ? 'cover' : 'peek') : 'open'} />
    </div>
    <form class="card" onsubmit={submit} data-testid="login-form">
      <h1>{t('login.title')}</h1>
      {#if error}
        <p class="error" role="alert" data-testid="login-error">{errorMessage(t, error)}</p>
      {/if}
      <label>
        <span>{t('login.email')}</span>
        <input class="input" type="email" name="email" autocomplete="username" required bind:value={email} />
      </label>
      <label>
        <span>{t('login.password')}</span>
        <input
          class="input"
          type="password"
          name="password"
          autocomplete="current-password"
          required
          bind:value={password}
          onfocus={() => (hideEyes = true)}
          onblur={() => (hideEyes = false)}
        />
      </label>
      <button class="btn btn-primary" type="submit" disabled={busy}>
        {busy ? t('login.submitting') : t('login.submit')}
      </button>
    </form>
    <select
      class="input language"
      value={i18n.locale}
      onchange={(e) => i18n.set(e.currentTarget.value)}
      aria-label={t('settings.language')}
      data-testid="lang-select"
    >
      {#each LOCALES as l (l)}
        <option value={l}>{t(`language.${l}`)}</option>
      {/each}
    </select>
  </main>
  <footer>
    <Prefs />
  </footer>
</div>

<style>
  .screen {
    display: flex;
    flex-direction: column;
    min-block-size: 100vh;
    background: var(--color-surface);
  }
  main {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: var(--space-4);
    padding: var(--space-4);
  }
  .brand {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--space-1);
    font-size: 1.5rem;
    font-weight: 600;
  }
  .card {
    inline-size: min(100%, 24rem);
    padding: var(--space-4);
    display: flex;
    flex-direction: column;
    gap: var(--space-3);
    background: var(--color-surface-raised);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-lg);
    box-shadow: var(--shadow);
  }
  h1 {
    margin: 0;
    font-size: 1.125rem;
    font-weight: 600;
  }
  label {
    display: flex;
    flex-direction: column;
    gap: var(--space-1);
    font-weight: 500;
  }
  .btn-primary {
    padding-block: 0.5rem;
  }
  .language {
    inline-size: auto;
    margin-block-start: calc(-1 * var(--space-2));
    padding-block: 0.3rem;
    color: var(--color-text-muted);
    background: transparent;
    border-color: transparent;
    box-shadow: none;
    cursor: pointer;
  }
  .error {
    margin: 0;
    color: var(--color-danger);
  }
  footer {
    align-self: flex-start;
  }
  footer :global(.prefs) {
    border-block-start: none;
  }
</style>
