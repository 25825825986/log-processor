// 閸忋劌鐪悩鑸碘偓?
let currentPage = 1;
let currentLimit = 20;
let currentTotal = 0;
let currentFilter = {};
let currentTab = 'dashboard';

// 鐠佸墽鐤嗘穱婵堟殌閺冨爼妫?
function setRetention(hours) {
    document.getElementById('storage-retention').value = hours;
    updateRetentionButtons(hours);
}

// 閺囧瓨鏌婃穱婵堟殌閺冨爼妫块幐澶愭尦閻樿埖鈧?
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

// 閻╂垵鎯夋穱婵堟殌閺冨爼妫挎潏鎾冲弳濡楀棗褰夐崠?
document.addEventListener('DOMContentLoaded', () => {
    const retentionInput = document.getElementById('storage-retention');
    if (retentionInput) {
        retentionInput.addEventListener('change', function() {
            updateRetentionButtons(parseInt(this.value));
        });
    }
});

// 閺冨爼妫块弽鐓庣础妫板嫯顔曢柊宥囩枂 (娴滆櫣琚崣顖濐嚢閻ㄥ嫬鎮曠粔鏉挎嫲缁€杞扮伐)
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

// 閺冦儱绻旈弽鐓庣础鐎电懓绨查惃鍕闂傚瓨鐗稿?
const FORMAT_TIME_MAPPING = {
    'nginx': '02/Jan/2006:15:04:05 -0700',
    'apache': '02/Jan/2006:15:04:05 -0700',
    'json': '2006-01-02T15:04:05Z07:00',
    'csv': '2006-01-02 15:04:05',
    'custom': ''
};

// 閺佹澘鐡ф潏鎾冲弳濡楀棝鐛欑拠渚€鍘ょ純?
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
    'storage-retention': { min: 1, max: 8760 }, // 閺堚偓婢?楠?8760鐏忓繑妞?
    'benchmark-duration': { min: 3, max: 300 },
    'benchmark-workers': { min: 1, max: 200 },
    'benchmark-target-qps': { min: 0, max: 1000000 }
};

// 閸掓繂顫愰崠鏍ㄦ殶鐎涙绶崗銉︻攱妤犲矁鐦?
function initNumberValidation() {
    Object.keys(NUMBER_INPUT_LIMITS).forEach(id => {
        const input = document.getElementById(id);
        if (input) {
            const limits = NUMBER_INPUT_LIMITS[id];
            
            // 鏉堟挸鍙嗛弮鍫曠崣鐠?
            input.addEventListener('input', function() {
                let value = parseInt(this.value);
                
                // 濞撳懘娅庨棃鐐存殶鐎涙鐡х粭?
                if (isNaN(value)) {
                    this.value = limits.min;
                    return;
                }
                
                // 闂勬劕鍩楅懠鍐ㄦ纯
                if (value < limits.min) {
                    this.value = limits.min;
                    showInputHint(this, `最小值为 ${limits.min}`);
                } else if (value > limits.max) {
                    this.value = limits.max;
                    showInputHint(this, `最大值为 ${limits.max}`);
                }
            });
            
            // 婢跺崬骞撻悞锔惧仯閺冨爼鐛欑拠?
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

// 閺勫墽銇氭潏鎾冲弳閹绘劗銇?
function showInputHint(input, message) {
    // 缁夊娅庨弮褏娈戦幓鎰仛
    const oldHint = input.parentElement.querySelector('.input-hint');
    if (oldHint) oldHint.remove();
    
    // 閸掓稑缂撻弬鐗堝絹缁€?
    const hint = document.createElement('span');
    hint.className = 'input-hint';
    hint.textContent = message;
    hint.style.cssText = 'color: #f5222d; font-size: 12px; margin-left: 8px;';
    
    input.parentElement.appendChild(hint);
    
    // 3缁夋帒鎮楃粔濠氭珟
    setTimeout(() => hint.remove(), 3000);
}

// 妤犲矁鐦夐幍鈧張澶嬫殶鐎涙绶崗?
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

// 閸掓繂顫愰崠?
document.addEventListener('DOMContentLoaded', () => {
    console.log('[App] Initializing...');
    
    initTabs();
    initConfigTabs();
    initUploadZone();
    initFormatListeners();
    initNumberValidation(); // 閸掓繂顫愰崠鏍ㄦ殶鐎涙鐛欑拠?
    initExportPreview(); // 閸掓繂顫愰崠鏍ь嚤閸戞椽顣╃憴?
    
    // 瀵ゆ儼绻滈崝鐘烘祰娴狀亣銆冮弶鎸庢殶閹诡噯绱濈涵顔荤箽DOM鐎瑰苯鍙忓〒鍙夌厠
    setTimeout(() => {
        console.log('[App] Loading dashboard...');
        loadDashboard();
    }, 100);
    
    loadConfig();
    
    console.log('[App] Initialization complete');
});

// 閸掓繂顫愰崠鏍ь嚤閸戞椽顣╃憴?
function initExportPreview() {
    // 閻╂垵鎯夐弮鍫曟？閼煎啫娲块崣妯哄
    const startTimeInput = document.getElementById('export-start-time');
    const endTimeInput = document.getElementById('export-end-time');
    
    if (startTimeInput) {
        startTimeInput.addEventListener('change', updateExportPreview);
    }
    if (endTimeInput) {
        endTimeInput.addEventListener('change', updateExportPreview);
    }
    
    // 閸掓繂顫愰弴瀛樻煀娑撯偓濞?
    updateExportPreview();
}

// 閸掓繂顫愰崠鏍ㄧ壐瀵繒娲冮崥顒€娅?
function initFormatListeners() {
    // 閺冦儱绻旈弽鐓庣础閸欐ê瀵查弮鎯板殰閸斻劏顔曠純顔碱嚠鎼存梻娈戦弮鍫曟？閺嶇厧绱?
    const formatSelect = document.getElementById('parser-format');
    if (formatSelect) {
        formatSelect.addEventListener('change', onLogFormatChange);
    }
    
    // 閸掓繂顫愰崠鏍ㄦ闂傚瓨鐗稿蹇撳幢閻?
    initTimeFormatCards();
    
    // 閸掓繂顫愰崠鏍帳缂冾噣銆夐棃顫唉娴?
    initConfigInteractions();
}

// 閸掓繂顫愰崠鏍帳缂冾噣銆夐棃顫唉娴?
function initConfigInteractions() {
    // 閺嶇厧绱￠崡锛勫闁瀚?
    document.querySelectorAll('.format-card').forEach(card => {
        card.addEventListener('click', () => {
            document.querySelectorAll('.format-card').forEach(c => c.classList.remove('active'));
            card.classList.add('active');
            const formatInput = document.getElementById('parser-format');
            if (formatInput) formatInput.value = card.dataset.format;
        });
    });
    
    // 閸掑棝娈х粭锕傗偓澶嬪
    document.querySelectorAll('.delimiter-btn').forEach(btn => {
        btn.addEventListener('click', () => {
            document.querySelectorAll('.delimiter-btn').forEach(b => b.classList.remove('active'));
            btn.classList.add('active');
            const delimiterInput = document.getElementById('parser-delimiter');
            if (delimiterInput) delimiterInput.value = btn.dataset.value;
        });
    });
    
    // 缂傛挸鍟块崠娲偓澶嬪
    document.querySelectorAll('.buffer-btn').forEach(btn => {
        btn.addEventListener('click', () => {
            document.querySelectorAll('.buffer-btn').forEach(b => b.classList.remove('active'));
            btn.classList.add('active');
            const bufferInput = document.getElementById('receiver-buffer');
            if (bufferInput) bufferInput.value = btn.dataset.value;
        });
    });
    
    // 娣囨繄鏆€缁涙牜鏆愰柅澶嬪
    document.querySelectorAll('.retention-btn').forEach(btn => {
        btn.addEventListener('click', () => {
            setRetention(parseInt(btn.dataset.hours));
        });
    });
    
    // 閸掓繂顫愰崠鏍帛鐠併倕鐡у▓鍨Ё鐏忓嫸绱欐俊鍌涚亯閸掓銆冩稉铏光敄閿?
    const mappingList = document.getElementById('mapping-list');
    if (mappingList && mappingList.children.length === 0) {
        addMappingRow('0', 'client_ip');
        addMappingRow('3', 'timestamp');
        addMappingRow('4', 'method');
        addMappingRow('5', 'path');
    }
}

// 閸掓繂顫愰崠鏍ㄦ闂傚瓨鐗稿蹇撳幢閻?
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

// 闁瀚ㄩ弮鍫曟？閺嶇厧绱?
function selectTimeFormat(format, cardElement) {
    const input = document.getElementById('parser-time-format');
    if (input) input.value = format;
    
    document.querySelectorAll('.time-format-card').forEach(card => {
        card.classList.remove('active');
    });
    if (cardElement) cardElement.classList.add('active');
    
    // 閺囧瓨鏌婃０鍕潔
    updateTimeFormatPreview(format);
}

// 閺冦儱绻旈弽鐓庣础閸欐ê瀵叉径鍕倞
function onLogFormatChange() {
    const format = document.getElementById('parser-format').value;
    const suggestedTimeFormat = FORMAT_TIME_MAPPING[format];
    
    if (suggestedTimeFormat) {
        // 閺屻儲澹樼€电懓绨查惃鍕幢閻?
        const cards = document.querySelectorAll('.time-format-card');
        cards.forEach(card => {
            if (card.dataset.format === suggestedTimeFormat) {
                selectTimeFormat(suggestedTimeFormat, card);
            }
        });
    }
}

// 閺囧瓨鏌婇弮鍫曟？閺嶇厧绱℃０鍕潔
function updateTimeFormatPreview(format) {
    const previewValue = document.getElementById('preview-value');
    if (!previewValue) return;
    
    // 閺屻儲澹樻０鍕啎閻ㄥ嫮銇氭笟?
    const preset = TIME_FORMAT_PRESETS.find(p => p.format === format);
    if (preset) {
        previewValue.textContent = preset.example;
    } else {
        // 閸斻劍鈧胶鏁撻幋鎰仛娓?
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

// 閺嶅洨顒锋い闈涘瀼閹?
function initTabs() {
    document.querySelectorAll('.nav-btn').forEach(btn => {
        btn.addEventListener('click', () => {
            document.querySelectorAll('.nav-btn').forEach(b => b.classList.remove('active'));
            document.querySelectorAll('.tab-content').forEach(t => t.classList.remove('active'));
            
            btn.classList.add('active');
            const tabId = btn.dataset.tab;
            document.getElementById(tabId).classList.add('active');
            currentTab = tabId;
            
            // 閸旂姾娴囩€电懓绨查弫鐗堝祦
            if (tabId === 'dashboard') {
                loadDashboard();
            } else if (tabId === 'query') {
                queryLogs();
            }
        });
    });
}

// 闁板秶鐤嗛弽鍥╊劮妞ら潧鍨忛幑?
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

// 鐠嬪啯鏆ｉ弫鏉跨摟鏉堟挸鍙嗛崐?
function adjustNumber(inputId, delta) {
    const input = document.getElementById(inputId);
    if (!input) return;
    const current = parseInt(input.value) || 0;
    const min = parseInt(input.min) || 0;
    const max = parseInt(input.max) || Infinity;
    const step = parseInt(input.step) || 1;
    input.value = Math.max(min, Math.min(max, current + delta * step));
}

// 閻㈢喐鍨氶梾蹇旀簚Token
function generateToken() {
    const token = 'tk_' + Math.random().toString(36).substring(2, 15) + Math.random().toString(36).substring(2, 15);
    const input = document.getElementById('receiver-http-token');
    if (input) input.value = token;
}

// 婢跺秴鍩楃捄顖氱窞閸掓澘澹€鐠愬瓨婢?
function copyPath() {
    const pathText = document.getElementById('storage-path-text');
    if (pathText) {
        navigator.clipboard.writeText(pathText.textContent).then(() => {
            alert('路径已复制到剪贴板');
        });
    }
}

// 閸掓繂顫愰崠鏍︾瑐娴肩姴灏崺?
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

// 婢跺嫮鎮婇弬鍥︽娑撳﹣绱?
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
    const startedAt = Date.now();

    if (selectedCountEl) selectedCountEl.textContent = String(fileList.length);
    if (successCountEl) successCountEl.textContent = '0';
    if (failCountEl) failCountEl.textContent = '0';

    progressSection.style.display = 'block';
    resultsSection.style.display = 'none';
    resultsDiv.innerHTML = '';

    const updateProgressStats = (doneFiles) => {
        const percent = totalBytes > 0
            ? Math.min(100, Math.round((uploadedBytes / totalBytes) * 100))
            : Math.round((doneFiles / fileList.length) * 100);
        progressPercent.textContent = `${percent}%`;
        progressFill.style.width = `${percent}%`;

        if (progressSize) {
            progressSize.textContent = `${formatBytes(uploadedBytes)} / ${formatBytes(totalBytes)}`;
        }

        if (progressSpeed) {
            const elapsedSeconds = Math.max((Date.now() - startedAt) / 1000, 0.1);
            progressSpeed.textContent = `${formatBytes(uploadedBytes / elapsedSeconds)}/s`;
        }
    };

    updateProgressStats(0);

    for (let i = 0; i < fileList.length; i++) {
        const file = fileList[i];
        const formData = new FormData();
        formData.append('file', file);
        progressFilename.textContent = file.name;

        try {
            const response = await fetch('/api/logs/import', {
                method: 'POST',
                body: formData
            });

            let result = {};
            const text = await response.text();
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
            uploadedBytes += (file.size || 0);
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
// 鍔犺浇浠〃鐩樻暟鎹?
async function loadDashboard() {
    try {
        const response = await fetch('/api/statistics');
        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }
        
        const stats = await response.json();
        console.log('Dashboard data:', stats);
        
        // 閺囧瓨鏌婄紒鐔活吀閸楋紕澧?
        const totalCount = stats.total_count || 0;
        const errorCount = stats.error_count || 0;
        const avgResponse = stats.avg_response_time || 0;
        
        document.getElementById('total-logs').textContent = totalCount.toLocaleString();
        document.getElementById('error-logs').textContent = errorCount.toLocaleString();
        document.getElementById('avg-response').textContent = Math.round(avgResponse) + 'ms';
        document.getElementById('system-status').textContent = '运行中';
        document.getElementById('system-status').className = 'stat-value';
        
        // 鐠侊紕鐣婚柨娆掝嚖閻?
        if (totalCount > 0) {
            const errorRate = ((errorCount / totalCount) * 100).toFixed(1);
            document.getElementById('error-rate').textContent = `错误率 ${errorRate}%`;
        } else {
            document.getElementById('error-rate').textContent = '';
        }
        
        // 閺囧瓨鏌婇弮鍫曟？閹?
        document.getElementById('last-update').textContent = '刚刚更新';
        
        // 濞撳弶鐓嬮崶鎹愩€?
        renderStatusChart(stats.status_code_dist || {});
        renderMethodChart(stats.method_dist || {});
        // 閺冨爼妫跨搾瀣◢閸ユ崘銆冨鑼╅梽?
        
    } catch (error) {
        console.error('Failed to load dashboard:', error);
        document.getElementById('system-status').textContent = '连接失败';
        document.getElementById('system-status').className = 'stat-value error';
        document.getElementById('last-update').textContent = '刷新失败';
        
        // 閺勫墽銇氱粚铏瑰Ц閹?
        renderEmptyChart('status-chart', '暂无数据');
        renderEmptyChart('method-chart', '暂无数据');
        // 閺冨爼妫跨搾瀣◢閸ユ崘銆冨鑼╅梽?
    }
}

// 濞撳弶鐓嬬粚鍝勬禈鐞涖劎濮搁幀?
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

// 閸掗攱鏌婃禒顏囥€冮弶?
function refreshDashboard() {
    const btn = document.querySelector('.btn-icon .fa-sync-alt');
    if (btn) {
        btn.classList.add('fa-spin');
        setTimeout(() => btn.classList.remove('fa-spin'), 1000);
    }
    loadDashboard();
}

// 閸掑洦宕查弽鍥╊劮妞?
function switchTab(tabName) {
    const tabBtn = document.querySelector(`.nav-btn[data-tab="${tabName}"]`);
    if (tabBtn) {
        tabBtn.click();
    }
}

// 濞撳弶鐓嬮悩鑸碘偓浣虹垳閸ユ崘銆?- 閸楋紕澧栧蹇氼啎鐠?
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

// 濞撳弶鐓嬮弬瑙勭《閸ユ崘銆?- 閻滎垰鑸伴崶鎹愵啎鐠?
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
    
    // 閸欘亝妯夌粈鐑樻付鏉?0娑擃亞鍋?
    const displayData = data.slice(-30);
    
    let html = '<div style="padding: 16px 0;">';
    
    // 缂佺喕顓告穱鈩冧紖
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
    
    // 閺岃京濮搁崶?
    html += '<div class="trend-chart" style="display: flex; align-items: flex-end; gap: 4px; height: 180px; padding: 10px 0; border-bottom: 1px solid var(--border-light);">';
    
    displayData.forEach((point, index) => {
        const count = point.count || 0;
        const height = max > 0 ? (count / max) * 100 : 0;
        const time = point.time || point.Time || '';
        const displayTime = formatTime(time);
        
        // 閺嶈宓侀弫浼村櫤鐠佸墽鐤嗘０婊嗗
        let color = 'var(--primary)';
        if (count >= max * 0.8) color = '#52c41a'; // 妤傛ê鍢?- 缂?
        else if (count >= max * 0.5) color = '#1890ff'; // 娑擃厾鐡?- 閽?
        else if (count >= max * 0.2) color = '#faad14'; // 鏉堝啩缍?- 姒?
        else color = '#d9d9d9'; // 瀵板牅缍?- 閻?
        
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
    
    // 閺冨爼妫挎潪瀛樼垼缁涙拝绱欓弰鍓с仛瀵偓婵鈧椒鑵戦梻娣偓浣虹波閺夌噦绱?
    const midIndex = Math.floor(displayData.length / 2);
    html += `
        <div style="display: flex; justify-content: space-between; margin-top: 8px; padding: 0 4px; font-size: 11px; color: var(--text-tertiary);">
            <span>${formatTime(displayData[0]?.time)}</span>
            <span>${formatTime(displayData[midIndex]?.time)}</span>
            <span>${formatTime(displayData[displayData.length - 1]?.time)}</span>
        </div>
    `;
    
    // 閸ュ彞绶?
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

// 閺嶇厧绱￠崠鏍ㄦ闂傚瓨妯夌粈?
function formatTime(timeStr) {
    if (!timeStr) return '-';
    // 婢跺嫮鎮婃稉宥呮倱閺嶇厧绱? 2026-03-10 13:45 閹?2026-03-10T13:45:00+08:00
    const date = new Date(timeStr.replace(' ', 'T'));
    if (isNaN(date.getTime())) return timeStr;
    
    const hours = date.getHours().toString().padStart(2, '0');
    const minutes = date.getMinutes().toString().padStart(2, '0');
    return `${hours}:${minutes}`;
}

// 閺屻儴顕楅弮銉ョ箶
async function queryLogs() {
    const startTime = document.getElementById('filter-start-time').value;
    const endTime = document.getElementById('filter-end-time').value;
    const methods = Array.from(document.querySelectorAll('#filter-method .method-tag.active')).map(btn => btn.dataset.value);
    const statusCodes = Array.from(document.querySelectorAll('#filter-status .status-tag.active')).flatMap(btn => btn.dataset.value.split(','));
    const keyword = document.getElementById('filter-keyword').value;
    
    // 閺勫墽銇氬鏌モ偓澶岀摣闁娼禒?
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

// 濞撳弶鐓嬮弮銉ョ箶鐞涖劍鐗?
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
    
    // 缂佹垵鐣炬禍瀣╂閻╂垵鎯夐崳?
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

// 閸掔娀娅庨崡鏇熸蒋閺冦儱绻?
async function deleteLog(id) {
    if (!confirm('确定要删除这条日志吗？')) {
        return;
    }
    
    try {
        // 鐎?ID 鏉╂稖顢?URL 缂傛牜鐖滈敍宀勪缉閸忓秶澹掑▓濠傜摟缁楋箓妫舵０?
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
            queryLogs(); // 閸掗攱鏌婇崚妤勩€?
            // 婵″倹鐏夎ぐ鎾冲閸︺劍顩х憴鍫ャ€夐敍灞肩瘍閸掗攱鏌婄紒鐔活吀閺佺増宓?
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

// 闁瀚ㄧ€电厧鍤弽鐓庣础
function selectExportFormat(format) {
    document.querySelectorAll('.export-format-card').forEach(card => {
        card.classList.remove('active');
    });
    document.querySelector(`.export-format-card[data-format="${format}"]`)?.classList.add('active');
    document.getElementById('export-format').value = format;
    
    // 閺囧瓨鏌婇弬鍥︽閸氬秴鎮楃紓鈧?
    const extMap = { excel: '.xlsx', csv: '.csv', json: '.json' };
    document.getElementById('filename-ext').textContent = extMap[format] || '.xlsx';
    
    // 閺嶇厧绱￠崣妯哄閺冭埖娲块弬浼搭暕鐟欏牞绱欓弬鍥︽婢堆冪毈娴兼壆鐣绘导姘綁閿?
    updateExportPreview();
}

// 閸掑洦宕茬€电厧鍤悩鑸碘偓浣虹摣闁?
function toggleExportStatus(btn) {
    btn.classList.toggle('active');
    updateExportStatusFilter();
}

// 閺囧瓨鏌婄€电厧鍤悩鑸碘偓浣虹摣闁鈧?
function updateExportStatusFilter() {
    const activeBtns = document.querySelectorAll('.status-filter-btn.active');
    const statuses = Array.from(activeBtns).map(btn => btn.dataset.status).join(',');
    document.getElementById('export-status').value = statuses;
    updateExportPreview(); // 缁涙盯鈧褰夐崠鏍ㄦ閺囧瓨鏌婃０鍕潔
}

// 閺囧瓨鏌婄€电厧鍤０鍕潔
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
            // 婵″倹鐏夋潻鏂挎礀閻ㄥ嫭妲?JSON閿涘矁顕╅弰搴㈡Ц闁挎瑨顕ゆ穱鈩冧紖
            if (contentType && contentType.includes('application/json')) {
                const result = await response.json();
                if (result.error) {
                    alert('导出失败: ' + result.error);
                    return;
                }
            }
            
            // 閼惧嘲褰?blob 楠炶埖顥呴弻銉ャ亣鐏?
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

// 濞撳懐鈹栭幍鈧張澶嬫）韫?
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
            queryLogs(); // 閸掗攱鏌婇崚妤勩€?
            // 閸掗攱鏌婂鍌濐潔閺佺増宓?
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

// 閹搭亝鏌囩€涙顑佹稉?
function truncate(str, length) {
    if (!str) return '-';
    return str.length > length ? str.substring(0, length) + '...' : str;
}

// 閺屻儳婀呴弮銉ョ箶鐠囷附鍎?
function viewLogDetail(log) {
    const modal = document.getElementById('log-modal');
    const detail = document.getElementById('log-detail');
    
    detail.textContent = JSON.stringify(log, null, 2);
    modal.classList.add('active');
}

// 閸忔娊妫村鍦崶
function closeModal() {
    document.getElementById('log-modal').classList.remove('active');
}

// 閸掑洦宕茬拠閿嬬湴閺傝纭堕柅澶嬪
function toggleMethod(btn) {
    btn.classList.toggle('active');
}

// 閸掑洦宕查悩鑸碘偓浣虹垳闁瀚?
function toggleStatus(btn) {
    btn.classList.toggle('active');
}

// 濞ｈ濮炵€涙顔岄弰鐘茬殸鐞?
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

// 閸掔娀娅庣€涙顔岄弰鐘茬殸鐞?
function removeMappingRow(btn) {
    btn.closest('.mapping-row').remove();
    updateMappingIndices();
    updateMappingJSON();
}

// 閺囧瓨鏌婄€涙顔岄弰鐘茬殸缁便垹绱?
function updateMappingIndices() {
    const rows = document.querySelectorAll('#mapping-list .mapping-row');
    rows.forEach((row, index) => {
        row.querySelector('.field-index').textContent = index;
    });
}

// 閺囧瓨鏌婄€涙顔岄弰鐘茬殸JSON
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

// 濞ｈ濮炲〒鍛鐟欏嫬鍨?
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

// 濞ｈ濮炴潻鍥ㄦ姢鐟欏嫬鍨?
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

// 閸掔娀娅庣憴鍕灟鐞?
function removeRule(btn, type) {
    btn.closest('.rule-row').remove();
    if (type === 'clean') updateCleanRulesJSON();
    else updateFilterRulesJSON();
}

// 閺囧瓨鏌婂〒鍛鐟欏嫬鍨疛SON
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

// 閺囧瓨鏌婃潻鍥ㄦ姢鐟欏嫬鍨疛SON
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

// 閸掓繂顫愰崠鏍х摟濞堝灚妲х亸鍕灙鐞?
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

// 閸掓繂顫愰崠鏍ㄧ濞叉顫夐崚娆忓灙鐞?
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

// 閸掓繂顫愰崠鏍箖濠娿倛顫夐崚娆忓灙鐞?
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

// 閼惧嘲褰囬悩鑸碘偓浣虹垳妫版粏澹婄猾?
function getStatusCodeClass(statusCode) {
    if (!statusCode) return '';
    const code = parseInt(statusCode);
    if (code >= 200 && code < 300) return 'status-success';
    if (code >= 300 && code < 400) return 'status-redirect';
    if (code >= 400 && code < 500) return 'status-client-error';
    if (code >= 500 && code < 600) return 'status-server-error';
    return '';
}

// 閺囧瓨鏌婂鏌モ偓澶岀摣闁娼禒鑸垫▔缁€?
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

// 闁插秶鐤嗙粵娑⑩偓?
function resetFilters() {
    document.getElementById('filter-start-time').value = '';
    document.getElementById('filter-end-time').value = '';
    document.querySelectorAll('#filter-method .method-tag').forEach(btn => btn.classList.remove('active'));
    document.querySelectorAll('#filter-status .status-tag').forEach(btn => btn.classList.remove('active'));
    document.getElementById('filter-keyword').value = '';
    currentPage = 1;
    queryLogs();
}

// 閸掑棝銆?
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

// 閺囧瓨鏌婇崚鍡涖€夐幐澶愭尦閻樿埖鈧?
function updatePagination() {
    const maxPage = Math.max(1, Math.ceil(currentTotal / currentLimit));
    document.getElementById('page-info').textContent = `第 ${currentPage} / ${maxPage} 页`;
    
    // 缁備胶鏁?閸氼垳鏁ら幐澶愭尦
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

// 閸旂姾娴囬柊宥囩枂
async function loadConfig() {
    try {
        const response = await fetch('/api/config');
        const config = await response.json();
        
        // Processor 闁板秶鐤?- 娴ｈ法鏁ゅ鎴濇健
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
        
        // 閺囧瓨鏌婂鎴濇健閺勫墽銇氶崐?
        updateSliderValue('processor-workers', workers);
        updateSliderValue('processor-batch-size', batchSize);
        updateSliderValue('processor-timeout', timeout);
        
        // 濡偓濞村鑻熸惔鏃傛暏閸栧綊鍘ら惃鍕暕鐠?
        detectAndApplyPreset(workers, batchSize, timeout);
        
        // Receiver 闁板秶鐤?
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
        
        // Storage 闁板秶鐤?
        const dbPath = config.storage?.db_path || './data/logs.db';
        const dbPathInput = document.getElementById('storage-db-path');
        if (dbPathInput) dbPathInput.value = dbPath;
        const pathText = document.getElementById('storage-path-text');
        if (pathText) {
            pathText.textContent = dbPath;
        }
        
        // 閺囧瓨鏌婃穱婵堟殌閺冨爼妫块獮璺烘倱濮濄儲瀵滈柦顔惧Ц閹?
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

// 閺嶈宓佽ぐ鎾冲閸婂吋顥呭ù瀣嫙鎼存梻鏁ゆ０鍕啎
function detectAndApplyPreset(workers, batchSize, timeout) {
    // 閺屻儲澹橀崠褰掑帳閻ㄥ嫰顣╃拋?
    let matchedPreset = null;
    for (const [name, preset] of Object.entries(PERFORMANCE_PRESETS)) {
        if (preset.workers === workers && preset.batchSize === batchSize && preset.timeout === timeout) {
            matchedPreset = name;
            break;
        }
    }
    
    // 閺囧瓨鏌婃０鍕啎閸楋紕澧栭悩鑸碘偓?
    document.querySelectorAll('.preset-card').forEach(card => {
        card.classList.toggle('active', card.dataset.preset === matchedPreset);
    });
}

// 娣囨繂鐡ㄩ柊宥囩枂
async function saveConfig() {
    // 閸忓牓鐛欑拠浣瑰閺堝鏆熺€涙绶崗?
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
        console.log('[Config] 濮濓絽婀穱婵嗙摠闁板秶鐤?', config);
        const response = await fetch('/api/config', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(config)
        });
        
        console.log('[Config] 閸濆秴绨查悩鑸碘偓?', response.status);
        
        if (response.ok) {
            const result = await response.json();
            console.log('[Config] 娣囨繂鐡ㄩ幋鎰:', result);
            alert('配置保存成功');
        } else {
            const result = await response.json().catch(() => ({ error: '未知错误' }));
            console.error('[Config] 娣囨繂鐡ㄦ径杈Е:', result);
            alert('保存失败: ' + (result.error || '服务器错误'));
        }
    } catch (error) {
        console.error('[Config] 鐠囬攱鐪板鍌氱埗:', error);
        alert('保存失败: ' + error.message);
    }
}

// 閻愮懓鍤鍦崶婢舵牠鍎撮崗鎶芥４
window.onclick = function(event) {
    const modal = document.getElementById('log-modal');
    if (event.target === modal) {
        closeModal();
    }
};

// 鐎规碍妞傞崚閿嬫煀娴狀亣銆冮弶?
setInterval(() => {
    if (currentTab === 'dashboard') {
        console.log('[App] Auto-refreshing dashboard...');
        loadDashboard();
    }
}, 30000);

// 妞ょ敻娼伴崣顖濐潌閹冨綁閸栨牗妞傞崚閿嬫煀
document.addEventListener('visibilitychange', () => {
    if (!document.hidden && currentTab === 'dashboard') {
        console.log('[App] Page visible, refreshing dashboard...');
        loadDashboard();
    }
});

// ========== 閺傛澘顤冮柊宥囩枂闂堛垺婢橀崝鐔诲厴 ==========

// 閹嗗厴妫板嫯顔曢柊宥囩枂
const PERFORMANCE_PRESETS = {
    dev: { workers: 2, batchSize: 50, timeout: 500 },
    standard: { workers: 10, batchSize: 100, timeout: 1000 },
    high: { workers: 20, batchSize: 200, timeout: 2000 },
    ultra: { workers: 50, batchSize: 500, timeout: 5000 }
};

// 鎼存梻鏁ら幀褑鍏樻０鍕啎
function applyPreset(presetName) {
    const preset = PERFORMANCE_PRESETS[presetName];
    if (!preset) return;
    
    // 閺囧瓨鏌婂鎴濇健閸?
    document.getElementById('processor-workers').value = preset.workers;
    document.getElementById('processor-batch-size').value = preset.batchSize;
    document.getElementById('processor-timeout').value = preset.timeout;
    
    // 閺囧瓨鏌婇弰鍓с仛閸?
    updateSliderValue('processor-workers', preset.workers);
    updateSliderValue('processor-batch-size', preset.batchSize);
    updateSliderValue('processor-timeout', preset.timeout);
    
    // 閺囧瓨鏌婃０鍕啎閸楋紕澧栭悩鑸碘偓?
    document.querySelectorAll('.preset-card').forEach(card => {
        card.classList.toggle('active', card.dataset.preset === presetName);
    });
}

// 閺囧瓨鏌婂鎴濇健閺勫墽銇氶崐?
function updateSliderValue(id, value) {
    const badge = document.getElementById(id + '-value');
    if (badge) {
        badge.textContent = value;
    }
}

// 娑撯偓闁款喖甯囧ù瀣嫙閺勫墽銇氶幎銉ユ啞
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

// 閸旂姾娴囩€涙ê鍋嶆穱鈩冧紖
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

// 鐎涙濡弽鐓庣础閸?
function formatBytes(bytes) {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
}




