<script>
  import { getApps, deleteApp } from '../lib/api.js';
  import { shortDate } from '../lib/format.js';

  let apps = $state([]);
  let loading = $state(true);
  let error = $state(null);
  let revoking = $state(null);

  $effect(() => {
    loadApps();
  });

  async function loadApps() {
    loading = true;
    error = null;
    try {
      const res = await getApps();
      apps = Array.isArray(res) ? res : (res?.apps || []);
    } catch (e) {
      error = e.message;
      apps = [];
    } finally {
      loading = false;
    }
  }

  async function handleRevoke(app) {
    const id = app.id || app.name;
    if (!confirm(`Revoke access for "${app.name || id}"? This will disconnect the app.`)) return;

    revoking = id;
    try {
      await deleteApp(id);
      apps = apps.filter(a => (a.id || a.name) !== id);
    } catch (e) {
      alert(`Failed to revoke: ${e.message}`);
    } finally {
      revoking = null;
    }
  }
</script>

<div class="page-header">
  <div class="flex-between">
    <div>
      <h2>Apps</h2>
      <p>Applications paired with this daemon</p>
    </div>
    <span class="badge badge-blue" style="font-size:12px">{apps.length} app{apps.length !== 1 ? 's' : ''}</span>
  </div>
</div>

{#if loading}
  <div class="loading">Loading apps...</div>
{:else if error}
  <div class="error-msg">{error}</div>
{:else if apps.length === 0}
  <div class="empty-state">
    <div class="icon">~</div>
    <h3>No paired apps</h3>
    <p>No applications have been paired with this daemon yet. Use the connect flow to pair an app.</p>
  </div>
{:else}
  <div class="table-wrap">
    <table>
      <thead>
        <tr>
          <th>Name</th>
          <th>URL</th>
          <th>Providers</th>
          <th>Access</th>
          <th>Paired</th>
          <th>Status</th>
          <th style="width:100px"></th>
        </tr>
      </thead>
      <tbody>
        {#each apps as app}
          {@const id = app.id || app.name}
          <tr>
            <td style="font-weight:500">{app.name || '--'}</td>
            <td>
              {#if app.url || app.origin}
                <code style="font-size:12px">{app.url || app.origin}</code>
              {:else}
                <span class="text-dim">--</span>
              {/if}
            </td>
            <td>
              {#if !app.providers || app.providers.length === 0}
                <span class="badge badge-blue">All</span>
              {:else}
                {#each app.providers as pid}
                  <span class="badge badge-gray" style="margin-right:4px">{pid}</span>
                {/each}
              {/if}
            </td>
            <td>
              {#if app.scope === 'full'}
                <span class="badge badge-yellow">Full</span>
              {:else}
                <span class="badge badge-blue">Chat</span>
              {/if}
            </td>
            <td class="text-dim">{shortDate(app.paired_at || app.created_at)}</td>
            <td>
              {#if app.status === 'active' || !app.status}
                <span class="badge badge-green">Active</span>
              {:else}
                <span class="badge badge-gray">{app.status}</span>
              {/if}
            </td>
            <td>
              <button
                class="btn btn-danger btn-sm"
                disabled={revoking === id}
                onclick={() => handleRevoke(app)}
              >
                {revoking === id ? 'Revoking...' : 'Revoke'}
              </button>
            </td>
          </tr>
        {/each}
      </tbody>
    </table>
  </div>
{/if}
