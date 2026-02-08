<script>
  import { getOnboardingStatus, testProvider, completeOnboarding } from '../lib/api.js';

  let { onComplete } = $props();

  let providers = $state([]);
  let loading = $state(true);
  let error = $state(null);
  let completing = $state(false);

  // Per-provider test state: 'idle' | 'testing' | 'success' | 'error'
  let testState = $state({});
  let testMessage = $state({});

  $effect(() => {
    loadProviders();
  });

  async function loadProviders() {
    loading = true;
    error = null;
    try {
      const data = await getOnboardingStatus();
      providers = data.providers || [];
      // Initialize test state for each provider
      for (const p of providers) {
        testState[p.id] = 'idle';
        testMessage[p.id] = '';
      }
    } catch (e) {
      error = e.message;
    } finally {
      loading = false;
    }
  }

  async function runTest(providerId) {
    testState[providerId] = 'testing';
    testMessage[providerId] = '';
    try {
      const result = await testProvider(providerId);
      if (result.success) {
        testState[providerId] = 'success';
        testMessage[providerId] = result.message || 'Provider is working';
      } else {
        testState[providerId] = 'error';
        testMessage[providerId] = result.error || 'Test failed';
      }
    } catch (e) {
      testState[providerId] = 'error';
      testMessage[providerId] = e.message || 'Request failed';
    }
  }

  let anySuccess = $derived(
    Object.values(testState).some(s => s === 'success')
  );
  let anyTesting = $derived(
    Object.values(testState).some(s => s === 'testing')
  );

  async function finish() {
    completing = true;
    try {
      await completeOnboarding();
      onComplete();
    } catch (e) {
      error = e.message;
      completing = false;
    }
  }
</script>

<div class="onboarding">
  <div class="onboarding-card">
    <div class="onboarding-header">
      <h1>Welcome to plug-my-ai</h1>
      <p>Let's verify your AI providers are working before you start.</p>
    </div>

    {#if loading}
      <div class="loading">Loading providers...</div>
    {:else if error}
      <div class="error-msg">{error}</div>
    {:else}
      <div class="provider-list">
        {#each providers as provider (provider.id)}
          <div class="provider-item">
            <div class="provider-info">
              <div class="provider-name">
                <span class="status-dot {provider.available ? 'available' : 'unavailable'}"></span>
                {provider.name}
              </div>
              <div class="provider-id"><code>{provider.id}</code></div>
            </div>

            {#if !provider.available}
              <div class="provider-unavailable">
                CLI not found in PATH. Install it first, then restart the daemon.
              </div>
            {:else}
              <div class="provider-actions">
                {#if testState[provider.id] === 'idle'}
                  <button class="btn btn-primary" onclick={() => runTest(provider.id)}>
                    Test
                  </button>
                {:else if testState[provider.id] === 'testing'}
                  <div class="test-running">
                    <span class="spinner"></span>
                    Testing...
                  </div>
                {:else if testState[provider.id] === 'success'}
                  <div class="test-result success">
                    <span class="result-icon">&#10003;</span>
                    <span>{testMessage[provider.id]}</span>
                  </div>
                {:else if testState[provider.id] === 'error'}
                  <div class="test-result error">
                    <span class="result-icon">&#10007;</span>
                    <span>{testMessage[provider.id]}</span>
                  </div>
                  <button class="btn btn-sm" onclick={() => runTest(provider.id)}>
                    Retry
                  </button>
                {/if}
              </div>
            {/if}
          </div>
        {/each}
      </div>

      <div class="onboarding-footer">
        <button
          class="btn btn-success"
          disabled={!anySuccess || anyTesting || completing}
          onclick={finish}
        >
          {completing ? 'Saving...' : 'Complete Setup'}
        </button>
        {#if !anySuccess}
          <p class="hint">Test at least one provider to continue.</p>
        {/if}
      </div>
    {/if}
  </div>
</div>

<style>
  .onboarding {
    display: flex;
    align-items: center;
    justify-content: center;
    min-height: 100vh;
    padding: 32px;
  }

  .onboarding-card {
    width: 100%;
    max-width: 520px;
    background: var(--bg-card);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    padding: 40px;
  }

  .onboarding-header {
    text-align: center;
    margin-bottom: 32px;
  }

  .onboarding-header h1 {
    font-size: 20px;
    font-weight: 600;
    letter-spacing: -0.02em;
    margin-bottom: 8px;
  }

  .onboarding-header p {
    color: var(--text-dim);
    font-size: 14px;
  }

  .provider-list {
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  .provider-item {
    background: var(--bg);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    padding: 16px;
  }

  .provider-info {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 12px;
  }

  .provider-name {
    display: flex;
    align-items: center;
    font-weight: 600;
    font-size: 14px;
  }

  .provider-id {
    font-size: 12px;
    color: var(--text-dim);
  }

  .provider-unavailable {
    font-size: 12px;
    color: var(--text-dim);
    padding: 8px 12px;
    background: rgba(255, 255, 255, 0.02);
    border-radius: var(--radius);
  }

  .provider-actions {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  .test-running {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 13px;
    color: var(--text-dim);
  }

  .spinner {
    display: inline-block;
    width: 14px;
    height: 14px;
    border: 2px solid var(--border);
    border-top-color: var(--accent);
    border-radius: 50%;
    animation: spin 0.6s linear infinite;
  }

  .test-result {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 13px;
  }

  .test-result.success {
    color: var(--green);
  }

  .test-result.error {
    color: var(--red);
  }

  .result-icon {
    font-weight: 700;
  }

  .onboarding-footer {
    margin-top: 28px;
    text-align: center;
  }

  .onboarding-footer .btn {
    width: 100%;
  }

  .hint {
    margin-top: 10px;
    font-size: 12px;
    color: var(--text-muted);
  }

  @keyframes spin {
    to { transform: rotate(360deg); }
  }
</style>
