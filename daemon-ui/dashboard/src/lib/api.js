const BASE_URL = window.location.origin;

function getToken() {
  return localStorage.getItem('pma_admin_token') || '';
}

async function request(method, path, { params, body } = {}) {
  let url = `${BASE_URL}${path}`;

  if (params) {
    const qs = new URLSearchParams();
    for (const [k, v] of Object.entries(params)) {
      if (v !== undefined && v !== null) {
        qs.set(k, String(v));
      }
    }
    const qsStr = qs.toString();
    if (qsStr) url += `?${qsStr}`;
  }

  const headers = {
    'Accept': 'application/json',
  };

  const token = getToken();
  if (token) {
    headers['Authorization'] = `Bearer ${token}`;
  }

  const opts = { method, headers };

  if (body !== undefined) {
    headers['Content-Type'] = 'application/json';
    opts.body = JSON.stringify(body);
  }

  const res = await fetch(url, opts);

  if (!res.ok) {
    const text = await res.text().catch(() => '');
    throw new Error(`API ${method} ${path}: ${res.status} ${res.statusText}${text ? ' - ' + text : ''}`);
  }

  const ct = res.headers.get('content-type') || '';
  if (ct.includes('application/json')) {
    return res.json();
  }
  return res.text();
}

// ── Status ──────────────────────────────────────────────────

export function getStatus() {
  return request('GET', '/v1/status');
}

// ── Providers ───────────────────────────────────────────────

export function getProviders() {
  return request('GET', '/v1/providers');
}

// ── Apps ────────────────────────────────────────────────────

export function getApps() {
  return request('GET', '/v1/apps');
}

export function deleteApp(id) {
  return request('DELETE', `/v1/apps/${encodeURIComponent(id)}`);
}

// ── History ─────────────────────────────────────────────────

export function getHistory(limit, offset, { app_name, sort } = {}) {
  return request('GET', '/v1/history', {
    params: { limit, offset, app_name, sort },
  });
}

export function getHistoryEntry(id) {
  return request('GET', `/v1/history/${encodeURIComponent(id)}`);
}

export function deleteHistory() {
  return request('DELETE', '/v1/history');
}

// ── Onboarding ──────────────────────────────────────────────

export function getOnboardingStatus() {
  return request('GET', '/v1/onboarding/status');
}

export function testProvider(providerId) {
  return request('POST', '/v1/onboarding/test-provider', {
    body: { provider_id: providerId },
  });
}

export function completeOnboarding() {
  return request('POST', '/v1/onboarding/complete');
}

// ── Connect / Approve ───────────────────────────────────────

export function getPendingConnects() {
  return request('GET', '/v1/connect/pending');
}

export function getConnectRequest(id) {
  return request('GET', `/v1/connect/${encodeURIComponent(id)}`);
}

export function getConnectProviders(id) {
  return request('GET', `/v1/connect/${encodeURIComponent(id)}/providers`);
}

export function approveConnect(id, providers, scope) {
  const body = {};
  if (providers && providers.length > 0) body.providers = providers;
  if (scope) body.scope = scope;
  return request('POST', `/v1/connect/${encodeURIComponent(id)}/approve`, {
    body: Object.keys(body).length > 0 ? body : undefined,
  });
}

export function denyConnect(id) {
  return request('POST', `/v1/connect/${encodeURIComponent(id)}/deny`);
}
