// 全局状态
let currentPage = 1;
let currentLimit = 20;
let currentTotal = 0;
let currentFilter = {};
let currentTab = 'dashboard';

// 时间格式预设配置 (人类可读的名称和示例)
const TIME_FORMAT_PRESETS = [
    {
        id: 'nginx',
        name: 'Nginx / Apache',
        format: '02/Jan/2006:15:04:05 -0700',
        example: '04/Mar/2024:10:30:00 +0800',
        desc: '常见 Web 服务器日志格式'
    },
    {
        id: 'iso',
        name: 'ISO 8601 标准',
        format: '2006-01-02T15:04:05Z07:00',
        example: '2024-03-04T10:30:00+08:00',
        desc: '国际通用标准格式'
    },
    {
        id: 'common',
        name: '标准日期时间',
        format: '2006-01-02 15:04:05',
        example: '2024-03-04 10:30:00',
        desc: '最常见的中文格式'
    },
    {
        id: 'syslog',
        name: 'Syslog',
        format: 'Jan 02 15:04:05',
        example: 'Mar 04 10:30:00',
        desc: '系统日志格式'
    },
    {
        id: 'slash',
        name: '斜杠分隔',
        format: '2006/01/02 15:04:05',
        example: '2024/03/04 10:30:00',
        desc: '使用 / 分隔日期'
    },
    {
        id: 'us',
        name: '美式日期',
        format: '01/02/2006 15:04:05',
        example: '03/04/2024 10:30:00',
        desc: '月/日/年 格式'
    }
];

// 日志格式对应的时间格式
const FORMAT_TIME_MAPPING = {
    'nginx': '02/Jan/2006:15:04:05 -0700',
    'apache': '02/Jan/2006:15:04:05 -0700',
    'json': '2006-01-02T15:04:05Z07:00',
    'csv': '2006-01-02 15:04:05',
    'custom': ''
};

// 初始化
document.addEventListener('DOMContentLoaded', () => {
    initTabs();
    initConfigTabs();
    initUploadZone();
    initFormatListeners();
    loadDashboard();
    loadConfig();
});

// 初始化格式监听器
function initFormatListeners() {
    // 日志格式变化时自动设置对应的时间格式
    const formatSelect = document.getElementById('parser-format');
    if (formatSelect) {
        formatSelect.addEventListener('change', onLogFormatChange);
    }
    
    // 初始化时间格式卡片
    initTimeFormatCards();
}

// 初始化时间格式卡片
function initTimeFormatCards() {
    const container = document.getElementById('time-format-cards');
    if (!container) return;
    
    container.innerHTML = TIME_FORMAT_PRESETS.map(preset => `
        <div class="time-format-card ${preset.id === 'nginx' ? 'selected' : ''}" 
             data-format="${preset.format}"
             onclick="selectTimeFormat('${preset.format}', this)">
            <div class="card-name">${preset.name}</div>
            <div class="card-example">${preset.example}</div>
        </div>
    `).join('');
}

// 选择时间格式
function selectTimeFormat(format, cardElement) {
    // 更新隐藏输入框
    const input = document.getElementById('parser-time-format');
    if (input) {
        input.value = format;
    }
    
    // 更新卡片样式
    document.querySelectorAll('.time-format-card').forEach(card => {
        card.classList.remove('selected');
    });
    if (cardElement) {
        cardElement.classList.add('selected');
    }
    
    // 更新预览
    updateTimeFormatPreview(format);
}

// 日志格式变化处理
function onLogFormatChange() {
    const format = document.getElementById('parser-format').value;
    const suggestedTimeFormat = FORMAT_TIME_MAPPING[format];
    
    if (suggestedTimeFormat) {
        // 查找对应的卡片
        const cards = document.querySelectorAll('.time-format-card');
        cards.forEach(card => {
            if (card.dataset.format === suggestedTimeFormat) {
                selectTimeFormat(suggestedTimeFormat, card);
            }
        });
    }
}

// 更新时间格式预览
function updateTimeFormatPreview(format) {
    const previewValue = document.getElementById('preview-value');
    if (!previewValue) return;
    
    // 查找预设的示例
    const preset = TIME_FORMAT_PRESETS.find(p => p.format === format);
    if (preset) {
        previewValue.textContent = preset.example;
    } else {
        // 动态生成示例
        const now = new Date();
        const example = format
            .replace('2006', now.getFullYear())
            .replace('01', String(now.getMonth() + 1).padStart(2, '0'))
            .replace('02', String(now.getDate()).padStart(2, '0'))
            .replace('15', String(now.getHours()).padStart(2, '0'))
            .replace('04', String(now.getMinutes()).padStart(2, '0'))
            .replace('05', String(now.getSeconds()).padStart(2, '0'))
            .replace('Jan', ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'][now.getMonth()])
            .replace('-0700', '+0800')
            .replace('Z07:00', '+08:00')
            .replace('.000', '.123');
        previewValue.textContent = example;
    }
}

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
                if (result.status === 'warning') {
                    // 格式不匹配警告
                    resultsDiv.innerHTML += `
                        <div class="upload-warning">
                            <i class="fas fa-exclamation-triangle"></i> 
                            <strong>${file.name}</strong>: ${result.warning}
                            <div style="margin-top: 8px; font-size: 12px; color: #666;">
                                检测到格式: ${result.detected_format || 'unknown'} | 当前配置: ${result.current_format || 'unknown'}
                            </div>
                        </div>`;
                } else if (result.lines > 0 && result.accepted === 0) {
                    resultsDiv.innerHTML += `<div class="upload-error"><i class="fas fa-times-circle"></i> ${file.name}: 导入失败，请检查配置格式是否匹配</div>`;
                } else {
                    resultsDiv.innerHTML += `<div class="upload-success"><i class="fas fa-check-circle"></i> ${file.name}: 成功导入 ${result.lines} 条记录 (接受 ${result.accepted || result.lines} 条)</div>`;
                    hasSuccess = true;
                }
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
            <td>
                <button class="btn-view" data-log='${JSON.stringify(log).replace(/'/g, "&#39;")}'><i class="fas fa-eye"></i></button>
                <button class="btn-delete" data-id="${log.id}"><i class="fas fa-trash"></i></button>
            </td>
        `;
        tbody.appendChild(row);
    });
    
    // 绑定事件监听器
    tbody.querySelectorAll('.btn-view').forEach(btn => {
        btn.addEventListener('click', () => {
            const log = JSON.parse(btn.dataset.log);
            viewLogDetail(log);
        });
    });
    
    tbody.querySelectorAll('.btn-delete').forEach(btn => {
        btn.addEventListener('click', () => {
            const id = btn.dataset.id;
            deleteLog(id);
        });
    });
}

// 删除单条日志
async function deleteLog(id) {
    if (!confirm('确定要删除这条日志吗？')) {
        return;
    }
    
    try {
        // 对 ID 进行 URL 编码，避免特殊字符问题
        const encodedId = encodeURIComponent(id);
        const response = await fetch(`/api/logs/${encodedId}`, {
            method: 'DELETE'
        });
        
        const text = await response.text();
        console.log('Delete response:', response.status, text);
        
        let result;
        try {
            result = JSON.parse(text);
        } catch (e) {
            result = { error: text || '解析响应失败' };
        }
        
        if (response.ok) {
            alert('删除成功');
            queryLogs(); // 刷新列表
            // 如果当前在概览页，也刷新统计数据
            if (currentTab === 'dashboard') {
                loadDashboard();
            }
        } else {
            alert('删除失败: ' + (result.error || `HTTP ${response.status}`));
        }
    } catch (error) {
        console.error('Delete error:', error);
        alert('删除失败: ' + error.message);
    }
}

// 导出日志
async function exportLogs() {
    const format = document.getElementById('export-format').value;
    const filename = document.getElementById('export-filename').value || 'logs_export';
    const startTime = document.getElementById('export-start-time').value;
    const endTime = document.getElementById('export-end-time').value;
    const statusCodesInput = document.getElementById('export-status').value;
    
    console.log('Export params:', { startTime, endTime, statusCodesInput });
    
    const filter = {};
    if (startTime) {
        filter.start_time = new Date(startTime).toISOString();
    }
    if (endTime) {
        filter.end_time = new Date(endTime).toISOString();
    }
    if (statusCodesInput) {
        const codes = statusCodesInput.split(',').map(s => parseInt(s.trim())).filter(n => !isNaN(n));
        if (codes.length > 0) {
            filter.status_codes = codes;
        }
    }
    
    console.log('Export filter:', filter);
    
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
            const contentType = response.headers.get('content-type');
            // 如果返回的是 JSON，说明是错误信息
            if (contentType && contentType.includes('application/json')) {
                const result = await response.json();
                if (result.error) {
                    alert('导出失败: ' + result.error);
                    return;
                }
            }
            
            // 获取 blob 并检查大小
            const blob = await response.blob();
            console.log('Export blob size:', blob.size, 'type:', blob.type);
            
            if (blob.size === 0) {
                alert('导出数据为空，请检查筛选条件');
                return;
            }
            
            const url = window.URL.createObjectURL(blob);
            const a = document.createElement('a');
            a.href = url;
            a.download = filename + (format === 'excel' ? '.xlsx' : format === 'csv' ? '.csv' : '.json');
            document.body.appendChild(a);
            a.click();
            window.URL.revokeObjectURL(url);
            document.body.removeChild(a);
            alert('导出成功，共导出数据到文件');
        } else {
            const text = await response.text();
            let result;
            try {
                result = JSON.parse(text);
            } catch (e) {
                result = { error: text };
            }
            alert('导出失败: ' + (result.error || '未知错误'));
        }
    } catch (error) {
        alert('导出失败: ' + error.message);
    }
}

// 清空所有日志
async function clearAllLogs() {
    const count = document.getElementById('results-count').textContent;
    if (!confirm(`确定要清空所有日志吗？\n\n${count}\n\n此操作不可恢复！`)) {
        return;
    }
    
    try {
        const response = await fetch('/api/logs', {
            method: 'DELETE'
        });
        
        const text = await response.text();
        let result;
        try {
            result = JSON.parse(text);
        } catch (e) {
            result = { error: text || 'Unknown error' };
        }
        
        if (response.ok) {
            alert('已清空所有日志');
            queryLogs(); // 刷新列表
            // 刷新概览数据
            if (currentTab === 'dashboard') {
                loadDashboard();
            }
        } else {
            alert('清空失败: ' + (result.error || '未知错误'));
        }
    } catch (error) {
        alert('清空失败: ' + error.message);
    }
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
        
        // 处理时间格式
        const timeFormat = config.parser?.time_format || '02/Jan/2006:15:04:05 -0700';
        const timeInput = document.getElementById('parser-time-format');
        if (timeInput) {
            timeInput.value = timeFormat;
        }
        
        // 选中对应的卡片
        const cards = document.querySelectorAll('.time-format-card');
        let matched = false;
        cards.forEach(card => {
            card.classList.remove('selected');
            if (card.dataset.format === timeFormat) {
                card.classList.add('selected');
                matched = true;
            }
        });
        
        // 如果没有匹配的预设，默认选中第一个
        if (!matched && cards.length > 0) {
            cards[0].classList.add('selected');
        }
        
        // 更新预览
        updateTimeFormatPreview(timeFormat);
        
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
        document.getElementById('receiver-http-token').value = config.receiver?.http_auth_token || '';
        document.getElementById('receiver-http-ips').value = (config.receiver?.http_allowed_ips || []).join(',');
        document.getElementById('receiver-http-rate').value = config.receiver?.http_rate_limit || 0;
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
            http_auth_token: document.getElementById('receiver-http-token').value,
            http_allowed_ips: document.getElementById('receiver-http-ips').value.split(',').map(s => s.trim()).filter(s => s),
            http_rate_limit: parseInt(document.getElementById('receiver-http-rate').value) || 0,
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
