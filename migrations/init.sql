-- Sisyphus 数据库智能平台 - 数据库初始化脚本
-- 创建数据库
CREATE DATABASE IF NOT EXISTS `sisyphus` DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

USE `sisyphus`;

-- 用户表
CREATE TABLE IF NOT EXISTS `sisyphus_users` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  `username` VARCHAR(100) NOT NULL UNIQUE COMMENT '用户名',
  `password` VARCHAR(255) NOT NULL COMMENT '密码(加密)',
  `email` VARCHAR(100) NOT NULL UNIQUE COMMENT '邮箱',
  `real_name` VARCHAR(100) NOT NULL COMMENT '真实姓名',
  `phone` VARCHAR(20) DEFAULT '' COMMENT '手机号',
  `department` VARCHAR(100) DEFAULT '' COMMENT '部门',
  `status` TINYINT NOT NULL DEFAULT 1 COMMENT '状态: 1-正常, 0-禁用',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX `idx_username` (`username`),
  INDEX `idx_email` (`email`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户表';

-- 角色表
CREATE TABLE IF NOT EXISTS `sisyphus_roles` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  `name` VARCHAR(50) NOT NULL UNIQUE COMMENT '角色名称',
  `code` VARCHAR(50) NOT NULL UNIQUE COMMENT '角色代码',
  `description` VARCHAR(255) DEFAULT '' COMMENT '角色描述',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='角色表';

-- 权限表
CREATE TABLE IF NOT EXISTS `sisyphus_permissions` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  `name` VARCHAR(100) NOT NULL COMMENT '权限名称',
  `code` VARCHAR(100) NOT NULL UNIQUE COMMENT '权限代码',
  `resource` VARCHAR(100) NOT NULL COMMENT '资源类型: sql_order, sql_query, binlog, redis, archive等',
  `action` VARCHAR(50) NOT NULL COMMENT '操作: create, read, update, delete, execute等',
  `description` VARCHAR(255) DEFAULT '' COMMENT '权限描述',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX `idx_resource_action` (`resource`, `action`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='权限表';

-- 角色权限关联表
CREATE TABLE IF NOT EXISTS `sisyphus_role_permissions` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  `role_id` BIGINT UNSIGNED NOT NULL,
  `permission_id` BIGINT UNSIGNED NOT NULL,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY `uk_role_permission` (`role_id`, `permission_id`),
  FOREIGN KEY (`role_id`) REFERENCES `sisyphus_roles`(`id`) ON DELETE CASCADE,
  FOREIGN KEY (`permission_id`) REFERENCES `sisyphus_permissions`(`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='角色权限关联表';

-- 用户角色关联表
CREATE TABLE IF NOT EXISTS `sisyphus_user_roles` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  `user_id` BIGINT UNSIGNED NOT NULL,
  `role_id` BIGINT UNSIGNED NOT NULL,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY `uk_user_role` (`user_id`, `role_id`),
  FOREIGN KEY (`user_id`) REFERENCES `sisyphus_users`(`id`) ON DELETE CASCADE,
  FOREIGN KEY (`role_id`) REFERENCES `sisyphus_roles`(`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户角色关联表';

-- 用户实例权限表(细粒度权限控制)
CREATE TABLE IF NOT EXISTS `sisyphus_user_instance_permissions` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  `user_id` BIGINT UNSIGNED NOT NULL,
  `instance_id` BIGINT UNSIGNED NOT NULL,
  `permission_type` VARCHAR(50) NOT NULL COMMENT '权限类型: ddl, dml, query, export',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY `uk_user_instance_permission` (`user_id`, `instance_id`, `permission_type`),
  INDEX `idx_user` (`user_id`),
  INDEX `idx_instance` (`instance_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户实例权限表';

-- 角色实例权限表（与 user 直授权合并后生效：角色 ∪ 用户）
CREATE TABLE IF NOT EXISTS `sisyphus_role_instance_permissions` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  `role_id` BIGINT UNSIGNED NOT NULL,
  `instance_id` BIGINT UNSIGNED NOT NULL,
  `permission_type` VARCHAR(50) NOT NULL COMMENT '权限类型: ddl, dml, query, export',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY `uk_role_instance_permission` (`role_id`, `instance_id`, `permission_type`),
  INDEX `idx_role` (`role_id`),
  INDEX `idx_instance` (`instance_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='角色实例权限表';

-- 数据库实例表
CREATE TABLE IF NOT EXISTS `sisyphus_database_instances` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  `name` VARCHAR(100) NOT NULL UNIQUE COMMENT '实例名称',
  `db_type` VARCHAR(32) NOT NULL COMMENT '数据库类型: mysql, redis, clickhouse等',
  `host` VARCHAR(512) NOT NULL COMMENT '主机地址',
  `port` INT NOT NULL COMMENT '端口',
  `username` VARCHAR(150) NOT NULL COMMENT '用户名',
  `password` VARCHAR(255) NOT NULL COMMENT '密码(加密存储)',
  `env` VARCHAR(50) NOT NULL COMMENT '环境: dev, test, prod',
  `region` VARCHAR(100) DEFAULT '' COMMENT '地区',
  `status` TINYINT NOT NULL DEFAULT 1 COMMENT '状态: 1-正常, 0-下线',
  `owner` VARCHAR(100) DEFAULT '' COMMENT '负责人',
  `description` VARCHAR(500) DEFAULT '' COMMENT '描述',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX `idx_name` (`name`),
  INDEX `idx_db_type` (`db_type`),
  INDEX `idx_env` (`env`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='数据库实例表';

-- SQL工单表
CREATE TABLE IF NOT EXISTS `sisyphus_sql_orders` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  `work_id` VARCHAR(50) NOT NULL UNIQUE COMMENT '工单ID',
  `instance_id` BIGINT UNSIGNED NOT NULL COMMENT '数据库实例ID',
  `database_name` VARCHAR(100) NOT NULL COMMENT '数据库名',
  `order_type` TINYINT NOT NULL COMMENT '工单类型: 0-DDL, 1-DML, 2-EXPORT',
  `status` TINYINT NOT NULL DEFAULT 0 COMMENT '状态: 0-待审核, 1-已通过, 2-已驳回, 3-执行中, 4-执行成功, 5-执行失败, 10-等待复核',
  `sql_content` TEXT NOT NULL COMMENT 'SQL内容',
  `remark` TEXT COMMENT '备注',
  `submitter_id` BIGINT UNSIGNED NOT NULL COMMENT '提交人ID',
  `auditor_id` BIGINT UNSIGNED DEFAULT NULL COMMENT '审核人ID',
  `executor_id` BIGINT UNSIGNED DEFAULT NULL COMMENT '执行人ID',
  `reject_reason` TEXT COMMENT '驳回原因',
  `delay_minutes` INT NOT NULL DEFAULT 0 COMMENT '延迟执行时间(分钟)',
  `need_backup` TINYINT NOT NULL DEFAULT 1 COMMENT '是否需要备份: 1-是, 0-否',
  `execute_method` TINYINT NOT NULL DEFAULT 0 COMMENT '执行方式: 0-native, 1-gh-ost, 2-goInception',
  `risk_level` TINYINT NOT NULL DEFAULT 0 COMMENT '风险等级: 0-正常, 1-危险',
  `execute_time` DATETIME DEFAULT NULL COMMENT '执行时间',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX `idx_work_id` (`work_id`),
  INDEX `idx_instance` (`instance_id`),
  INDEX `idx_submitter` (`submitter_id`),
  INDEX `idx_status` (`status`),
  INDEX `idx_created_at` (`created_at`),
  FOREIGN KEY (`instance_id`) REFERENCES `sisyphus_database_instances`(`id`),
  FOREIGN KEY (`submitter_id`) REFERENCES `sisyphus_users`(`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='SQL工单表';

-- SQL工单明细表(拆分后的单条SQL)
CREATE TABLE IF NOT EXISTS `sisyphus_sql_order_items` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  `order_id` BIGINT UNSIGNED NOT NULL COMMENT '工单ID',
  `sequence` INT NOT NULL COMMENT '序号',
  `sql_text` TEXT NOT NULL COMMENT 'SQL文本',
  `status` TINYINT NOT NULL DEFAULT 0 COMMENT '状态: 0-待执行, 1-执行中, 2-执行成功, 3-执行失败',
  `execute_type` VARCHAR(50) DEFAULT '' COMMENT '执行方式',
  `execute_time` DECIMAL(10,2) DEFAULT 0 COMMENT '执行耗时(秒)',
  `affected_rows` INT DEFAULT 0 COMMENT '影响行数',
  `result_text` TEXT COMMENT '执行结果',
  `error_message` TEXT COMMENT '错误信息',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX `idx_order` (`order_id`),
  FOREIGN KEY (`order_id`) REFERENCES `sisyphus_sql_orders`(`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='SQL工单明细表';

-- SQL审核记录表(goInception审核结果)
CREATE TABLE IF NOT EXISTS `sisyphus_sql_review_records` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  `order_id` BIGINT UNSIGNED NOT NULL,
  `item_id` BIGINT UNSIGNED DEFAULT NULL,
  `sequence` VARCHAR(50) DEFAULT NULL COMMENT 'goInception返回的sequence',
  `stage` VARCHAR(20) DEFAULT NULL COMMENT '阶段: CHECKED, EXECUTED',
  `error_level` TINYINT DEFAULT NULL COMMENT '错误级别: 0-正常, 1-警告, 2-错误',
  `stage_status` VARCHAR(20) DEFAULT NULL COMMENT '阶段状态',
  `error_message` TEXT COMMENT '错误信息',
  `sql_text` TEXT COMMENT 'SQL文本',
  `affected_rows` INT DEFAULT 0 COMMENT '影响行数',
  `backup_dbname` VARCHAR(100) DEFAULT NULL COMMENT '备份库名',
  `execute_time` VARCHAR(50) DEFAULT NULL COMMENT '执行时间',
  `sql_hash` VARCHAR(64) DEFAULT NULL COMMENT 'SQL哈希值',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX `idx_order` (`order_id`),
  INDEX `idx_item` (`item_id`),
  FOREIGN KEY (`order_id`) REFERENCES `sisyphus_sql_orders`(`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='SQL审核记录表';

-- Binlog回滚工单表
CREATE TABLE IF NOT EXISTS `sisyphus_binlog_rollback_orders` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  `work_id` VARCHAR(50) NOT NULL UNIQUE COMMENT '工单ID',
  `instance_id` BIGINT UNSIGNED NOT NULL,
  `database_name` VARCHAR(100) NOT NULL,
  `start_time` DATETIME NOT NULL COMMENT '开始时间',
  `stop_time` DATETIME NOT NULL COMMENT '结束时间',
  `binlog_file` VARCHAR(255) DEFAULT '' COMMENT 'binlog文件',
  `table_list` TEXT COMMENT '表列表(JSON)',
  `status` TINYINT NOT NULL DEFAULT 0 COMMENT '状态: 0-解析中, 1-解析成功, 2-解析失败',
  `rollback_sql_count` INT DEFAULT 0 COMMENT '回滚SQL数量',
  `submitter_id` BIGINT UNSIGNED NOT NULL,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX `idx_work_id` (`work_id`),
  INDEX `idx_instance` (`instance_id`),
  FOREIGN KEY (`instance_id`) REFERENCES `sisyphus_database_instances`(`id`),
  FOREIGN KEY (`submitter_id`) REFERENCES `sisyphus_users`(`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Binlog回滚工单表';

-- Binlog回滚SQL表
CREATE TABLE IF NOT EXISTS `sisyphus_binlog_rollback_sqls` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  `order_id` BIGINT UNSIGNED NOT NULL,
  `table_name` VARCHAR(100) NOT NULL,
  `rollback_sql` TEXT NOT NULL COMMENT '回滚SQL',
  `original_sql` TEXT COMMENT '原始SQL',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX `idx_order` (`order_id`),
  FOREIGN KEY (`order_id`) REFERENCES `sisyphus_binlog_rollback_orders`(`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Binlog回滚SQL表';

-- Redis热key分析任务表
CREATE TABLE IF NOT EXISTS `sisyphus_redis_hotkey_tasks` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  `instance_id` BIGINT UNSIGNED NOT NULL,
  `task_type` VARCHAR(50) NOT NULL COMMENT '任务类型: hotkey, bigkey等',
  `status` TINYINT NOT NULL DEFAULT 0 COMMENT '状态: 0-待执行, 1-执行中, 2-完成, 3-失败',
  `result` JSON COMMENT '分析结果',
  `submitter_id` BIGINT UNSIGNED NOT NULL,
  `started_at` DATETIME DEFAULT NULL,
  `finished_at` DATETIME DEFAULT NULL,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX `idx_instance` (`instance_id`),
  INDEX `idx_status` (`status`),
  FOREIGN KEY (`instance_id`) REFERENCES `sisyphus_database_instances`(`id`),
  FOREIGN KEY (`submitter_id`) REFERENCES `sisyphus_users`(`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Redis热key分析任务表';

-- 数据归档配置表
CREATE TABLE IF NOT EXISTS `sisyphus_archive_configs` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  `instance_id` BIGINT UNSIGNED NOT NULL,
  `source_database` VARCHAR(100) NOT NULL,
  `source_table` VARCHAR(100) NOT NULL,
  `dest_instance_id` BIGINT UNSIGNED DEFAULT NULL,
  `dest_database` VARCHAR(100) DEFAULT NULL,
  `dest_table` VARCHAR(100) DEFAULT NULL,
  `archive_field` VARCHAR(100) NOT NULL COMMENT '归档字段',
  `retention_days` INT NOT NULL COMMENT '保留天数',
  `archive_condition` VARCHAR(500) DEFAULT '' COMMENT '归档条件',
  `is_enabled` TINYINT NOT NULL DEFAULT 1 COMMENT '是否启用: 1-是, 0-否',
  `cron_expression` VARCHAR(100) NOT NULL COMMENT '定时表达式',
  `owner` VARCHAR(100) DEFAULT '' COMMENT '负责人',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX `idx_instance` (`instance_id`),
  INDEX `idx_enabled` (`is_enabled`),
  FOREIGN KEY (`instance_id`) REFERENCES `sisyphus_database_instances`(`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='数据归档配置表';

-- 数据归档执行记录表
CREATE TABLE IF NOT EXISTS `sisyphus_archive_records` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  `config_id` BIGINT UNSIGNED NOT NULL,
  `work_id` VARCHAR(50) NOT NULL,
  `status` TINYINT NOT NULL DEFAULT 0 COMMENT '状态: 0-待执行, 1-执行中, 2-成功, 3-失败',
  `archive_count` INT DEFAULT 0 COMMENT '归档数量',
  `execute_time` DATETIME DEFAULT NULL,
  `finished_time` DATETIME DEFAULT NULL,
  `error_message` TEXT COMMENT '错误信息',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX `idx_config` (`config_id`),
  INDEX `idx_work_id` (`work_id`),
  INDEX `idx_status` (`status`),
  FOREIGN KEY (`config_id`) REFERENCES `sisyphus_archive_configs`(`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='数据归档执行记录表';

-- 查询历史记录表
CREATE TABLE IF NOT EXISTS `sisyphus_query_history` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  `user_id` BIGINT UNSIGNED NOT NULL,
  `instance_id` BIGINT UNSIGNED NOT NULL,
  `database_name` VARCHAR(100) NOT NULL,
  `sql_text` TEXT NOT NULL,
  `execute_time` DECIMAL(10,2) DEFAULT 0 COMMENT '执行耗时(秒)',
  `affected_rows` INT DEFAULT 0,
  `result_count` INT DEFAULT 0 COMMENT '结果数量',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX `idx_user` (`user_id`),
  INDEX `idx_instance` (`instance_id`),
  INDEX `idx_created_at` (`created_at`),
  FOREIGN KEY (`user_id`) REFERENCES `sisyphus_users`(`id`),
  FOREIGN KEY (`instance_id`) REFERENCES `sisyphus_database_instances`(`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='查询历史记录表';

-- 分区表信息表
CREATE TABLE IF NOT EXISTS `sisyphus_partition_table_info` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  `instance_id` BIGINT UNSIGNED NOT NULL,
  `instance_name` VARCHAR(100) NOT NULL COMMENT '实例名称',
  `connection_string` VARCHAR(512) NOT NULL COMMENT '连接地址',
  `database_name` VARCHAR(100) NOT NULL COMMENT '数据库名',
  `table_name` VARCHAR(100) NOT NULL COMMENT '表名',
  `part_type` VARCHAR(32) DEFAULT '' COMMENT '分区类型: RANGE, HASH等',
  `part_key` VARCHAR(255) DEFAULT '' COMMENT '分区键表达式',
  `part_cycle` VARCHAR(10) DEFAULT '' COMMENT '分区周期: 天, 周, 月, 季, 半年, 年',
  `part_num` INT DEFAULT 0 COMMENT '分区数量',
  `part_maxvalue` VARCHAR(128) DEFAULT '' COMMENT '最大分区值',
  `part_maxvalue_day` VARCHAR(16) DEFAULT '未到期' COMMENT '最大分区距离当前日期的天数状态: 已到期, 未到期',
  `have_maxvalue` VARCHAR(8) DEFAULT '否' COMMENT '是否有MAXVALUE分区: 是, 否',
  `table_size` DECIMAL(12,2) DEFAULT 0 COMMENT '表大小(GB)',
  `table_rows` BIGINT DEFAULT 0 COMMENT '总行数',
  `maxpart_rows` BIGINT DEFAULT 0 COMMENT '最大分区行数',
  `is_crontab` VARCHAR(8) DEFAULT '否' COMMENT '是否定时任务维护: 是, 否',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY `uk_instance_table` (`instance_id`, `database_name`, `table_name`),
  INDEX `idx_expire` (`part_maxvalue_day`),
  FOREIGN KEY (`instance_id`) REFERENCES `sisyphus_database_instances`(`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='分区表信息表';

-- 分区操作记录表
CREATE TABLE IF NOT EXISTS `sisyphus_partition_operation_records` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  `work_id` VARCHAR(50) NOT NULL COMMENT '工单ID',
  `instance_id` BIGINT UNSIGNED NOT NULL,
  `instance_name` VARCHAR(100) NOT NULL COMMENT '实例名称',
  `connection_string` VARCHAR(512) NOT NULL COMMENT '连接地址',
  `table_schema` VARCHAR(100) NOT NULL COMMENT '数据库名',
  `table_name` VARCHAR(100) NOT NULL COMMENT '表名',
  `operate_type` TINYINT NOT NULL COMMENT '操作类型: 1-初始化, 2-添加分区, 3-数据归档, 4-数据校验, 5-分区删除, 6-数据删除',
  `exe_status` TINYINT NOT NULL DEFAULT 1 COMMENT '执行状态: 1-创建, 2-取消, 3-执行中, 4-成功, 5-失败',
  `check_status` TINYINT NOT NULL DEFAULT 0 COMMENT '确认状态: 0-未确认, 1-确认',
  `exe_pct` VARCHAR(10) DEFAULT '' COMMENT '执行进度',
  `exe_result` TEXT COMMENT '执行结果',
  `exe_cmd` VARCHAR(1000) DEFAULT '' COMMENT '执行命令',
  `where_condition` VARCHAR(500) DEFAULT '' COMMENT '归档条件',
  `add_user` VARCHAR(50) DEFAULT '' COMMENT '添加用户',
  `update_user` VARCHAR(50) DEFAULT '' COMMENT '更新用户',
  `exe_time` DATETIME DEFAULT NULL COMMENT '执行时间',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX `idx_work_id` (`work_id`),
  INDEX `idx_instance` (`instance_id`),
  INDEX `idx_operate_type` (`operate_type`),
  INDEX `idx_status` (`exe_status`),
  INDEX `idx_created_at` (`created_at`),
  FOREIGN KEY (`instance_id`) REFERENCES `sisyphus_database_instances`(`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='分区操作记录表';

-- 初始化默认角色
INSERT INTO `sisyphus_roles` (`name`, `code`, `description`) VALUES
('超级管理员', 'admin', '拥有所有权限'),
('DBA', 'dba', '数据库管理员'),
('开发人员', 'developer', '普通开发人员'),
('只读用户', 'guest', '只读权限');

-- 初始化默认权限
INSERT INTO `sisyphus_permissions` (`name`, `code`, `resource`, `action`, `description`) VALUES
-- SQL工单权限
('创建SQL工单', 'sql_order:create', 'sql_order', 'create', '创建SQL工单'),
('审核SQL工单', 'sql_order:audit', 'sql_order', 'audit', '审核SQL工单'),
('执行SQL工单', 'sql_order:execute', 'sql_order', 'execute', '执行SQL工单'),
('查看SQL工单', 'sql_order:read', 'sql_order', 'read', '查看SQL工单'),
-- 数据查询权限
('执行查询', 'sql_query:execute', 'sql_query', 'execute', '执行SQL查询'),
('导出数据', 'sql_query:export', 'sql_query', 'export', '导出查询结果'),
-- Binlog权限
('创建回滚工单', 'binlog:create', 'binlog', 'create', '创建binlog回滚工单'),
('查看回滚工单', 'binlog:read', 'binlog', 'read', '查看binlog回滚工单'),
-- Redis权限
('热key分析', 'redis:hotkey', 'redis', 'hotkey', 'Redis热key分析'),
-- 归档权限
('配置归档', 'archive:config', 'archive', 'config', '配置数据归档'),
('查看归档', 'archive:read', 'archive', 'read', '查看归档记录'),
-- 用户管理权限
('用户管理', 'user:manage', 'user', 'manage', '用户管理'),
('角色管理', 'role:manage', 'role', 'manage', '角色管理'),
('实例管理', 'instance:manage', 'instance', 'manage', '实例管理');

-- 为角色分配权限（admin 拥有全部，dba 拥有审核/执行/查询/实例/归档/binlog/redis，developer 拥有提交/查询/查看）
-- admin: 所有权限
INSERT INTO `sisyphus_role_permissions` (`role_id`, `permission_id`)
SELECT r.id, p.id
FROM `sisyphus_roles` r
CROSS JOIN `sisyphus_permissions` p
WHERE r.code = 'admin';

-- dba: 工单审核/执行/查看、查询/导出、binlog、redis、归档、用户管理、角色管理、实例管理
INSERT INTO `sisyphus_role_permissions` (`role_id`, `permission_id`)
SELECT r.id, p.id
FROM `sisyphus_roles` r
JOIN `sisyphus_permissions` p ON p.code IN (
  'sql_order:audit', 'sql_order:execute', 'sql_order:read', 'sql_order:create',
  'sql_query:execute', 'sql_query:export',
  'binlog:create', 'binlog:read',
  'redis:hotkey',
  'archive:config', 'archive:read',
  'instance:manage'
)
WHERE r.code = 'dba';

-- developer: 创建/查看工单、查询、binlog查看、归档查看
INSERT INTO `sisyphus_role_permissions` (`role_id`, `permission_id`)
SELECT r.id, p.id
FROM `sisyphus_roles` r
JOIN `sisyphus_permissions` p ON p.code IN (
  'sql_order:create', 'sql_order:read',
  'sql_query:execute',
  'binlog:read',
  'archive:read'
)
WHERE r.code = 'developer';

-- guest: 只读
INSERT INTO `sisyphus_role_permissions` (`role_id`, `permission_id`)
SELECT r.id, p.id
FROM `sisyphus_roles` r
JOIN `sisyphus_permissions` p ON p.code IN (
  'sql_order:read',
  'binlog:read',
  'archive:read'
)
WHERE r.code = 'guest';

-- 查询收藏表
CREATE TABLE IF NOT EXISTS `sisyphus_query_favorites` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint unsigned NOT NULL COMMENT '用户ID',
  `name` varchar(100) NOT NULL COMMENT '收藏名称',
  `sql_text` text NOT NULL COMMENT 'SQL内容',
  `instance_id` bigint unsigned NOT NULL DEFAULT 0 COMMENT '实例ID',
  `database_name` varchar(100) NOT NULL DEFAULT '' COMMENT '数据库名',
  `db_type` varchar(20) NOT NULL DEFAULT 'mysql' COMMENT '数据库类型',
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  KEY `idx_user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='查询收藏';

-- DML回滚SQL表
CREATE TABLE IF NOT EXISTS `sisyphus_sql_order_rollback` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `order_id` bigint unsigned NOT NULL COMMENT '工单ID',
  `order_item_id` bigint unsigned DEFAULT NULL COMMENT '工单条目ID',
  `backup_db_name` varchar(100) NOT NULL COMMENT 'goInception备份库名',
  `opid_time` varchar(50) NOT NULL COMMENT 'goInception操作ID',
  `table_name` varchar(100) NOT NULL COMMENT '表名',
  `rollback_sql` text NOT NULL COMMENT '回滚SQL',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_order_id` (`order_id`),
  KEY `idx_order_item_id` (`order_item_id`),
  KEY `idx_opid_time` (`opid_time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='DML回滚SQL记录';

-- 宽表配置表
CREATE TABLE IF NOT EXISTS `sisyphus_widetable_configs` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(128) NOT NULL COMMENT '宽表名称',
  `description` varchar(512) DEFAULT '' COMMENT '描述',
  `source_instance_id` bigint unsigned NOT NULL COMMENT '源实例ID',
  `main_database` varchar(64) NOT NULL COMMENT '主库名',
  `main_table` varchar(64) NOT NULL COMMENT '主表名',
  `join_config` text COMMENT 'JOIN配置(JSON)',
  `field_config` text COMMENT '字段映射配置(JSON)',
  `sync_condition` varchar(512) DEFAULT '' COMMENT '同步WHERE条件',
  `target_instance_id` bigint unsigned DEFAULT NULL COMMENT '目标实例ID',
  `target_database` varchar(64) DEFAULT '' COMMENT '目标库名',
  `target_table` varchar(64) DEFAULT '' COMMENT '目标表名',
  `status` tinyint NOT NULL DEFAULT 0 COMMENT '状态: 0-未同步, 1-同步中, 2-已完成',
  `created_by` bigint unsigned NOT NULL COMMENT '创建人ID',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_source_instance` (`source_instance_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='宽表配置';

-- 宽表同步记录表
CREATE TABLE IF NOT EXISTS `sisyphus_widetable_sync_records` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `config_id` bigint unsigned NOT NULL COMMENT '宽表配置ID',
  `sync_type` tinyint NOT NULL COMMENT '同步类型: 1-全量',
  `status` tinyint NOT NULL DEFAULT 0 COMMENT '状态: 0-待执行, 1-执行中, 2-成功, 3-失败',
  `processed_rows` bigint NOT NULL DEFAULT 0 COMMENT '处理行数',
  `error_message` text COMMENT '错误信息',
  `cost_seconds` decimal(10,2) DEFAULT NULL COMMENT '耗时(秒)',
  `started_at` datetime DEFAULT NULL COMMENT '开始时间',
  `finished_at` datetime DEFAULT NULL COMMENT '完成时间',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_config` (`config_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='宽表同步记录';

-- 通知渠道配置表
CREATE TABLE IF NOT EXISTS `sisyphus_notification_configs` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `channel` varchar(50) NOT NULL COMMENT '通知渠道: wechatwork, feishu, dingtalk',
  `enabled` tinyint(1) NOT NULL DEFAULT 0 COMMENT '是否启用',
  `webhook_url` varchar(500) NOT NULL DEFAULT '' COMMENT 'Webhook地址',
  `extra_config` json DEFAULT NULL COMMENT '额外配置，如@手机号列表等',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_channel` (`channel`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='通知渠道配置';
