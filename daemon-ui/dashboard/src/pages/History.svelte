<script>
  import { getHistory, getHistoryEntry, deleteHistory } from '../lib/api.js';
  import { relativeTime, formatNumber, formatDuration, truncate } from '../lib/format.js';

  const PAGE_SIZE = 20;

  let entries = $state([]);
  let loading = $state(true);
  let error = $state(null);
  let page = $state(0);
  let total = $state(0);

  let filterApp = $state('');
  let filterSort = $state('recent');

  let expandedId = $state(null);
  let expandedEntry = $state(null);
  let expandLoading = $state(false);

  let pollInterval = $state(null);

  $effect(() => {
    // Re-run when page, filterApp, or filterSort change
    loadPage(page, filterApp, filterSort);
  });

  // Live reload: poll every 5s when page is visible
  $effect(() => {
    pollInterval = setInterval(() => {
      if (!document.hidden) {
        loadPageQuiet(page, filterApp, filterSort);
      }
    }, 5000);
    return () => clearInterval(pollInterval);
  });

  // Silent reload — doesn't show loading spinner
  async function loadPageQuiet(p, appName, sort) {
    try {
      const res = await getHistory(PAGE_SIZE, p * PAGE_SIZE, {
        app_name: appName || undefined,
        sort: sort !== 'recent' ? sort : undefined,
      });
      entries = res?.entries || [];
      total = res?.total ?? 0;
    } catch {
      // Silently fail on background poll
    }
  }

  async function loadPage(p, appName, sort) {
    loading = true;
    error = null;
    try {
      const res = await getHistory(PAGE_SIZE, p * PAGE_SIZE, {
        app_name: appName || undefined,
        sort: sort !== 'recent' ? sort : undefined,
      });
      entries = res?.entries || [];
      total = res?.total ?? 0;
    } catch (e) {
      error = e.message;
      entries = [];
      total = 0;
    } finally {
      loading = false;
    }
  }

  function totalPages() {
    return Math.max(1, Math.ceil(total / PAGE_SIZE));
  }

  function nextPage() {
    page += 1;
    expandedId = null;
    expandedEntry = null;
  }

  function prevPage() {
    if (page > 0) {
      page -= 1;
      expandedId = null;
      expandedEntry = null;
    }
  }

  function onFilterChange() {
    page = 0;
    expandedId = null;
    expandedEntry = null;
  }

  async function clearHistory() {
    if (!confirm('Delete all history entries? This cannot be undone.')) return;
    try {
      await deleteHistory();
      entries = [];
      total = 0;
      page = 0;
    } catch (e) {
      alert('Failed to clear history: ' + e.message);
    }
  }

  async function toggleExpand(entry) {
    const id = entry.id || entry.request_id;
    if (expandedId === id) {
      expandedId = null;
      expandedEntry = null;
      return;
    }
    expandedId = id;
    expandedEntry = null;
    expandLoading = true;
    try {
      expandedEntry = await getHistoryEntry(id);
    } catch {
      // Use inline data if detail fetch fails
      expandedEntry = entry;
    } finally {
      expandLoading = false;
    }
  }

  function getPromptText(entry) {
    if (entry.prompt) return entry.prompt;
    if (entry.messages && entry.messages.length > 0) {
      const last = entry.messages.filter(m => m.role === 'user').pop();
      return last?.content || entry.messages[entry.messages.length - 1]?.content || '';
    }
    return '';
  }

  function getResponseText(entry) {
    const r = entry.response;
    if (r) {
      // Response is stored as {"content": "..."} object
      if (typeof r === 'object' && r.content !== undefined) return r.content;
      if (typeof r === 'string') return r;
    }
    if (entry.completion) return entry.completion;
    if (entry.choices && entry.choices.length > 0) {
      return entry.choices[0]?.message?.content || entry.choices[0]?.text || '';
    }
    return '';
  }
</script>

<div class="page-header">
  <div class="flex-between">
    <div>
      <h2>History</h2>
      <p>Request and response log</p>
    </div>
    {#if entries.length > 0 || total > 0}
      <button class="btn btn-sm btn-danger" onclick={clearHistory}>Clear History</button>
    {/if}
  </div>
</div>

<div class="filter-row" style="margin-bottom:16px;display:flex;gap:12px;align-items:center;flex-wrap:wrap">
  <input
    class="input"
    type="text"
    placeholder="Filter by app name..."
    style="max-width:240px"
    bind:value={filterApp}
    oninput={onFilterChange}
  />
  <select class="input" style="max-width:160px" bind:value={filterSort} onchange={onFilterChange}>
    <option value="recent">Most recent</option>
    <option value="tokens">Most tokens</option>
  </select>
  {#if total > 0}
    <span style="font-size:13px;color:var(--text-muted)">{total} {total === 1 ? 'entry' : 'entries'}</span>
  {/if}
</div>

{#if loading}
  <div class="loading">Loading history...</div>
{:else if error}
  <div class="error-msg">{error}</div>
{:else if entries.length === 0}
  <div class="empty-state">
    <div class="icon">~</div>
    <h3>No history yet</h3>
    <p>Requests will appear here once apps start making API calls through the proxy.</p>
  </div>
{:else}
  <div class="table-wrap">
    <table>
      <thead>
        <tr>
          <th>Time</th>
          <th>App</th>
          <th>Prompt</th>
          <th>Model</th>
          <th style="text-align:right">Tokens In</th>
          <th style="text-align:right">Tokens Out</th>
          <th style="text-align:right">Duration</th>
        </tr>
      </thead>
      <tbody>
        {#each entries as entry}
          {@const id = entry.id || entry.request_id}
          <tr class="clickable" onclick={() => toggleExpand(entry)}>
            <td class="mono text-dim">{relativeTime(entry.timestamp || entry.created_at)}</td>
            <td>{entry.app_name || entry.app || '--'}</td>
            <td class="text-dim" style="max-width:280px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap">
              {truncate(getPromptText(entry), 60)}
            </td>
            <td><code>{entry.model || '--'}</code></td>
            <td class="mono" style="text-align:right">{formatNumber(entry.tokens_in ?? entry.prompt_tokens)}</td>
            <td class="mono" style="text-align:right">{formatNumber(entry.tokens_out ?? entry.completion_tokens)}</td>
            <td class="mono" style="text-align:right">{formatDuration(entry.duration)}</td>
          </tr>

          {#if expandedId === id}
            <tr class="expanded-row">
              <td colspan="7">
                {#if expandLoading}
                  <div class="loading" style="padding:16px">Loading details...</div>
                {:else if expandedEntry}
                  <div class="expanded-content">
                    <h4>Prompt</h4>
                    <pre>{getPromptText(expandedEntry) || '(empty)'}</pre>
                    <h4>Response</h4>
                    <pre>{getResponseText(expandedEntry) || '(empty)'}</pre>
                    {#if expandedEntry.model}
                      <div style="margin-top:12px;font-size:12px" class="text-dim">
                        Model: <code>{expandedEntry.model}</code>
                        {#if expandedEntry.tokens_in ?? expandedEntry.prompt_tokens}
                          &middot; Tokens: {formatNumber(expandedEntry.tokens_in ?? expandedEntry.prompt_tokens)} in / {formatNumber(expandedEntry.tokens_out ?? expandedEntry.completion_tokens)} out
                        {/if}
                        {#if expandedEntry.duration}
                          &middot; Duration: {formatDuration(expandedEntry.duration)}
                        {/if}
                      </div>
                    {/if}
                  </div>
                {/if}
              </td>
            </tr>
          {/if}
        {/each}
      </tbody>
    </table>
  </div>

  <div class="pagination">
    <button class="btn btn-sm" disabled={page === 0} onclick={prevPage}>Previous</button>
    <span>Page {page + 1} of {totalPages()}</span>
    <button class="btn btn-sm" disabled={(page + 1) >= totalPages()} onclick={nextPage}>Next</button>
  </div>
{/if}
