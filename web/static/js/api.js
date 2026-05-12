/**
 * API Service Layer
 */

const API_BASE = '';

export async function post(path, payload) {
    const res = await fetch(API_BASE + path, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
    });
    const data = await res.json();
    if (!res.ok) throw new Error(data.error || 'Server error');
    return data;
}

export async function get(path) {
    const res = await fetch(API_BASE + path);
    const data = await res.json();
    if (!res.ok) throw new Error(data.error || 'Server error');
    return data;
}

export async function fetchJobs() {
    return get('/api/jobs');
}

export async function fetchJobDetail(id) {
    return get('/api/job/' + id);
}

export async function fetchFiles() {
    return get('/api/list-files');
}

export async function fetchSchemas(prefix, config) {
    return post('/api/list-schemas', config);
}

export async function testConnection(config) {
    return post('/api/test-conn', config);
}

export async function startBackup(payload) {
    return post('/api/backup', payload);
}

export async function startRestore(payload) {
    return post('/api/restore', payload);
}

export async function uploadFile(file) {
    const formData = new FormData();
    formData.append('file', file);
    const res = await fetch(API_BASE + '/api/upload', {
        method: 'POST',
        body: formData,
    });
    const data = await res.json();
    if (!res.ok) throw new Error(data.error || 'Upload failed');
    return data;
}

export function getDownloadUrl(path) {
    return API_BASE + '/api/download?file=' + encodeURIComponent(path);
}
