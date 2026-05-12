const API = '';

// ── Icons (SVG Paths) ─────────────────────────────────────────
const ICONS = {
    backup: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>`,
    restore: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="17 8 12 3 7 8"/><line x1="12" y1="3" x2="12" y2="15"/></svg>`,
    jobs: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="2" y="3" width="20" height="14" rx="2" ry="2"/><line x1="8" y1="21" x2="16" y2="21"/><line x1="12" y1="17" x2="12" y2="21"/></svg>`,
    database: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><ellipse cx="12" cy="5" rx="9" ry="3"/><path d="M3 5V19A9 3 0 0 0 21 19V5"/><path d="M3 12A9 3 0 0 0 21 12"/></svg>`,
    check: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="20 6 9 17 4 12"/></svg>`,
    alert: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/></svg>`
};

// ── Navigation ─────────────────────────────────────────────
function showPage(name) {
    document.querySelectorAll('.page').forEach(p => p.classList.remove('active'));
    document.querySelectorAll('.nav-item').forEach(n => n.classList.remove('active'));
    
    document.getElementById('page-' + name).classList.add('active');
    const navItem = document.querySelector(`.nav-item[data-page="${name}"]`);
    if (navItem) navItem.classList.add('active');
    
    if (name === 'jobs') loadJobs();
    
    // Save to history/URL if needed
    window.location.hash = name;
}

// ── Connection test ─────────────────────────────────────────
async function testConn(prefix) {
    const el = document.getElementById(prefix + '-conn-status');
    el.style.display = 'flex';
    el.className = 'conn-status checking';
    el.innerHTML = '<div class="dot blink"></div> Menghubungkan...';

    const payload = {
        host: val(prefix + '-host'),
        port: val(prefix + '-port'),
        user: val(prefix + '-user'),
        password: val(prefix + '-password'),
        db: val(prefix + '-db'),
    };

    try {
        const res = await fetch(API + '/api/test-conn', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(payload),
        });
        const data = await res.json();
        if (!res.ok) throw new Error(data.error);

        el.className = 'conn-status ok';
        el.innerHTML = `<div class="dot"></div> Connected: ${data.version}`;

        if (data.has_timescale && prefix === 'b') {
            const tsInfo = document.getElementById('b-ts-info');
            if (tsInfo) tsInfo.style.display = 'block';
            const tsCheck = document.getElementById('b-timescale');
            if (tsCheck) tsCheck.checked = true;
        }

        loadSchemas(prefix);
    } catch (e) {
        el.className = 'conn-status err';
        el.innerHTML = `<div class="dot"></div> Gagal: ${e.message}`;
    }
}

// ── Load schemas ────────────────────────────────────────────
async function loadSchemas(prefix) {
    const payload = {
        host: val(prefix + '-host'),
        port: val(prefix + '-port'),
        user: val(prefix + '-user'),
        password: val(prefix + '-password'),
        db: val(prefix + '-db'),
    };

    try {
        const res = await fetch(API + '/api/list-schemas', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(payload),
        });
        const data = await res.json();
        if (!res.ok) throw new Error(data.error);

        const hypertableSchemas = new Set(
            (data.hypertables || []).map(h => h.split('.')[0])
        );

        const select = document.getElementById(prefix + '-schema');
        if (select) {
            select.innerHTML = '<option value="">★ Full Backup (Semua Schema)</option>';
            (data.schemas || []).forEach(s => {
                const isHyper = hypertableSchemas.has(s);
                const opt = document.createElement('option');
                opt.value = s;
                opt.textContent = (isHyper ? '⚡ ' : '') + s + (isHyper ? ' (TimescaleDB)' : '');
                select.appendChild(opt);
            });
        }

        const container = document.getElementById(prefix + '-schema-chips');
        if (container) {
            container.innerHTML = '';
            (data.schemas || []).forEach(s => {
                const chip = document.createElement('div');
                const isHyper = hypertableSchemas.has(s);
                chip.className = 'schema-chip' + (isHyper ? ' hypertable' : '');
                chip.innerHTML = isHyper ? `⚡ ${s}` : s;
                chip.onclick = () => {
                    if (select) select.value = s;
                    container.querySelectorAll('.schema-chip').forEach(c => c.classList.remove('selected'));
                    chip.classList.add('selected');
                };
                container.appendChild(chip);
            });
        }
    } catch (e) {
        console.error('Gagal load schema:', e);
    }
}

// ── Run backup ──────────────────────────────────────────────
async function runBackup() {
    const btn = document.getElementById('b-run-btn');
    btn.disabled = true;
    const originalText = btn.innerHTML;
    btn.innerHTML = '⏳ Memproses...';

    clearLog();
    appendLog('info', 'Mengirim permintaan backup...');

    const payload = {
        host: val('b-host'),
        port: val('b-port'),
        user: val('b-user'),
        password: val('b-password'),
        db: val('b-db'),
        schema: val('b-schema'),
        output_file: val('b-output'),
        format: val('b-format'),
        jobs: parseInt(val('b-jobs')) || 4,
        compress: parseInt(val('b-compress')) || 6,
        timescale: document.getElementById('b-timescale').checked,
    };

    try {
        const res = await fetch(API + '/api/backup', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(payload),
        });
        const data = await res.json();
        if (!res.ok) throw new Error(data.error);

        appendLog('success', 'Job dibuat: ' + data.job_id);
        watchJob(data.job_id, () => {
            btn.disabled = false;
            btn.innerHTML = originalText;
            updateJobCount();
        });
    } catch (e) {
        appendLog('error', 'Error: ' + e.message);
        btn.disabled = false;
        btn.innerHTML = originalText;
    }
}

// ── Run restore ─────────────────────────────────────────────
async function runRestore() {
    const btn = document.getElementById('r-run-btn');
    btn.disabled = true;
    const originalText = btn.innerHTML;
    btn.innerHTML = '⏳ Memproses...';

    clearLog();
    appendLog('info', 'Mengirim permintaan restore...');

    const payload = {
        host: val('r-host'),
        port: val('r-port'),
        user: val('r-user'),
        password: val('r-password'),
        db: val('r-db'),
        schema: val('r-schema'),
        file: val('r-file'),
        jobs: parseInt(val('r-jobs')) || 4,
        timescale: document.getElementById('r-timescale').checked,
        create_db: document.getElementById('r-create-db').checked,
        src_version: val('r-src-version'),
    };

    try {
        const res = await fetch(API + '/api/restore', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(payload),
        });
        const data = await res.json();
        if (!res.ok) throw new Error(data.error);

        appendLog('success', 'Job dibuat: ' + data.job_id);
        watchJob(data.job_id, () => {
            btn.disabled = false;
            btn.innerHTML = originalText;
            updateJobCount();
        });
    } catch (e) {
        appendLog('error', 'Error: ' + e.message);
        btn.disabled = false;
        btn.innerHTML = originalText;
    }
}

// ── Watch job via polling ───────────────────────────────────
let pollTimer = null;
function watchJob(jobId, onDone) {
    updateJobStatus('running');
    if (pollTimer) clearInterval(pollTimer);

    let lastLogCount = 0;

    pollTimer = setInterval(async () => {
        try {
            const res = await fetch(API + '/api/job/' + jobId);
            const job = await res.json();

            if (job.logs && job.logs.length > lastLogCount) {
                for (let i = lastLogCount; i < job.logs.length; i++) {
                    const l = job.logs[i];
                    appendLog(l.level, l.message, l.time);
                }
                lastLogCount = job.logs.length;
            }

            if (job.status !== 'running') {
                clearInterval(pollTimer);
                updateJobStatus(job.status);
                if (onDone) onDone();
            }
        } catch (e) {
            // silent network error
        }
    }, 400);
}

// ── Log panel ───────────────────────────────────────────────
function clearLog() {
    document.getElementById('log-body').innerHTML = '';
    document.getElementById('log-job-status').style.display = 'none';
}

function appendLog(level, message, time) {
    const body = document.getElementById('log-body');
    const t = time ? new Date(time) : new Date();
    const timeStr = t.toLocaleTimeString('id-ID', { hour12: false });

    const line = document.createElement('div');
    line.className = 'log-line ' + (level || 'info');
    line.innerHTML = `
        <span class="log-time">${timeStr}</span>
        <span class="log-prefix">${level.toUpperCase()}</span>
        <span class="log-msg">${escHtml(message)}</span>
    `;
    body.appendChild(line);
    body.scrollTop = body.scrollHeight;
}

function updateJobStatus(status) {
    const el = document.getElementById('log-job-status');
    el.style.display = 'inline-block';
    el.className = 'job-status-tag status-' + status;
    const labels = { running: '⏳ Berjalan', done: '✅ Selesai', failed: '❌ Gagal' };
    el.textContent = labels[status] || status;
}

// ── Jobs page ───────────────────────────────────────────────
async function loadJobs() {
    try {
        const res = await fetch(API + '/api/jobs');
        const jobs = await res.json();

        const container = document.getElementById('jobs-list');
        if (!jobs || jobs.length === 0) {
            container.innerHTML = '<div style="padding:80px;text-align:center;color:var(--text-muted);font-size:14px">Belum ada jobs yang tercatat</div>';
            updateJobBadge(0);
            return;
        }

        updateJobBadge(jobs.length);

        container.innerHTML = jobs.map(j => `
            <div class="job-item" onclick="viewJobDetail('${j.id}')">
                <div class="job-type-icon ${j.type}">${ICONS[j.type] || ''}</div>
                <div class="job-main">
                    <div class="job-db">${escHtml(j.db)}${j.schema ? ' <span style="color:var(--text-muted);font-weight:400">/</span> ' + escHtml(j.schema) : ''}</div>
                    <div class="job-meta">
                        <span>ID: ${j.id.split('-')[0]}...</span>
                        <span>${new Date(j.started_at).toLocaleString('id-ID')}</span>
                    </div>
                </div>
                <div class="job-status-tag status-${j.status}">${statusLabel(j.status)}</div>
            </div>
        `).join('');
    } catch (e) {
        console.error(e);
    }
}

function statusLabel(s) {
    return { running: 'Berjalan', done: 'Selesai', failed: 'Gagal' }[s] || s;
}

async function viewJobDetail(jobId) {
    try {
        const res = await fetch(API + '/api/job/' + jobId);
        const job = await res.json();
        clearLog();
        (job.logs || []).forEach(l => appendLog(l.level, l.message, l.time));
        updateJobStatus(job.status);
        // Scroll ke log panel
        document.querySelector('.log-panel').scrollIntoView({ behavior: 'smooth' });
    } catch (e) {
        console.error(e);
    }
}

async function updateJobCount() {
    try {
        const res = await fetch(API + '/api/jobs');
        const jobs = await res.json();
        updateJobBadge((jobs || []).length);
    } catch (e) {}
}

function updateJobBadge(count) {
    const badges = document.querySelectorAll('.job-count-badge');
    badges.forEach(b => b.textContent = count);
}

// ── File Browser ───────────────────────────────────────────
async function loadBackupFiles() {
    const list = document.getElementById('file-browser-list');
    list.innerHTML = '<div style="padding:20px;text-align:center;color:var(--text-muted)">Memuat file...</div>';
    
    try {
        const res = await fetch(API + '/api/list-files');
        const files = await res.json();
        
        if (!files || files.length === 0) {
            list.innerHTML = '<div style="padding:20px;text-align:center;color:var(--text-muted)">Tidak ada file backup ditemukan</div>';
            return;
        }
        
        list.innerHTML = files.map(f => `
            <div class="file-item" onclick="selectFile('${f.path}')">
                <div class="file-icon">${ICONS.database}</div>
                <div class="file-info">
                    <div class="file-name">${escHtml(f.name)}</div>
                    <div class="file-meta">${formatBytes(f.size)} • ${new Date(f.time).toLocaleString('id-ID')}</div>
                </div>
            </div>
        `).join('');
    } catch (e) {
        list.innerHTML = `<div style="padding:20px;text-align:center;color:var(--error)">Gagal memuat file: ${e.message}</div>`;
    }
}

function toggleFileBrowser(show) {
    const modal = document.getElementById('file-browser-modal');
    modal.style.display = show ? 'flex' : 'none';
    if (show) loadBackupFiles();
}

function selectFile(path) {
    document.getElementById('r-file').value = path;
    toggleFileBrowser(false);
}

function formatBytes(bytes) {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
}

// ── Helpers ─────────────────────────────────────────────────
function val(id) {
    const el = document.getElementById(id);
    return el ? el.value.trim() : '';
}

function escHtml(s) {
    return String(s)
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;');
}

// Init
window.addEventListener('DOMContentLoaded', () => {
    // Check hash for page
    const hash = window.location.hash.substring(1) || 'backup';
    showPage(hash);
    updateJobCount();
    
    // Auto refresh job count
    setInterval(updateJobCount, 10000);
});
