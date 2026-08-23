-- MySQL 数据库初始化脚本
-- 创建数据库和用户

-- 1. 创建数据库
CREATE DATABASE IF NOT EXISTS log_processor DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- 2. 创建用户（请修改密码）
CREATE USER IF NOT EXISTS 'log_processor'@'localhost' IDENTIFIED BY 'your_password';
CREATE USER IF NOT EXISTS 'log_processor'@'%' IDENTIFIED BY 'your_password';

-- 3. 授权
GRANT ALL PRIVILEGES ON log_processor.* TO 'log_processor'@'localhost';
GRANT ALL PRIVILEGES ON log_processor.* TO 'log_processor'@'%';
FLUSH PRIVILEGES;

-- 4. 使用数据库
USE log_processor;

-- 5. 创建日志表
CREATE TABLE IF NOT EXISTS logs (
    id VARCHAR(36) PRIMARY KEY COMMENT '日志ID',
    timestamp DATETIME(3) NOT NULL COMMENT '日志时间',
    
    -- 项目维度
    project_id VARCHAR(64) DEFAULT '' COMMENT '项目ID',
    project_name VARCHAR(128) DEFAULT '' COMMENT '项目名称',
    environment VARCHAR(32) DEFAULT 'prod' COMMENT '环境',
    service_name VARCHAR(128) DEFAULT '' COMMENT '服务名称',
    
    -- 日志分类
    source VARCHAR(64) DEFAULT '' COMMENT '日志来源',
    level VARCHAR(16) DEFAULT 'INFO' COMMENT '日志级别',
    log_type VARCHAR(32) DEFAULT '' COMMENT '日志类型',
    
    -- 请求信息
    method VARCHAR(16) DEFAULT '' COMMENT 'HTTP方法',
    path TEXT COMMENT '请求路径',
    status_code INT DEFAULT 0 COMMENT 'HTTP状态码',
    response_time BIGINT DEFAULT 0 COMMENT '响应时间(ms)',
    client_ip VARCHAR(64) DEFAULT '' COMMENT '客户端IP',
    user_agent TEXT COMMENT 'User-Agent',
    referer TEXT COMMENT 'Referer',
    request_size BIGINT DEFAULT 0 COMMENT '请求大小',
    response_size BIGINT DEFAULT 0 COMMENT '响应大小',
    
    -- 错误追踪
    error_message TEXT COMMENT '错误信息',
    error_code VARCHAR(64) DEFAULT '' COMMENT '错误码',
    stack_trace TEXT COMMENT '堆栈跟踪',
    
    -- 上下文信息
    request_id VARCHAR(64) DEFAULT '' COMMENT '请求ID',
    user_id VARCHAR(64) DEFAULT '' COMMENT '用户ID',
    session_id VARCHAR(64) DEFAULT '' COMMENT '会话ID',
    
    -- AI 分析
    ai_analyzed BOOLEAN DEFAULT FALSE COMMENT '是否已AI分析',
    ai_analysis TEXT COMMENT 'AI分析结果',
    ai_suggestions TEXT COMMENT 'AI建议',
    ai_analyzed_at DATETIME(3) NULL COMMENT 'AI分析时间',
    
    -- 其他
    extra_fields JSON COMMENT '额外字段',
    raw_data TEXT COMMENT '原始日志',
    created_at DATETIME(3) DEFAULT CURRENT_TIMESTAMP(3) COMMENT '创建时间',
    
    INDEX idx_timestamp (timestamp),
    INDEX idx_project_env (project_id, environment),
    INDEX idx_status_code (status_code),
    INDEX idx_level (level),
    INDEX idx_log_type (log_type),
    INDEX idx_method (method),
    INDEX idx_client_ip (client_ip),
    INDEX idx_ai_analyzed (ai_analyzed, level),
    INDEX idx_error (level, ai_analyzed) COMMENT '错误日志查询',
    INDEX idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='日志表';

-- 6. 创建项目表（可选，用于多项目管理）
CREATE TABLE IF NOT EXISTS projects (
    id VARCHAR(64) PRIMARY KEY COMMENT '项目ID',
    name VARCHAR(128) NOT NULL COMMENT '项目名称',
    description TEXT COMMENT '项目描述',
    api_token VARCHAR(128) DEFAULT '' COMMENT 'API Token',
    environments JSON COMMENT '环境列表',
    log_types JSON COMMENT '日志类型',
    created_at DATETIME(3) DEFAULT CURRENT_TIMESTAMP(3) COMMENT '创建时间',
    updated_at DATETIME(3) DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3) COMMENT '更新时间',
    
    INDEX idx_name (name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='项目表';

-- 7. 创建问题工单表（可选，用于问题跟踪）
CREATE TABLE IF NOT EXISTS issues (
    id VARCHAR(36) PRIMARY KEY COMMENT '工单ID',
    project_id VARCHAR(64) NOT NULL COMMENT '项目ID',
    title VARCHAR(256) NOT NULL COMMENT '标题',
    description TEXT COMMENT '描述',
    severity VARCHAR(16) DEFAULT 'medium' COMMENT '严重程度',
    status VARCHAR(16) DEFAULT 'open' COMMENT '状态',
    assigned_to VARCHAR(64) DEFAULT '' COMMENT '负责人',
    created_by VARCHAR(64) DEFAULT '' COMMENT '创建人',
    log_ids JSON COMMENT '关联日志ID列表',
    ai_analysis TEXT COMMENT 'AI分析',
    root_cause TEXT COMMENT '根本原因',
    suggestions JSON COMMENT '修复建议',
    created_at DATETIME(3) DEFAULT CURRENT_TIMESTAMP(3) COMMENT '创建时间',
    resolved_at DATETIME(3) NULL COMMENT '解决时间',
    updated_at DATETIME(3) DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3) COMMENT '更新时间',
    
    INDEX idx_project_id (project_id),
    INDEX idx_status (status),
    INDEX idx_severity (severity),
    INDEX idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='问题工单表';

-- 8. 显示表结构
SHOW TABLES;
DESCRIBE logs;
