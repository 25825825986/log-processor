// 调试脚本 - 在浏览器控制台中运行

// 1. 测试 API 连接
async function testAPI() {
    console.log('=== 测试 API 连接 ===');
    try {
        const response = await fetch('/api/statistics');
        console.log('状态码:', response.status);
        const data = await response.json();
        console.log('返回数据:', data);
        return data;
    } catch (e) {
        console.error('API 错误:', e);
        return null;
    }
}

// 2. 检查 DOM 元素
function checkDOM() {
    console.log('=== 检查 DOM 元素 ===');
    const elements = [
        'total-logs',
        'error-logs', 
        'avg-response',
        'system-status',
        'error-rate',
        'last-update',
        'status-chart',
        'method-chart',
        'trend-chart'
    ];
    
    elements.forEach(id => {
        const el = document.getElementById(id);
        console.log(`${id}: ${el ? '✅ 存在' : '❌ 不存在'}`);
    });
}

// 3. 手动更新测试
function testUpdate() {
    console.log('=== 手动更新测试 ===');
    document.getElementById('total-logs').textContent = '116';
    document.getElementById('error-logs').textContent = '0';
    document.getElementById('avg-response').textContent = '45ms';
    console.log('已手动填入测试数据，观察界面是否更新');
}

// 4. 完整诊断
async function diagnose() {
    console.clear();
    console.log('🔍 开始诊断...\n');
    
    checkDOM();
    console.log('');
    
    const data = await testAPI();
    console.log('');
    
    if (data) {
        console.log('✅ API 正常，尝试手动更新...');
        testUpdate();
    } else {
        console.log('❌ API 连接失败');
    }
}

// 在控制台运行 diagnose()
console.log('调试脚本已加载，运行 diagnose() 开始诊断');
