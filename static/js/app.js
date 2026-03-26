'use strict';

let txnType = 'withdraw';

document.addEventListener('DOMContentLoaded', () => {
  checkServer();
  loadCards();
  setInterval(loadCards, 5000);

  document.getElementById('seg').addEventListener('click', e => {
    const btn = e.target.closest('.seg-btn');
    if (!btn) return;
    document.querySelectorAll('.seg-btn').forEach(b => b.classList.remove('active'));
    btn.classList.add('active');
    txnType = btn.dataset.v;
  });

  document.getElementById('btnExec').addEventListener('click', runTransaction);
  document.getElementById('btnBal').addEventListener('click', fetchBalance);
  document.getElementById('btnHist').addEventListener('click', fetchHistory);
  document.getElementById('btnClr').addEventListener('click', () => {
    document.getElementById('feedList').innerHTML = '<div class="placeholder">no activity yet</div>';
  });
});

async function checkServer() {
  try {
    const r = await fetch('/api/health');
    if (r.ok) {
      document.getElementById('pulse').classList.add('on');
      document.getElementById('srvLabel').textContent = 'online';
    }
  } catch {
    document.getElementById('srvLabel').textContent = 'offline';
  }
}

async function loadCards() {
  try {
    const r    = await fetch('/api/cards');
    const list = await r.json();
    paintCards(list);
  } catch {
    document.getElementById('cardList').innerHTML =
      '<div class="placeholder" style="color:var(--red)">connection error</div>';
  }
}

function paintCards(list) {
  document.getElementById('cardBadge').textContent = list.length;
  if (!list.length) {
    document.getElementById('cardList').innerHTML = '<div class="placeholder">no cards</div>';
    return;
  }
  document.getElementById('cardList').innerHTML = list
    .sort((a, b) => a.cardHolder.localeCompare(b.cardHolder))
    .map(c => `
      <div class="card-tile ${c.status === 'BLOCKED' ? 'is-blocked' : ''}"
           onclick="pickCard('${c.cardNumber}')">
        <div class="ct-num">${groupDigits(c.cardNumber)}</div>
        <div class="ct-name">${c.cardHolder}</div>
        <div class="ct-row">
          <span class="ct-bal">₹${money(c.balance)}</span>
          <span class="status-tag ${c.status === 'ACTIVE' ? 'active' : 'blocked'}">${c.status}</span>
        </div>
      </div>`).join('');
}

function pickCard(n) {
  document.getElementById('iCard').value   = n;
  document.getElementById('iLookup').value = n;
  document.getElementById('iPin').focus();
}

async function runTransaction() {
  const card = document.getElementById('iCard').value.trim();
  const pin  = document.getElementById('iPin').value.trim();
  const amt  = parseFloat(document.getElementById('iAmt').value);

  if (!card) { flash('iCard');  return; }
  if (!pin)  { flash('iPin');   return; }
  if (!amt || amt <= 0) { flash('iAmt'); return; }

  const btn = document.getElementById('btnExec');
  btn.disabled = true;
  btn.textContent = 'Processing...';

  try {
    const r    = await fetch('/api/transaction', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ cardNumber: card, pin, type: txnType, amount: amt }),
    });
    const data = await r.json();
    showResult(data, txnType, amt);
    pushFeed(data, card, txnType, amt);
    if (data.status === 'SUCCESS') loadCards();
  } catch {
    showResult({ status:'FAILED', respCode:'XX', message:'Network error' }, txnType, amt);
  } finally {
    btn.disabled = false;
    btn.textContent = 'Process Transaction';
  }
}

function showResult(d, type, amt) {
  const ok   = d.status === 'SUCCESS';
  const box  = document.getElementById('resultBox');
  const ttl  = document.getElementById('resultTitle');
  const tbl  = document.getElementById('resultTable');
  const pill = document.getElementById('rcPill');

  box.classList.remove('hidden');
  ttl.className   = 'result-title ' + (ok ? 'ok' : 'err');
  ttl.textContent = ok ? '✓  Approved' : '✗  Declined';
  pill.textContent = 'RC: ' + (d.respCode || '—');

  if (ok) {
    tbl.innerHTML = `
      <tr><td>Status</td>   <td class="val-green">SUCCESS</td></tr>
      <tr><td>Resp Code</td><td>${d.respCode}</td></tr>
      <tr><td>Type</td>     <td>${type}</td></tr>
      <tr><td>Amount</td>   <td>₹${money(amt)}</td></tr>
      <tr><td>New Balance</td><td class="val-green">₹${money(d.balance)}</td></tr>`;
  } else {
    tbl.innerHTML = `
      <tr><td>Status</td>   <td class="val-red">FAILED</td></tr>
      <tr><td>Resp Code</td><td class="val-red">${d.respCode}</td></tr>
      <tr><td>Reason</td>   <td>${d.message || 'unknown'}</td></tr>`;
  }
}

async function fetchBalance() {
  const n   = document.getElementById('iLookup').value.trim();
  if (!n) { flash('iLookup'); return; }

  const out = document.getElementById('lookupResult');
  out.classList.remove('hidden');
  out.innerHTML = 'loading…';

  try {
    const r = await fetch('/api/card/balance/' + n);
    const d = await r.json();
    if (r.ok) {
      out.innerHTML = `
        <table>
          <tr><td>Card</td>   <td>${groupDigits(d.cardNumber)}</td></tr>
          <tr><td>Holder</td> <td>${d.cardHolder}</td></tr>
          <tr><td>Balance</td><td style="color:var(--green);font-weight:800;font-family:var(--mono)">₹${money(d.balance)}</td></tr>
          <tr><td>Status</td> <td><span class="tag ${d.status === 'ACTIVE' ? 'success' : 'failed'}">${d.status}</span></td></tr>
        </table>`;
    } else {
      out.innerHTML = `<span style="color:var(--red);font-weight:700">${d.message}</span>`;
    }
  } catch {
    out.innerHTML = '<span style="color:var(--red)">network error</span>';
  }
}

async function fetchHistory() {
  const n   = document.getElementById('iLookup').value.trim();
  if (!n) { flash('iLookup'); return; }

  const out = document.getElementById('lookupResult');
  out.classList.remove('hidden');
  out.innerHTML = 'loading…';

  try {
    const r    = await fetch('/api/card/transactions/' + n);
    const data = await r.json();

    if (!r.ok) { out.innerHTML = `<span style="color:var(--red);font-weight:700">${data.message}</span>`; return; }
    if (!data.length) { out.innerHTML = '<span style="color:var(--ink-3)">No transactions on record.</span>'; return; }

    out.innerHTML = [...data].reverse().slice(0, 30).map(t => `
      <div class="txn-row">
        <div class="txn-meta">
          <span class="tag ${t.type}">${t.type}</span>
          <span class="tag ${t.status.toLowerCase()}">${t.status}</span>
        </div>
        <div class="txn-sub">
          <span>₹${money(t.amount)}</span>
          <span>${hms(t.timestamp)}</span>
        </div>
        <div class="txn-id">${t.transactionId}</div>
      </div>`).join('');
  } catch {
    out.innerHTML = '<span style="color:var(--red)">network error</span>';
  }
}

function pushFeed(d, card, type, amt) {
  const list  = document.getElementById('feedList');
  const empty = list.querySelector('.placeholder');
  if (empty) empty.remove();

  const ok   = d.status === 'SUCCESS';
  const item = document.createElement('div');
  item.className = 'feed-card ' + (ok ? 'ok' : 'err');
  item.innerHTML = `
    <div class="fc-top">
      <span class="fc-type">${type}</span>
      <span class="fc-time">${hms(new Date())}</span>
    </div>
    <div class="fc-card">${groupDigits(card)}</div>
    <div class="fc-amt">₹${money(amt)} <span class="fc-rc">RC:${d.respCode}</span></div>`;

  list.prepend(item);

  const all = list.querySelectorAll('.feed-card');
  if (all.length > 50) all[all.length - 1].remove();
}

function groupDigits(n) {
  return String(n).replace(/(\d{4})(?=\d)/g, '$1 ');
}

function money(n) {
  return Number(n).toLocaleString('en-IN', { minimumFractionDigits: 2, maximumFractionDigits: 2 });
}

function hms(ts) {
  return new Date(ts).toLocaleTimeString('en-IN', { hour:'2-digit', minute:'2-digit', second:'2-digit', hour12: false });
}

function flash(id) {
  const el = document.getElementById(id);
  el.classList.add('shake');
  el.focus();
  setTimeout(() => el.classList.remove('shake'), 1200);
}
