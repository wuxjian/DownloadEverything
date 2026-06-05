/* Download Everything - 前端交互逻辑 */

// ========== Toast 通知 ==========
function toast(message, type = 'info') {
    const container = document.getElementById('toast-container');
    if (!container) return;
    const el = document.createElement('div');
    el.className = `toast ${type}`;
    el.textContent = message;
    container.appendChild(el);
    setTimeout(() => { el.style.opacity = '0'; setTimeout(() => el.remove(), 300); }, 3000);
}

// ========== 通用工具 ==========
async function api(url, method = 'GET', body = null) {
    const opts = { method, headers: { 'Content-Type': 'application/json' } };
    if (body) opts.body = JSON.stringify(body);
    const resp = await fetch(url, opts);
    return resp.json();
}

function formatSize(bytes) {
    if (!bytes || bytes <= 0) return '';
    const units = ['B', 'KB', 'MB', 'GB'];
    let i = 0;
    let size = bytes;
    while (size >= 1024 && i < units.length - 1) { size /= 1024; i++; }
    return size.toFixed(1) + ' ' + units[i];
}

function formatSpeed(bytesPerSec) {
    if (!bytesPerSec || bytesPerSec <= 0) return '0 B/s';
    const units = ['B/s', 'KB/s', 'MB/s', 'GB/s'];
    let i = 0;
    let speed = bytesPerSec;
    while (speed >= 1024 && i < units.length - 1) { speed /= 1024; i++; }
    return speed.toFixed(1) + ' ' + units[i];
}

function esc(s) {
    if (!s) return '';
    const d = document.createElement('div');
    d.textContent = s;
    return d.innerHTML;
}

// ========== SVG Icons ==========
const icons = {
    file: '<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/></svg>',
    pause: '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="6" y="4" width="4" height="16"/><rect x="14" y="4" width="4" height="16"/></svg>',
    play: '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polygon points="5 3 19 12 5 21 5 3"/></svg>',
    trash: '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></svg>',
    download: '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>',
};

// ========== Init ==========
document.addEventListener('DOMContentLoaded', () => {
    const btn = (id) => document.getElementById(id);
    btn('btn-add-task')?.addEventListener('click', addTask);
    btn('btn-search')?.addEventListener('click', startSearch);
    btn('btn-parse')?.addEventListener('click', parseURL);
    btn('btn-save-settings')?.addEventListener('click', saveSettings);
    if (document.getElementById('task-list')) {
        listenProgress();
        loadTasks();
    }
    // 恢复搜索页状态
    if (document.getElementById('progress-steps')) restoreSearchState();
});

// ========== 打开下载目录 ==========
async function openDownloadDir() {
    try {
        const data = await api('/api/open-dir', 'POST');
        if (data.error) toast(data.error, 'error');
        else toast('已打开下载目录', 'success');
    } catch (e) {
        toast('打开失败: ' + e.message, 'error');
    }
}

// ========== 下载管理 ==========

function toggleAdvanced() {
    const el = document.getElementById('advanced-options');
    const btn = document.getElementById('btn-toggle-adv');
    if (el.style.display === 'none') {
        el.style.display = 'block';
        btn.classList.add('open');
    } else {
        el.style.display = 'none';
        btn.classList.remove('open');
    }
}

function parseHeaders(text) {
    if (!text || !text.trim()) return null;
    const headers = {};
    text.trim().split('\n').forEach(line => {
        const idx = line.indexOf(':');
        if (idx > 0) {
            const key = line.substring(0, idx).trim();
            const val = line.substring(idx + 1).trim();
            if (key && val) headers[key] = val;
        }
    });
    return Object.keys(headers).length > 0 ? headers : null;
}

function parseCookies(text) {
    if (!text || !text.trim()) return null;
    const cookies = [];
    text.trim().split('\n').forEach(line => {
        const idx = line.indexOf('=');
        if (idx > 0) {
            const name = line.substring(0, idx).trim();
            const value = line.substring(idx + 1).trim();
            if (name) cookies.push({ Name: name, Value: value });
        }
    });
    return cookies.length > 0 ? cookies : null;
}
async function addTask() {
    const url = document.getElementById('input-url').value.trim();
    if (!url) return toast('请输入下载链接', 'error');
    const name = document.getElementById('input-name').value.trim();
    const headers = parseHeaders(document.getElementById('input-headers')?.value);
    const cookies = parseCookies(document.getElementById('input-cookies')?.value);

    const btn = document.getElementById('btn-add-task');
    btn.disabled = true;
    btn.innerHTML = '<span class="spinner"></span>';

    try {
        const result = await api('/api/tasks', 'POST', { url, name, headers, cookies });
        if (result.error) {
            toast(result.error, 'error');
        } else {
            document.getElementById('input-url').value = '';
            document.getElementById('input-name').value = '';
            if (document.getElementById('input-headers')) document.getElementById('input-headers').value = '';
            if (document.getElementById('input-cookies')) document.getElementById('input-cookies').value = '';
            toast('任务已添加', 'success');
            loadTasks();
        }
    } catch (e) {
        toast('添加失败: ' + e.message, 'error');
    } finally {
        btn.disabled = false;
        btn.innerHTML = icons.download + ' 添加下载';
    }
}

function renderTask(t) {
    let actions = '';
    if (t.status === 'downloading') {
        actions += `<button class="btn btn-ghost btn-icon" onclick="pauseTask('${t.id}')" title="暂停">${icons.pause}</button>`;
    } else if (t.status === 'paused') {
        actions += `<button class="btn btn-ghost btn-icon" onclick="resumeTask('${t.id}')" title="恢复">${icons.play}</button>`;
    } else if (t.status === 'failed') {
        actions += `<button class="btn btn-ghost btn-icon" onclick="retryTask('${t.id}')" title="重试">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="23 4 23 10 17 10"/><path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10"/></svg>
        </button>`;
    }
    actions += `<button class="btn btn-danger btn-icon" onclick="deleteTask('${t.id}')" title="删除">${icons.trash}</button>`;

    const speedText = t.speed ? formatSpeed(t.speed) : '';
    const sizeText = t.file_size ? formatSize(t.file_size) : '';

    const progress = Math.min(t.progress, 100);

    return `<div class="task-item" data-id="${t.id}">
        <div class="task-icon ${t.status}">${icons.file}</div>
        <div class="task-info">
            <div class="task-name">${esc(t.name)}</div>
            <div class="task-url">${esc(t.url)}</div>
        </div>
        <div class="task-progress-wrap">
            <div class="progress-bar"><div class="progress-fill ${t.status}" style="width:${progress}%"></div></div>
            <div class="task-meta">
                <span class="status-badge ${t.status}"><span class="status-dot ${t.status}"></span>${t.status}</span>
                <span>${progress.toFixed(1)}%</span>
                <span>${t.speed ? formatSpeed(t.speed) : '0 B/s'}</span>
                ${sizeText ? '<span>' + sizeText + '</span>' : ''}
            </div>
        </div>
        <div class="task-actions">${actions}</div>
    </div>`;
}

async function loadTasks() {
    try {
        const data = await api('/api/tasks');
        const list = document.getElementById('task-list');
        if (!data.tasks || data.tasks.length === 0) {
            list.innerHTML = `<div class="empty-state">
                <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>
                <p>暂无下载任务，添加链接开始下载</p>
            </div>`;
        } else {
            list.innerHTML = data.tasks.map(renderTask).join('');
        }
        // 更新统计卡片
        const counts = { downloading: 0, done: 0, paused: 0, failed: 0 };
        if (data.tasks) {
            data.tasks.forEach(t => { if (counts[t.status] !== undefined) counts[t.status]++; });
        }
        const id = (el) => document.getElementById(el);
        id('count-downloading').textContent = counts.downloading;
        id('count-done').textContent = counts.done;
        id('count-paused').textContent = counts.paused;
        id('count-failed').textContent = counts.failed;
    } catch (e) {
        console.error('加载任务失败:', e);
    }
}

async function pauseTask(id) {
    await api(`/api/tasks/${id}/pause`, 'POST');
    toast('任务已暂停', 'info');
    loadTasks();
}

async function resumeTask(id) {
    await api(`/api/tasks/${id}/resume`, 'POST');
    toast('任务已恢复', 'success');
    loadTasks();
}

async function retryTask(id) {
    await api(`/api/tasks/${id}/retry`, 'POST');
    toast('任务已加入重试', 'success');
    loadTasks();
}

async function clearAllTasks() {
    if (!confirm('确定清空所有已完成/失败/暂停的历史任务？')) return;
    await api('/api/tasks', 'DELETE');
    toast('历史任务已清空', 'success');
    loadTasks();
}

async function deleteTask(id) {
    if (!confirm('确定删除此任务？')) return;
    await api(`/api/tasks/${id}`, 'DELETE');
    toast('任务已删除', 'info');
    loadTasks();
}

// SSE 进度监听
let taskStatusCache = {}; // 缓存任务状态，检测状态变化

function listenProgress() {
    const es = new EventSource('/api/tasks/events');
    es.onmessage = (e) => {
        try {
            const event = JSON.parse(e.data);
            const p = event.P;
            if (!p) return;

            const prevStatus = taskStatusCache[event.TaskID];
            taskStatusCache[event.TaskID] = p.status;

            // 状态变化时重新加载任务列表
            if (prevStatus && prevStatus !== p.status) {
                loadTasks();
                return;
            }

            const item = document.querySelector(`.task-item[data-id="${event.TaskID}"]`);
            if (item) {
                const fill = item.querySelector('.progress-fill');
                const meta = item.querySelector('.task-meta');
                const prog = Math.min(p.progress, 100);
                if (fill) fill.style.width = prog + '%';
                if (meta) {
                    const speed = p.speed ? formatSpeed(p.speed) : '0 B/s';
                    const size = p.total ? formatSize(p.total) : '';
                    meta.innerHTML = `<span class="status-badge ${p.status}"><span class="status-dot ${p.status}"></span>${p.status}</span>
                        <span>${prog.toFixed(1)}%</span>
                        <span>${speed}</span>
                        ${size ? '<span>' + size + '</span>' : ''}`;
                }
            }
        } catch (e) {}
    };
    es.onerror = () => setTimeout(listenProgress, 3000);
}

// ========== AI 搜索 ==========
let searchResults = [];

// 搜索状态持久化
function saveSearchState(stepsHtml, links, query) {
    try {
        localStorage.setItem('de_search', JSON.stringify({
            steps: stepsHtml,
            links: links,
            query: query,
            time: Date.now()
        }));
    } catch(e) {}
}

function restoreSearchState() {
    try {
        const raw = localStorage.getItem('de_search');
        if (!raw) return;
        const state = JSON.parse(raw);
        // 超过30分钟则不恢复
        if (Date.now() - state.time > 30 * 60 * 1000) {
            localStorage.removeItem('de_search');
            return;
        }
        if (state.query) {
            const input = document.getElementById('input-query');
            if (input) input.value = state.query;
        }
        if (state.steps) {
            const stepsEl = document.getElementById('progress-steps');
            const progressEl = document.getElementById('search-progress');
            if (stepsEl && progressEl) {
                stepsEl.innerHTML = state.steps;
                progressEl.style.display = 'block';
            }
        }
        if (state.links && state.links.length > 0) {
            searchResults = state.links;
            showDownloadLinks(state.links, false); // false = 不滚动不toast
        }
    } catch(e) {
        localStorage.removeItem('de_search');
    }
}

function clearSearchState() {
    localStorage.removeItem('de_search');
}

function clearSearchUI() {
    clearSearchState();
    searchResults = [];
    const progressEl = document.getElementById('search-progress');
    const resultsCard = document.getElementById('search-results-card');
    const stepsEl = document.getElementById('progress-steps');
    const input = document.getElementById('input-query');
    if (progressEl) progressEl.style.display = 'none';
    if (resultsCard) resultsCard.style.display = 'none';
    if (stepsEl) stepsEl.innerHTML = '';
    if (input) input.value = '';
    toast('搜索状态已清空', 'info');
}

async function startSearch() {
    const query = document.getElementById('input-query').value.trim();
    if (!query) return toast('请输入搜索关键词', 'error');

    const btn = document.getElementById('btn-search');
    btn.disabled = true;
    btn.innerHTML = '<span class="spinner"></span> 搜索中...';

    const progressEl = document.getElementById('search-progress');
    const resultsCard = document.getElementById('search-results-card');
    const stepsEl = document.getElementById('progress-steps');

    progressEl.style.display = 'block';
    resultsCard.style.display = 'none';
    stepsEl.innerHTML = '';
    clearSearchState();

    try {
        const resp = await fetch('/api/ai/search', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ query })
        });

        const reader = resp.body.getReader();
        const decoder = new TextDecoder();
        let buffer = '';

        while (true) {
            const { done, value } = await reader.read();
            if (done) break;
            buffer += decoder.decode(value, { stream: true });
            const lines = buffer.split('\n');
            buffer = lines.pop();
            for (const line of lines) {
                if (!line.startsWith('data: ')) continue;
                try {
                    const data = JSON.parse(line.substring(6));
                    updateStep(data, stepsEl);
                    if (data.done) {
                        if (data.error) {
                            addErrorStep(stepsEl, data.error);
                        } else if (data.data && data.data.links) {
                            showDownloadLinks(data.data.links);
                        }
                    }
                } catch (e) {}
            }
        }
    } catch (e) {
        addErrorStep(stepsEl, e.message);
    } finally {
        btn.disabled = false;
        btn.innerHTML = '<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg> 搜索';
        // 保存搜索状态到 localStorage
        saveSearchState(stepsEl.innerHTML, searchResults, query);
    }
}

function addErrorStep(container, msg) {
    container.innerHTML += `<div class="step-item">
        <span class="step-badge error">!</span>
        <div class="step-content"><div class="step-title" style="color:var(--danger)">错误: ${esc(msg)}</div></div>
    </div>`;
}

const stepNames = { 1: '搜索引擎搜索', 2: 'AI 筛选网页', 3: '抓取网页内容', 4: 'AI 提取链接' };
const stepIcons = {
    1: '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg>',
    2: '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polygon points="22 3 2 3 10 12.46 10 19 14 21 14 12.46 22 3"/></svg>',
    3: '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>',
    4: '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"/><path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"/></svg>',
};

function updateStep(data, container) {
    const existing = container.querySelector(`[data-step="${data.step}"]`);
    const hasDetail = data.data != null && data.step !== 4 && (Array.isArray(data.data) ? data.data.length > 0 : true);
    const status = data.done ? 'done' : 'active';
    const checkmark = data.done ? '✓' : data.step;

    // 有数据就渲染折叠内容，不依赖 done
    let detailHtml = '';
    if (hasDetail) {
        detailHtml = renderStepDetail(data.step, data.data);
    }

    const toggleBtn = hasDetail
        ? `<button class="step-toggle" onclick="toggleStepDetail(this)" aria-label="展开详情">
            <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="6 9 12 15 18 9"/></svg>
           </button>`
        : '';

    const html = `<div class="step-item ${hasDetail ? 'has-detail' : ''}" data-step="${data.step}">
        <span class="step-badge ${status}">${data.done ? checkmark : data.step}</span>
        <div class="step-content" style="flex:1">
            <div class="step-header">
                <span class="step-title">${stepIcons[data.step] || ''} ${stepNames[data.step] || '步骤 ' + data.step}</span>
                ${toggleBtn}
            </div>
            <div class="step-desc">${esc(data.message)}</div>
            ${detailHtml}
        </div>
    </div>`;

    if (existing) { existing.outerHTML = html; } else { container.innerHTML += html; }
}

function renderStepDetail(step, data) {
    switch (step) {
        case 1: return renderTavilyResults(data);
        case 2: return renderFilteredURLs(data);
        default: return '';
    }
}

function renderTavilyResults(results) {
    if (!Array.isArray(results) || results.length === 0) return '';
    const items = results.map(r => `
        <div class="search-result-item">
            <div class="sr-title">${esc(r.title)}</div>
            <div class="sr-url">${esc(r.url)}</div>
            ${r.content ? `<div class="sr-snippet">${esc(r.content.substring(0, 200))}${r.content.length > 200 ? '...' : ''}</div>` : ''}
        </div>
    `).join('');
    return `<div class="step-detail">${items}</div>`;
}

function renderFilteredURLs(urls) {
    if (!Array.isArray(urls) || urls.length === 0) return '';
    const items = urls.map(u => `
        <div class="filtered-url-item">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"/><path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"/></svg>
            <span>${esc(u)}</span>
        </div>
    `).join('');
    return `<div class="step-detail">${items}</div>`;
}

function toggleStepDetail(btn) {
    const stepItem = btn.closest('.step-item');
    const detail = stepItem.querySelector('.step-detail');
    if (!detail) return;
    const isOpen = detail.classList.toggle('open');
    btn.classList.toggle('open', isOpen);
}

function showDownloadLinks(links, showToast = true) {
    searchResults = links;
    const card = document.getElementById('search-results-card');
    const list = document.getElementById('results-list');
    card.style.display = 'block';

    if (links.length === 0) {
        list.innerHTML = '<div class="empty-state"><p>未找到下载链接</p></div>';
        return;
    }

    list.innerHTML = links.map((link, i) => `
        <div class="result-item">
            <div class="result-info">
                <div class="result-name">${esc(link.name)}</div>
                <div class="result-url">${esc(link.url)}</div>
            </div>
            <div class="result-meta">
                ${link.type ? `<span class="badge accent">${esc(link.type)}</span>` : ''}
                ${link.size ? `<span class="badge">${esc(link.size)}</span>` : ''}
                <button class="btn btn-primary btn-sm" onclick="downloadOne(${i})">${icons.download} 下载</button>
            </div>
        </div>
    `).join('');

    if (showToast) {
        toast(`找到 ${links.length} 个下载链接`, 'success');
        card.scrollIntoView({ behavior: 'smooth', block: 'start' });
    }
}

async function downloadOne(idx) {
    if (!searchResults[idx]) return;
    await api('/api/ai/download', 'POST', { links: [searchResults[idx]] });
    toast('已添加到下载队列', 'success');
}

async function downloadAll() {
    if (!searchResults.length) return;
    const data = await api('/api/ai/download', 'POST', { links: searchResults });
    toast(`已添加 ${data.count} 个下载任务`, 'success');
}

// URL 解析
async function parseURL() {
    const url = document.getElementById('input-parse-url').value.trim();
    if (!url) return toast('请输入URL', 'error');

    const btn = document.getElementById('btn-parse');
    btn.disabled = true;
    btn.innerHTML = '<span class="spinner"></span>';

    try {
        const data = await api('/api/ai/parse-url', 'POST', { url });
        const el = document.getElementById('parse-results');
        const list = document.getElementById('parse-results-list');
        el.style.display = 'block';

        if (data.error) {
            list.innerHTML = `<div class="empty-state"><p>解析失败: ${esc(data.error)}</p></div>`;
        } else if (!data.links || data.links.length === 0) {
            list.innerHTML = '<div class="empty-state"><p>未找到下载链接</p></div>';
        } else {
            list.innerHTML = data.links.map(link => `
                <div class="result-item">
                    <div class="result-info">
                        <div class="result-name">${esc(link.name)}</div>
                        <div class="result-url">${esc(link.url)}</div>
                    </div>
                    <div class="result-meta">
                        ${link.type ? `<span class="badge accent">${esc(link.type)}</span>` : ''}
                        ${link.size ? `<span class="badge">${esc(link.size)}</span>` : ''}
                    </div>
                </div>
            `).join('');
            toast(`找到 ${data.links.length} 个链接`, 'success');
        }
    } catch (e) {
        toast('解析失败: ' + e.message, 'error');
    } finally {
        btn.disabled = false;
        btn.innerHTML = '<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"/><path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"/></svg> 解析链接';
    }
}

// ========== 密码显隐切换 ==========
function togglePasswordVisibility(id) {
    const input = document.getElementById(id);
    const btn = input.parentElement.querySelector('.password-toggle');
    if (input.type === 'password') {
        input.type = 'text';
        btn.classList.add('visible');
    } else {
        input.type = 'password';
        btn.classList.remove('visible');
    }
}

// ========== 设置 ==========
async function saveSettings() {
    const settings = {
        port: parseInt(document.getElementById('set-port').value),
        down_dir: document.getElementById('set-down-dir').value,
        ai_endpoint: document.getElementById('set-ai-endpoint').value,
        ai_model: document.getElementById('set-ai-model').value,
        ai_key: document.getElementById('set-ai-key').value,
        tavily_key: document.getElementById('set-tavily-key').value,
        serper_key: document.getElementById('set-serper-key').value,
        max_concurrent: parseInt(document.getElementById('set-max-concurrent').value),
        threads_per_file: parseInt(document.getElementById('set-threads-per-file').value),
        proxy_url: document.getElementById('set-proxy-url').value.trim(),
        max_retries: parseInt(document.getElementById('set-max-retries').value),
        retry_interval: parseInt(document.getElementById('set-retry-interval').value),
    };

    const btn = document.getElementById('btn-save-settings');
    btn.disabled = true;
    btn.innerHTML = '<span class="spinner"></span> 保存中...';

    try {
        const data = await api('/api/settings', 'PUT', settings);
        if (data.error) {
            toast(data.error, 'error');
        } else {
            toast('设置已保存，部分设置需重启后生效', 'success');
        }
    } catch (e) {
        toast('保存失败: ' + e.message, 'error');
    } finally {
        btn.disabled = false;
        btn.innerHTML = '<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M19 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11l5 5v11a2 2 0 0 1-2 2z"/><polyline points="17 21 17 13 7 13 7 21"/><polyline points="7 3 7 8 15 8"/></svg> 保存设置';
    }
}
