// 页面状态
let currentPage = 1;
let currentLimit = 20;
let currentTotal = 0;
let currentFilter = {};
let currentTab = 'dashboard';

// 设置存储保留时间
function setRetention(hours) {
    document.getElementById('storage-retention').value = hours;
    updateRetentionButtons(hours);
}

// 根据当前保留时间更新快捷按钮状态
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

// 初始化存储保留时间输入框事件
document.addEventListener('DOMContentLoaded', () => {
    const retentionInput = document.getElementById('storage-retention');
    if (retentionInput) {
        retentionInput.addEventListener('change', function() {
            updateRetentionButtons(parseInt(this.value));
        });
    }
});

// 时间格式预设（用于日志解析配置）
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

// 日志格式与建议时间格式映射
const FORMAT_TIME_MAPPING = {
    'nginx': '02/Jan/2006:15:04:05 -0700',
    'apache': '02/Jan/2006:15:04:05 -0700',
    'json': '2006-01-02T15:04:05Z07:00',
    'csv': '2006-01-02 15:04:05',
    'custom': ''
};

const TIME_PICKER_GROUPS = {
    filter: {
        startDate: 'filter-start-date',
        startClock: 'filter-start-clock',
        startHidden: 'filter-start-time',
        endDate: 'filter-end-date',
        endClock: 'filter-end-clock',
        endHidden: 'filter-end-time'
    },
    export: {
        startDate: 'export-start-date',
        startClock: 'export-start-clock',
        startHidden: 'export-start-time',
        endDate: 'export-end-date',
        endClock: 'export-end-clock',
        endHidden: 'export-end-time'
    }
};

function padTimeNumber(value) {
    return String(value).padStart(2, '0');
}

function formatDateInputValue(date) {
    return `${date.getFullYear()}-${padTimeNumber(date.getMonth() + 1)}-${padTimeNumber(date.getDate())}`;
}

function formatTimeInputValue(date) {
    return `${padTimeNumber(date.getHours())}:${padTimeNumber(date.getMinutes())}`;
}

function formatDateTimeLocalValue(date) {
    return `${formatDateInputValue(date)}T${formatTimeInputValue(date)}`;
}

function getTimePickerElements(target) {
    return TIME_PICKER_GROUPS[target] || null;
}

function syncTimePickerHidden(target, side) {
    const group = getTimePickerElements(target);
    if (!group) return;

    const dateInput = document.getElementById(group[`${side}Date`]);
    const clockInput = document.getElementById(group[`${side}Clock`]);
    const hiddenInput = document.getElementById(group[`${side}Hidden`]);
    if (!dateInput || !clockInput || !hiddenInput) return;

    if (!dateInput.value) {
        hiddenInput.value = '';
        return;
    }

    const fallbackTime = side === 'start' ? '00:00' : '23:59';
    hiddenInput.value = `${dateInput.value}T${clockInput.value || fallbackTime}`;
}

function setTimePickerField(target, side, date) {
    const group = getTimePickerElements(target);
    if (!group) return;

    const dateInput = document.getElementById(group[`${side}Date`]);
    const clockInput = document.getElementById(group[`${side}Clock`]);
    const hiddenInput = document.getElementById(group[`${side}Hidden`]);
    if (!dateInput || !clockInput || !hiddenInput) return;

    if (!date) {
        dateInput.value = '';
        clockInput.value = '';
        hiddenInput.value = '';
        return;
    }

    dateInput.value = formatDateInputValue(date);
    clockInput.value = formatTimeInputValue(date);
    hiddenInput.value = formatDateTimeLocalValue(date);
}

function setTimePickerRange(target, startDate, endDate) {
    setTimePickerField(target, 'start', startDate);
    setTimePickerField(target, 'end', endDate);
}

function updateTimePresetButtons(target, activeRange) {
    document.querySelectorAll(`.time-preset-btn[data-target="${target}"]`).forEach(btn => {
        btn.classList.toggle('active', btn.dataset.range === activeRange);
    });
}

function applyTimePreset(target, range) {
    const now = new Date();
    let startDate = null;
    let endDate = null;

    switch (range) {
        case '1h':
            endDate = new Date(now);
            startDate = new Date(now.getTime() - 60 * 60 * 1000);
            break;
        case '24h':
            endDate = new Date(now);
            startDate = new Date(now.getTime() - 24 * 60 * 60 * 1000);
            break;
        case '7d':
            endDate = new Date(now);
            startDate = new Date(now.getTime() - 7 * 24 * 60 * 60 * 1000);
            break;
        case 'today':
            startDate = new Date(now.getFullYear(), now.getMonth(), now.getDate(), 0, 0, 0);
            endDate = new Date(now);
            break;
        case 'clear':
            break;
        default:
            return;
    }

    setTimePickerRange(target, startDate, endDate);
    updateTimePresetButtons(target, range === 'clear' ? '' : range);

    if (target === 'export') {
        updateExportPreview();
    }
}

function initTimePickers() {
    Object.keys(TIME_PICKER_GROUPS).forEach(target => {
        const group = getTimePickerElements(target);
        ['start', 'end'].forEach(side => {
            const dateInput = document.getElementById(group[`${side}Date`]);
            const clockInput = document.getElementById(group[`${side}Clock`]);
            const hiddenInput = document.getElementById(group[`${side}Hidden`]);

            if (!dateInput || !clockInput || !hiddenInput) return;

            const sync = () => {
                syncTimePickerHidden(target, side);
                updateTimePresetButtons(target, '');
                if (target === 'export') {
                    updateExportPreview();
                }
            };

            dateInput.addEventListener('change', sync);
            clockInput.addEventListener('change', sync);
        });
    });

    document.querySelectorAll('.time-preset-btn').forEach(btn => {
        btn.addEventListener('click', () => {
            applyTimePreset(btn.dataset.target, btn.dataset.range);
        });
    });

    applyTimePreset('filter', '24h');
}

// 数字输入项的取值范围
const NUMBER_INPUT_LIMITS = {
    'processor-workers': { min: 1, max: 100 },
    'processor-batch-size': { min: 10, max: 10000 },
    'processor-timeout': { min: 10, max: 60000 },
    'processor-overflow-max-mb': { min: 16, max: 20480 },
    'processor-overflow-drain-batch': { min: 1, max: 20000 },
    'processor-overflow-drain-interval': { min: 10, max: 5000 },
    'receiver-tcp-port': { min: 1, max: 65535 },
    'receiver-udp-port': { min: 1, max: 65535 },
    'receiver-http-port': { min: 1, max: 65535 },
    'receiver-http-rate': { min: 0, max: 100000 },
    'receiver-buffer': { min: 1024, max: 65536 },
    'storage-retention': { min: 1, max: 8760 }, // 最长支持 8760 小时
    'benchmark-duration': { min: 3, max: 300 },
    'benchmark-workers': { min: 1, max: 200 },
    'benchmark-target-qps': { min: 0, max: 1000000 }
};

// 初始化数字输入校验
function initNumberValidation() {
    Object.keys(NUMBER_INPUT_LIMITS).forEach(id => {
        const input = document.getElementById(id);
        if (input) {
            const limits = NUMBER_INPUT_LIMITS[id];
            
            // 输入时即时限制范围
            input.addEventListener('input', function() {
                let value = parseInt(this.value);
                
                // 非数字时回退到最小值
                if (isNaN(value)) {
                    this.value = limits.min;
                    return;
                }
                
                // 超出范围时给出提示
                if (value < limits.min) {
                    this.value = limits.min;
                    showInputHint(this, `最小值为 ${limits.min}`);
                } else if (value > limits.max) {
                    this.value = limits.max;
                    showInputHint(this, `最大值为 ${limits.max}`);
                }
            });
            
            // 失焦时再次兜底
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
    // 先移除旧提示
    const oldHint = input.parentElement.querySelector('.input-hint');
    if (oldHint) oldHint.remove();
    
    // 创建新的提示节点
    const hint = document.createElement('span');
    hint.className = 'input-hint';
    hint.textContent = message;
    hint.style.cssText = 'color: #f5222d; font-size: 12px; margin-left: 8px;';
    
    input.parentElement.appendChild(hint);
    
    // 3 秒后自动消失
    setTimeout(() => hint.remove(), 3000);
}

// 保存前统一校验数字输入
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

// 页面初始化
document.addEventListener('DOMContentLoaded', () => {
    console.log('[App] Initializing...');
    
    initTabs();
    initConfigTabs();
    initUploadZone();
    initTimePickers();
    initFormatListeners();
    initNumberValidation(); // 初始化数字输入校验
    initExportPreview(); // 初始化导出预览
    
    // 等 DOM 和样式稳定后再加载概览，避免首屏抖动
    setTimeout(() => {
        console.log('[App] Loading dashboard...');
        loadDashboard();
    }, 100);
    
    loadConfig();
    
    console.log('[App] Initialization complete');
});

// 初始化导出预览
function initExportPreview() {
    // 监听时间范围变化
    const startTimeInput = document.getElementById('export-start-time');
    const endTimeInput = document.getElementById('export-end-time');
    
    if (startTimeInput) {
        startTimeInput.addEventListener('change', updateExportPreview);
    }
    if (endTimeInput) {
        endTimeInput.addEventListener('change', updateExportPreview);
    }
    
    // 首次进入时先刷新一次
    updateExportPreview();
}

// 初始化格式相关交互
function initFormatListeners() {
    // 监听日志格式变化，自动推荐匹配的时间格式
    const formatSelect = document.getElementById('parser-format');
    if (formatSelect) {
        formatSelect.addEventListener('change', onLogFormatChange);
    }
    
    // 初始化时间格式卡片
    initTimeFormatCards();
    
    // 初始化配置区按钮与映射编辑器
    initConfigInteractions();
}

// 初始化配置区交互
function initConfigInteractions() {
    // 日志格式卡片选择
    document.querySelectorAll('.format-card').forEach(card => {
        card.addEventListener('click', () => {
            document.querySelectorAll('.format-card').forEach(c => c.classList.remove('active'));
            card.classList.add('active');
            const formatInput = document.getElementById('parser-format');
            if (formatInput) formatInput.value = card.dataset.format;
        });
    });
    
    // 分隔符快捷按钮
    document.querySelectorAll('.delimiter-btn').forEach(btn => {
        btn.addEventListener('click', () => {
            document.querySelectorAll('.delimiter-btn').forEach(b => b.classList.remove('active'));
            btn.classList.add('active');
            const delimiterInput = document.getElementById('parser-delimiter');
            if (delimiterInput) delimiterInput.value = btn.dataset.value;
        });
    });
    
    // 缓冲区快捷按钮
    document.querySelectorAll('.buffer-btn').forEach(btn => {
        btn.addEventListener('click', () => {
            document.querySelectorAll('.buffer-btn').forEach(b => b.classList.remove('active'));
            btn.classList.add('active');
            const bufferInput = document.getElementById('receiver-buffer');
            if (bufferInput) bufferInput.value = btn.dataset.value;
        });
    });
    
    // 保留时间快捷按钮
    document.querySelectorAll('.retention-btn').forEach(btn => {
        btn.addEventListener('click', () => {
            setRetention(parseInt(btn.dataset.hours));
        });
    });
    
    // 如果字段映射为空，填入常用默认映射
    const mappingList = document.getElementById('mapping-list');
    if (mappingList && mappingList.children.length === 0) {
        addMappingRow('0', 'client_ip');
        addMappingRow('3', 'timestamp');
        addMappingRow('4', 'method');
        addMappingRow('5', 'path');
    }
}

// 初始化时间格式卡片
function initTimeFormatCards() {
    const container = document.getElementById('time-format-cards');
    if (!container) return;
    
    container.innerHTML = TIME_FORMAT_PRESETS.map(preset => `
        <div class="time-format-card ${preset.id === 'nginx' ? 'active' : ''}" 
             data-format="${preset.format}"
             onclick="selectTimeFormat('${preset.format}', this)">
            <div class="format-name">${preset.name}</div>
            <div class="format-example">${preset.example}</div>
        </div>
    `).join('');
}

// 选择时间格式
function selectTimeFormat(format, cardElement) {
    const input = document.getElementById('parser-time-format');
    if (input) input.value = format;
    
    document.querySelectorAll('.time-format-card').forEach(card => {
        card.classList.remove('active');
    });
    if (cardElement) cardElement.classList.add('active');
    
    // 同步更新时间格式预览
    updateTimeFormatPreview(format);
}

// 日志格式变化时自动匹配时间格式
function onLogFormatChange() {
    const format = document.getElementById('parser-format').value;
    const suggestedTimeFormat = FORMAT_TIME_MAPPING[format];
    
    if (suggestedTimeFormat) {
        // 找到匹配卡片后自动选中
        const cards = document.querySelectorAll('.time-format-card');
        cards.forEach(card => {
            if (card.dataset.format === suggestedTimeFormat) {
                selectTimeFormat(suggestedTimeFormat, card);
            }
        });
    }
}

// 更新时间格式示例预览
function updateTimeFormatPreview(format) {
    const previewValue = document.getElementById('preview-value');
    if (!previewValue) return;
    
    // 优先使用预设示例
    const preset = TIME_FORMAT_PRESETS.find(p => p.format === format);
    if (preset) {
        previewValue.textContent = preset.example;
    } else {
        // 否则按当前时间拼出一个示例
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

// 初始化顶部标签切换
function initTabs() {
    document.querySelectorAll('.nav-btn').forEach(btn => {
        btn.addEventListener('click', () => {
            document.querySelectorAll('.nav-btn').forEach(b => b.classList.remove('active'));
            document.querySelectorAll('.tab-content').forEach(t => t.classList.remove('active'));
            
            btn.classList.add('active');
            const tabId = btn.dataset.tab;
            document.getElementById(tabId).classList.add('active');
            currentTab = tabId;
            
            // 切换标签后加载对应数据
            if (tabId === 'dashboard') {
                loadDashboard();
            } else if (tabId === 'query') {
                queryLogs();
            }
        });
    });
}

// 初始化配置子标签
function initConfigTabs() {
    document.querySelectorAll('.config-tab').forEach(tab => {
        tab.addEventListener('click', () => {
            document.querySelectorAll('.config-tab').forEach(t => t.classList.remove('active'));
            document.querySelectorAll('.config-panel').forEach(p => p.style.display = 'none');
            tab.classList.add('active');
            const configId = 'config-' + tab.dataset.config;
            const panel = document.getElementById(configId);
            if (panel) panel.style.display = 'block';
        });
    });
}

// 调整数值输入框
function adjustNumber(inputId, delta) {
    const input = document.getElementById(inputId);
    if (!input) return;
    const current = parseInt(input.value) || 0;
    const min = parseInt(input.min) || 0;
    const max = parseInt(input.max) || Infinity;
    const step = parseInt(input.step) || 1;
    input.value = Math.max(min, Math.min(max, current + delta * step));
}

// 生成访问令牌
function generateToken() {
    const token = 'tk_' + Math.random().toString(36).substring(2, 15) + Math.random().toString(36).substring(2, 15);
    const input = document.getElementById('receiver-http-token');
    if (input) input.value = token;
}

// 复制数据库路径
function copyPath() {
    const pathText = document.getElementById('storage-path-text');
    if (pathText) {
        navigator.clipboard.writeText(pathText.textContent).then(() => {
            alert('路径已复制到剪贴板');
        });
    }
}

// 初始化导入上传区域
function initUploadZone() {
    const zone = document.getElementById('upload-zone');
    const input = document.getElementById('file-input');
    
    if (!zone || !input) return;
    
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

function createImportId(file, index) {
    const suffix = Math.random().toString(36).slice(2, 8);
    return `import_${Date.now()}_${index}_${file.size || 0}_${suffix}`;
}

function startImportProgressPolling(importId, onProgress) {
    let stopped = false;

    const poll = async () => {
        while (!stopped) {
            try {
                const response = await fetch(`/api/logs/import/progress?id=${encodeURIComponent(importId)}`, {
                    cache: 'no-store'
                });

                if (response.ok) {
                    const progress = await response.json();
                    if (typeof onProgress === 'function') {
                        onProgress(progress);
                    }

                    if (['completed', 'partial', 'warning', 'error'].includes(progress.phase)) {
                        break;
                    }
                }
            } catch (error) {
                // 忽略短暂轮询失败，等待下一轮
            }

            await new Promise(resolve => setTimeout(resolve, 500));
        }
    };

    poll();

    return () => {
        stopped = true;
    };
}

function importFileWithProgress(file, importId, onUploadProgress, onServerProcessing) {
    return new Promise((resolve, reject) => {
        const xhr = new XMLHttpRequest();
        xhr.open('POST', '/api/logs/import', true);

        let markedProcessing = false;
        const markProcessing = () => {
            if (!markedProcessing) {
                markedProcessing = true;
                if (typeof onServerProcessing === 'function') {
                    onServerProcessing();
                }
            }
        };

        xhr.upload.addEventListener('progress', (event) => {
            if (!event.lengthComputable) return;
            if (typeof onUploadProgress === 'function') {
                onUploadProgress(event.loaded, event.total);
            }
            if (event.loaded >= event.total) {
                markProcessing();
            }
        });

        xhr.upload.addEventListener('load', () => {
            markProcessing();
        });

        xhr.addEventListener('load', () => {
            resolve({
                ok: xhr.status >= 200 && xhr.status < 300,
                status: xhr.status,
                text: xhr.responseText || ''
            });
        });

        xhr.addEventListener('error', () => {
            reject(new Error('网络错误，导入请求失败'));
        });

        xhr.addEventListener('abort', () => {
            reject(new Error('导入请求已取消'));
        });

        const formData = new FormData();
        formData.append('file', file);
        formData.append('import_id', importId);
        xhr.send(formData);
    });
}

// 处理导入文件
async function handleFiles(files) {
    const fileList = Array.from(files || []);
    if (fileList.length === 0) return;

    const progressSection = document.getElementById('upload-progress-section');
    const progressFill = document.getElementById('progress-fill');
    const progressFilename = document.getElementById('progress-filename');
    const progressPercent = document.getElementById('progress-percent');
    const progressSize = document.getElementById('progress-size');
    const progressSpeed = document.getElementById('progress-speed');
    const resultsSection = document.getElementById('upload-results-section');
    const resultsDiv = document.getElementById('upload-results');
    const selectedCountEl = document.getElementById('import-selected-count');
    const successCountEl = document.getElementById('import-success-count');
    const failCountEl = document.getElementById('import-fail-count');
    const fileInput = document.getElementById('file-input');

    if (!progressSection || !progressFill || !progressFilename || !progressPercent || !resultsSection || !resultsDiv) {
        return;
    }

    let successFiles = 0;
    let failFiles = 0;
    let hasSuccess = false;
    const totalBytes = fileList.reduce((sum, file) => sum + (file.size || 0), 0);
    let uploadedBytes = 0;
    let currentFileUploadedBytes = 0;
    let requestPending = false;
    let speedHint = '';
    let processingProgress = null;
    const startedAt = Date.now();

    if (selectedCountEl) selectedCountEl.textContent = String(fileList.length);
    if (successCountEl) successCountEl.textContent = '0';
    if (failCountEl) failCountEl.textContent = '0';

    progressSection.style.display = 'block';
    resultsSection.style.display = 'none';
    resultsDiv.innerHTML = '';

    const updateProgressStats = (doneFiles) => {
        const backendPercent = requestPending && processingProgress
            ? Math.max(0, Math.min(100, Number(processingProgress.percent || 0)))
            : null;
        const displayUploaded = uploadedBytes + currentFileUploadedBytes;
        const percent = backendPercent !== null
            ? Math.round(backendPercent)
            : totalBytes > 0
                ? Math.min(100, Math.round((displayUploaded / totalBytes) * 100))
                : Math.round((doneFiles / fileList.length) * 100);
        const displayPercent = percent;
        progressPercent.textContent = `${displayPercent}%`;
        progressFill.style.width = `${displayPercent}%`;

        if (progressSize) {
            if (requestPending && processingProgress) {
                const parsed = Number(processingProgress.parsed_lines || 0).toLocaleString();
                const target = Number(processingProgress.target_lines || 0).toLocaleString();
                progressSize.textContent = `已解析 ${parsed} / ${target} 条`;
            } else {
                progressSize.textContent = `${formatBytes(displayUploaded)} / ${formatBytes(totalBytes)}`;
            }
        }

        if (progressSpeed) {
            if (requestPending && processingProgress) {
                const written = Number(processingProgress.written_lines || 0).toLocaleString();
                const skipped = Number(processingProgress.skipped_lines || 0).toLocaleString();
                progressSpeed.textContent = `已写入 ${written} 条 · 已跳过 ${skipped} 条`;
            } else if (speedHint) {
                progressSpeed.textContent = speedHint;
            } else {
                const elapsedSeconds = Math.max((Date.now() - startedAt) / 1000, 0.1);
                progressSpeed.textContent = `${formatBytes(displayUploaded / elapsedSeconds)}/s`;
            }
        }
    };

    updateProgressStats(0);

    for (let i = 0; i < fileList.length; i++) {
        const file = fileList[i];
        const importId = createImportId(file, i);
        let stopProgressPolling = null;
        progressFilename.textContent = file.name;
        currentFileUploadedBytes = 0;
        requestPending = true;
        speedHint = '上传中...';
        processingProgress = null;
        updateProgressStats(i);

        try {
            const response = await importFileWithProgress(
                file,
                importId,
                (loaded, total) => {
                    const safeTotal = total > 0 ? total : (file.size || 1);
                    const ratio = Math.min(1, Math.max(0, loaded / safeTotal));
                    currentFileUploadedBytes = Math.round((file.size || 0) * ratio);
                    speedHint = '上传中...';
                    updateProgressStats(i);
                },
                () => {
                    currentFileUploadedBytes = file.size || currentFileUploadedBytes;
                    speedHint = '';
                    if (!stopProgressPolling) {
                        stopProgressPolling = startImportProgressPolling(importId, (progress) => {
                            processingProgress = progress;
                            currentFileUploadedBytes = Math.round((file.size || 0) * (Number(progress.percent || 0) / 100));
                            updateProgressStats(i);
                        });
                    }
                    updateProgressStats(i);
                }
            );

            if ((file.size || 0) > 0 && currentFileUploadedBytes === 0) {
                currentFileUploadedBytes = file.size;
                speedHint = '';
                if (!stopProgressPolling) {
                    stopProgressPolling = startImportProgressPolling(importId, (progress) => {
                        processingProgress = progress;
                        currentFileUploadedBytes = Math.round((file.size || 0) * (Number(progress.percent || 0) / 100));
                        updateProgressStats(i);
                    });
                }
                updateProgressStats(i);
            }

            let result = {};
            const text = response.text;
            try {
                result = JSON.parse(text);
            } catch (e) {
                result = { error: text || 'Unknown error' };
            }

            const imported = Number(result.imported ?? result.accepted ?? 0);
            const accepted = Number(result.accepted ?? imported);
            const dropped = Number(result.dropped ?? 0);
            const hasWarning = Boolean(result.warning);
            const isPartial = result.status === 'partial' || (accepted > 0 && imported < accepted);
            const isSuccess = response.ok && imported > 0;

            const resultItem = document.createElement('div');
            resultItem.className = 'result-item';

            if (isSuccess) {
                if (isPartial || hasWarning) {
                    resultItem.classList.add('warning');
                }

                let detailText = `成功导入 ${imported} 条`;
                if (accepted > 0 && accepted !== imported) {
                    detailText = `提交 ${accepted} 条，实际导入 ${imported} 条`;
                }
                if (dropped > 0) {
                    detailText += `（丢弃 ${dropped} 条）`;
                }
                if (hasWarning) {
                    detailText += `<br><span style="color: var(--warning); font-size: 12px;">${result.warning}</span>`;
                }

                resultItem.innerHTML = `
                    <div class="result-icon">${isPartial ? '<i class="fas fa-exclamation-triangle"></i>' : '<i class="fas fa-check"></i>'}</div>
                    <div class="result-info">
                        <div class="result-filename">${file.name}</div>
                        <div class="result-detail">${detailText}</div>
                    </div>
                    <div class="result-count">${imported}</div>
                `;

                successFiles += 1;
                hasSuccess = true;
            } else if (result.status === 'warning') {
                resultItem.classList.add('warning');
                resultItem.innerHTML = `
                    <div class="result-icon"><i class="fas fa-exclamation-triangle"></i></div>
                    <div class="result-info">
                        <div class="result-filename">${file.name}</div>
                        <div class="result-detail">${result.warning || '格式不匹配'}</div>
                    </div>
                    <div class="result-count">0</div>
                `;
                failFiles += 1;
            } else {
                resultItem.classList.add('error');
                resultItem.innerHTML = `
                    <div class="result-icon"><i class="fas fa-times"></i></div>
                    <div class="result-info">
                        <div class="result-filename">${file.name}</div>
                        <div class="result-detail">${result.error || '导入失败'}</div>
                    </div>
                `;
                failFiles += 1;
            }

            resultsDiv.appendChild(resultItem);
        } catch (error) {
            const resultItem = document.createElement('div');
            resultItem.className = 'result-item error';
            resultItem.innerHTML = `
                <div class="result-icon"><i class="fas fa-times"></i></div>
                <div class="result-info">
                    <div class="result-filename">${file.name}</div>
                    <div class="result-detail">${error.message}</div>
                </div>
            `;
            resultsDiv.appendChild(resultItem);
            failFiles += 1;
        } finally {
            if (stopProgressPolling) {
                stopProgressPolling();
            }
            requestPending = false;
            speedHint = '';
            processingProgress = null;
            uploadedBytes += (file.size || 0);
            currentFileUploadedBytes = 0;
            updateProgressStats(i + 1);
            if (successCountEl) successCountEl.textContent = String(successFiles);
            if (failCountEl) failCountEl.textContent = String(failFiles);
        }
    }

    progressFill.style.width = '100%';
    progressPercent.textContent = '100%';
    if (progressSize) {
        progressSize.textContent = `${formatBytes(totalBytes)} / ${formatBytes(totalBytes)}`;
    }
    resultsSection.style.display = 'block';

    setTimeout(() => {
        progressSection.style.display = 'none';
    }, 1500);

    if (fileInput) fileInput.value = '';

    if (hasSuccess && currentTab === 'dashboard') {
        setTimeout(() => loadDashboard(), 500);
    }
}
// 加载概览统计数据
async function loadDashboard() {
    try {
        const response = await fetch('/api/statistics');
        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }
        
        const stats = await response.json();
        console.log('Dashboard data:', stats);
        
        // 更新顶部统计卡片
        const totalCount = stats.total_count || 0;
        const errorCount = stats.error_count || 0;
        const avgResponse = stats.avg_response_time || 0;
        
        document.getElementById('total-logs').textContent = totalCount.toLocaleString();
        document.getElementById('error-logs').textContent = errorCount.toLocaleString();
        document.getElementById('avg-response').textContent = Math.round(avgResponse) + 'ms';
        document.getElementById('system-status').textContent = '运行中';
        document.getElementById('system-status').className = 'stat-value';
        
        // 计算错误率
        if (totalCount > 0) {
            const errorRate = ((errorCount / totalCount) * 100).toFixed(1);
            document.getElementById('error-rate').textContent = `错误率 ${errorRate}%`;
        } else {
            document.getElementById('error-rate').textContent = '';
        }
        
        // 更新刷新时间
        document.getElementById('last-update').textContent = '刚刚更新';
        
        // 渲染状态码与方法分布
        renderStatusChart(stats.status_code_dist || {});
        renderMethodChart(stats.method_dist || {});
        // 趋势图接口可按需补充
        
    } catch (error) {
        console.error('Failed to load dashboard:', error);
        document.getElementById('system-status').textContent = '连接失败';
        document.getElementById('system-status').className = 'stat-value error';
        document.getElementById('last-update').textContent = '刷新失败';
        
        // 加载失败时显示空态
        renderEmptyChart('status-chart', '暂无数据');
        renderEmptyChart('method-chart', '暂无数据');
        // 趋势图接口可按需补充
    }
}

// 渲染空状态图表
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

// 手动刷新概览
function refreshDashboard() {
    const btn = document.querySelector('.btn-icon .fa-sync-alt');
    if (btn) {
        btn.classList.add('fa-spin');
        setTimeout(() => btn.classList.remove('fa-spin'), 1000);
    }
    loadDashboard();
}

// 切换主导航标签
function switchTab(tabName) {
    const tabBtn = document.querySelector(`.nav-btn[data-tab="${tabName}"]`);
    if (tabBtn) {
        tabBtn.click();
    }
}

// 渲染状态码分布卡片
function renderStatusChart(data) {
    const container = document.getElementById('status-chart');
    const totalEl = document.getElementById('status-total');

    if (!container) return;

    if (!data || Object.keys(data).length === 0) {
        container.innerHTML = `
            <div class="empty-state">
                <i class="fas fa-shield-alt"></i>
                <h4>暂无数据</h4>
                <p>导入或接收日志后将显示状态码分布</p>
            </div>
        `;
        if (totalEl) totalEl.textContent = '总计 0 条';
        return;
    }

    const total = Object.values(data).reduce((a, b) => a + b, 0);
    if (totalEl) totalEl.textContent = `总计 ${total.toLocaleString()} 条`;

    const sortedData = Object.entries(data).sort((a, b) => parseInt(a[0], 10) - parseInt(b[0], 10));
    const getStatusInfo = (code) => {
        const c = parseInt(code, 10);
        if (c >= 200 && c < 300) return { class: 'success', label: '成功' };
        if (c >= 300 && c < 400) return { class: 'redirect', label: '重定向' };
        if (c >= 400 && c < 500) return { class: 'client-error', label: '客户端错误' };
        if (c >= 500) return { class: 'server-error', label: '服务端错误' };
        return { class: 'success', label: '其他' };
    };

    let html = '<div class="status-cards">';
    for (const [code, count] of sortedData) {
        const percent = total > 0 ? ((count / total) * 100).toFixed(1) : '0.0';
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
// 格式化数字为紧凑形式（如 1.2K, 3.5M）
function formatCompactNumber(num) {
    if (num >= 1000000) {
        return (num / 1000000).toFixed(1) + 'M';
    } else if (num >= 1000) {
        return (num / 1000).toFixed(1) + 'K';
    }
    return num.toString();
}

// 渲染请求方法分布
function renderMethodChart(data) {
    const container = document.getElementById('method-chart');
    const totalEl = document.getElementById('method-total');

    if (!container) return;

    if (!data || Object.keys(data).length === 0) {
        container.innerHTML = `
            <div class="empty-state">
                <i class="fas fa-code-branch"></i>
                <h4>暂无数据</h4>
                <p>导入或接收日志后将显示请求方法分布</p>
            </div>
        `;
        if (totalEl) totalEl.textContent = '总计 0 条';
        return;
    }

    const total = Object.values(data).reduce((a, b) => a + b, 0);
    if (totalEl) totalEl.textContent = `总计 ${total.toLocaleString()} 条`;

    const sortedData = Object.entries(data).sort((a, b) => b[1] - a[1]);
    const colors = ['#4472C4', '#52C41A', '#FAAD14', '#F5222D', '#722ED1', '#13C2C2', '#EB2F96', '#FA541C'];

    let currentAngle = 0;
    const arcs = sortedData.map(([method, count], index) => {
        const percentage = total > 0 ? count / total : 0;
        const angle = percentage * 360;
        const startAngle = currentAngle;
        const endAngle = currentAngle + angle;
        currentAngle += angle;

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

    let html = '<div class="method-donut">';
    html += `
        <div class="donut-chart">
            <svg class="donut-svg" viewBox="0 0 180 180">
                ${arcs.map(arc => `
                    <path d="${arc.path}" fill="${arc.color}" opacity="0.9">
                        <title>${arc.method}: ${arc.count} (${arc.percentage}%)</title>
                    </path>
                `).join('')}
                <circle cx="90" cy="90" r="45" fill="white"/>
            </svg>
            <div class="donut-center">
                <div class="donut-value" title="${total.toLocaleString()}">${formatCompactNumber(total)}</div>
                <div class="donut-label">总请求</div>
            </div>
        </div>
    `;

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
    
    // 只展示最近 30 个时间点
    const displayData = data.slice(-30);
    
    let html = '<div style="padding: 16px 0;">';
    
    // 顶部统计信息
    html += `
        <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 20px; padding: 12px 16px; background: var(--bg-secondary); border-radius: 8px;">
            <div style="font-size: 13px; color: var(--text-secondary);">
                <i class="fas fa-clock" style="margin-right: 6px;"></i>
                时间范围: ${formatTime(displayData[0]?.time)} ~ ${formatTime(displayData[displayData.length - 1]?.time)}
            </div>
            <div style="font-size: 13px; color: var(--text-secondary);">
                总计: <strong style="color: var(--primary);">${total.toLocaleString()}</strong> 条
            </div>
        </div>
    `;
    
    // 柱状图主体
    html += '<div class="trend-chart" style="display: flex; align-items: flex-end; gap: 4px; height: 180px; padding: 10px 0; border-bottom: 1px solid var(--border-light);">';
    
    displayData.forEach((point, index) => {
        const count = point.count || 0;
        const height = max > 0 ? (count / max) * 100 : 0;
        const time = point.time || point.Time || '';
        const displayTime = formatTime(time);
        
        // 根据数量区间设置颜色
        let color = 'var(--primary)';
        if (count >= max * 0.8) color = '#52c41a'; // 高峰
        else if (count >= max * 0.5) color = '#1890ff'; // 较高
        else if (count >= max * 0.2) color = '#faad14'; // 中等
        else color = '#d9d9d9'; // 较低
        
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
    
    // 底部只显示首、中、尾三个时间刻度，避免过密
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
                <span>高峰 (>=80%)</span>
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
                <span>低谷 (<20%)</span>
            </div>
        </div>
    `;
    
    html += '</div>';
    container.innerHTML = html;
}

// 格式化时间显示
function formatTime(timeStr) {
    if (!timeStr) return '-';
    // 兼容 2026-03-10 13:45 和 2026-03-10T13:45:00+08:00
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
    const methods = Array.from(document.querySelectorAll('#filter-method .method-tag.active')).map(btn => btn.dataset.value);
    const statusCodes = Array.from(document.querySelectorAll('#filter-status .status-tag.active')).flatMap(btn => btn.dataset.value.split(','));
    const keyword = document.getElementById('filter-keyword').value;
    
    // 更新当前筛选标签
    updateActiveFilters({ startTime, endTime, methods, statusCodes, keyword });
    
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
        const statusClass = getStatusCodeClass(log.status_code);
        row.innerHTML = `
            <td>${new Date(log.timestamp).toLocaleString()}</td>
            <td><span class="method-badge method-${log.method}">${log.method || '-'}</span></td>
            <td class="path-cell" title="${log.path || '-'}">${truncate(log.path, 30)}</td>
            <td><span class="status-badge ${statusClass}">${log.status_code || '-'}</span></td>
            <td>${log.response_time ? log.response_time + 'ms' : '-'}</td>
            <td>${log.client_ip || '-'}</td>
            <td>
                <button class="btn-view" data-log='${JSON.stringify(log).replace(/'/g, "&#39;")}'><i class="fas fa-eye"></i></button>
                <button class="btn-delete" data-id="${log.id}"><i class="fas fa-trash"></i></button>
            </td>
        `;
        tbody.appendChild(row);
    });
    
    // 绑定查看和删除按钮事件
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
        // ID 可能包含特殊字符，先编码再拼接 URL
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
            queryLogs(); // 删除后刷新列表
            // 如果当前在概览页，也同步刷新统计
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

// 选择导出格式
function selectExportFormat(format) {
    document.querySelectorAll('.export-format-card').forEach(card => {
        card.classList.remove('active');
    });
    document.querySelector(`.export-format-card[data-format="${format}"]`)?.classList.add('active');
    document.getElementById('export-format').value = format;
    
    // 更新文件扩展名
    const extMap = { excel: '.xlsx', csv: '.csv', json: '.json' };
    document.getElementById('filename-ext').textContent = extMap[format] || '.xlsx';
    
    // 切换格式后同步刷新导出预览
    updateExportPreview();
}

// 切换导出状态码筛选
function toggleExportStatus(btn) {
    btn.classList.toggle('active');
    updateExportStatusFilter();
}

// 汇总导出状态码筛选值
function updateExportStatusFilter() {
    const activeBtns = document.querySelectorAll('.status-filter-btn.active');
    const statuses = Array.from(activeBtns).map(btn => btn.dataset.status).join(',');
    document.getElementById('export-status').value = statuses;
    updateExportPreview(); // 变更后立即刷新预览
}

// 更新导出预览
async function updateExportPreview() {
    const startTime = document.getElementById('export-start-time').value;
    const endTime = document.getElementById('export-end-time').value;
    const statusCodes = document.getElementById('export-status').value;
    const format = document.getElementById('export-format').value;

    const countEl = document.getElementById('export-count');
    const sizeEl = document.getElementById('export-size');
    const rangeEl = document.getElementById('export-range');

    const params = new URLSearchParams();
    params.append('limit', '1');
    if (startTime) params.append('start_time', new Date(startTime).toISOString());
    if (endTime) params.append('end_time', new Date(endTime).toISOString());
    if (statusCodes) {
        statusCodes.split(',').forEach(code => {
            code.split(',').forEach(c => params.append('status_codes', c.trim()));
        });
    }

    try {
        const response = await fetch(`/api/logs?${params}`);
        const result = await response.json();
        const total = result.total || 0;

        countEl.textContent = `${total.toLocaleString()} 条`;

        let bytesPerRecord;
        switch (format) {
            case 'json':
                bytesPerRecord = 300;
                break;
            case 'excel':
                bytesPerRecord = 200;
                break;
            case 'csv':
                bytesPerRecord = 150;
                break;
            default:
                bytesPerRecord = 200;
        }

        const totalBytes = total * bytesPerRecord;
        let sizeText;
        if (totalBytes < 1024) {
            sizeText = `${totalBytes} B`;
        } else if (totalBytes < 1024 * 1024) {
            sizeText = `${(totalBytes / 1024).toFixed(1)} KB`;
        } else if (totalBytes < 1024 * 1024 * 1024) {
            sizeText = `${(totalBytes / (1024 * 1024)).toFixed(1)} MB`;
        } else {
            sizeText = `${(totalBytes / (1024 * 1024 * 1024)).toFixed(2)} GB`;
        }
        sizeEl.textContent = sizeText;

        if (startTime && endTime) {
            const start = new Date(startTime).toLocaleDateString();
            const end = new Date(endTime).toLocaleDateString();
            rangeEl.textContent = `${start} 至 ${end}`;
        } else if (startTime) {
            rangeEl.textContent = `${new Date(startTime).toLocaleDateString()} 之后`;
        } else if (endTime) {
            rangeEl.textContent = `${new Date(endTime).toLocaleDateString()} 之前`;
        } else {
            rangeEl.textContent = '全部';
        }
    } catch (error) {
        console.error('Failed to update export preview:', error);
        countEl.textContent = '-';
        sizeEl.textContent = '-';
        rangeEl.textContent = '-';
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
            // 某些失败场景会返回 JSON 错误信息，先解析再处理
            if (contentType && contentType.includes('application/json')) {
                const result = await response.json();
                if (result.error) {
                    alert('导出失败: ' + result.error);
                    return;
                }
            }
            
            // 再读取 blob 作为真正导出内容
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
            alert('导出成功，已生成文件');
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
    if (!confirm(`确认清空所有日志吗？\n当前结果：${count}\n\n此操作不可恢复。`)) {
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
            queryLogs(); // 清空后刷新列表
            // 如果当前在概览页，也同步刷新统计
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

// 截断过长文本
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

// 关闭详情弹窗
function closeModal() {
    document.getElementById('log-modal').classList.remove('active');
}

// 切换查询方法标签
function toggleMethod(btn) {
    btn.classList.toggle('active');
}

// 切换查询状态码标签
function toggleStatus(btn) {
    btn.classList.toggle('active');
}

// 添加字段映射行
function addMappingRow(index = '', field = '') {
    const list = document.getElementById('mapping-list');
    if (!list) return;
    
    const row = document.createElement('div');
    row.className = 'mapping-row';
    row.innerHTML = `
        <span class="field-index">${index || list.children.length}</span>
        <i class="fas fa-arrow-right field-arrow"></i>
        <input type="text" class="field-name" placeholder="字段名" value="${field}">
        <button type="button" class="btn-remove" onclick="removeMappingRow(this)">
            <i class="fas fa-times"></i>
        </button>
    `;
    list.appendChild(row);
    updateMappingJSON();
}

// 删除字段映射行
function removeMappingRow(btn) {
    btn.closest('.mapping-row').remove();
    updateMappingIndices();
    updateMappingJSON();
}

// 更新字段映射序号
function updateMappingIndices() {
    const rows = document.querySelectorAll('#mapping-list .mapping-row');
    rows.forEach((row, index) => {
        row.querySelector('.field-index').textContent = index;
    });
}

// 同步字段映射 JSON
function updateMappingJSON() {
    const rows = document.querySelectorAll('#mapping-list .mapping-row');
    const mapping = {};
    rows.forEach(row => {
        const index = row.querySelector('.field-index').textContent;
        const field = row.querySelector('.field-name').value.trim();
        if (field) mapping[index] = field;
    });
    const textarea = document.getElementById('parser-mapping');
    if (textarea) textarea.value = JSON.stringify(mapping, null, 2);
}

// 添加清洗规则
function addCleanRule() {
    const list = document.getElementById('clean-rules-list');
    if (!list) return;
    
    const row = document.createElement('div');
    row.className = 'rule-row';
    row.innerHTML = `
        <select onchange="updateCleanRulesJSON()">
            <option value="">选择字段</option>
            <option value="client_ip">客户端 IP</option>
            <option value="method">请求方法</option>
            <option value="path">请求路径</option>
            <option value="status_code">状态码</option>
            <option value="user_agent">User-Agent</option>
        </select>
        <select onchange="updateCleanRulesJSON()">
            <option value="">选择操作</option>
            <option value="trim">去除首尾空白</option>
            <option value="lower">转小写</option>
            <option value="upper">转大写</option>
            <option value="replace">替换字符</option>
        </select>
        <input type="text" placeholder="规则值" oninput="updateCleanRulesJSON()">
        <button type="button" class="btn-remove" onclick="removeRule(this, 'clean')">
            <i class="fas fa-times"></i>
        </button>
    `;
    list.appendChild(row);
    updateCleanRulesJSON();
}

// 添加过滤规则
function addFilterRule() {
    const list = document.getElementById('filter-rules-list');
    if (!list) return;
    
    const row = document.createElement('div');
    row.className = 'rule-row';
    row.innerHTML = `
        <select onchange="updateFilterRulesJSON()">
            <option value="">选择字段</option>
            <option value="client_ip">客户端 IP</option>
            <option value="method">请求方法</option>
            <option value="path">请求路径</option>
            <option value="status_code">状态码</option>
            <option value="response_time">响应时间</option>
        </select>
        <select onchange="updateFilterRulesJSON()">
            <option value="">选择条件</option>
            <option value="eq">等于</option>
            <option value="ne">不等于</option>
            <option value="gt">大于</option>
            <option value="lt">小于</option>
            <option value="contains">包含</option>
            <option value="regex">正则匹配</option>
        </select>
        <input type="text" placeholder="条件值" oninput="updateFilterRulesJSON()">
        <button type="button" class="btn-remove" onclick="removeRule(this, 'filter')">
            <i class="fas fa-times"></i>
        </button>
    `;
    list.appendChild(row);
    updateFilterRulesJSON();
}

// 删除规则行
function removeRule(btn, type) {
    btn.closest('.rule-row').remove();
    if (type === 'clean') updateCleanRulesJSON();
    else updateFilterRulesJSON();
}

// 同步清洗规则 JSON
function updateCleanRulesJSON() {
    const rows = document.querySelectorAll('#clean-rules-list .rule-row');
    const rules = [];
    rows.forEach(row => {
        const selects = row.querySelectorAll('select');
        const input = row.querySelector('input');
        if (selects[0].value && selects[1].value) {
            rules.push({
                field: selects[0].value,
                operation: selects[1].value,
                value: input.value
            });
        }
    });
    const textarea = document.getElementById('processor-clean-rules');
    if (textarea) textarea.value = JSON.stringify(rules, null, 2);
}

// 同步过滤规则 JSON
function updateFilterRulesJSON() {
    const rows = document.querySelectorAll('#filter-rules-list .rule-row');
    const rules = [];
    rows.forEach(row => {
        const selects = row.querySelectorAll('select');
        const input = row.querySelector('input');
        if (selects[0].value && selects[1].value) {
            rules.push({
                field: selects[0].value,
                operator: selects[1].value,
                value: input.value
            });
        }
    });
    const textarea = document.getElementById('processor-filter-rules');
    if (textarea) textarea.value = JSON.stringify(rules, null, 2);
}

// 根据配置初始化字段映射列表
function initMappingList(mapping) {
    const list = document.getElementById('mapping-list');
    if (!list) return;
    list.innerHTML = '';
    
    Object.entries(mapping).forEach(([index, field]) => {
        const row = document.createElement('div');
        row.className = 'mapping-row';
        row.innerHTML = `
            <span class="field-index">${index}</span>
            <i class="fas fa-arrow-right field-arrow"></i>
            <input type="text" class="field-name" placeholder="字段名" value="${field}" oninput="updateMappingJSON()">
            <button type="button" class="btn-remove" onclick="removeMappingRow(this)">
                <i class="fas fa-times"></i>
            </button>
        `;
        list.appendChild(row);
    });
}

// 根据配置初始化清洗规则列表
function initCleanRulesList(rules) {
    const list = document.getElementById('clean-rules-list');
    if (!list) return;
    list.innerHTML = '';
    
    if (rules.length === 0) return;
    
    rules.forEach(rule => {
        const row = document.createElement('div');
        row.className = 'rule-row';
        row.innerHTML = `
            <select onchange="updateCleanRulesJSON()">
                <option value="">选择字段</option>
                <option value="client_ip" ${rule.field === 'client_ip' ? 'selected' : ''}>客户端 IP</option>
                <option value="method" ${rule.field === 'method' ? 'selected' : ''}>请求方法</option>
                <option value="path" ${rule.field === 'path' ? 'selected' : ''}>请求路径</option>
                <option value="status_code" ${rule.field === 'status_code' ? 'selected' : ''}>状态码</option>
                <option value="user_agent" ${rule.field === 'user_agent' ? 'selected' : ''}>User-Agent</option>
            </select>
            <select onchange="updateCleanRulesJSON()">
                <option value="">选择操作</option>
                <option value="trim" ${rule.operation === 'trim' ? 'selected' : ''}>去除首尾空白</option>
                <option value="lower" ${rule.operation === 'lower' ? 'selected' : ''}>转小写</option>
                <option value="upper" ${rule.operation === 'upper' ? 'selected' : ''}>转大写</option>
                <option value="replace" ${rule.operation === 'replace' ? 'selected' : ''}>替换字符</option>
            </select>
            <input type="text" placeholder="规则值" value="${rule.value || ''}" oninput="updateCleanRulesJSON()">
            <button type="button" class="btn-remove" onclick="removeRule(this, 'clean')">
                <i class="fas fa-times"></i>
            </button>
        `;
        list.appendChild(row);
    });
}

// 根据配置初始化过滤规则列表
function initFilterRulesList(rules) {
    const list = document.getElementById('filter-rules-list');
    if (!list) return;
    list.innerHTML = '';
    
    if (rules.length === 0) return;
    
    rules.forEach(rule => {
        const row = document.createElement('div');
        row.className = 'rule-row';
        row.innerHTML = `
            <select onchange="updateFilterRulesJSON()">
                <option value="">选择字段</option>
                <option value="client_ip" ${rule.field === 'client_ip' ? 'selected' : ''}>客户端 IP</option>
                <option value="method" ${rule.field === 'method' ? 'selected' : ''}>请求方法</option>
                <option value="path" ${rule.field === 'path' ? 'selected' : ''}>请求路径</option>
                <option value="status_code" ${rule.field === 'status_code' ? 'selected' : ''}>状态码</option>
                <option value="response_time" ${rule.field === 'response_time' ? 'selected' : ''}>响应时间</option>
            </select>
            <select onchange="updateFilterRulesJSON()">
                <option value="">选择条件</option>
                <option value="eq" ${rule.operator === 'eq' ? 'selected' : ''}>等于</option>
                <option value="ne" ${rule.operator === 'ne' ? 'selected' : ''}>不等于</option>
                <option value="gt" ${rule.operator === 'gt' ? 'selected' : ''}>大于</option>
                <option value="lt" ${rule.operator === 'lt' ? 'selected' : ''}>小于</option>
                <option value="contains" ${rule.operator === 'contains' ? 'selected' : ''}>包含</option>
                <option value="regex" ${rule.operator === 'regex' ? 'selected' : ''}>正则匹配</option>
            </select>
            <input type="text" placeholder="条件值" value="${rule.value || ''}" oninput="updateFilterRulesJSON()">
            <button type="button" class="btn-remove" onclick="removeRule(this, 'filter')">
                <i class="fas fa-times"></i>
            </button>
        `;
        list.appendChild(row);
    });
}

// 根据状态码返回样式类
function getStatusCodeClass(statusCode) {
    if (!statusCode) return '';
    const code = parseInt(statusCode);
    if (code >= 200 && code < 300) return 'status-success';
    if (code >= 300 && code < 400) return 'status-redirect';
    if (code >= 400 && code < 500) return 'status-client-error';
    if (code >= 500 && code < 600) return 'status-server-error';
    return '';
}

// 更新当前筛选条件展示
function updateActiveFilters(filters) {
    const container = document.getElementById('active-filters');
    const list = document.getElementById('active-filters-list');
    const { startTime, endTime, methods, statusCodes, keyword } = filters;
    
    const tags = [];
    
    if (startTime) {
        tags.push(`<span class="active-filter-tag"><i class="fas fa-calendar"></i> 开始: ${new Date(startTime).toLocaleString()}</span>`);
    }
    if (endTime) {
        tags.push(`<span class="active-filter-tag"><i class="fas fa-calendar"></i> 结束: ${new Date(endTime).toLocaleString()}</span>`);
    }
    if (methods.length > 0) {
        tags.push(`<span class="active-filter-tag"><i class="fas fa-code-branch"></i> 方法: ${methods.join(', ')}</span>`);
    }
    if (statusCodes.length > 0) {
        const statusNames = [];
        if (statusCodes.includes('200')) statusNames.push('200 成功');
        if (statusCodes.includes('301') || statusCodes.includes('302')) statusNames.push('30x 重定向');
        if (statusCodes.includes('400') || statusCodes.includes('401') || statusCodes.includes('403') || statusCodes.includes('404')) statusNames.push('40x 客户端错误');
        if (statusCodes.includes('500') || statusCodes.includes('502') || statusCodes.includes('503')) statusNames.push('50x 服务端错误');
        tags.push(`<span class="active-filter-tag"><i class="fas fa-shield-alt"></i> 状态: ${statusNames.join(', ')}</span>`);
    }
    if (keyword) {
        tags.push(`<span class="active-filter-tag"><i class="fas fa-search"></i> 关键词: ${keyword}</span>`);
    }
    
    if (tags.length > 0) {
        list.innerHTML = tags.join('');
        container.style.display = 'flex';
    } else {
        container.style.display = 'none';
    }
}

// 重置查询条件
function resetFilters() {
    applyTimePreset('filter', 'clear');
    document.querySelectorAll('#filter-method .method-tag').forEach(btn => btn.classList.remove('active'));
    document.querySelectorAll('#filter-status .status-tag').forEach(btn => btn.classList.remove('active'));
    document.getElementById('filter-keyword').value = '';
    currentPage = 1;
    queryLogs();
}

// 上一页
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

// 更新分页状态
function updatePagination() {
    const maxPage = Math.max(1, Math.ceil(currentTotal / currentLimit));
    document.getElementById('page-info').textContent = `第 ${currentPage} / ${maxPage} 页`;
    
    // 同步分页按钮状态
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

// 加载系统配置
async function loadConfig() {
    try {
        const response = await fetch('/api/config');
        const config = await response.json();
        
        // Processor 配置
        const workers = config.processor?.worker_count || 10;
        const batchSize = config.processor?.batch_size || 100;
        const timeout = config.processor?.batch_timeout || 1000;
        const overflowEnabled = config.processor?.overflow_enabled ?? true;
        const overflowDir = config.processor?.overflow_dir || './data/overflow';
        const overflowMaxMB = config.processor?.overflow_max_disk_mb || 512;
        const overflowDrainBatch = config.processor?.overflow_drain_batch || 1000;
        const overflowDrainInterval = config.processor?.overflow_drain_interval_ms || 200;
        
        const workersInput = document.getElementById('processor-workers');
        const batchSizeInput = document.getElementById('processor-batch-size');
        const timeoutInput = document.getElementById('processor-timeout');
        if (workersInput) workersInput.value = workers;
        if (batchSizeInput) batchSizeInput.value = batchSize;
        if (timeoutInput) timeoutInput.value = timeout;

        const overflowEnabledInput = document.getElementById('processor-overflow-enabled');
        const overflowDirInput = document.getElementById('processor-overflow-dir');
        const overflowMaxMBInput = document.getElementById('processor-overflow-max-mb');
        const overflowDrainBatchInput = document.getElementById('processor-overflow-drain-batch');
        const overflowDrainIntervalInput = document.getElementById('processor-overflow-drain-interval');
        if (overflowEnabledInput) overflowEnabledInput.checked = overflowEnabled;
        if (overflowDirInput) overflowDirInput.value = overflowDir;
        if (overflowMaxMBInput) overflowMaxMBInput.value = overflowMaxMB;
        if (overflowDrainBatchInput) overflowDrainBatchInput.value = overflowDrainBatch;
        if (overflowDrainIntervalInput) overflowDrainIntervalInput.value = overflowDrainInterval;
        
        // 更新性能参数展示值
        updateSliderValue('processor-workers', workers);
        updateSliderValue('processor-batch-size', batchSize);
        updateSliderValue('processor-timeout', timeout);
        
        // 根据当前参数自动高亮匹配的预设
        detectAndApplyPreset(workers, batchSize, timeout);
        
        // Receiver 配置
        const tcpEnabledInput = document.getElementById('receiver-tcp');
        const tcpPortInput = document.getElementById('receiver-tcp-port');
        const udpEnabledInput = document.getElementById('receiver-udp');
        const udpPortInput = document.getElementById('receiver-udp-port');
        const httpEnabledInput = document.getElementById('receiver-http');
        const httpPortInput = document.getElementById('receiver-http-port');
        const httpTokenInput = document.getElementById('receiver-http-token');
        const httpIpsInput = document.getElementById('receiver-http-ips');
        if (tcpEnabledInput) tcpEnabledInput.checked = config.receiver?.tcp_enabled ?? true;
        if (tcpPortInput) tcpPortInput.value = config.receiver?.tcp_port || 9000;
        if (udpEnabledInput) udpEnabledInput.checked = config.receiver?.udp_enabled ?? true;
        if (udpPortInput) udpPortInput.value = config.receiver?.udp_port || 9001;
        if (httpEnabledInput) httpEnabledInput.checked = config.receiver?.http_enabled ?? true;
        if (httpPortInput) httpPortInput.value = config.receiver?.http_port || 9002;
        if (httpTokenInput) httpTokenInput.value = config.receiver?.http_auth_token || '';
        if (httpIpsInput) httpIpsInput.value = (config.receiver?.http_allowed_ips || []).join(', ');
        
        // Storage 配置
        const dbPath = config.storage?.db_path || './data/logs.db';
        const dbPathInput = document.getElementById('storage-db-path');
        if (dbPathInput) dbPathInput.value = dbPath;
        const pathText = document.getElementById('storage-path-text');
        if (pathText) {
            pathText.textContent = dbPath;
        }
        
        // 初始化保留时间快捷按钮
        const retention = config.storage?.retention_hours || 720;
        const retentionInput = document.getElementById('storage-retention');
        if (retentionInput) retentionInput.value = retention;
        updateRetentionButtons(retention);

    } catch (error) {
        console.error('Failed to load config:', error);
    } finally {
        // 即使配置加载失败，也尝试刷新存储大小与压测报告
        loadStorageInfo();
        loadBenchmarkReport();
    }
}

// 根据当前性能参数自动识别预设
function detectAndApplyPreset(workers, batchSize, timeout) {
    // 查找匹配的预设项
    let matchedPreset = null;
    for (const [name, preset] of Object.entries(PERFORMANCE_PRESETS)) {
        if (preset.workers === workers && preset.batchSize === batchSize && preset.timeout === timeout) {
            matchedPreset = name;
            break;
        }
    }
    
    // 更新预设卡片高亮
    document.querySelectorAll('.preset-card').forEach(card => {
        card.classList.toggle('active', card.dataset.preset === matchedPreset);
    });
}

// 保存配置
async function saveConfig() {
    // 保存前先校验所有数字输入
    if (!validateNumberInputs()) {
        alert('请检查输入，有些数值超出了允许范围');
        return;
    }
    
    const config = {
        processor: {
            worker_count: parseInt(document.getElementById('processor-workers')?.value) || 10,
            batch_size: parseInt(document.getElementById('processor-batch-size')?.value) || 100,
            batch_timeout: parseInt(document.getElementById('processor-timeout')?.value) || 1000,
            overflow_enabled: document.getElementById('processor-overflow-enabled')?.checked ?? true,
            overflow_dir: document.getElementById('processor-overflow-dir')?.value || './data/overflow',
            overflow_max_disk_mb: parseInt(document.getElementById('processor-overflow-max-mb')?.value) || 512,
            overflow_drain_batch: parseInt(document.getElementById('processor-overflow-drain-batch')?.value) || 1000,
            overflow_drain_interval_ms: parseInt(document.getElementById('processor-overflow-drain-interval')?.value) || 200
        },
        receiver: {
            tcp_enabled: document.getElementById('receiver-tcp')?.checked ?? true,
            tcp_port: parseInt(document.getElementById('receiver-tcp-port')?.value) || 9000,
            udp_enabled: document.getElementById('receiver-udp')?.checked ?? true,
            udp_port: parseInt(document.getElementById('receiver-udp-port')?.value) || 9001,
            http_enabled: document.getElementById('receiver-http')?.checked ?? true,
            http_port: parseInt(document.getElementById('receiver-http-port')?.value) || 9002,
            http_auth_token: document.getElementById('receiver-http-token')?.value || '',
            http_allowed_ips: (document.getElementById('receiver-http-ips')?.value || '').split(',').map(s => s.trim()).filter(s => s)
        },
        storage: {
            db_path: document.getElementById('storage-db-path')?.value || './data/logs.db',
            retention_hours: parseInt(document.getElementById('storage-retention')?.value) || 720
        }
    };
    
    try {
        console.log('[Config] 准备保存配置:', config);
        const response = await fetch('/api/config', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(config)
        });
        
        console.log('[Config] 保存接口响应状态:', response.status);
        
        if (response.ok) {
            const result = await response.json();
            console.log('[Config] 配置保存成功:', result);
            alert('配置保存成功');
        } else {
            const result = await response.json().catch(() => ({ error: '未知错误' }));
            console.error('[Config] 配置保存失败:', result);
            alert('保存失败: ' + (result.error || '服务器错误'));
        }
    } catch (error) {
        console.error('[Config] 保存配置请求异常:', error);
        alert('保存失败: ' + error.message);
    }
}

// 点击弹窗外部区域时关闭详情
window.onclick = function(event) {
    const modal = document.getElementById('log-modal');
    if (event.target === modal) {
        closeModal();
    }
};

// 定时自动刷新概览
setInterval(() => {
    if (currentTab === 'dashboard') {
        console.log('[App] Auto-refreshing dashboard...');
        loadDashboard();
    }
}, 30000);

// 页面重新可见时刷新概览
document.addEventListener('visibilitychange', () => {
    if (!document.hidden && currentTab === 'dashboard') {
        console.log('[App] Page visible, refreshing dashboard...');
        loadDashboard();
    }
});

// ========== 性能预设与压测工具 ==========

// 性能预设
const PERFORMANCE_PRESETS = {
    dev: { workers: 2, batchSize: 50, timeout: 500 },
    standard: { workers: 10, batchSize: 100, timeout: 1000 },
    high: { workers: 20, batchSize: 200, timeout: 2000 },
    ultra: { workers: 50, batchSize: 500, timeout: 5000 }
};

// 应用性能预设
function applyPreset(presetName) {
    const preset = PERFORMANCE_PRESETS[presetName];
    if (!preset) return;
    
    // 写入预设参数
    document.getElementById('processor-workers').value = preset.workers;
    document.getElementById('processor-batch-size').value = preset.batchSize;
    document.getElementById('processor-timeout').value = preset.timeout;
    
    // 同步数值徽标
    updateSliderValue('processor-workers', preset.workers);
    updateSliderValue('processor-batch-size', preset.batchSize);
    updateSliderValue('processor-timeout', preset.timeout);
    
    // 更新预设卡片高亮
    document.querySelectorAll('.preset-card').forEach(card => {
        card.classList.toggle('active', card.dataset.preset === presetName);
    });
}

// 更新滑块/数字展示值
function updateSliderValue(id, value) {
    const badge = document.getElementById(id + '-value');
    if (badge) {
        badge.textContent = value;
    }
}

// 运行快速压测
async function runQuickBenchmark() {
    const btn = document.getElementById('benchmark-run-btn');
    const reportEl = document.getElementById('benchmark-report');
    if (!btn || !reportEl) return;

    const payload = {
        duration_seconds: parseInt(document.getElementById('benchmark-duration')?.value) || 10,
        workers: parseInt(document.getElementById('benchmark-workers')?.value) || 20,
        target_qps: parseInt(document.getElementById('benchmark-target-qps')?.value) || 0
    };

    btn.disabled = true;
    reportEl.value = '压测执行中，请稍候...';

    try {
        const response = await fetch('/api/benchmark/run', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(payload)
        });

        const result = await response.json().catch(() => ({}));
        if (!response.ok) {
            reportEl.value = `压测失败: ${result.error || '未知错误'}`;
            return;
        }

        reportEl.value = formatBenchmarkReport(result);
    } catch (error) {
        reportEl.value = `压测失败: ${error.message}`;
    } finally {
        btn.disabled = false;
    }
}

function formatBenchmarkReport(report) {
    const processorDelta = report.processor_delta || {};
    return [
        `开始时间: ${report.started_at || '-'}`,
        `结束时间: ${report.finished_at || '-'}`,
        `持续时间: ${report.duration_seconds || '-'} 秒`,
        `并发协程: ${report.workers || '-'}`,
        `目标QPS: ${report.target_qps || 0}`,
        '---',
        `提交总数: ${report.submitted ?? 0}`,
        `拒绝总数: ${report.rejected ?? 0}`,
        `接收率: ${(report.accept_rate ?? 0).toFixed(2)}%`,
        `提交QPS: ${(report.submit_qps ?? 0).toFixed(2)}`,
        `入库新增: ${report.stored_added ?? 0}`,
        `入库QPS: ${(report.stored_qps ?? 0).toFixed(2)}`,
        '--- 处理器增量 ---',
        `received_delta: ${processorDelta.received_delta ?? 0}`,
        `processed_delta: ${processorDelta.processed_delta ?? 0}`,
        `dropped_delta: ${processorDelta.dropped_delta ?? 0}`,
        `parse_error_delta: ${processorDelta.parse_error_delta ?? 0}`,
        `spill_delta: ${processorDelta.spill_delta ?? 0}`,
        `overflow_recovered_delta: ${processorDelta.overflow_recovered_delta ?? 0}`,
        `overflow_pending: ${processorDelta.overflow_pending ?? 0}`
    ].join('\n');
}

async function loadBenchmarkReport() {
    const reportEl = document.getElementById('benchmark-report');
    if (!reportEl) return;

    try {
        const response = await fetch('/api/benchmark/report');
        if (!response.ok) return;

        const result = await response.json();
        if (result?.report) {
            reportEl.value = formatBenchmarkReport(result.report);
        }
    } catch (error) {
        console.error('Failed to load benchmark report:', error);
    }
}

async function compactDB() {
    if (!confirm('确定要压缩数据库吗？这将释放未使用空间。')) {
        return;
    }
    
    try {
        const response = await fetch('/api/storage/compact', {
            method: 'POST'
        });
        
        if (response.ok) {
            const result = await response.json();
            alert(`数据库压缩成功，释放空间: ${formatBytes(result.freed_bytes || 0)}`);
            loadStorageInfo();
        } else {
            alert('压缩失败: ' + (await response.text()));
        }
    } catch (error) {
        alert('压缩请求失败: ' + error.message);
    }
}

// 加载存储信息
async function loadStorageInfo() {
    try {
        const response = await fetch('/api/storage/info');
        const info = await response.json();
        
        const sizeEl = document.getElementById('storage-size');
        if (sizeEl && info.size_bytes !== undefined) {
            sizeEl.textContent = formatBytes(info.size_bytes);
        }
    } catch (error) {
        console.error('Failed to load storage info:', error);
    }
}

// 格式化字节大小
function formatBytes(bytes) {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
}




