/**
 * UI Manipulation Layer
 */
import { escHtml, ICONS, formatBytes, statusLabel } from './utils.js';
import { getDownloadUrl } from './api.js';

export function showPage(name) {
    document.querySelectorAll('.page').forEach(p => p.classList.remove('active'));
    document.querySelectorAll('.nav-item').forEach(n => n.classList.remove('active'));
    
    const target = document.getElementById('page-' + name);
    if (target) target.classList.add('active');
    
    const navItem = document.querySelector(`.nav-item[data-page="${name}"]`);
    if (navItem) navItem.classList.add('active');
    
    window.location.hash = name;
}

export function clearLog() {
    const body = document.getElementById('log-body');
    if (body) body.innerHTML = '';
    const status = document.getElementById('log-job-status');
    if (status) status.style.display = 'none';
}

export function appendLog(level, message, time) {
    const body = document.getElementById('log-body');
    if (!body) return;
    
    const t = time ? new Date(time) : new Date();
    const timeStr = t.toLocaleTimeString('id-ID', { hour12: false });

    const line = document.createElement('div');
    line.className = 'log-line ' + (level || 'info');
    line.innerHTML = `
        <span class="log-time">${timeStr}</span>
        <span class="log-prefix">${(level || 'info').toUpperCase()}</span>
        <span class="log-msg">${escHtml(message)}</span>
    `;
    body.appendChild(line);
    body.scrollTop = body.scrollHeight;
}

export function updateJobStatus(status) {
    const el = document.getElementById('log-job-status');
    if (!el) return;
    el.style.display = 'inline-block';
    el.className = 'job-status-tag status-' + status;
    const labels = { running: 'Running', done: 'Success', failed: 'Failed' };
    el.textContent = labels[status] || status;
}

export function updateJobBadge(count) {
    document.querySelectorAll('.job-count-badge').forEach(b => b.textContent = count);
}

export function renderJobs(jobs, onViewDetail) {
    const container = document.getElementById('jobs-list');
    if (!container) return;
    
    if (!jobs || jobs.length === 0) {
        container.innerHTML = '<div style="padding:80px;text-align:center;color:var(--text-muted);font-size:14px">No jobs recorded</div>';
        return;
    }

    container.innerHTML = jobs.map(j => `
        <div class="job-item" id="job-${j.id}">
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

    // Attach listeners
    jobs.forEach(j => {
        const el = document.getElementById(`job-${j.id}`);
        if (el) el.onclick = () => onViewDetail(j.id);
    });
}

export function renderFileList(files, onSelect) {
    const list = document.getElementById('file-browser-list');
    if (!list) return;
    
    if (!files || files.length === 0) {
        list.innerHTML = '<div style="padding:20px;text-align:center;color:var(--text-muted)">No backup files found</div>';
        return;
    }
    
    list.innerHTML = files.map(f => `
        <div class="file-item" data-path="${f.path}">
            <div class="file-icon">${ICONS.database}</div>
            <div class="file-info">
                <div class="file-name">${escHtml(f.name)}</div>
                <div class="file-meta">${formatBytes(f.size)} • ${new Date(f.time).toLocaleString('id-ID')}</div>
            </div>
            <a href="${getDownloadUrl(f.path)}" class="btn-download" title="Download" onclick="event.stopPropagation()">
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>
            </a>
        </div>
    `).join('');

    // Attach listeners
    list.querySelectorAll('.file-item').forEach(el => {
        el.onclick = () => onSelect(el.dataset.path);
    });
}

export function toggleFileBrowser(show) {
    const modal = document.getElementById('file-browser-modal');
    if (modal) modal.style.display = show ? 'flex' : 'none';
}
