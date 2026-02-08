<script>
  import { getStatus, getProviders, getApps, getHistory } from '../lib/api.js';
  import { relativeTime, formatNumber, formatDuration, formatUptime, truncate } from '../lib/format.js';

  let status = $state(null);
  let providers = $state([]);
  let apps = $state([]);
  let recentHistory = $state([]);
  let loading = $state(true);
  let error = $state(null);

  $effect(() => {
    loadData();
  });

  async function loadData() {
    loading = true;
    error = null;
    try {
      const results = await Promise.allSettled([
        getStatus(),
        getProviders(),
        getApps(),
        getHistory(5, 0),
      ]);

      if (results[0].status === 'fulfilled') status = results[0].value;
      if (results[1].status === 'fulfilled') {
        providers = Array.isArray(results[1].value) ? results[1].value : (results[1].value?.providers || []);
      }
      if (results[2].status === 'fulfilled') {
        apps = Array.isArray(results[2].value) ? results[2].value : (results[2].value?.apps || []);
      }
      if (results[3].status === 'fulfilled') {
        const hv = results[3].value;
        recentHistory = Array.isArray(hv) ? hv : (hv?.entries || hv?.history || []);
      }
    } catch (e) {
      error = e.message;
    } finally {
      loading = false;
    }
  }

  let availableCount = $derived(providers.filter(p => p.status === 'available').length);
</script>

<div class="page-header">
  <h2>Overview</h2>
  <p>Dashboard summary for your local AI proxy</p>
</div>

{#if loading}
  <div class="loading">Loading dashboard...</div>
{:else}

  {#if error}
    <div class="error-msg mb-16">{error}</div>
  {/if}

  <div class="card-grid">
    <div class="stat-card">
      <div class="label">Version</div>
      <div class="value" style="font-size:20px">{status?.version || '--'}</div>
      {#if status?.uptime}
        <div class="meta">Uptime: {formatUptime(status.uptime)}</div>
      {/if}
    </div>

    <div class="stat-card">
      <div class="label">Providers</div>
      <div class="value">{providers.length}</div>
      <div class="meta">
        <span class="text-green">{availableCount} available</span>
        {#if providers.length - availableCount > 0}
          <span class="text-dim"> / {providers.length - availableCount} unavailable</span>
        {/if}
      </div>
    </div>

    <div class="stat-card">
      <div class="label">Connected Apps</div>
      <div class="value">{apps.length}</div>
      <div class="meta">
        <a href="#/apps" class="text-accent" style="text-decoration:none;font-size:12px">View all &rarr;</a>
      </div>
    </div>

    <div class="stat-card">
      <div class="label">Recent Requests</div>
      <div class="value">{recentHistory.length > 0 ? formatNumber(status?.total_requests ?? recentHistory.length) : '0'}</div>
      <div class="meta">
        <a href="#/history" class="text-accent" style="text-decoration:none;font-size:12px">View history &rarr;</a>
      </div>
    </div>
  </div>

  <!-- Provider Status -->
  {#if providers.length > 0}
    <div class="flex-between mb-16" style="margin-top:8px">
      <h3 style="font-size:14px;font-weight:600;">Provider Status</h3>
    </div>
    <div class="card-grid">
      {#each providers as provider}
        <div class="card" style="padding:16px">
          <div class="flex" style="align-items:center;gap:10px">
            <span class="status-dot {provider.status === 'available' ? 'available' : 'unavailable'}"></span>
            <div>
              <div style="font-weight:600;font-size:14px">{provider.name || provider.id || 'Unknown'}</div>
              <div class="text-dim" style="font-size:12px">{provider.type || provider.provider || '--'}</div>
            </div>
          </div>
        </div>
      {/each}
    </div>
  {/if}

  <!-- Recent Requests -->
  {#if recentHistory.length > 0}
    <div class="flex-between mb-16" style="margin-top:8px">
      <h3 style="font-size:14px;font-weight:600;">Recent Requests</h3>
      <a href="#/history" class="text-accent" style="text-decoration:none;font-size:12px">View all &rarr;</a>
    </div>
    <div class="table-wrap">
      <table>
        <thead>
          <tr>
            <th>Time</th>
            <th>App</th>
            <th>Prompt</th>
            <th>Model</th>
            <th>Duration</th>
          </tr>
        </thead>
        <tbody>
          {#each recentHistory as entry}
            <tr>
              <td class="mono text-dim">{relativeTime(entry.timestamp || entry.created_at)}</td>
              <td>{entry.app_name || entry.app || '--'}</td>
              <td class="text-dim" style="max-width:300px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap">
                {truncate(entry.prompt || entry.messages?.[0]?.content || '', 60)}
              </td>
              <td><code>{entry.model || '--'}</code></td>
              <td class="mono">{formatDuration(entry.duration)}</td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  {/if}

  {#if !error && providers.length === 0 && apps.length === 0 && recentHistory.length === 0}
    <div class="empty-state" style="margin-top:24px">
      <div class="icon">~</div>
      <h3>No data yet</h3>
      <p>The daemon is running but no providers, apps, or requests have been recorded.</p>
    </div>
  {/if}

{/if}
