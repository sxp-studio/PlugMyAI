<script>
  import { getStatus } from '../lib/api.js';
  import { formatUptime } from '../lib/format.js';

  let status = $state(null);
  let loading = $state(true);
  let copied = $state(false);

  const UNINSTALL_CMD = 'curl -fsSL https://plugmy.ai/uninstall.sh | sh';

  $effect(() => {
    getStatus()
      .then(data => { status = data; })
      .catch(() => {})
      .finally(() => { loading = false; });
  });

  function copyCommand() {
    navigator.clipboard.writeText(UNINSTALL_CMD).then(() => {
      copied = true;
      setTimeout(() => { copied = false; }, 2000);
    });
  }
</script>

<div class="page-header">
  <h2>Settings</h2>
  <p>Daemon info and configuration</p>
</div>

{#if loading}
  <div class="loading">Loading...</div>
{:else}

  <!-- Daemon Info -->
  <h3 class="section-title">Daemon</h3>
  <div class="card" style="padding:20px; margin-bottom:32px">
    <dl class="info-grid">
      <dt>Version</dt>
      <dd>{status?.version || '--'}</dd>
      <dt>Uptime</dt>
      <dd>{status?.uptime ? formatUptime(status.uptime) : '--'}</dd>
      <dt>Endpoint</dt>
      <dd><code>http://localhost:21110</code></dd>
      <dt>Data directory</dt>
      <dd><code>~/.plug-my-ai/</code></dd>
    </dl>
  </div>

  <!-- Danger Zone -->
  <h3 class="section-title" style="color: var(--red)">Danger Zone</h3>
  <div class="danger-card">
    <div class="danger-header">
      <div>
        <div class="danger-title">Uninstall plug-my-ai</div>
        <div class="danger-desc">
          Stops the daemon, removes the app, deletes all data and configuration. This cannot be undone.
        </div>
      </div>
    </div>
    <div class="uninstall-cmd-wrap">
      <pre class="uninstall-cmd">{UNINSTALL_CMD}</pre>
      <button class="btn btn-sm" class:copied onclick={copyCommand}>
        {copied ? 'Copied!' : 'Copy'}
      </button>
    </div>
    <div class="danger-details">
      <p>Run this command in Terminal. It will:</p>
      <ul>
        <li>Stop running processes (daemon + menu bar app)</li>
        <li>Remove the app from /Applications</li>
        <li>Delete <code>~/.plug-my-ai/</code> (config, database, tokens)</li>
        <li>Clean up PATH from shell profiles</li>
        <li>Unregister the URL scheme from Launch Services</li>
      </ul>
    </div>
  </div>

{/if}

<style>
  .section-title {
    font-size: 13px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: var(--text-dim);
    margin-bottom: 12px;
  }

  .info-grid {
    display: grid;
    grid-template-columns: 120px 1fr;
    gap: 8px 16px;
  }

  .info-grid dt {
    font-size: 13px;
    color: var(--text-dim);
  }

  .info-grid dd {
    font-size: 13px;
  }

  .danger-card {
    background: var(--bg-card);
    border: 1px solid rgba(239, 68, 68, 0.25);
    border-radius: var(--radius);
    padding: 20px;
  }

  .danger-header {
    margin-bottom: 16px;
  }

  .danger-title {
    font-size: 14px;
    font-weight: 600;
    color: var(--text);
    margin-bottom: 4px;
  }

  .danger-desc {
    font-size: 13px;
    color: var(--text-dim);
  }

  .uninstall-cmd-wrap {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-bottom: 16px;
  }

  .uninstall-cmd {
    flex: 1;
    margin: 0;
    padding: 10px 14px;
    font-size: 13px;
    user-select: all;
  }

  .copied {
    color: var(--green) !important;
    border-color: var(--green) !important;
  }

  .danger-details {
    font-size: 12px;
    color: var(--text-muted);
  }

  .danger-details p {
    margin-bottom: 6px;
  }

  .danger-details ul {
    list-style: none;
    padding: 0;
  }

  .danger-details li {
    padding: 2px 0;
  }

  .danger-details li::before {
    content: '—';
    margin-right: 6px;
    opacity: 0.5;
  }
</style>
