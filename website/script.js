// Copy to clipboard
function copyText(text, btn) {
  navigator.clipboard.writeText(text).then(function () {
    btn.classList.add('copied');
    setTimeout(function () { btn.classList.remove('copied'); }, 1500);
  });
}

// GitHub stars
(function () {
  var el = document.getElementById('github-stars');
  if (!el) return;
  fetch('https://api.github.com/repos/sxp-studio/PlugMyAI')
    .then(function (r) { return r.json(); })
    .then(function (data) {
      if (data.stargazers_count !== undefined) {
        el.textContent = data.stargazers_count;
      }
    })
    .catch(function () { /* silently fail */ });
})();

// Hero slot animation
(function () {
  var slot = document.getElementById('hero-slot');
  if (!slot) return;
  var items = slot.children;
  var current = 0;
  setInterval(function () {
    current = (current + 1) % items.length;
    slot.style.transform = 'translateY(-' + (current * 1.15) + 'em)';
  }, 3000);
})();

// ── Plug My AI button ───────────────────────────────────────
//
// State machine:
//   'offline'    → daemon not reachable → click goes to #install
//   'ready'      → daemon reachable, no token → click starts pairing
//   'connecting' → pairing in progress
//   'connected'  → has valid token → button shows current provider, click toggles dropdown
//

var DAEMON_URL = 'http://localhost:21110';
var _state = 'offline';
var _models = [];       // [{id, owned_by}, ...]
var _currentModel = ''; // currently selected model ID
var _pollTimer = null;

// All pma-btn instances on the page (integration preview + demo)
function allBtns() {
  return [
    { btn: document.getElementById('demo-btn'), wrap: document.getElementById('demo-wrap'), dropdown: document.getElementById('demo-dropdown') },
    { btn: document.getElementById('demo-header-btn'), wrap: document.getElementById('demo-header-wrap'), dropdown: document.getElementById('demo-header-dropdown') },
  ].filter(function (b) { return b.btn; });
}

function setState(state) {
  _state = state;
  allBtns().forEach(function (b) {
    b.btn.classList.remove('connected', 'connecting');
    b.wrap.classList.remove('open');
    var span = b.btn.querySelector('span');

    if (state === 'connected') {
      b.btn.classList.add('connected');
      var label = _currentModel || 'Connected';
      // Show provider name instead of model ID if possible
      var model = _models.find(function (m) { return m.id === _currentModel; });
      if (model) label = model.owned_by + ' — ' + model.id;
      if (span) span.textContent = label;
    } else if (state === 'connecting') {
      b.btn.classList.add('connecting');
      if (span) span.textContent = 'Connecting...';
    } else {
      if (span) span.textContent = 'Plug My AI';
    }
  });
}

function setStatus(msg, isError) {
  var el = document.getElementById('demo-status');
  if (!el) return;
  el.textContent = msg;
  el.classList.toggle('error', !!isError);
}

function populateDropdowns() {
  allBtns().forEach(function (b) {
    b.dropdown.innerHTML = '';

    // Group by provider
    var groups = {};
    _models.forEach(function (m) {
      var p = m.owned_by || 'unknown';
      if (!groups[p]) groups[p] = [];
      groups[p].push(m);
    });

    var providers = Object.keys(groups);
    providers.forEach(function (provider) {
      if (providers.length > 1) {
        var label = document.createElement('div');
        label.className = 'pma-dropdown-group';
        label.textContent = provider;
        b.dropdown.appendChild(label);
      }
      groups[provider].forEach(function (m) {
        var item = document.createElement('button');
        item.className = 'pma-dropdown-item';
        if (m.id === _currentModel) item.classList.add('active');
        item.textContent = m.id;
        item.onclick = function (e) {
          e.stopPropagation();
          selectModel(m.id);
        };
        b.dropdown.appendChild(item);
      });
    });
  });
}

function selectModel(modelId) {
  _currentModel = modelId;
  localStorage.setItem('pma_demo_model', modelId);
  setState('connected');
  populateDropdowns();

  // Hide onboarding, show header, unblur input, enable chat
  var onboarding = document.getElementById('demo-onboarding');
  if (onboarding) onboarding.style.display = 'none';
  var header = document.getElementById('demo-chat-header');
  if (header) header.style.display = '';
  var inputRow = document.getElementById('demo-input-row');
  if (inputRow) inputRow.classList.remove('blurred');
  var input = document.getElementById('demo-input');
  if (input) { input.disabled = false; input.focus(); }
  var sendBtn = document.getElementById('demo-send');
  if (sendBtn) sendBtn.disabled = false;
}

// Close dropdowns when clicking outside
document.addEventListener('click', function (e) {
  allBtns().forEach(function (b) {
    if (!b.wrap.contains(e.target)) {
      b.wrap.classList.remove('open');
    }
  });
});

// ── Button click handler ────────────────────────────────────

function handlePlugClick() {
  if (_state === 'offline') {
    // Try launching PlugMyAI via custom URL scheme.
    // If the app is installed, macOS will open it; if not, nothing visible happens.
    // We poll for the daemon to come online. If it doesn't after a few seconds,
    // fall back to the install section.
    setState('connecting');
    setStatus('Launching PlugMyAI...');
    window.location.href = 'plugmyai://connect';
    waitForDaemon(0);
    return;
  }

  if (_state === 'connecting') {
    return; // already in progress
  }

  if (_state === 'connected') {
    // Toggle provider dropdown
    allBtns().forEach(function (b) {
      b.wrap.classList.toggle('open');
    });
    return;
  }

  // state === 'ready' → start pairing
  startPairing();
}

// Poll for daemon to come online after URL scheme launch attempt.
// Cold-starting the app + daemon can take 10-15s, so we poll generously.
// If it doesn't appear within ~15s, assume not installed → go to #install.
function waitForDaemon(attempt) {
  if (attempt >= 15) {
    setState('offline');
    setStatus('');
    window.location.href = '#install';
    return;
  }

  setTimeout(function () {
    fetch(DAEMON_URL + '/v1/status', { mode: 'cors' })
      .then(function (r) { return r.json(); })
      .then(function () {
        // Daemon is up — start pairing
        setState('ready');
        setStatus('');
        startPairing();
      })
      .catch(function () {
        waitForDaemon(attempt + 1);
      });
  }, 1000);
}

// ── Init: detect daemon or validate existing token ──────────

(function () {
  var token = localStorage.getItem('pma_demo_token');
  if (token) {
    validateToken(token);
  } else {
    detectDaemon();
  }
})();

function detectDaemon() {
  fetch(DAEMON_URL + '/v1/status', { mode: 'cors' })
    .then(function (r) { return r.json(); })
    .then(function () {
      setState('ready');
    })
    .catch(function () {
      setState('offline');
    });
}

function validateToken(token) {
  fetch(DAEMON_URL + '/v1/models', {
    headers: { 'Authorization': 'Bearer ' + token },
    mode: 'cors',
  })
    .then(function (r) {
      if (!r.ok) throw new Error('invalid');
      return r.json();
    })
    .then(function (data) {
      _models = data.data || [];
      var saved = localStorage.getItem('pma_demo_model');

      if (_models.length === 1) {
        selectModel(_models[0].id);
      } else if (saved && _models.some(function (m) { return m.id === saved; })) {
        selectModel(saved);
      } else if (_models.length > 0) {
        selectModel(_models[0].id);
      }

      setState('connected');
      populateDropdowns();
      setStatus('');
    })
    .catch(function () {
      localStorage.removeItem('pma_demo_token');
      localStorage.removeItem('pma_demo_model');
      detectDaemon();
    });
}

// ── Pairing flow ────────────────────────────────────────────

function startPairing() {
  setState('connecting');
  setStatus('Requesting connection...');

  fetch(DAEMON_URL + '/v1/connect', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    mode: 'cors',
    body: JSON.stringify({
      app_name: 'plugmy.ai Demo',
      app_url: 'https://plugmy.ai',
    }),
  })
    .then(function (r) {
      if (!r.ok) throw new Error('Failed to connect');
      return r.json();
    })
    .then(function (data) {
      setStatus('Waiting for approval — check for a dialog on your Mac...');
      pollForApproval(data.request_id, 0);
    })
    .catch(function (err) {
      setState('ready');
      setStatus(err.message || 'Connection failed', true);
    });
}

function pollForApproval(reqId, attempt) {
  if (attempt >= 100) {
    setState('ready');
    setStatus('Timed out waiting for approval.', true);
    return;
  }

  _pollTimer = setTimeout(function () {
    fetch(DAEMON_URL + '/v1/connect/' + reqId, { mode: 'cors' })
      .then(function (r) { return r.json(); })
      .then(function (data) {
        if (data.status === 'approved' && data.token) {
          localStorage.setItem('pma_demo_token', data.token);
          setStatus('');
          validateToken(data.token);
        } else if (data.status === 'denied') {
          setState('ready');
          setStatus('Connection denied.', true);
        } else if (data.status === 'expired') {
          setState('ready');
          setStatus('Request expired. Try again.', true);
        } else {
          pollForApproval(reqId, attempt + 1);
        }
      })
      .catch(function () {
        pollForApproval(reqId, attempt + 1);
      });
  }, 3000);
}

// ── Chat (demo section only) ────────────────────────────────

function sendDemo(e) {
  e.preventDefault();
  var input = document.getElementById('demo-input');
  var msg = input.value.trim();
  if (!msg) return;

  var token = localStorage.getItem('pma_demo_token');
  if (!token || !_currentModel) return;

  input.value = '';
  appendMessage('user', msg);
  var sendBtn = document.getElementById('demo-send');
  sendBtn.disabled = true;
  input.disabled = true;

  var assistantEl = appendMessage('assistant', '');
  assistantEl.innerHTML = '<span class="thinking-dots"><span></span><span></span><span></span></span>';

  fetch(DAEMON_URL + '/v1/chat/completions', {
    method: 'POST',
    headers: {
      'Authorization': 'Bearer ' + token,
      'Content-Type': 'application/json',
    },
    mode: 'cors',
    body: JSON.stringify({
      model: _currentModel,
      messages: [{ role: 'user', content: msg }],
      stream: true,
    }),
  })
    .then(function (r) {
      if (!r.ok) throw new Error('Request failed: ' + r.status);
      return readSSE(r.body, assistantEl);
    })
    .then(function () {
      sendBtn.disabled = false;
      input.disabled = false;
      input.focus();
    })
    .catch(function (err) {
      assistantEl.textContent = 'Error: ' + (err.message || 'unknown error');
      assistantEl.style.color = '#ef4444';
      sendBtn.disabled = false;
      input.disabled = false;
    });
}

function appendMessage(role, text) {
  var messages = document.getElementById('demo-messages');
  var el = document.createElement('div');
  el.className = 'demo-msg ' + role;
  el.textContent = text;
  messages.appendChild(el);
  messages.scrollTop = messages.scrollHeight;
  return el;
}

function readSSE(body, el) {
  var reader = body.getReader();
  var decoder = new TextDecoder();
  var buffer = '';
  var content = '';

  function processChunk(result) {
    if (result.done) return;

    buffer += decoder.decode(result.value, { stream: true });
    var lines = buffer.split('\n');
    buffer = lines.pop();

    for (var i = 0; i < lines.length; i++) {
      var line = lines[i].trim();
      if (!line.startsWith('data: ')) continue;
      var data = line.substring(6);
      if (data === '[DONE]') return;

      try {
        var parsed = JSON.parse(data);
        if (parsed.error) {
          el.textContent = 'Error: ' + (parsed.error.message || 'unknown');
          el.style.color = '#ef4444';
          return;
        }
        var delta = parsed.choices && parsed.choices[0] && parsed.choices[0].delta;
        if (delta && delta.content) {
          content += delta.content;
          el.textContent = content;
          document.getElementById('demo-messages').scrollTop = document.getElementById('demo-messages').scrollHeight;
        }
      } catch (e) {
        // skip
      }
    }

    return reader.read().then(processChunk);
  }

  return reader.read().then(processChunk);
}
