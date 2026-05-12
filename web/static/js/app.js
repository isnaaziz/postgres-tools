/**
 * Main Application Entry Point
 */
import * as api from './api.js';
import * as ui from './ui.js';
import { val } from './utils.js';

let pollTimer = null;

// ── Lifecycle ──────────────────────────────────────────────
async function init() {
    // Nav handling
    const hash = window.location.hash.substring(1) || 'backup';
    ui.showPage(hash);
    
    // Initial data
    updateJobCount();
    
    // Global Listeners
    setupListeners();
    
    // Auto refresh job count
    setInterval(updateJobCount, 10000);
}

function setupListeners() {
    // Navigation
    document.querySelectorAll('.nav-item').forEach(item => {
        item.addEventListener('click', () => {
            const page = item.getAttribute('data-page');
            ui.showPage(page);
            if (page === 'jobs') loadJobs();
        });
    });

    // Forms
    window.testConn = async (prefix) => {
        const el = document.getElementById(prefix + '-conn-status');
        el.style.display = 'flex';
        el.className = 'conn-status checking';
        el.innerHTML = '<div class="dot blink"></div> Connecting...';

        const config = {
            host: val(prefix + '-host'),
            port: val(prefix + '-port'),
            user: val(prefix + '-user'),
            password: val(prefix + '-password'),
            db: val(prefix + '-db'),
        };

        try {
            const data = await api.testConnection(config);
            el.className = 'conn-status ok';
            el.innerHTML = `<div class="dot"></div> Connected: ${data.version}`;
            
            if (data.has_timescale && prefix === 'b') {
                document.getElementById('b-ts-info').style.display = 'block';
                document.getElementById('b-timescale').checked = true;
            }
            loadSchemas(prefix, config);
        } catch (e) {
            el.className = 'conn-status err';
            el.innerHTML = `<div class="dot"></div> Failed: ${e.message}`;
        }
    };

    window.runBackup = async () => {
        const btn = document.getElementById('b-run-btn');
        btn.disabled = true;
        const originalText = btn.innerHTML;
        btn.innerHTML = '⏳ Processing...';

        ui.clearLog();
        ui.appendLog('info', 'Requesting backup...');

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
            const data = await api.startBackup(payload);
            ui.appendLog('success', 'Job created: ' + data.job_id);
            watchJob(data.job_id, () => {
                btn.disabled = false;
                btn.innerHTML = originalText;
                updateJobCount();
            });
        } catch (e) {
            ui.appendLog('error', 'Error: ' + e.message);
            btn.disabled = false;
            btn.innerHTML = originalText;
        }
    };

    window.runRestore = async () => {
        const btn = document.getElementById('r-run-btn');
        btn.disabled = true;
        const originalText = btn.innerHTML;
        btn.innerHTML = '⏳ Processing...';

        ui.clearLog();
        ui.appendLog('info', 'Requesting restore...');

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
            const data = await api.startRestore(payload);
            ui.appendLog('success', 'Job created: ' + data.job_id);
            watchJob(data.job_id, () => {
                btn.disabled = false;
                btn.innerHTML = originalText;
                updateJobCount();
            });
        } catch (e) {
            ui.appendLog('error', 'Error: ' + e.message);
            btn.disabled = false;
            btn.innerHTML = originalText;
        }
    };

    window.toggleFileBrowser = async (show) => {
        ui.toggleFileBrowser(show);
        if (show) {
            const files = await api.fetchFiles();
            ui.renderFileList(files, (path) => {
                document.getElementById('r-file').value = path;
                ui.toggleFileBrowser(false);
            });
        }
    };

    // File Upload Handling
    const uploadInput = document.getElementById('file-upload-input');
    if (uploadInput) {
        uploadInput.addEventListener('change', async (e) => {
            const file = e.target.files[0];
            if (!file) return;

            const btn = e.target.nextElementSibling;
            const originalText = btn.innerHTML;
            btn.innerHTML = '...';
            btn.disabled = true;

            try {
                await api.uploadFile(file);
                // Refresh list
                const files = await api.fetchFiles();
                ui.renderFileList(files, (path) => {
                    document.getElementById('r-file').value = path;
                    ui.toggleFileBrowser(false);
                });
                alert('File berhasil diimpor!');
            } catch (err) {
                alert('Gagal mengimpor file: ' + err.message);
            } finally {
                btn.innerHTML = originalText;
                btn.disabled = false;
                uploadInput.value = '';
            }
        });
    }
}

// ── Logic ──────────────────────────────────────────────────
async function loadSchemas(prefix, config) {
    try {
        const data = await api.fetchSchemas(prefix, config);
        const hypertableSchemas = new Set((data.hypertables || []).map(h => h.split('.')[0]));
        
        const select = document.getElementById(prefix + '-schema');
        if (select) {
            select.innerHTML = '<option value="">★ Full Backup (All Schemas)</option>';
            (data.schemas || []).forEach(s => {
                const isHyper = hypertableSchemas.has(s);
                const opt = document.createElement('option');
                opt.value = s;
                opt.textContent = s + (isHyper ? ' (TimescaleDB)' : '');
                select.appendChild(opt);
            });
        }
    } catch (e) { console.error('Failed to load schemas', e); }
}

async function loadJobs() {
    try {
        const jobs = await api.fetchJobs();
        ui.renderJobs(jobs, viewJobDetail);
        ui.updateJobBadge(jobs.length);
    } catch (e) { console.error(e); }
}

async function viewJobDetail(id) {
    try {
        const job = await api.fetchJobDetail(id);
        ui.clearLog();
        (job.logs || []).forEach(l => ui.appendLog(l.level, l.message, l.time));
        ui.updateJobStatus(job.status);
        document.querySelector('.log-panel').scrollIntoView({ behavior: 'smooth' });
    } catch (e) { console.error(e); }
}

async function updateJobCount() {
    try {
        const jobs = await api.fetchJobs();
        ui.updateJobBadge(jobs.length);
    } catch (e) {}
}

function watchJob(jobId, onDone) {
    ui.updateJobStatus('running');
    if (pollTimer) clearInterval(pollTimer);

    let lastLogCount = 0;
    pollTimer = setInterval(async () => {
        try {
            const job = await api.fetchJobDetail(jobId);
            if (job.logs && job.logs.length > lastLogCount) {
                for (let i = lastLogCount; i < job.logs.length; i++) {
                    const l = job.logs[i];
                    ui.appendLog(l.level, l.message, l.time);
                }
                lastLogCount = job.logs.length;
            }

            if (job.status !== 'running') {
                clearInterval(pollTimer);
                ui.updateJobStatus(job.status);
                if (onDone) onDone();
            }
        } catch (e) {}
    }, 400);
}

// Start
init();
