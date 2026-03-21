-- Sisyphus 测试数据 / 开发环境初始数据
-- 密码均为: Admin@123 (bcrypt cost=10)
USE `sisyphus`;

-- 默认用户
INSERT INTO `sisyphus_users` (`username`, `password`, `email`, `real_name`, `phone`, `department`, `status`) VALUES
('admin',     '$2b$10$pb4zMrWBXkcLKfKqKCeiXeq4Ew4vjzQXckBpobtJit6FCg.JZ3rLi', 'admin@example.com', '管理员',   '13800138000', '技术部', 1),
('dba',       '$2b$10$pb4zMrWBXkcLKfKqKCeiXeq4Ew4vjzQXckBpobtJit6FCg.JZ3rLi', 'dba@example.com',   'DBA',      '13800138001', '技术部', 1),
('developer', '$2b$10$pb4zMrWBXkcLKfKqKCeiXeq4Ew4vjzQXckBpobtJit6FCg.JZ3rLi', 'dev@example.com',   '开发人员', '13800138002', '研发部', 1)
ON DUPLICATE KEY UPDATE `updated_at` = CURRENT_TIMESTAMP;

-- 用户角色绑定
INSERT INTO `sisyphus_user_roles` (`user_id`, `role_id`)
SELECT u.id, r.id
FROM `sisyphus_users` u
CROSS JOIN `sisyphus_roles` r
WHERE (u.username = 'admin'     AND r.code = 'admin')
   OR (u.username = 'dba'       AND r.code = 'dba')
   OR (u.username = 'developer' AND r.code = 'developer')
ON DUPLICATE KEY UPDATE `user_id` = `user_id`;

-- 示例数据库实例（本地开发用）
INSERT INTO `sisyphus_database_instances` (`name`, `db_type`, `host`, `port`, `username`, `password`, `env`, `status`, `owner`, `description`) VALUES
('本地MySQL', 'mysql', '127.0.0.1', 3306, 'root', 'root', 'dev', 1, 'admin', '本地开发MySQL实例'),
('本地Redis', 'redis', '127.0.0.1', 6379, '',     '',     'dev', 1, 'admin', '本地开发Redis实例')
ON DUPLICATE KEY UPDATE `updated_at` = CURRENT_TIMESTAMP;

-- 示例SQL工单
INSERT INTO `sisyphus_sql_orders` (`work_id`, `instance_id`, `database_name`, `order_type`, `status`, `sql_content`, `remark`, `submitter_id`, `delay_minutes`, `need_backup`, `execute_method`, `risk_level`) VALUES
('20240101000001',
 (SELECT id FROM `sisyphus_database_instances` WHERE db_type = 'mysql' LIMIT 1),
 'test_db', 0, 0,
 'CREATE TABLE IF NOT EXISTS `test_orders` (\n  `id` INT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,\n  `amount` DECIMAL(10,2) NOT NULL DEFAULT 0.00,\n  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP\n) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;',
 '示例DDL工单：创建测试表',
 (SELECT id FROM `sisyphus_users` WHERE username = 'developer' LIMIT 1),
 0, 0, 0, 0),
('20240101000002',
 (SELECT id FROM `sisyphus_database_instances` WHERE db_type = 'mysql' LIMIT 1),
 'test_db', 1, 0,
 'INSERT INTO `test_orders` (`amount`) VALUES (100.00), (200.00), (300.00);',
 '示例DML工单：写入测试数据',
 (SELECT id FROM `sisyphus_users` WHERE username = 'developer' LIMIT 1),
 0, 1, 0, 0)
ON DUPLICATE KEY UPDATE `updated_at` = CURRENT_TIMESTAMP;
