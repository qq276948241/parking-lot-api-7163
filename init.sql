-- 停车场管理系统数据库初始化脚本
-- 创建数据库
CREATE DATABASE IF NOT EXISTS parking_system DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

USE parking_system;

-- 用户表
CREATE TABLE IF NOT EXISTS users (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    username VARCHAR(50) NOT NULL UNIQUE COMMENT '用户名',
    password VARCHAR(255) NOT NULL COMMENT '密码(BCrypt加密)',
    real_name VARCHAR(50) COMMENT '真实姓名',
    role VARCHAR(20) NOT NULL DEFAULT 'operator' COMMENT '角色: admin/operator',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_username (username),
    INDEX idx_role (role)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户表';

-- 车位表
CREATE TABLE IF NOT EXISTS parking_spaces (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    space_no VARCHAR(20) NOT NULL UNIQUE COMMENT '车位编号',
    status VARCHAR(20) NOT NULL DEFAULT 'free' COMMENT '状态: free/occupied',
    plate_no VARCHAR(20) COMMENT '当前停放车牌号',
    record_id BIGINT UNSIGNED COMMENT '关联停车记录ID',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_status (status),
    INDEX idx_plate_no (plate_no)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='车位表';

-- 停车记录表
CREATE TABLE IF NOT EXISTS parking_records (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    plate_no VARCHAR(20) NOT NULL COMMENT '车牌号',
    space_no VARCHAR(20) COMMENT '车位编号',
    entry_time DATETIME NOT NULL COMMENT '进场时间',
    exit_time DATETIME COMMENT '出场时间',
    duration INT COMMENT '停车时长(分钟)',
    fee DECIMAL(10,2) DEFAULT 0 COMMENT '停车费用',
    is_monthly TINYINT(1) DEFAULT 0 COMMENT '是否月卡车辆',
    operator VARCHAR(50) COMMENT '进场操作员',
    exit_operator VARCHAR(50) COMMENT '出场操作员',
    status VARCHAR(20) NOT NULL DEFAULT 'parking' COMMENT '状态: parking/exited',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_plate_no (plate_no),
    INDEX idx_status (status),
    INDEX idx_entry_time (entry_time),
    INDEX idx_exit_time (exit_time)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='停车记录表';

-- 月卡表
CREATE TABLE IF NOT EXISTS monthly_cards (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    plate_no VARCHAR(20) NOT NULL UNIQUE COMMENT '车牌号',
    owner_name VARCHAR(50) COMMENT '车主姓名',
    phone VARCHAR(20) COMMENT '联系电话',
    start_date DATETIME NOT NULL COMMENT '生效日期',
    end_date DATETIME NOT NULL COMMENT '到期日期',
    price DECIMAL(10,2) NOT NULL COMMENT '总金额',
    status VARCHAR(20) NOT NULL DEFAULT 'active' COMMENT '状态: active/expired',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_plate_no (plate_no),
    INDEX idx_status (status),
    INDEX idx_end_date (end_date)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='月卡表';

-- 收费账单表
CREATE TABLE IF NOT EXISTS bills (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    record_id BIGINT UNSIGNED NOT NULL COMMENT '停车记录ID',
    plate_no VARCHAR(20) NOT NULL COMMENT '车牌号',
    amount DECIMAL(10,2) NOT NULL COMMENT '收费金额',
    pay_type VARCHAR(20) NOT NULL DEFAULT 'cash' COMMENT '支付方式: cash/wechat/alipay',
    pay_time DATETIME NOT NULL COMMENT '支付时间',
    is_monthly TINYINT(1) DEFAULT 0 COMMENT '是否月卡相关',
    operator VARCHAR(50) COMMENT '操作员',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_record_id (record_id),
    INDEX idx_plate_no (plate_no),
    INDEX idx_pay_time (pay_time)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='收费账单表';

-- 插入默认管理员 (密码: admin123, BCrypt哈希)
INSERT INTO users (username, password, real_name, role) VALUES
('admin', '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', '超级管理员', 'admin')
ON DUPLICATE KEY UPDATE username=username;

-- 插入示例操作员 (密码: operator123)
INSERT INTO users (username, password, real_name, role) VALUES
('operator01', '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', '岗亭操作员01', 'operator')
ON DUPLICATE KEY UPDATE username=username;
