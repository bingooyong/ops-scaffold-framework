-- ============================================
-- Metrics 表分区迁移脚本
-- 功能：将 metrics 表转换为按日期分区的 RANGE 分区表
-- 创建时间：2025-12-04
-- ============================================

-- 步骤 1: 创建分区表
-- 使用 RANGE 分区，按日期分区（TO_DAYS(timestamp)）
-- 初始创建未来 7 天的分区
CREATE TABLE IF NOT EXISTS metrics_partitioned (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    created_at DATETIME(3) NOT NULL,
    deleted_at DATETIME(3) NULL,
    node_id VARCHAR(50) NOT NULL,
    type VARCHAR(20) NOT NULL,
    timestamp DATETIME(3) NOT NULL,
    values JSON NOT NULL,
    PRIMARY KEY (id, timestamp),
    KEY idx_node_type_time (node_id, type, timestamp),
    KEY idx_node_time (node_id, timestamp),
    KEY idx_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
PARTITION BY RANGE (TO_DAYS(timestamp)) (
    PARTITION p20251204 VALUES LESS THAN (TO_DAYS('2025-12-05')),
    PARTITION p20251205 VALUES LESS THAN (TO_DAYS('2025-12-06')),
    PARTITION p20251206 VALUES LESS THAN (TO_DAYS('2025-12-07')),
    PARTITION p20251207 VALUES LESS THAN (TO_DAYS('2025-12-08')),
    PARTITION p20251208 VALUES LESS THAN (TO_DAYS('2025-12-09')),
    PARTITION p20251209 VALUES LESS THAN (TO_DAYS('2025-12-10')),
    PARTITION p20251210 VALUES LESS THAN (TO_DAYS('2025-12-11')),
    PARTITION p_future VALUES LESS THAN MAXVALUE
);

-- 步骤 2: 数据迁移（分批插入，按日期分批）
-- 注意：对于大数据量，建议按日期分批迁移，避免长时间锁表
-- 示例：迁移 2025-12-01 的数据
-- INSERT INTO metrics_partitioned SELECT * FROM metrics WHERE DATE(timestamp) = '2025-12-01';

-- 完整迁移（适用于数据量较小的情况）
-- 如果数据量较大，请使用分批迁移方式
INSERT INTO metrics_partitioned (id, created_at, deleted_at, node_id, type, timestamp, values)
SELECT id, created_at, deleted_at, node_id, type, timestamp, values
FROM metrics
WHERE timestamp >= DATE_SUB(CURDATE(), INTERVAL 30 DAY);

-- 步骤 3: 数据完整性验证
-- 验证迁移的数据行数
SELECT 
    'Original table' as table_name,
    COUNT(*) as row_count
FROM metrics
UNION ALL
SELECT 
    'Partitioned table' as table_name,
    COUNT(*) as row_count
FROM metrics_partitioned;

-- 验证数据范围
SELECT 
    'Original table' as table_name,
    MIN(timestamp) as min_timestamp,
    MAX(timestamp) as max_timestamp
FROM metrics
UNION ALL
SELECT 
    'Partitioned table' as table_name,
    MIN(timestamp) as min_timestamp,
    MAX(timestamp) as max_timestamp
FROM metrics_partitioned;

-- 步骤 4: 表重命名（原子操作，最小化停机时间）
-- 注意：执行前请确保数据验证通过
-- RENAME TABLE metrics TO metrics_old, metrics_partitioned TO metrics;

-- 步骤 5: 验证分区表结构
-- SHOW CREATE TABLE metrics;
-- 应该看到 PARTITION BY RANGE (TO_DAYS(timestamp)) 定义

-- 步骤 6: 验证索引使用（使用 EXPLAIN 分析查询计划）
-- EXPLAIN SELECT * FROM metrics 
-- WHERE node_id='test' AND type='cpu' AND timestamp BETWEEN '2025-12-01' AND '2025-12-04';
-- 应该看到：
-- - partitions 列仅显示相关分区（分区裁剪生效）
-- - key 列显示使用了 idx_node_type_time 索引
-- - type 列显示 ref 或 range（避免 ALL 全表扫描）

-- ============================================
-- 回滚方案（如果迁移失败）
-- ============================================
-- 如果迁移出现问题，执行以下命令回滚：
-- RENAME TABLE metrics TO metrics_partitioned_failed, metrics_old TO metrics;

-- ============================================
-- 后续维护：创建新分区
-- ============================================
-- 每天自动创建新分区（通过定时任务执行）
-- ALTER TABLE metrics ADD PARTITION (
--     PARTITION p20251211 VALUES LESS THAN (TO_DAYS('2025-12-12'))
-- );

