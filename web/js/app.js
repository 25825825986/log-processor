// 全局状态
let currentPage = 1;
let currentLimit = 20;
let currentTotal = 0;
let currentFilter = {};
let currentTab = 'dashboard';

// 初始化
document.addEventListener('DOMContentLoaded', () => {
    initTabs();
    initConfigTabs();
    initUploadZone();
    loadDashboard();
    loadConfig();
});

// 标签页切换
function initTabs() {
    document.querySelectorAll('.nav-btn').forEach(btn => {
        btn.addEventListener('click', () => {
            document.querySelectorAll('.nav-btn').forEach(b => b.classList.remove('active'));
            document.querySelectorAll('.tab-content').forEach(t => t.classList.remove('active'));
            
            btn.classList.add('active');
            const tabId = btn.dataset.tab;
            document.getElementById(tabId).classList.add('active');
            currentTab = tabId;
            
            // 加载对应数据
            if (tabId === 'dashboard') {
                loadDashboard();
            } else if (tabId === 'query') {
                queryLogs();
            }
        });
    });
}

// 配置标签页切换
function initConfigTabs() {
    document.querySelectorAll('.config-tab').forEach(tab => {
        tab.addEventListener('click', () => {
            document.querySelectorAll('.config-tab').forEach(t => t.classList.remove('active'));
            document.querySelectorAll('.config-panel').forEach(p => p.classList.remove('active'));
            
            tab.classList.add('active');
            const configId = 'config-' + tab.dataset.config;
            document.getElementById(configId).classList.add('active');
        });
    });
}

// 初始化上传区域
function initUploadZone() {
    const zone = document.getElementById('upload-zone');
    const input = document.getElementById('file-input');
    
    zone.addEventListener('click', () => input.click());
    
    zone.addEventListener('dragover', (e) => {
        e.preventDefault();
        zone.classList.add('dragover');
    });
    
    zone.addEventListener('dragleave', () => {
        zone.classList.remove('dragover');
    });
    
    zone.addEventListener('drop', (e) => {
        e.preventDefault();
        zone.classList.remove('dragover');
        handleFiles(e.dataTransfer.files);
    });
    
    input.addEventListener('change', () => {
        handleFiles(input.files);
    });
}

// 处理文件上传
async function handleFiles(files) {
    const progressDiv = document.getElementById('upload-progress');
    const progressFill = document.getElementById('progress-fill');
    const progressText = document.getElementById('progress-text');
    const resultsDiv = document.getElementById('upload-results');
    
    progressDiv.style.display = 'block';
    resultsDiv.innerHTML = '';
    let hasSuccess = false;
    
    for (let i = 0; i < files.length; i++) {
        const file = files[i];
        const formData = new FormData();
        formData.append('file', file);
        
        progressText.textContent = `正在上传 ${file.name}...`;
        progressFill.style.width = `${(i / files.length) * 100}%`;
        
        try {
            const response = await fetch('/api/logs/import', {
                method: 'POST',
                body: formData
            });
            
            let result;
            const text = await response.text();
            try {
                result = JSON.parse(text);
            } catch (e) {
                result = { error: text || 'Unknown error' };
            }
            
            if (response.ok) {
                resultsDiv.innerHTML += `<div class="upload-success"><i class="fas fa-check-circle"></i> ${file.name}: 成功导入 ${result.lines} 条记录 (接受 ${result.accepted || result.lines} 条)</div>`;
                hasSuccess = true;
            } else {
                resultsDiv.innerHTML += `<div class="upload-error"><i class="fas fa-times-circle"></i> ${file.name}: ${result.error || '导入失败'}</div>`;
            }
        } catch (error) {
            resultsDiv.innerHTML += `<div class="upload-error"><i class="fas fa-times-circle"></i> ${file.name}: ${error.message}</div>`;
        }
    }
    
    progressFill.style.width = '100%';
    progressText.textContent = hasSuccess ? '上传完成' : '上传失败';
    
    // 如果导入成功，刷新数据
    if (hasSuccess && currentTab === 'dashboard') {
        setTimeout(() => loadDashboard(), 500);
    }
    
    setTimeout(() => {
        progressDiv.style.display = 'none';
    }, 3000);
}

// 加载仪表板数据
async function loadDashboard() {
    try {
        const response = await fetch('/api/statistics');
        const stats = await response.json();
        
        document.getElementById('total-logs').textContent = stats.total_count?.toLocaleString() || '-';
        document.getElementById('error-logs').textContent = stats.error_count?.toLocaleString() || '-';
        document.getElementById('avg-response').textContent = stats.avg_response_time ? 
            Math.round(stats.avg_response_time) + 'ms' : '-';
        
        // 渲染图表
        renderStatusChart(stats.status_code_dist);
        renderMethodChart(stats.method_dist);
        renderTrendChart(stats.time_series);
    } catch (error) {
        console.error('Failed to load dashboard:', error);
    }
}

// 渲染状态码图表
function renderStatusChart(data) {
    const container = document.getElementById('status-chart');
    if (!data || Object.keys(data).length === 0) {
        container.innerHTML = '<p class="no-data">暂无数据</p>';
        return;
    }
    
    let html = '<div class="chart-bars">';
    const max = Math.max(...Object.values(data));
    
    for (const [code, count] of Object.entries(data)) {
        const percentage = (count / max) * 100;
        const barClass = code >= 500 ? 'error' : code >= 400 ? 'warning' : 'success';
        html += `
            <div class="chart-bar-row">
                <span class="chart-label">${code}</span>
                <div class="chart-bar-wrapper">
                    <div class="chart-bar ${barClass}" style="width: ${percentage}%"></div>
                </div>
                <span class="chart-value">${count}</span>
            </div>
        `;
    }
    html += '</div>';
    container.innerHTML = html;
}

// 渲染方法图表
function renderMethodChart(data) {
    const container = document.getElementById('method-chart');
    if (!data || Object.keys(data).length === 0) {
        container.innerHTML = '<p class="no-data">暂无数据</p>';
        return;
    }
    
    let html = '<div class="chart-bars">';
    const max = Math.max(...Object.values(data));
    
    for (const [method, count] of Object.entries(data)) {
        const percentage = (count / max) * 100;
        html += `
            <div class="chart-bar-row">
                <span class="chart-label">${method}</span>
                <div class="chart-bar-wrapper">
                    <div class="chart-bar" style="width: ${percentage}%"></div>
                </div>
                <span class="chart-value">${count}</span>
            </div>
        `;
    }
    html += '</div>';
    container.innerHTML = html;
}

// 渲染趋势图
function renderTrendChart(data) {
    const container = document.getElementById('trend-chart');
    if (!data || data.length === 0) {
        container.innerHTML = '<p class="no-data">暂无数据</p>';
        return;
    }
    
    // 简化的趋势显示
    const max = Math.max(...data.map(d => d.count));
    let html = '<div class="trend-chart">';
    
    data.slice(-20).forEach(point => {
        const height = (point.count / max) * 100;
        html += `
            <div class="trend-bar" style="height: ${height}%;" title="${point.time}: ${point.count}"></div>
        `;
    });
    
    html += '</div>';
    container.innerHTML = html;
}

// 查询日志
async function queryLogs() {
    const startTime = document.getElementById('filter-start-time').value;
    const endTime = document.getElementById('filter-end-time').value;
    const methods = Array.from(document.getElementById('filter-method').selectedOptions).map(o => o.value);
    const statusCodes = document.getElementById('filter-status').value.split(',').filter(s => s);
    const keyword = document.getElementById('filter-keyword').value;
    
    const params = new URLSearchParams();
    if (startTime) params.append('start_time', new Date(startTime).toISOString());
    if (endTime) params.append('end_time', new Date(endTime).toISOString());
    methods.forEach(m => params.append('methods', m));
    statusCodes.forEach(s => params.append('status_codes', s.trim()));
    if (keyword) params.append('keyword', keyword);
    params.append('limit', currentLimit);
    params.append('offset', (currentPage - 1) * currentLimit);
    
    try {
        const response = await fetch(`/api/logs?${params}`);
        const result = await response.json();
        
        currentTotal = result.total || 0;
        renderLogsTable(result.data);
        document.getElementById('results-count').textContent = `共 ${currentTotal} 条记录`;
        updatePagination();
    } catch (error) {
        console.error('Failed to query logs:', error);
    }
}

// 渲染日志表格
function renderLogsTable(logs) {
    const tbody = document.querySelector('#logs-table tbody');
    tbody.innerHTML = '';
    
    if (!logs || logs.length === 0) {
        tbody.innerHTML = '<tr><td colspan="7" class="no-data">暂无数据</td></tr>';
        return;
    }
    
    logs.forEach(log => {
        const row = document.createElement('tr');
        row.innerHTML = `
            <td>${new Date(log.timestamp).toLocaleString()}</td>
            <td>${log.method || '-'}</td>
            <td class="path-cell" title="${log.path || '-'}">${truncate(log.path, 30)}</td>
            <td><span class="status-tag status-${log.status_code}">${log.status_code || '-'}</span></td>
            <td>${log.response_time ? log.response_time + 'ms' : '-'}</td>
            <td>${log.client_ip || '-'}</td>
            <td><button class="btn-view" onclick='viewLogDetail(${JSON.stringify(log)})'>查看</button></td>
        `;
        tbody.appendChild(row);
    });
}

// 截断字符串
function truncate(str, length) {
    if (!str) return '-';
    return str.length > length ? str.substring(0, length) + '...' : str;
}

// 查看日志详情
function viewLogDetail(log) {
    const modal = document.getElementById('log-modal');
    const detail = document.getElementById('log-detail');
    
    detail.textContent = JSON.stringify(log, null, 2);
    modal.classList.add('active');
}

// 关闭弹窗
function closeModal() {
    document.getElementById('log-modal').classList.remove('active');
}

// 重置筛选
function resetFilters() {
    document.getElementById('filter-start-time').value = '';
    document.getElementById('filter-end-time').value = '';
    document.getElementById('filter-method').selectedIndex = -1;
    document.getElementById('filter-status').value = '';
    document.getElementById('filter-keyword').value = '';
    currentPage = 1;
    queryLogs();
}

// 分页
function prevPage() {
    if (currentPage > 1) {
        currentPage--;
        queryLogs();
    }
}

function nextPage() {
    const maxPage = Math.ceil(currentTotal / currentLimit);
    if (currentPage < maxPage) {
        currentPage++;
        queryLogs();
    }
}

// 更新分页按钮状态
function updatePagination() {
    const maxPage = Math.max(1, Math.ceil(currentTotal / currentLimit));
    document.getElementById('page-info').textContent = `第 ${currentPage} / ${maxPage} 页`;
    
    // 禁用/启用按钮
    const prevBtn = document.querySelector('.pagination button:first-child');
    const nextBtn = document.querySelector('.pagination button:last-child');
    
    if (prevBtn) {
        prevBtn.disabled = currentPage <= 1;
        prevBtn.style.opacity = currentPage <= 1 ? '0.5' : '1';
    }
    if (nextBtn) {
        nextBtn.disabled = currentPage >= maxPage || currentTotal === 0;
        nextBtn.style.opacity = (currentPage >= maxPage || currentTotal === 0) ? '0.5' : '1';
    }
}

// 加载配置
async function loadConfig() {
    try {
        const response = await fetch('/api/config');
        const config = await response.json();
        
        // 填充表单
        document.getElementById('parser-format').value = config.parser?.format || 'nginx';
        document.getElementById('parser-delimiter').value = config.parser?.delimiter || ' ';
        document.getElementById('parser-time-format').value = config.parser?.time_format || '';
        document.getElementById('parser-mapping').value = JSON.stringify(config.parser?.field_mapping || {}, null, 2);
        
        document.getElementById('processor-workers').value = config.processor?.worker_count || 10;
        document.getElementById('processor-batch-size').value = config.processor?.batch_size || 100;
        document.getElementById('processor-timeout').value = config.processor?.batch_timeout || 1000;
        document.getElementById('processor-clean-rules').value = JSON.stringify(config.processor?.clean_rules || [], null, 2);
        document.getElementById('processor-filter-rules').value = JSON.stringify(config.processor?.filter_rules || [], null, 2);
        
        document.getElementById('receiver-tcp').checked = config.receiver?.tcp_enabled ?? true;
        document.getElementById('receiver-tcp-port').value = config.receiver?.tcp_port || 9000;
        document.getElementById('receiver-udp').checked = config.receiver?.udp_enabled ?? true;
        document.getElementById('receiver-udp-port').value = config.receiver?.udp_port || 9001;
        document.getElementById('receiver-http').checked = config.receiver?.http_enabled ?? true;
        document.getElementById('receiver-http-port').value = config.receiver?.http_port || 9002;
        document.getElementById('receiver-buffer').value = config.receiver?.buffer_size || 8192;
        
        document.getElementById('storage-type').value = config.storage?.type || 'sqlite';
        document.getElementById('storage-db-path').value = config.storage?.db_path || './data/logs.db';
        document.getElementById('storage-retention').value = config.storage?.retention_hours || 168;
    } catch (error) {
        console.error('Failed to load config:', error);
    }
}

// 保存配置
async function saveConfig() {
    const config = {
        server: {
            host: "0.0.0.0",
            port: 8080
        },
        parser: {
            format: document.getElementById('parser-format').value,
            delimiter: document.getElementById('parser-delimiter').value,
            time_format: document.getElementById('parser-time-format').value,
            field_mapping: JSON.parse(document.getElementById('parser-mapping').value || '{}'),
            parse_user_agent: false
        },
        processor: {
            worker_count: parseInt(document.getElementById('processor-workers').value),
            batch_size: parseInt(document.getElementById('processor-batch-size').value),
            batch_timeout: parseInt(document.getElementById('processor-timeout').value),
            clean_rules: JSON.parse(document.getElementById('processor-clean-rules').value || '[]'),
            filter_rules: JSON.parse(document.getElementById('processor-filter-rules').value || '[]')
        },
        receiver: {
            tcp_enabled: document.getElementById('receiver-tcp').checked,
            tcp_port: parseInt(document.getElementById('receiver-tcp-port').value),
            udp_enabled: document.getElementById('receiver-udp').checked,
            udp_port: parseInt(document.getElementById('receiver-udp-port').value),
            http_enabled: document.getElementById('receiver-http').checked,
            http_port: parseInt(document.getElementById('receiver-http-port').value),
            buffer_size: parseInt(document.getElementById('receiver-buffer').value),
            file_watcher_enabled: false,
            watch_paths: [],
            max_connections: 1000
        },
        storage: {
            type: document.getElementById('storage-type').value,
            db_path: document.getElementById('storage-db-path').value,
            retention_hours: parseInt(document.getElementById('storage-retention').value)
        }
    };
    
    try {
        const response = await fetch('/api/config', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(config)
        });
        
        if (response.ok) {
            alert('配置保存成功！');
        } else {
            const result = await response.json();
            alert('保存失败: ' + result.error);
        }
    } catch (error) {
        alert('保存失败: ' + error.message);
    }
}

// 导出日志
async function exportLogs() {
    const format = document.getElementById('export-format').value;
    const filename = document.getElementById('export-filename').value || 'logs_export';
    const startTime = document.getElementById('export-start-time').value;
    const endTime = document.getElementById('export-end-time').value;
    const statusCodes = document.getElementById('export-status').value.split(',').filter(s => s).map(s => parseInt(s.trim()));
    
    const filter = {};
    if (startTime) filter.start_time = new Date(startTime).toISOString();
    if (endTime) filter.end_time = new Date(endTime).toISOString();
    if (statusCodes.length) filter.status_codes = statusCodes;
    
    const request = {
        format: format,
        file_name: filename,
        filter: filter
    };
    
    try {
        const response = await fetch('/api/export', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(request)
        });
        
        if (response.ok) {
            const blob = await response.blob();
            const url = window.URL.createObjectURL(blob);
            const a = document.createElement('a');
            a.href = url;
            a.download = filename + (format === 'excel' ? '.xlsx' : format === 'csv' ? '.csv' : '.json');
            document.body.appendChild(a);
            a.click();
            window.URL.revokeObjectURL(url);
            document.body.removeChild(a);
        } else {
            const result = await response.json();
            alert('导出失败: ' + result.error);
        }
    } catch (error) {
        alert('导出失败: ' + error.message);
    }
}

// 点击弹窗外部关闭
window.onclick = function(event) {
    const modal = document.getElementById('log-modal');
    if (event.target === modal) {
        closeModal();
    }
};

// 定时刷新仪表板
setInterval(() => {
    if (currentTab === 'dashboard') {
        loadDashboard();
    }
}, 30000);
