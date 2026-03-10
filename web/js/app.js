// 全局状态
let currentPage = 1;
let currentLimit = 20;
let currentTotal = 0;
let currentFilter = {};
let currentTab = 'dashboard';

// 设置保留时间
function setRetention(hours) {
    document.getElementById('storage-retention').value = hours;
    updateRetentionButtons(hours);
}

// 更新保留时间按钮状态
function updateRetentionButtons(currentHours) {
    document.querySelectorAll('.retention-btn').forEach(btn => {
        const btnHours = parseInt(btn.dataset.hours);
        if (btnHours === currentHours) {
            btn.classList.add('active');
        } else {
            btn.classList.remove('active');
        }
    });
}

// 监听保留时间输入框变化
document.addEventListener('DOMContentLoaded', () => {
    const retentionInput = document.getElementById('storage-retention');
    if (retentionInput) {
        retentionInput.addEventListener('change', function() {
            updateRetentionButtons(parseInt(this.value));
        });
    }
});

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

// 数字输入框验证配置
const NUMBER_INPUT_LIMITS = {
    'processor-workers': { min: 1, max: 100 },
    'processor-batch-size': { min: 10, max: 10000 },
    'processor-timeout': { min: 100, max: 60000 },
    'receiver-tcp-port': { min: 1, max: 65535 },
    'receiver-udp-port': { min: 1, max: 65535 },
    'receiver-http-port': { min: 1, max: 65535 },
    'receiver-http-rate': { min: 0, max: 100000 },
    'receiver-buffer': { min: 1024, max: 65536 },
    'storage-retention': { min: 1, max: 8760 } // 最多1年(8760小时)
};

// 初始化数字输入框验证
function initNumberValidation() {
    Object.keys(NUMBER_INPUT_LIMITS).forEach(id => {
        const input = document.getElementById(id);
        if (input) {
            const limits = NUMBER_INPUT_LIMITS[id];
            
            // 输入时验证
            input.addEventListener('input', function() {
                let value = parseInt(this.value);
                
                // 清除非数字字符
                if (isNaN(value)) {
                    this.value = limits.min;
                    return;
                }
                
                // 限制范围
                if (value < limits.min) {
                    this.value = limits.min;
                    showInputHint(this, `最小值为 ${limits.min}`);
                } else if (value > limits.max) {
                    this.value = limits.max;
                    showInputHint(this, `最大值为 ${limits.max}`);
                }
            });
            
            // 失去焦点时验证
            input.addEventListener('blur', function() {
                let value = parseInt(this.value);
                if (isNaN(value) || value < limits.min) {
                    this.value = limits.min;
                } else if (value > limits.max) {
                    this.value = limits.max;
                }
            });
        }
    });
}

// 显示输入提示
function showInputHint(input, message) {
    // 移除旧的提示
    const oldHint = input.parentElement.querySelector('.input-hint');
    if (oldHint) oldHint.remove();
    
    // 创建新提示
    const hint = document.createElement('span');
    hint.className = 'input-hint';
    hint.textContent = message;
    hint.style.cssText = 'color: #f5222d; font-size: 12px; margin-left: 8px;';
    
    input.parentElement.appendChild(hint);
    
    // 3秒后移除
    setTimeout(() => hint.remove(), 3000);
}

// 验证所有数字输入
function validateNumberInputs() {
    let isValid = true;
    Object.keys(NUMBER_INPUT_LIMITS).forEach(id => {
        const input = document.getElementById(id);
        if (input) {
            const limits = NUMBER_INPUT_LIMITS[id];
            let value = parseInt(input.value);
            
            if (isNaN(value) || value < limits.min || value > limits.max) {
                input.style.borderColor = '#f5222d';
                isValid = false;
            } else {
                input.style.borderColor = '';
            }
        }
    });
    return isValid;
}

// 初始化
document.addEventListener('DOMContentLoaded', () => {
    console.log('[App] Initializing...');
    
    initTabs();
    initConfigTabs();
    initUploadZone();
    initFormatListeners();
    initNumberValidation(); // 初始化数字验证
    
    // 延迟加载仪表板数据，确保DOM完全渲染
    setTimeout(() => {
        console.log('[App] Loading dashboard...');
        loadDashboard();
    }, 100);
    
    loadConfig();
    
    console.log('[App] Initialization complete');
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
            // 移除所有活动状态
            document.querySelectorAll('.config-tab').forEach(t => {
                t.classList.remove('active');
                t.style.background = '';
                t.style.color = 'var(--text-secondary)';
            });
            document.querySelectorAll('.config-panel').forEach(p => {
                p.style.display = 'none';
            });
            
            // 激活当前标签
            tab.classList.add('active');
            tab.style.background = 'var(--primary-bg)';
            tab.style.color = 'var(--primary)';
            
            const configId = 'config-' + tab.dataset.config;
            const panel = document.getElementById(configId);
            if (panel) {
                panel.style.display = 'block';
            }
        });
    });
    
    // 初始化第一个为活动状态
    const firstTab = document.querySelector('.config-tab.active');
    if (firstTab) {
        firstTab.style.background = 'var(--primary-bg)';
        firstTab.style.color = 'var(--primary)';
    }
}

// 初始化上传区域
function initUploadZone() {
    const zone = document.getElementById('upload-zone');
    const input = document.getElementById('file-input');
    
    if (!zone || !input) return;
    
    zone.addEventListener('click', () => input.click());
    
    zone.addEventListener('dragover', (e) => {
        e.preventDefault();
        zone.style.borderColor = 'var(--primary)';
        zone.style.background = 'var(--primary-bg)';
    });
    
    zone.addEventListener('dragleave', () => {
        zone.style.borderColor = '';
        zone.style.background = '';
    });
    
    zone.addEventListener('drop', (e) => {
        e.preventDefault();
        zone.style.borderColor = '';
        zone.style.background = '';
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
        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }
        
        const stats = await response.json();
        console.log('Dashboard data:', stats);
        
        // 更新统计卡片
        const totalCount = stats.total_count || 0;
        const errorCount = stats.error_count || 0;
        const avgResponse = stats.avg_response_time || 0;
        
        document.getElementById('total-logs').textContent = totalCount.toLocaleString();
        document.getElementById('error-logs').textContent = errorCount.toLocaleString();
        document.getElementById('avg-response').textContent = Math.round(avgResponse) + 'ms';
        
        // 计算错误率
        if (totalCount > 0) {
            const errorRate = ((errorCount / totalCount) * 100).toFixed(1);
            document.getElementById('error-rate').textContent = `错误率: ${errorRate}%`;
        } else {
            document.getElementById('error-rate').textContent = '';
        }
        
        // 更新时间戳
        document.getElementById('last-update').textContent = '刚刚更新';
        
        // 渲染图表
        renderStatusChart(stats.status_code_dist || {});
        renderMethodChart(stats.method_dist || {});
        renderTrendChart(stats.time_series || []);
        
    } catch (error) {
        console.error('Failed to load dashboard:', error);
        document.getElementById('system-status').textContent = '连接失败';
        document.getElementById('system-status').className = 'stat-value error';
        document.getElementById('last-update').textContent = '刷新失败';
        
        // 显示空状态
        renderEmptyChart('status-chart', '暂无数据');
        renderEmptyChart('method-chart', '暂无数据');
        renderEmptyChart('trend-chart', '暂无数据');
    }
}

// 渲染空图表状态
function renderEmptyChart(containerId, message) {
    const container = document.getElementById(containerId);
    if (container) {
        container.innerHTML = `
            <div class="empty-state">
                <i class="fas fa-chart-bar"></i>
                <p>${message}</p>
            </div>
        `;
    }
}

// 刷新仪表板
function refreshDashboard() {
    const btn = document.querySelector('.btn-icon .fa-sync-alt');
    if (btn) {
        btn.classList.add('fa-spin');
        setTimeout(() => btn.classList.remove('fa-spin'), 1000);
    }
    loadDashboard();
}

// 切换标签页
function switchTab(tabName) {
    const tabBtn = document.querySelector(`.nav-btn[data-tab="${tabName}"]`);
    if (tabBtn) {
        tabBtn.click();
    }
}

// 渲染状态码图表 - 卡片式设计
function renderStatusChart(data) {
    const container = document.getElementById('status-chart');
    const totalEl = document.getElementById('status-total');
    
    if (!data || Object.keys(data).length === 0) {
        container.innerHTML = `
            <div class="empty-state">
                <i class="fas fa-shield-alt"></i>
                <h4>暂无数据</h4>
                <p>导入或接收日志后将显示状态码分布</p>
            </div>
        `;
        if (totalEl) totalEl.textContent = '共 0 条';
        return;
    }
    
    // 计算总数和百分比
    const total = Object.values(data).reduce((a, b) => a + b, 0);
    if (totalEl) totalEl.textContent = `共 ${total.toLocaleString()} 条`;
    
    // 按状态码排序
    const sortedData = Object.entries(data).sort((a, b) => parseInt(a[0]) - parseInt(b[0]));
    
    // 状态码分类和标签
    const getStatusInfo = (code) => {
        const c = parseInt(code);
        if (c >= 200 && c < 300) return { class: 'success', label: '成功' };
        if (c >= 300 && c < 400) return { class: 'redirect', label: '重定向' };
        if (c >= 400 && c < 500) return { class: 'client-error', label: '客户端错误' };
        if (c >= 500) return { class: 'server-error', label: '服务器错误' };
        return { class: 'success', label: '其他' };
    };
    
    let html = '<div class="status-cards">';
    for (const [code, count] of sortedData) {
        const percent = total > 0 ? ((count / total) * 100).toFixed(1) : 0;
        const info = getStatusInfo(code);
        
        html += `
            <div class="status-card ${info.class}">
                <div class="status-code-badge">${code}</div>
                <div class="status-info">
                    <div class="status-label">${info.label}</div>
                    <div class="status-count">${count.toLocaleString()} 条</div>
                </div>
                <div class="status-percent">${percent}%</div>
            </div>
        `;
    }
    html += '</div>';
    container.innerHTML = html;
}

// 渲染方法图表 - 环形图设计
function renderMethodChart(data) {
    const container = document.getElementById('method-chart');
    const totalEl = document.getElementById('method-total');
    
    if (!data || Object.keys(data).length === 0) {
        container.innerHTML = `
            <div class="empty-state">
                <i class="fas fa-code-branch"></i>
                <h4>暂无数据</h4>
                <p>导入或接收日志后将显示请求方法分布</p>
            </div>
        `;
        if (totalEl) totalEl.textContent = '共 0 条';
        return;
    }
    
    // 计算总数
    const total = Object.values(data).reduce((a, b) => a + b, 0);
    if (totalEl) totalEl.textContent = `共 ${total.toLocaleString()} 条`;
    
    // 排序：按数量从大到小
    const sortedData = Object.entries(data).sort((a, b) => b[1] - a[1]);
    
    // 颜色方案
    const colors = [
        '#4472C4', '#52C41A', '#FAAD14', '#F5222D', 
        '#722ED1', '#13C2C2', '#EB2F96', '#FA541C'
    ];
    
    // 计算环形图的圆弧
    let currentAngle = 0;
    const arcs = sortedData.map(([method, count], index) => {
        const percentage = total > 0 ? count / total : 0;
        const angle = percentage * 360;
        const startAngle = currentAngle;
        const endAngle = currentAngle + angle;
        currentAngle += angle;
        
        // 计算SVG路径
        const startRad = (startAngle * Math.PI) / 180;
        const endRad = (endAngle * Math.PI) / 180;
        const x1 = 90 + 70 * Math.cos(startRad);
        const y1 = 90 + 70 * Math.sin(startRad);
        const x2 = 90 + 70 * Math.cos(endRad);
        const y2 = 90 + 70 * Math.sin(endRad);
        const largeArc = angle > 180 ? 1 : 0;
        
        return {
            method,
            count,
            percentage: (percentage * 100).toFixed(1),
            color: colors[index % colors.length],
            path: `M 90 90 L ${x1} ${y1} A 70 70 0 ${largeArc} 1 ${x2} ${y2} Z`,
            barWidth: Math.max(percentage * 100, 5)
        };
    });
    
    // 生成HTML
    let html = '<div class="method-donut">';
    
    // 环形图 SVG
    html += `
        <div class="donut-chart">
            <svg class="donut-svg" viewBox="0 0 180 180">
                ${arcs.map((arc, i) => `
                    <path d="${arc.path}" fill="${arc.color}" opacity="0.9">
                        <title>${arc.method}: ${arc.count} (${arc.percentage}%)</title>
                    </path>
                `).join('')}
                <circle cx="90" cy="90" r="45" fill="white"/>
            </svg>
            <div class="donut-center">
                <div class="donut-value">${total.toLocaleString()}</div>
                <div class="donut-label">总请求</div>
            </div>
        </div>
    `;
    
    // 图例
    html += '<div class="method-legend">';
    arcs.forEach(arc => {
        html += `
            <div class="method-item">
                <div class="method-color" style="background: ${arc.color}"></div>
                <span class="method-name">${arc.method}</span>
                <div class="method-bar-bg">
                    <div class="method-bar-fill" style="width: ${arc.barWidth}%; background: ${arc.color}"></div>
                </div>
                <span class="method-count">${arc.count.toLocaleString()}</span>
                <span class="method-percent">${arc.percentage}%</span>
            </div>
        `;
    });
    html += '</div></div>';
    
    container.innerHTML = html;
}

// 渲染趋势图
function renderTrendChart(data) {
    const container = document.getElementById('trend-chart');
    if (!data || data.length === 0) {
        container.innerHTML = `
            <div class="empty-state">
                <i class="fas fa-chart-line"></i>
                <h4>暂无数据</h4>
                <p>导入或接收日志后将显示时间趋势</p>
            </div>
        `;
        return;
    }
    
    const max = Math.max(...data.map(d => d.count || 0));
    const total = data.reduce((sum, d) => sum + (d.count || 0), 0);
    
    // 只显示最近30个点
    const displayData = data.slice(-30);
    
    let html = '<div style="padding: 16px 0;">';
    
    // 统计信息
    html += `
        <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 20px; padding: 12px 16px; background: var(--bg-secondary); border-radius: 8px;">
            <div style="font-size: 13px; color: var(--text-secondary);">
                <i class="fas fa-clock" style="margin-right: 6px;"></i>
                时间范围: ${formatTime(displayData[0]?.time)} ~ ${formatTime(displayData[displayData.length - 1]?.time)}
            </div>
            <div style="font-size: 13px; color: var(--text-secondary);">
                怼计: <strong style="color: var(--primary);">${total.toLocaleString()}</strong> 条
            </div>
        </div>
    `;
    
    // 柱状图
    html += '<div class="trend-chart" style="display: flex; align-items: flex-end; gap: 4px; height: 180px; padding: 10px 0; border-bottom: 1px solid var(--border-light);">';
    
    displayData.forEach((point, index) => {
        const count = point.count || 0;
        const height = max > 0 ? (count / max) * 100 : 0;
        const time = point.time || point.Time || '';
        const displayTime = formatTime(time);
        
        // 根据数量设置颜色
        let color = 'var(--primary)';
        if (count >= max * 0.8) color = '#52c41a'; // 高峰 - 绿
        else if (count >= max * 0.5) color = '#1890ff'; // 中等 - 蓝
        else if (count >= max * 0.2) color = '#faad14'; // 较低 - 黄
        else color = '#d9d9d9'; // 很低 - 灰
        
        html += `
            <div style="flex: 1; display: flex; flex-direction: column; align-items: center; gap: 4px; min-width: 8px;">
                <div style="width: 100%; height: ${Math.max(height, 3)}%; background: ${color}; border-radius: 3px 3px 0 0; transition: all 0.3s; cursor: pointer;" 
                     title="${displayTime}: ${count}条"
                     onmouseover="this.style.opacity='0.8'" 
                     onmouseout="this.style.opacity='1'"></div>
            </div>
        `;
    });
    
    html += '</div>';
    
    // 时间轴标签（显示开始、中间、结束）
    const midIndex = Math.floor(displayData.length / 2);
    html += `
        <div style="display: flex; justify-content: space-between; margin-top: 8px; padding: 0 4px; font-size: 11px; color: var(--text-tertiary);">
            <span>${formatTime(displayData[0]?.time)}</span>
            <span>${formatTime(displayData[midIndex]?.time)}</span>
            <span>${formatTime(displayData[displayData.length - 1]?.time)}</span>
        </div>
    `;
    
    // 图例
    html += `
        <div style="display: flex; justify-content: center; gap: 16px; margin-top: 16px; padding-top: 16px; border-top: 1px solid var(--border-light);">
            <div style="display: flex; align-items: center; gap: 6px; font-size: 12px; color: var(--text-secondary);">
                <div style="width: 12px; height: 12px; background: #52c41a; border-radius: 2px;"></div>
                <span>高峰 (≥80%)</span>
            </div>
            <div style="display: flex; align-items: center; gap: 6px; font-size: 12px; color: var(--text-secondary);">
                <div style="width: 12px; height: 12px; background: #1890ff; border-radius: 2px;"></div>
                <span>正常 (50-80%)</span>
            </div>
            <div style="display: flex; align-items: center; gap: 6px; font-size: 12px; color: var(--text-secondary);">
                <div style="width: 12px; height: 12px; background: #faad14; border-radius: 2px;"></div>
                <span>较低 (20-50%)</span>
            </div>
            <div style="display: flex; align-items: center; gap: 6px; font-size: 12px; color: var(--text-secondary);">
                <div style="width: 12px; height: 12px; background: #d9d9d9; border-radius: 2px;"></div>
                <span>低调 (<20%)</span>
            </div>
        </div>
    `;
    
    html += '</div>';
    container.innerHTML = html;
}

// 格式化时间显示
function formatTime(timeStr) {
    if (!timeStr) return '-';
    // 处理不同格式: 2026-03-10 13:45 或 2026-03-10T13:45:00+08:00
    const date = new Date(timeStr.replace(' ', 'T'));
    if (isNaN(date.getTime())) return timeStr;
    
    const hours = date.getHours().toString().padStart(2, '0');
    const minutes = date.getMinutes().toString().padStart(2, '0');
    return `${hours}:${minutes}`;
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
        
        // 更新存储路径显示
        const dbPath = config.storage?.db_path || './data/logs.db';
        document.getElementById('storage-db-path').value = dbPath;
        const pathText = document.getElementById('storage-path-text');
        if (pathText) {
            pathText.textContent = dbPath;
        }
        
        // 更新保留时间并同步按钮状态
        const retention = config.storage?.retention_hours || 168;
        document.getElementById('storage-retention').value = retention;
        updateRetentionButtons(retention);
    } catch (error) {
        console.error('Failed to load config:', error);
    }
}

// 保存配置
async function saveConfig() {
    // 先验证所有数字输入
    if (!validateNumberInputs()) {
        alert('请检查输入，有些数值超出了允许的范围');
        return;
    }
    
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
        console.log('[App] Auto-refreshing dashboard...');
        loadDashboard();
    }
}, 30000);

// 页面可见性变化时刷新
document.addEventListener('visibilitychange', () => {
    if (!document.hidden && currentTab === 'dashboard') {
        console.log('[App] Page visible, refreshing dashboard...');
        loadDashboard();
    }
});
