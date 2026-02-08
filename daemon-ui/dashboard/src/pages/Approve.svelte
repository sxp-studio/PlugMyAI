<script>
  import { getConnectRequest, getConnectProviders, getPendingConnects, approveConnect, denyConnect } from '../lib/api.js';

  let { hash } = $props();

  let requestId = $derived.by(() => {
    const match = hash.match(/[?&]req=([^&]+)/);
    return match ? decodeURIComponent(match[1]) : null;
  });

  let pending = $state([]);
  let loading = $state(true);
  let error = $state(null);

  // Single-request approval flow
  let requestInfo = $state(null);
  let requestLoading = $state(true);
  let actionState = $state(null); // null | 'approving' | 'denying' | 'approved' | 'denied' | 'error'
  let actionError = $state(null);

  // Provider picker state
  let availableProviders = $state([]);
  let allProvidersMode = $state(true); // "All providers" checkbox
  let selectedProviders = $state({}); // { providerID: boolean }
  let showProviderPicker = $derived(availableProviders.length >= 2);

  // Scope picker state
  let scope = $state('chat'); // "chat" or "full"

  let canApprove = $derived(
    allProvidersMode || Object.values(selectedProviders).some(v => v)
  );

  $effect(() => {
    const reqId = requestId;
    if (reqId) {
      loadRequestInfo(reqId);
    } else {
      loadPending();
    }
  });

  async function loadPending() {
    loading = true;
    error = null;
    try {
      const res = await getPendingConnects();
      pending = Array.isArray(res) ? res : [];
    } catch (e) {
      error = e.message;
      pending = [];
    } finally {
      loading = false;
    }
  }

  async function loadRequestInfo(reqId) {
    requestLoading = true;
    error = null;
    try {
      // Fetch connect request details and available providers in parallel
      const [res, providerData] = await Promise.all([
        getConnectRequest(reqId),
        getConnectProviders(reqId).catch(() => ({ providers: [] })),
      ]);
      requestInfo = res;
      // providerData may be {providers, requested_scope} or a flat array (backwards compat)
      const providers = Array.isArray(providerData) ? providerData : (providerData.providers || []);
      availableProviders = providers;
      // Pre-select scope from the app's requested scope (or from request info)
      const reqScope = providerData?.requested_scope || res?.requested_scope || 'chat';
      scope = reqScope === 'full' ? 'full' : 'chat';
      // Initialize selection state for each provider
      const sel = {};
      for (const p of availableProviders) {
        sel[p.id] = true;
      }
      selectedProviders = sel;
    } catch (e) {
      error = e.message;
      requestInfo = null;
    } finally {
      requestLoading = false;
    }
  }

  async function handleApprove(id) {
    actionState = 'approving';
    actionError = null;
    try {
      // Determine which providers to send
      let providers = null;
      if (!allProvidersMode && showProviderPicker) {
        providers = Object.entries(selectedProviders)
          .filter(([, v]) => v)
          .map(([k]) => k);
      }
      await approveConnect(id, providers, scope);
      actionState = 'approved';
    } catch (e) {
      actionState = 'error';
      actionError = e.message;
    }
  }

  async function handleDeny(id) {
    actionState = 'denying';
    actionError = null;
    try {
      await denyConnect(id);
      actionState = 'denied';
    } catch (e) {
      actionState = 'error';
      actionError = e.message;
    }
  }

  async function handleListApprove(req) {
    const id = req.id || req.request_id;
    try {
      await approveConnect(id);
      pending = pending.filter(r => (r.id || r.request_id) !== id);
    } catch (e) {
      alert(`Failed to approve: ${e.message}`);
    }
  }

  async function handleListDeny(req) {
    const id = req.id || req.request_id;
    try {
      await denyConnect(id);
      pending = pending.filter(r => (r.id || r.request_id) !== id);
    } catch (e) {
      alert(`Failed to deny: ${e.message}`);
    }
  }

  function faviconUrl(appUrl) {
    if (!appUrl) return null;
    try {
      const u = new URL(appUrl);
      return `${u.origin}/favicon.ico`;
    } catch {
      return null;
    }
  }

  function resolveIcon(info) {
    if (!info) return null;
    return info.app_icon || faviconUrl(info.app_url);
  }
</script>

{#snippet appBadge(info)}
  {#if info}
    <div class="app-identity">
      {#if resolveIcon(info)}
        <img
          src={resolveIcon(info)}
          alt=""
          class="app-favicon"
          onerror={(e) => { e.target.style.display = 'none'; e.target.nextElementSibling.style.display = 'flex'; }}
        />
        <div class="globe-fallback" style="display:none">
          <svg xmlns="http://www.w3.org/2000/svg" width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
            <circle cx="12" cy="12" r="10"/>
            <path d="M2 12h20"/>
            <path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/>
          </svg>
        </div>
      {:else}
        <div class="globe-fallback">
          <svg xmlns="http://www.w3.org/2000/svg" width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
            <circle cx="12" cy="12" r="10"/>
            <path d="M2 12h20"/>
            <path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/>
          </svg>
        </div>
      {/if}
      <div class="app-identity-text">
        {#if info.app_name}
          <div class="app-name">{info.app_name}</div>
        {/if}
        {#if info.app_url}
          <div class="app-url"><code>{info.app_url}</code></div>
        {/if}
      </div>
    </div>
  {/if}
{/snippet}

<div class="page-header">
  <h2>Approve</h2>
  <p>Manage app pairing requests</p>
</div>

{#if requestId}
  <!-- Single request approval flow -->
  {#if requestLoading}
    <div class="loading">Loading request...</div>
  {:else if error && !requestInfo}
    <div class="approve-card">
      <h3>Request Not Found</h3>
      <p class="description">This pairing request could not be loaded. It may have expired or the daemon was restarted.</p>
      <div class="error-msg">{error}</div>
    </div>
  {:else if requestInfo?.status === 'approved' && !actionState}
    <div class="approve-card">
      <div class="result-state">
        <div class="result-icon text-green">&#10003;</div>
        <h3>Already Approved</h3>
        <p class="text-dim" style="margin-top:8px">This pairing request has already been approved.</p>
      </div>
    </div>
  {:else if requestInfo?.status === 'denied' && !actionState}
    <div class="approve-card">
      <div class="result-state">
        <div class="result-icon text-red">&#10005;</div>
        <h3>Already Denied</h3>
        <p class="text-dim" style="margin-top:8px">This pairing request was denied.</p>
      </div>
    </div>
  {:else if requestInfo?.status === 'expired' && !actionState}
    <div class="approve-card">
      <div class="result-state">
        <div class="result-icon text-dim">&#8987;</div>
        <h3>Request Expired</h3>
        <p class="text-dim" style="margin-top:8px">This pairing request has expired. The app will need to request access again.</p>
      </div>
    </div>
  {:else if actionState === 'approved'}
    <div class="approve-card">
      <div class="result-state">
        {@render appBadge(requestInfo)}
        <div class="result-icon text-green">&#10003;</div>
        <h3>Access Approved</h3>
        <p class="text-dim" style="margin-top:8px">
          {requestInfo?.app_name || 'The app'} has been granted access to your AI.
        </p>
        <a href="#/apps" class="btn" style="margin-top:20px;text-decoration:none">View Connected Apps</a>
      </div>
    </div>
  {:else if actionState === 'denied'}
    <div class="approve-card">
      <div class="result-state">
        {@render appBadge(requestInfo)}
        <div class="result-icon text-red">&#10005;</div>
        <h3>Access Denied</h3>
        <p class="text-dim" style="margin-top:8px">
          The pairing request from {requestInfo?.app_name || 'the app'} has been rejected.
        </p>
      </div>
    </div>
  {:else}
    <div class="approve-card">
      <h3>Pairing Request</h3>
      <p class="description">An application is requesting access to your AI proxy.</p>

      {@render appBadge(requestInfo)}

      <dl class="app-info">
        <dt>Request ID</dt>
        <dd><code style="font-size:13px">{requestId}</code></dd>
      </dl>

      <div class="scope-picker">
        <h4>Access level</h4>
        <label class="scope-option" class:scope-active={scope === 'chat'}>
          <input type="radio" name="scope" value="chat" bind:group={scope} />
          <div>
            <span class="scope-label">Chat only</span>
            <span class="scope-desc">Can send prompts and receive responses. No file or system access.</span>
          </div>
        </label>
        <label class="scope-option" class:scope-active={scope === 'full'} class:scope-warning={scope === 'full'}>
          <input type="radio" name="scope" value="full" bind:group={scope} />
          <div>
            <span class="scope-label">Full access</span>
            <span class="scope-desc">Can use AI tools including reading/writing files and running commands.</span>
          </div>
        </label>
      </div>

      {#if showProviderPicker}
        <div class="provider-picker">
          <h4>Provider access</h4>
          <label class="provider-option">
            <input
              type="checkbox"
              bind:checked={allProvidersMode}
            />
            <span>All providers (unrestricted)</span>
          </label>
          {#if !allProvidersMode}
            <div class="provider-list">
              {#each availableProviders as p}
                <label class="provider-option">
                  <input
                    type="checkbox"
                    bind:checked={selectedProviders[p.id]}
                  />
                  <span>{p.name}</span>
                  <code class="text-dim">{p.id}</code>
                </label>
              {/each}
            </div>
          {/if}
        </div>
      {/if}

      {#if actionState === 'error'}
        <div class="error-msg mb-16">{actionError}</div>
      {/if}

      <div class="actions">
        <button
          class="btn btn-success"
          disabled={actionState === 'approving' || actionState === 'denying' || !canApprove}
          onclick={() => handleApprove(requestId)}
        >
          {actionState === 'approving' ? 'Approving...' : 'Allow'}
        </button>
        <button
          class="btn btn-danger"
          disabled={actionState === 'approving' || actionState === 'denying'}
          onclick={() => handleDeny(requestId)}
        >
          {actionState === 'denying' ? 'Denying...' : 'Deny'}
        </button>
      </div>
    </div>
  {/if}
{:else}
  <!-- Pending requests list -->
  {#if loading}
    <div class="loading">Loading pending requests...</div>
  {:else if error}
    <div class="error-msg">{error}</div>
  {:else if pending.length === 0}
    <div class="empty-state">
      <div class="icon">~</div>
      <h3>No pending requests</h3>
      <p>There are no pairing requests waiting for approval.</p>
    </div>
  {:else}
    <div class="table-wrap">
      <table>
        <thead>
          <tr>
            <th>App</th>
            <th>URL</th>
            <th>Request ID</th>
            <th style="width:180px"></th>
          </tr>
        </thead>
        <tbody>
          {#each pending as req}
            {@const id = req.id || req.request_id}
            <tr>
              <td style="font-weight:500">{req.app_name || req.name || '--'}</td>
              <td><code style="font-size:12px">{req.app_url || req.url || '--'}</code></td>
              <td class="mono text-dim" style="font-size:12px">{id}</td>
              <td>
                <div class="flex gap-8" style="justify-content:flex-end">
                  <button class="btn btn-success btn-sm" onclick={() => handleListApprove(req)}>Allow</button>
                  <button class="btn btn-danger btn-sm" onclick={() => handleListDeny(req)}>Deny</button>
                </div>
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  {/if}
{/if}

<style>
  .app-identity {
    display: flex;
    align-items: center;
    gap: 16px;
    padding: 20px;
    background: var(--bg-card, #141414);
    border: 1px solid var(--border, #222);
    border-radius: 8px;
    margin: 20px 0;
  }
  .app-favicon {
    width: 40px;
    height: 40px;
    border-radius: 8px;
    flex-shrink: 0;
  }
  .app-name {
    font-size: 18px;
    font-weight: 600;
  }
  .app-url code {
    font-size: 13px;
    color: var(--text-dim, #888);
  }
  .globe-fallback {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 40px;
    height: 40px;
    flex-shrink: 0;
    color: var(--text-dim, #888);
  }
  .result-state .app-identity {
    margin: 0 0 16px 0;
    justify-content: center;
  }
  .provider-picker {
    margin: 16px 0;
    padding: 16px;
    background: var(--bg-card, #141414);
    border: 1px solid var(--border, #222);
    border-radius: 8px;
  }
  .provider-picker h4 {
    margin: 0 0 12px 0;
    font-size: 14px;
    font-weight: 600;
  }
  .provider-option {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 6px 0;
    cursor: pointer;
    font-size: 14px;
  }
  .provider-option code {
    font-size: 12px;
    margin-left: 4px;
  }
  .provider-list {
    margin-top: 8px;
    padding-left: 24px;
  }
  .scope-picker {
    margin: 16px 0;
    padding: 16px;
    background: var(--bg-card, #141414);
    border: 1px solid var(--border, #222);
    border-radius: 8px;
  }
  .scope-picker h4 {
    margin: 0 0 12px 0;
    font-size: 14px;
    font-weight: 600;
  }
  .scope-option {
    display: flex;
    align-items: flex-start;
    gap: 10px;
    padding: 10px 12px;
    border: 1px solid var(--border, #222);
    border-radius: 6px;
    cursor: pointer;
    margin-bottom: 8px;
    transition: border-color 0.15s;
  }
  .scope-option:last-child {
    margin-bottom: 0;
  }
  .scope-option input[type="radio"] {
    margin-top: 2px;
    flex-shrink: 0;
  }
  .scope-active {
    border-color: var(--accent, #58a6ff);
  }
  .scope-warning {
    border-color: #d29922;
    background: rgba(210, 153, 34, 0.06);
  }
  .scope-label {
    font-weight: 500;
    display: block;
    font-size: 14px;
  }
  .scope-desc {
    font-size: 12px;
    color: var(--text-dim, #888);
    display: block;
    margin-top: 2px;
  }
</style>
