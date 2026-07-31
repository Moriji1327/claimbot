let paused = false;
let logBuffer = [];
let config = null;
let botsRunning = false;
let logLineCount = 0;

let currentValue = { claims: 0, servers: 0, channels: 0 };
let audioCtx = null;
let soundEnabled = false;

// ======================== BOOT SEQUENCE ========================

function glitchHeader() {
  const el = document.querySelector('.logo .glitch-text');
  if (!el) return;
  el.classList.add('glitching');
}

function playBootSequence() {
  const lines = document.querySelectorAll('.boot-line, .boot-banner');
  let maxDelay = 0;

  lines.forEach(el => {
    const delay = parseInt(el.dataset.delay) || 0;
    maxDelay = Math.max(maxDelay, delay);
    if (el.classList.contains('boot-glitch-line')) {
      setTimeout(() => {
        el.classList.add('visible');
        setTimeout(() => {
          const glitch = el.querySelector('.glitch-text');
          if (glitch) glitch.classList.add('glitching');
        }, 300);
      }, delay);
    } else {
      setTimeout(() => el.classList.add('visible'), delay);
    }
  });

  setTimeout(() => {
    const overlay = document.getElementById('boot-overlay');
    overlay.classList.add('hidden');
    setTimeout(() => overlay.style.display = 'none', 400);
  }, maxDelay + 800);
}

// ======================== MATRIX RAIN ========================

function startMatrixRain() {
  const canvas = document.getElementById('matrixCanvas');
  if (!canvas) return;

  const sidebar = canvas.parentElement;
  canvas.width = sidebar.offsetWidth;
  canvas.height = sidebar.offsetHeight;

  const ctx = canvas.getContext('2d');
  const chars = 'アイウエオカキクケコサシスセソタチツテトナニヌネノハヒフヘホマミムメモヤユヨラリルレロワヲン0123456789ABCDEF';

  const fontSize = 10;
  const columns = Math.floor(canvas.width / fontSize);
  const drops = new Array(columns).fill(1);

  function draw() {
    ctx.fillStyle = 'rgba(5, 5, 5, 0.05)';
    ctx.fillRect(0, 0, canvas.width, canvas.height);

    ctx.font = fontSize + 'px monospace';

    for (let i = 0; i < drops.length; i++) {
      const char = chars[Math.floor(Math.random() * chars.length)];
      const x = i * fontSize;
      const y = drops[i] * fontSize;

      ctx.fillStyle = y < canvas.height / 3 ? 'rgba(0, 255, 65, 0.15)' :
                      y < canvas.height * 2 / 3 ? 'rgba(0, 255, 65, 0.3)' :
                      'rgba(0, 255, 65, 0.05)';
      ctx.fillText(char, x, y);

      if (y > canvas.height && Math.random() > 0.975) {
        drops[i] = 0;
      }
      drops[i]++;
    }
  }

  setInterval(draw, 60);
}

// ======================== TICKER ========================

const tickerMessages = [
  'all channels nominal',
  'scanning ticket queue...',
  '3 active connections',
  'monitoring 47 channels',
  'packet received: handshake ok',
  'system integrity: verified',
  '0 anomalies detected',
  'listening on 8 endpoints',
  'worker threads: active',
  'pulse interval: nominal',
  'cache synchronized',
  'ready for incoming signals',
  'network latency: 12ms',
  'gateway: connected',
  'protocol: v2.0 compliant',
];

let tickerIndex = 0;

async function startTicker() {
  const el = document.getElementById('tickerText');
  if (!el) return;

  function nextMessage() {
    const msg = tickerMessages[tickerIndex];
    tickerIndex = (tickerIndex + 1) % tickerMessages.length;
    typeText(el, msg, 25, () => {
      setTimeout(() => {
        el.textContent = '';
        setTimeout(nextMessage, 800);
      }, 2500);
    });
  }

  // Show real status when bots are running
  setInterval(() => {
    if (botsRunning) {
      const claims = document.getElementById('statClaims')?.textContent || '0';
      const chans = document.getElementById('statChannels')?.textContent || '0';
      typeText(el, claims + ' claims | ' + chans + ' channels tracked', 20);
    }
  }, 8000);

  setTimeout(nextMessage, 1500);
}

function typeText(el, text, speed, cb) {
  let i = 0;
  el.textContent = '';
  function type() {
    if (i < text.length) {
      el.textContent += text[i];
      i++;
      setTimeout(type, speed);
    } else if (cb) {
      cb();
    }
  }
  type();
}

// ======================== CONSOLE WIDGET ========================

const consoleIdleMessages = [
  { text: 'scanning ticket queue...', type: 'dim' },
  { text: 'checking channel integrity', type: 'dim' },
  { text: '3 active connections', type: 'info' },
  { text: 'monitoring 47 channels', type: 'info' },
  { text: 'packet received @ 192.168.x.x', type: 'dim' },
  { text: 'handshake established', type: 'info' },
  { text: 'syncing cache...', type: 'dim' },
  { text: 'cache sync: complete', type: 'info' },
  { text: 'listening on 8 endpoints', type: 'info' },
  { text: 'pulse: nominal', type: 'info' },
  { text: 'system integrity verified', type: 'info' },
  { text: '0 errors in last 60s', type: 'info' },
  { text: 'gateway latency: 11ms', type: 'dim' },
  { text: 'protocol heartbeat: ok', type: 'info' },
  { text: 'scanning for new tickets...', type: 'dim' },
  { text: '2 tickets detected', type: 'warn' },
  { text: '1 claim in queue', type: 'claim' },
  { text: 'processing...', type: 'dim' },
  { text: 'claim injected', type: 'claim' },
  { text: 'awaiting next signal', type: 'dim' },
];

let consoleIdleIndex = 0;
let consoleRealMode = false;
const maxConsoleLines = 14;

function addConsoleLine(text, type) {
  const body = document.getElementById('consoleBody');
  if (!body || body.style.display === 'none') return;

  const div = document.createElement('div');
  div.className = 'console-line';
  div.innerHTML = '<span class="console-prompt">root@sentry:~$</span> <span class="text-' + (type || 'info') + '">' + escapeHtml(text) + '</span>';
  body.appendChild(div);

  while (body.children.length > maxConsoleLines) {
    body.removeChild(body.firstChild);
  }
  body.scrollTop = body.scrollHeight;
}

function startConsoleLoop() {
  function tick() {
    if (!consoleRealMode) {
      const msg = consoleIdleMessages[consoleIdleIndex];
      addConsoleLine(msg.text, msg.type);
      consoleIdleIndex = (consoleIdleIndex + 1) % consoleIdleMessages.length;
    }
    const delay = 1800 + Math.random() * 2200;
    setTimeout(tick, delay);
  }
  tick();
}

function feedRealEvent(entry) {
  consoleRealMode = true;
  const type = entry.level === 'info' ? 'dim' :
               entry.level === 'warn' ? 'warn' :
               entry.level === 'error' ? 'error' : 'info';
  addConsoleLine('[BOT' + entry.botID + '] ' + entry.message, type);
  setTimeout(() => { consoleRealMode = false; }, 2000);
}

function toggleConsole() {
  const widget = document.getElementById('consoleWidget');
  widget.classList.toggle('minimized');
  const btn = widget.querySelector('.console-toggle');
  btn.textContent = widget.classList.contains('minimized') ? '□' : '_';
}

// ======================== CYBER EYE ========================

function startCyberEye() {
  const canvas = document.getElementById('eyeCanvas');
  if (!canvas) return;

  const ctx = canvas.getContext('2d');
  const w = 200, h = 200;
  const cx = w/2, cy = h/2;
  const scleraR = 56, irisR = 30, pupilR = 10;

  let angle = 0;
  let blinkProgress = 1;
  let blinkTimer = 0;
  let nextBlink = 2500 + Math.random() * 3000;
  let targetPupilX = 0, targetPupilY = 0;
  let pupilX = 0, pupilY = 0;

  canvas.addEventListener('mousemove', function(e) {
    const rect = canvas.getBoundingClientRect();
    targetPupilX = ((e.clientX - rect.left) / w - 0.5) * 2;
    targetPupilY = ((e.clientY - rect.top) / h - 0.5) * 2;
    const d = Math.sqrt(targetPupilX*targetPupilX + targetPupilY*targetPupilY);
    if (d > 1) { targetPupilX /= d; targetPupilY /= d; }
  });

  canvas.addEventListener('mouseleave', function() {
    targetPupilX = 0; targetPupilY = 0;
  });

  function draw() {
    ctx.clearRect(0, 0, w, h);

    pupilX += (targetPupilX - pupilX) * 0.08;
    pupilY += (targetPupilY - pupilY) * 0.08;

    blinkTimer++;
    if (blinkProgress >= 1 && blinkTimer > nextBlink) {
      blinkProgress = 0;
      blinkTimer = 0;
      nextBlink = 2000 + Math.random() * 4000;
    }
    if (blinkProgress < 1) {
      blinkProgress += 0.04;
      if (blinkProgress > 1) blinkProgress = 1;
    }
    const blinkValue = Math.sin(blinkProgress * Math.PI);
    angle += 0.015;

    // Background glow
    const bgGlow = ctx.createRadialGradient(cx, cy, 0, cx, cy, 87);
    bgGlow.addColorStop(0, 'rgba(0, 255, 65, 0.04)');
    bgGlow.addColorStop(1, 'transparent');
    ctx.fillStyle = bgGlow;
    ctx.fillRect(0, 0, w, h);

    // Outer orbit rings
    ctx.beginPath();
    ctx.arc(cx, cy, 76, 0, Math.PI * 2);
    ctx.strokeStyle = 'rgba(0, 255, 65, 0.1)';
    ctx.lineWidth = 1.5;
    ctx.setLineDash([3, 7]);
    ctx.lineDashOffset = -angle * 30;
    ctx.stroke();
    ctx.setLineDash([]);

    ctx.beginPath();
    ctx.arc(cx, cy, 69, 0, Math.PI * 2);
    ctx.strokeStyle = 'rgba(0, 255, 65, 0.05)';
    ctx.lineWidth = 1;
    ctx.stroke();

    // Tick marks
    for (let i = 0; i < 12; i++) {
      const a = (i / 12) * Math.PI * 2 + angle;
      ctx.beginPath();
      ctx.moveTo(cx + Math.cos(a) * 76, cy + Math.sin(a) * 76);
      ctx.lineTo(cx + Math.cos(a) * 80, cy + Math.sin(a) * 80);
      ctx.strokeStyle = i % 3 === 0 ? 'rgba(0, 255, 65, 0.25)' : 'rgba(0, 255, 65, 0.07)';
      ctx.lineWidth = 1.5;
      ctx.stroke();
    }

    // Sclera
    ctx.beginPath();
    ctx.arc(cx, cy, scleraR, 0, Math.PI * 2);
    const scleraGrad = ctx.createRadialGradient(cx - 8, cy - 8, 0, cx, cy, scleraR);
    scleraGrad.addColorStop(0, '#0d1a0d');
    scleraGrad.addColorStop(0.6, '#080f08');
    scleraGrad.addColorStop(1, '#020402');
    ctx.fillStyle = scleraGrad;
    ctx.fill();
    ctx.strokeStyle = 'rgba(0, 255, 65, 0.12)';
    ctx.lineWidth = 1;
    ctx.stroke();

    // Iris glow
    ctx.save();
    ctx.shadowColor = '#00ff41';
    ctx.shadowBlur = 20;
    ctx.beginPath();
    ctx.arc(cx, cy, irisR, 0, Math.PI * 2);
    const irisGrad = ctx.createRadialGradient(cx, cy, 0, cx, cy, irisR);
    irisGrad.addColorStop(0, '#00ff41');
    irisGrad.addColorStop(0.3, '#00cc33');
    irisGrad.addColorStop(0.6, '#00881a');
    irisGrad.addColorStop(0.85, '#005511');
    irisGrad.addColorStop(1, '#002a00');
    ctx.fillStyle = irisGrad;
    ctx.fill();
    ctx.restore();

    // Iris fibers
    for (let i = 0; i < 24; i++) {
      const a = (i / 24) * Math.PI * 2 + angle * 0.3;
      ctx.beginPath();
      ctx.moveTo(cx + Math.cos(a) * 4, cy + Math.sin(a) * 4);
      ctx.lineTo(cx + Math.cos(a) * irisR, cy + Math.sin(a) * irisR);
      ctx.strokeStyle = 'rgba(0, 255, 65, 0.07)';
      ctx.lineWidth = 0.8;
      ctx.stroke();
    }

    // Pupil
    const maxOff = 14;
    const pdx = pupilX * maxOff;
    const pdy = pupilY * maxOff;
    ctx.beginPath();
    ctx.arc(cx + pdx, cy + pdy, pupilR, 0, Math.PI * 2);
    ctx.fillStyle = '#000000';
    ctx.fill();
    ctx.strokeStyle = 'rgba(0, 255, 65, 0.25)';
    ctx.lineWidth = 0.5;
    ctx.stroke();

    // Highlight
    ctx.beginPath();
    ctx.arc(cx + pdx - 3, cy + pdy - 3, 2.5, 0, Math.PI * 2);
    ctx.fillStyle = 'rgba(255, 255, 255, 0.15)';
    ctx.fill();

    // Blink eyelids
    if (blinkValue > 0.01) {
      const topLid = cy - scleraR + blinkValue * scleraR * 2;
      const botLid = cy + scleraR - blinkValue * scleraR * 2;
      ctx.fillStyle = '#000000';
      ctx.fillRect(0, 0, w, Math.max(0, topLid));
      ctx.fillRect(0, botLid, w, Math.max(0, h - botLid));
    }

    requestAnimationFrame(draw);
  }

  draw();
}

// ======================== HEARTBEAT ========================

// ======================== API ========================

async function api(method, path, body) {
  try {
    const opts = { method, headers: {} };
    if (body) {
      opts.headers['Content-Type'] = 'application/json';
      opts.body = JSON.stringify(body);
    }
    const res = await fetch('/api' + path, opts);
    if (!res.ok) throw new Error(await res.text());
    return await res.json();
  } catch (e) {
    console.error('API error:', e);
    return null;
  }
}

async function loadConfig() {
  config = await api('GET', '/config');
  if (!config) return;
  const count = Object.keys(config.servers || {}).length;
  document.getElementById('bootCount').textContent = count + ' servers loaded.';
  renderServers();
  renderSettings();
}

// ======================== DASHBOARD RENDERING ========================

let _uptimeStart = null;

function formatUptime(ms) {
  const t = Math.floor(ms / 1000);
  const h = String(Math.floor(t / 3600)).padStart(2, '0');
  const m = String(Math.floor((t % 3600) / 60)).padStart(2, '0');
  const s = String(t % 60).padStart(2, '0');
  return h + ':' + m + ':' + s;
}

function tickUptime() {
  const el = document.getElementById('dashUptime');
  if (!el) return;
  if (_uptimeStart) {
    el.textContent = formatUptime(Date.now() - _uptimeStart);
  } else {
    el.textContent = '--:--:--';
  }
}

function statusClass(s) {
  if (s === 'connected') return 'connected';
  if (s === 'connecting' || s === 'starting') return 'connecting';
  return 'disconnected';
}

function renderDashboard(status) {
  if (!status) return;
  const grid = document.getElementById('botsGrid');
  const bots = status.bots || [];

  if (bots.length === 0) {
    grid.innerHTML = '<div class="bot-card empty"><div class="bot-header">// NO BOTS ACTIVE</div><div class="bot-body">$ bootstrap --start-all</div></div>';
  } else {
    grid.innerHTML = bots.map(b => {
      const cls = statusClass(b.status);
      return '<div class="bot-card">'
        + '<div class="bot-header">'
        + '<span>$ BOT_' + b.id + '</span>'
        + '<span class="bot-status"><span class="status-dot ' + cls + '"></span> ' + b.status + '</span>'
        + '</div>'
        + '<div class="bot-controls">'
        + '<button class="btn btn-small btn-start" onclick="startBot(' + b.id + ')" ' + (b.status === 'connected' ? 'disabled' : '') + '>START</button>'
        + '<button class="btn btn-small btn-stop" onclick="stopBot(' + b.id + ')" ' + (b.status !== 'connected' ? 'disabled' : '') + '>STOP</button>'
        + '</div>'
        + '</div>';
    }).join('');
  }

  const newClaims = status.totalClaims || 0;
  const newServers = status.serverCount || 0;
  const newChannels = status.channelCount || 0;

  animateValue(document.getElementById('dashClaims'), currentValue.claims, newClaims, 400);
  animateValue(document.getElementById('dashServers'), currentValue.servers, newServers, 400);
  animateValue(document.getElementById('dashChannels'), currentValue.channels, newChannels, 400);
  animateValue(document.getElementById('statClaims'), currentValue.claims, newClaims, 400);
  animateValue(document.getElementById('statServers'), currentValue.servers, newServers, 400);
  animateValue(document.getElementById('statChannels'), currentValue.channels, newChannels, 400);

  if (newClaims > currentValue.claims) {
    showToast('claim recorded: ' + newClaims + ' total', 'claim');
    playBlip(880, 0.08, 'square');
  }

  currentValue.claims = newClaims;
  currentValue.servers = newServers;
  currentValue.channels = newChannels;

  const anyConnected = (status.bots || []).some(b => b.status === 'connected');
  if (anyConnected && !_uptimeStart) {
    _uptimeStart = Date.now();
  } else if (!anyConnected) {
    _uptimeStart = null;
  }
  tickUptime();
}

function renderServers() {
  if (!config || !config.servers) return;
  const tbody = document.getElementById('serverBody');
  const entries = Object.entries(config.servers);
  tbody.innerHTML = entries.map(([id, srv]) => {
    const msgs = (srv.messages || []).join(', ') || '-';
    const unclaim = srv.unclaimReply || '-';
    const raffle = srv.raffleReply || '-';
    const disabled = srv.disabled || false;
    const ticketBadge = srv.useTicketNumber ? '<span class="badge badge-yes">TICKET</span>' : '<span class="badge badge-no">MSG</span>';
    const aggroBadge = srv.aggressiveMode ? '<span class="badge badge-yes">AGGRO</span>' : '<span class="badge badge-no">NORM</span>';
    const statusBadge = disabled ? '<span class="badge badge-no">DISABLED</span>' : '<span class="badge badge-yes">ACTIVE</span>';
    const rowClass = disabled ? ' class="row-disabled"' : '';
    return '<tr' + rowClass + '>'
      + '<td><span class="server-name">' + (srv.name || id) + '</span></td>'
      + '<td><span class="server-id">' + id + '</span></td>'
      + '<td>' + (srv.categoryNamePattern || '-') + '</td>'
      + '<td>' + msgs + '</td>'
      + '<td>' + unclaim + '</td>'
      + '<td>' + raffle + '</td>'
      + '<td>' + ticketBadge + ' ' + aggroBadge + '</td>'
      + '<td>' + statusBadge + '</td>'
      + '<td><button class="btn btn-small" onclick="editServer(\'' + id + '\')">EDIT</button> <button class="btn btn-small btn-danger" onclick="deleteServer(\'' + id + '\')">DEL</button></td>'
      + '</tr>';
  }).join('');
}

function renderSettings() {
  if (!config) return;
  document.getElementById('tokensInput').value = (config.sessionTokens || []).join('\n');
  document.getElementById('defClaim').value = config.defaultTriggers?.claim || '';
  document.getElementById('defUnclaim').value = config.defaultTriggers?.unclaim || '';
  document.getElementById('defReopened').value = config.defaultTriggers?.reopened || '';
  document.getElementById('defRaffle').value = config.defaultTriggers?.raffle || '';
  document.getElementById('portInput').value = config.port || 8080;
  document.getElementById('cfCookieInput').value = config.cfCookie || '';
}

async function saveCfCookie() {
  const val = document.getElementById('cfCookieInput').value.trim();
  await api('POST', '/config', { _cfCookie: val });
  const msg = document.getElementById('cfCookieSaved');
  msg.textContent = 'saved. restart bots to apply.';
  addConsoleLine('config: cf cookie updated', 'info');
  setTimeout(() => msg.textContent = '', 3000);
  await loadConfig();
}

// ======================== BOT CONTROLS ========================

async function fetchStatus() {
  const st = await api('GET', '/status');
  renderDashboard(st);
}

function switchTab(name) {
  document.querySelectorAll('.tab').forEach(t => t.classList.remove('active'));
  document.querySelectorAll('.tab-content').forEach(t => t.classList.remove('active'));
  document.querySelector('.tab[data-tab="' + name + '"]').classList.add('active');
  document.getElementById('tab-' + name).classList.add('active');
  document.querySelector('.content')?.scrollTo(0, 0);
  const hidden = name === 'servers';
  document.getElementById('eyeWidget').style.display = hidden ? 'none' : '';
  document.getElementById('consoleWidget').style.display = hidden ? 'none' : '';
}

async function startAll() {
  await api('POST', '/bot/start');
  updateButtons(true);
  addConsoleLine('bootstrap: starting all bots...', 'claim');
  fetchStatus();
}

async function stopAll() {
  await api('POST', '/bot/stop');
  updateButtons(false);
  addConsoleLine('shutdown: all bots terminated', 'warn');
  fetchStatus();
}

async function restartAll() {
  addConsoleLine('reboot: restarting all bots...', 'warn');
  await api('POST', '/bot/restart');
  fetchStatus();
}

async function startBot(id) {
  await api('POST', '/bot/start');
  fetchStatus();
}

async function stopBot(id) {
  await api('POST', '/bot/stop');
  fetchStatus();
}

function updateButtons(running) {
  botsRunning = running;
  document.getElementById('btnStartAll').disabled = running;
  document.getElementById('btnStopAll').disabled = !running;
  document.getElementById('btnRestartAll').disabled = !running;
  const dot = document.getElementById('globalStatus');
  const txt = document.getElementById('statusText');
  if (running) {
    dot.className = 'status-dot connected';
    txt.textContent = 'ACTIVE';
    addConsoleLine('system: all bots operational', 'claim');
  } else {
    dot.className = 'status-dot disconnected';
    txt.textContent = 'IDLE';
  }
}

// ======================== LOG ========================

function togglePause() {
  paused = !paused;
  document.getElementById('btnPause').textContent = paused ? 'RESUME' : 'PAUSE';
}

function clearLogs() {
  logBuffer = [];
  logLineCount = 0;
  document.getElementById('logContainer').innerHTML = '';
}

function applyLogFilter() {
  const filter = document.getElementById('logFilter').value.toLowerCase();
  const container = document.getElementById('logContainer');
  container.innerHTML = '';
  const entries = filter ? logBuffer.filter(e => e.message.toLowerCase().includes(filter)) : logBuffer;
  entries.forEach(e => container.appendChild(createLogEl(e)));
  container.scrollTop = container.scrollHeight;
}

function createLogEl(entry) {
  logLineCount++;
  const div = document.createElement('div');
  div.className = 'log-entry';
  const levelClass = 'level-' + (entry.level || 'info');
  div.innerHTML = '<span class="log-line-num">' + logLineCount + '</span><span class="log-prompt">$</span><span class="timestamp">[' + entry.timestamp + ']</span> <span class="bot-tag b' + (entry.botID % 4) + '">BOT' + entry.botID + '</span> <span class="' + levelClass + '">' + escapeHtml(entry.message) + '</span>';
  return div;
}

function escapeHtml(s) {
  const d = document.createElement('div');
  d.textContent = s;
  return d.innerHTML;
}

function addLogEntry(entry) {
  logBuffer.push(entry);
  if (logBuffer.length > 5000) logBuffer.splice(0, logBuffer.length - 5000);
  if (paused) return;
  const filter = document.getElementById('logFilter').value.toLowerCase();
  if (filter && !entry.message.toLowerCase().includes(filter)) return;
  const container = document.getElementById('logContainer');
  container.appendChild(createLogEl(entry));
  container.scrollTop = container.scrollHeight;
  feedRealEvent(entry);
}

function setupSSE() {
  const evtSource = new EventSource('/api/logs');
  evtSource.onmessage = function(e) {
    try {
      const entry = JSON.parse(e.data);
      addLogEntry(entry);
    } catch (_) {}
  };
  evtSource.onerror = function() {
    setTimeout(setupSSE, 2000);
  };
}

// ======================== ANIMATED COUNTERS (Phase 1) ========================

function animateValue(el, from, to, duration) {
  const start = performance.now();
  const diff = to - from;
  if (diff === 0) { el.textContent = to; return; }

  function tick(now) {
    const t = Math.min((now - start) / duration, 1);
    const ease = 1 - Math.pow(1 - t, 3);
    el.textContent = Math.round(from + diff * ease);
    if (t < 1) requestAnimationFrame(tick);
  }
  requestAnimationFrame(tick);
}

// ======================== TOAST SYSTEM (Phase 1) ========================

function showToast(message, type) {
  const container = document.getElementById('toastContainer');
  if (!container) return;
  const el = document.createElement('div');
  el.className = 'toast toast-' + (type || 'info');
  el.textContent = message;
  container.appendChild(el);
  setTimeout(() => {
    el.classList.add('toast-out');
    setTimeout(() => el.remove(), 200);
  }, 3000);
}

// ======================== THEME SWITCHER (Phase 3) ========================

function setTheme(name) {
  document.documentElement.setAttribute('data-theme', name);
  localStorage.setItem('sentry-theme', name);
  document.querySelectorAll('.theme-dot').forEach(d => {
    d.classList.toggle('active', d.dataset.theme === name);
  });

}

function loadTheme() {
  const saved = localStorage.getItem('sentry-theme');
  if (saved) setTheme(saved);
}

// ======================== COMMAND PALETTE (Phase 4) ========================

const commands = [
  { id: 'start-all', label: 'start all bots', hint: 'POST /api/bot/start', action: startAll },
  { id: 'stop-all', label: 'stop all bots', hint: 'POST /api/bot/stop', action: stopAll },
  { id: 'restart-all', label: 'restart all bots', hint: 'POST /api/bot/restart', action: restartAll },
  { id: 'tab-dashboard', label: 'switch to dashboard', hint: 'Ctrl+1', action: () => switchTab('dashboard') },
  { id: 'tab-logs', label: 'switch to live log', hint: 'Ctrl+2', action: () => switchTab('logs') },
  { id: 'tab-servers', label: 'switch to servers', hint: 'Ctrl+3', action: () => switchTab('servers') },
  { id: 'tab-settings', label: 'switch to settings', hint: 'Ctrl+4', action: () => switchTab('settings') },
  { id: 'theme-green', label: 'theme: green phosphor', hint: '', action: () => setTheme('green') },
  { id: 'theme-amber', label: 'theme: amber terminal', hint: '', action: () => setTheme('amber') },
  { id: 'theme-cyan', label: 'theme: cyan hacker', hint: '', action: () => setTheme('cyan') },
  { id: 'theme-ibm', label: 'theme: ibm 3270 blue', hint: '', action: () => setTheme('ibm') },
  { id: 'clear-logs', label: 'clear logs', hint: '', action: clearLogs },
  { id: 'pause-logs', label: 'toggle log pause', hint: '', action: togglePause },
];

function openCommandPalette() {
  const overlay = document.getElementById('commandPalette');
  const input = document.getElementById('cmdInput');
  const results = document.getElementById('cmdResults');
  if (!overlay) return;
  overlay.classList.add('open');
  results.innerHTML = '';
  input.value = '';
  input.focus();
  filterCommands('', results);
}

function closeCommandPalette() {
  const overlay = document.getElementById('commandPalette');
  if (overlay) overlay.classList.remove('open');
}

function filterCommands(query, resultsEl) {
  const q = query.toLowerCase();
  const filtered = commands.filter(c => c.label.includes(q));
  resultsEl.innerHTML = '';
  if (filtered.length === 0) {
    resultsEl.innerHTML = '<div class="cmd-empty">no matching commands</div>';
    return;
  }
  filtered.forEach((cmd, i) => {
    const div = document.createElement('div');
    div.className = 'cmd-item' + (i === 0 ? ' selected' : '');
    div.dataset.id = cmd.id;
    div.innerHTML = '<span class="cmd-label">' + cmd.label + '</span><span class="cmd-hint">' + cmd.hint + '</span>';
    div.onclick = () => { closeCommandPalette(); cmd.action(); };
    div.onmouseenter = function() {
      document.querySelectorAll('#cmdResults .cmd-item').forEach(function(e) { e.classList.remove('selected'); });
      this.classList.add('selected');
    };
    resultsEl.appendChild(div);
  });
}

function getCSSVar(name) {
  return getComputedStyle(document.documentElement).getPropertyValue(name).trim();
}

// ======================== SOUND EFFECTS (Phase 3) ========================

function initAudio() {
  try { audioCtx = new (window.AudioContext || window.webkitAudioContext)(); } catch (_) {}
}

function playBlip(freq, duration, type) {
  if (!soundEnabled || !audioCtx) return;
  const osc = audioCtx.createOscillator();
  const gain = audioCtx.createGain();
  osc.type = type || 'square';
  osc.frequency.value = freq;
  gain.gain.setValueAtTime(0.04, audioCtx.currentTime);
  gain.gain.exponentialRampToValueAtTime(0.001, audioCtx.currentTime + duration);
  osc.connect(gain);
  gain.connect(audioCtx.destination);
  osc.start();
  osc.stop(audioCtx.currentTime + duration);
}

function toggleSound() {
  soundEnabled = document.getElementById('soundEnabled').checked;
  localStorage.setItem('sentry-sound', soundEnabled ? '1' : '0');
  if (soundEnabled && !audioCtx) initAudio();
}

function loadSoundPref() {
  soundEnabled = localStorage.getItem('sentry-sound') === '1';
  const cb = document.getElementById('soundEnabled');
  if (cb) cb.checked = soundEnabled;
}

// ======================== KEYBOARD SHORTCUTS ========================

document.addEventListener('keydown', function(e) {
  if ((e.ctrlKey || e.metaKey) && e.key === 'k') {
    e.preventDefault();
    openCommandPalette();
  }
  if ((e.ctrlKey || e.metaKey) && e.key >= '1' && e.key <= '4') {
    e.preventDefault();
    const tabs = ['dashboard', 'logs', 'servers', 'settings'];
    switchTab(tabs[parseInt(e.key) - 1]);
  }
  if (e.key === 'Escape') {
    const palette = document.getElementById('commandPalette');
    if (palette && palette.classList.contains('open')) closeCommandPalette();
  }
  if (e.key === 'Enter') {
    const palette = document.getElementById('commandPalette');
    if (palette && palette.classList.contains('open')) {
      const selected = document.querySelector('#cmdResults .cmd-item.selected');
      if (selected) {
        const cmd = commands.find(c => c.id === selected.dataset.id);
        if (cmd) { closeCommandPalette(); cmd.action(); }
      }
    }
  }
  if (e.key === 'ArrowDown' || e.key === 'ArrowUp') {
    const palette = document.getElementById('commandPalette');
    if (palette && palette.classList.contains('open')) {
      e.preventDefault();
      const items = document.querySelectorAll('#cmdResults .cmd-item');
      const sel = document.querySelector('#cmdResults .cmd-item.selected');
      let idx = Array.from(items).indexOf(sel);
      if (e.key === 'ArrowDown') idx = Math.min(idx + 1, items.length - 1);
      else idx = Math.max(idx - 1, 0);
      items.forEach(i => i.classList.remove('selected'));
      if (items[idx]) items[idx].classList.add('selected');
    }
  }
});

window.addEventListener('beforeunload', function(e) {
  if (botsRunning) {
    e.preventDefault();
    e.returnValue = 'Sentry is still running! Closing this tab will NOT stop the bot.\nReopen http://localhost:8080 to control it.';
  }
});

// ======================== SERVER CONFIG ========================

async function showAddServer() {
  document.getElementById('modalTitle').textContent = '> add server';
  document.getElementById('editServerId').value = '';
  document.getElementById('srvId').value = '';
  document.getElementById('srvId').disabled = false;
  document.getElementById('srvName').value = '';
  document.getElementById('srvCat').value = 'ticket';
  document.getElementById('srvMsgs').value = '/claim';
  document.getElementById('srvUnclaim').value = '';
  document.getElementById('srvRaffle').value = '';
  document.getElementById('srvTicketNum').checked = false;
  document.getElementById('srvTicketPrefix').value = '';
  document.getElementById('srvDisabled').checked = false;
  document.getElementById('srvAggro').checked = false;
  document.getElementById('srvTrigClaim').value = '';
  document.getElementById('srvTrigUnclaim').value = '';
  document.getElementById('srvTrigReopened').value = '';
  document.getElementById('srvTrigRaffle').value = '';
  document.getElementById('serverModal').classList.add('open');
}

function editServer(id) {
  const srv = config.servers[id];
  if (!srv) return;
  document.getElementById('modalTitle').textContent = '> edit server';
  document.getElementById('editServerId').value = id;
  document.getElementById('srvId').value = id;
  document.getElementById('srvId').disabled = true;
  document.getElementById('srvName').value = srv.name || '';
  document.getElementById('srvCat').value = srv.categoryNamePattern || '';
  document.getElementById('srvMsgs').value = (srv.messages || []).join('\n');
  document.getElementById('srvUnclaim').value = srv.unclaimReply || '';
  document.getElementById('srvRaffle').value = srv.raffleReply || '';
  document.getElementById('srvTicketNum').checked = srv.useTicketNumber || false;
  document.getElementById('srvTicketPrefix').value = srv.ticketPrefix || '';
  document.getElementById('srvDisabled').checked = srv.disabled || false;
  document.getElementById('srvAggro').checked = srv.aggressiveMode || false;
  document.getElementById('srvTrigClaim').value = srv.triggerClaim || '';
  document.getElementById('srvTrigUnclaim').value = srv.triggerUnclaim || '';
  document.getElementById('srvTrigReopened').value = srv.triggerReopened || '';
  document.getElementById('srvTrigRaffle').value = srv.triggerRaffle || '';
  document.getElementById('serverModal').classList.add('open');
}

function closeModal() {
  document.getElementById('serverModal').classList.remove('open');
}

function collectServerForm() {
  const msgs = document.getElementById('srvMsgs').value.split('\n').map(s => s.trim()).filter(Boolean);
  return {
    name: document.getElementById('srvName').value.trim(),
    categoryNamePattern: document.getElementById('srvCat').value.trim(),
    messages: msgs,
    unclaimReply: document.getElementById('srvUnclaim').value.trim(),
    raffleReply: document.getElementById('srvRaffle').value.trim(),
    useTicketNumber: document.getElementById('srvTicketNum').checked,
    ticketPrefix: document.getElementById('srvTicketPrefix').value.trim(),
    aggressiveMode: document.getElementById('srvAggro').checked,
    disabled: document.getElementById('srvDisabled').checked,
    triggerClaim: document.getElementById('srvTrigClaim').value.trim(),
    triggerUnclaim: document.getElementById('srvTrigUnclaim').value.trim(),
    triggerReopened: document.getElementById('srvTrigReopened').value.trim(),
    triggerRaffle: document.getElementById('srvTrigRaffle').value.trim(),
  };
}

async function saveServer() {
  const id = document.getElementById('srvId').value.trim();
  if (!id) return alert('server id is required');
  const data = collectServerForm();
  data.serverId = id;
  await api('POST', '/config', data);
  addConsoleLine('config: server ' + id.slice(0, 8) + '... saved', 'info');
  await loadConfig();
  closeModal();
}

async function deleteServer(id) {
  if (!confirm('delete server "' + (config.servers[id]?.name || id) + '"?')) return;
  await api('POST', '/config', { serverId: id, _delete: true });
  addConsoleLine('config: server ' + id.slice(0, 8) + '... removed', 'warn');
  await loadConfig();
}

async function saveTokens() {
  const val = document.getElementById('tokensInput').value.trim();
  const tokens = val ? val.split('\n').map(s => s.trim()).filter(Boolean) : [];
  await api('POST', '/config', { _tokens: tokens });
  const msg = document.getElementById('tokensSaved');
  msg.textContent = 'saved.';
  addConsoleLine('config: tokens updated', 'info');
  setTimeout(() => msg.textContent = '', 2000);
  await loadConfig();
}

async function saveDefaults() {
  await api('POST', '/config', {
    _defaults: {
      claim: document.getElementById('defClaim').value.trim(),
      unclaim: document.getElementById('defUnclaim').value.trim(),
      reopened: document.getElementById('defReopened').value.trim(),
      raffle: document.getElementById('defRaffle').value.trim(),
    }
  });
  const msg = document.getElementById('defaultsSaved');
  msg.textContent = 'saved.';
  addConsoleLine('config: default triggers updated', 'info');
  setTimeout(() => msg.textContent = '', 2000);
  await loadConfig();
}

async function savePort() {
  const port = parseInt(document.getElementById('portInput').value);
  if (!port || port < 1024 || port > 65535) return alert('port must be 1024-65535');
  await api('POST', '/config', { _port: port });
  const msg = document.getElementById('portSaved');
  msg.textContent = 'saved. restart to apply.';
  addConsoleLine('config: port changed to ' + port + ' (restart required)', 'warn');
  setTimeout(() => msg.textContent = '', 3000);
  await loadConfig();
}

// ======================== INIT ========================

document.addEventListener('DOMContentLoaded', async () => {
  loadTheme();
  loadSoundPref();

  startMatrixRain();
  await loadConfig();
  const st = await api('GET', '/status');
  if (st) {
    const running = st.bots && st.bots.length > 0;
    updateButtons(running);
    renderDashboard(st);
  }
  setupSSE();
  setInterval(fetchStatus, 3000);
  startConsoleLoop();
  startTicker();
  startCyberEye();
  setInterval(tickUptime, 1000);
  playBootSequence();
  glitchHeader();
  document.getElementById('cmdInput')?.addEventListener('input', function() {
    filterCommands(this.value, document.getElementById('cmdResults'));
  });
});
