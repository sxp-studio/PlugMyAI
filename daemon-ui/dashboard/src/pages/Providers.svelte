<script>
  import { getProviders } from '../lib/api.js';

  let providers = $state([]);
  let loading = $state(true);
  let error = $state(null);

  $effect(() => {
    loadProviders();
  });

  async function loadProviders() {
    loading = true;
    error = null;
    try {
      const res = await getProviders();
      providers = Array.isArray(res) ? res : (res?.providers || []);
    } catch (e) {
      error = e.message;
      providers = [];
    } finally {
      loading = false;
    }
  }
</script>

<div class="page-header">
  <h2>Providers</h2>
  <p>Configured AI providers and their current status</p>
</div>

{#if loading}
  <div class="loading">Loading providers...</div>
{:else if error}
  <div class="error-msg">{error}</div>
{:else if providers.length === 0}
  <div class="empty-state">
    <div class="icon">~</div>
    <h3>No providers configured</h3>
    <p>Add a provider in the daemon configuration file to get started.</p>
  </div>
{:else}
  <div class="card-grid">
    {#each providers as provider}
      <div class="card">
        <div class="flex" style="align-items:center;gap:12px;margin-bottom:14px">
          <span class="status-dot {provider.status === 'available' ? 'available' : 'unavailable'}"></span>
          <div>
            <div style="font-weight:600;font-size:15px">{provider.name || provider.id || 'Unknown'}</div>
          </div>
        </div>

        <div style="display:flex;flex-direction:column;gap:8px">
          <div class="flex-between">
            <span class="text-dim" style="font-size:12px">Type</span>
            <code>{provider.type || provider.provider || '--'}</code>
          </div>

          <div class="flex-between">
            <span class="text-dim" style="font-size:12px">Status</span>
            {#if provider.status === 'available'}
              <span class="badge badge-green">Available</span>
            {:else}
              <span class="badge badge-gray">Unavailable</span>
            {/if}
          </div>

          {#if provider.models}
            <div class="flex-between">
              <span class="text-dim" style="font-size:12px">Models</span>
              <span class="mono" style="font-size:12px">{Array.isArray(provider.models) ? provider.models.length : '--'}</span>
            </div>
          {/if}

          {#if provider.base_url}
            <div class="flex-between">
              <span class="text-dim" style="font-size:12px">Endpoint</span>
              <span class="mono text-dim" style="font-size:11px;max-width:160px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap">
                {provider.base_url}
              </span>
            </div>
          {/if}
        </div>
      </div>
    {/each}
  </div>
{/if}
