# Metrics 表分区迁移文档

## 概述

本文档说明如何将 `metrics` 表从普通表转换为按日期分区的 RANGE 分区表，以优化大数据量查询性能和数据管理效率。

**适用场景**：
- 数据量较大（200 节点 × 30 天 ≈ 35GB）
- 需要按时间范围频繁查询
- 需要定期清理历史数据

**预期收益**：
- 查询性能提升（分区裁剪，仅扫描相关分区）
- 数据管理简化（按分区删除，无需 DELETE 语句）
- 索引效率提升（每个分区独立索引）

---

## 分区策略说明

### 分区方式
- **类型**: RANGE 分区
- **分区键**: `TO_DAYS(timestamp)`（按日期分区）
- **分区粒度**: 每天一个分区

### 分区命名规则
- 格式: `pYYYYMMDD`
- 示例: `p20251204` 表示 2025-12-04 这一天的数据

### 初始分区
- 创建未来 7 天的分区
- 包含一个 `p_future` 分区（`VALUES LESS THAN MAXVALUE`）用于处理未来数据

### 索引设计
- **主索引**: `idx_node_type_time (node_id, type, timestamp)` - 用于时间范围查询
- **辅助索引**: `idx_node_time (node_id, timestamp)` - 用于最新指标查询
- **软删除索引**: `idx_deleted_at (deleted_at)` - 用于软删除查询

---

## 迁移前准备

### 1. 备份现有数据
```bash
# 使用 mysqldump 备份
mysqldump -h 127.0.0.1 -u root -p ops_manager_dev metrics > metrics_backup_$(date +%Y%m%d).sql

# 或使用 MySQL 物理备份
# 确保有完整的数据库备份
```

### 2. 评估数据量和迁移时间
```sql
-- 查看当前数据量
SELECT COUNT(*) as total_rows, 
       MIN(timestamp) as min_time,
       MAX(timestamp) as max_time,
       COUNT(DISTINCT DATE(timestamp)) as days
FROM metrics;

-- 估算迁移时间（根据数据量）
-- 小数据量（< 1GB）: 几分钟
-- 中等数据量（1-10GB）: 10-30 分钟
-- 大数据量（> 10GB）: 30 分钟以上，建议分批迁移
```

### 3. 选择维护窗口
- **推荐时间**: 业务低峰期（如凌晨 2-4 点）
- **预计停机时间**: 5-10 分钟（表重命名操作）
- **通知**: 提前通知相关团队维护窗口时间

### 4. 检查 MySQL 版本
```sql
SELECT VERSION();
-- 需要 MySQL 5.7+ 或 MySQL 8.0+
```

---

## 迁移步骤

### 步骤 1: 执行迁移脚本
```bash
# 连接到 MySQL
mysql -h 127.0.0.1 -u root -p ops_manager_dev

# 执行迁移脚本
source manager/migrations/add_metrics_partitions.sql;
```

### 步骤 2: 验证数据完整性
```sql
-- 验证行数一致
SELECT 
    (SELECT COUNT(*) FROM metrics) as original_count,
    (SELECT COUNT(*) FROM metrics_partitioned) as partitioned_count;

-- 验证数据范围一致
SELECT 
    (SELECT MIN(timestamp) FROM metrics) as original_min,
    (SELECT MIN(timestamp) FROM metrics_partitioned) as partitioned_min,
    (SELECT MAX(timestamp) FROM metrics) as original_max,
    (SELECT MAX(timestamp) FROM metrics_partitioned) as partitioned_max;
```

**预期结果**:
- `original_count` = `partitioned_count`
- 时间范围一致

### 步骤 3: 性能验证
```sql
-- 验证分区裁剪
EXPLAIN SELECT * FROM metrics_partitioned 
WHERE node_id='test-node' AND type='cpu' 
AND timestamp BETWEEN '2025-12-01' AND '2025-12-04';

-- 应该看到：
-- - partitions 列仅显示相关分区（如 p20251201, p20251202, p20251203, p20251204）
-- - key 列显示 idx_node_type_time
-- - type 列显示 ref 或 range
```

### 步骤 4: 执行表重命名（切换到分区表）
```sql
-- 原子操作，最小化停机时间
RENAME TABLE metrics TO metrics_old, metrics_partitioned TO metrics;
```

**注意**: 此操作会短暂锁定表，建议在维护窗口执行。

### 步骤 5: 验证分区表结构
```sql
-- 查看分区表定义
SHOW CREATE TABLE metrics;

-- 查看分区列表
SELECT 
    PARTITION_NAME,
    PARTITION_DESCRIPTION,
    TABLE_ROWS
FROM information_schema.PARTITIONS
WHERE TABLE_SCHEMA = DATABASE()
  AND TABLE_NAME = 'metrics'
ORDER BY PARTITION_ORDINAL_POSITION;
```

### 步骤 6: 测试查询性能
```sql
-- 测试时间范围查询
SELECT COUNT(*) FROM metrics 
WHERE node_id='test-node' AND type='cpu' 
AND timestamp BETWEEN '2025-12-01' AND '2025-12-04';

-- 测试最新指标查询
SELECT * FROM metrics 
WHERE node_id='test-node' AND type='cpu' 
ORDER BY timestamp DESC LIMIT 1;
```

---

## 回滚方案

如果迁移过程中出现问题，可以执行以下回滚操作：

### 回滚步骤
```sql
-- 1. 如果表重命名还未执行，直接删除分区表
DROP TABLE IF EXISTS metrics_partitioned;

-- 2. 如果表重命名已执行，回滚到原始表
RENAME TABLE metrics TO metrics_partitioned_failed, metrics_old TO metrics;
```

### 回滚后验证
```sql
-- 验证原始表数据完整
SELECT COUNT(*) FROM metrics;
-- 应该与备份时的数据量一致
```

---

## 生产部署注意事项

### 1. 停止 Daemon 数据上报
- 在迁移期间，建议暂停 Daemon 的数据上报
- 或确保 Daemon 有重试机制，可以容忍短暂的写入失败

### 2. API 维护状态
- 在迁移期间，API 可能返回维护状态
- 或使用只读模式，允许查询但不允许写入

### 3. 分批迁移大数据量
如果数据量很大（> 10GB），建议分批迁移：

```sql
-- 按日期分批迁移
-- 第一天
INSERT INTO metrics_partitioned SELECT * FROM metrics WHERE DATE(timestamp) = '2025-12-01';

-- 第二天
INSERT INTO metrics_partitioned SELECT * FROM metrics WHERE DATE(timestamp) = '2025-12-02';

-- ... 依此类推
```

### 4. 监控迁移进度
```sql
-- 查看迁移进度
SELECT 
    DATE(timestamp) as date,
    COUNT(*) as row_count
FROM metrics_partitioned
GROUP BY DATE(timestamp)
ORDER BY date;
```

### 5. 验证应用功能
迁移完成后，验证以下功能：
- ✅ 节点最新指标查询
- ✅ 历史数据查询
- ✅ 统计摘要查询
- ✅ 数据写入（Daemon 上报）

---

## 后续维护

### 1. 定期创建新分区
通过定时任务（Task 1.5 实现）自动创建新分区，或手动执行：

```sql
-- 创建明天的分区
ALTER TABLE metrics ADD PARTITION (
    PARTITION p20251211 VALUES LESS THAN (TO_DAYS('2025-12-12'))
);
```

### 2. 监控分区数量
```sql
-- 查看当前分区数量
SELECT COUNT(*) as partition_count
FROM information_schema.PARTITIONS
WHERE TABLE_SCHEMA = DATABASE()
  AND TABLE_NAME = 'metrics'
  AND PARTITION_NAME IS NOT NULL;
```

**建议**: 保持未来 7-14 天的分区，避免分区过多影响性能。

### 3. 自动清理过期分区
通过定时任务（Task 1.5 实现）自动删除过期分区：

```sql
-- 删除 30 天前的分区
ALTER TABLE metrics DROP PARTITION p20251104;
```

### 4. 分区维护最佳实践
- **定期检查**: 每周检查分区数量和大小
- **提前创建**: 提前 7 天创建新分区
- **及时清理**: 定期清理过期分区，避免分区数量过多
- **监控性能**: 使用 `EXPLAIN` 定期验证查询计划

---

## 性能优化建议

### 1. 分区裁剪验证
定期使用 `EXPLAIN` 验证查询是否使用了分区裁剪：

```sql
EXPLAIN SELECT * FROM metrics 
WHERE node_id='node-001' AND type='cpu' 
AND timestamp BETWEEN '2025-12-01' AND '2025-12-04';
```

**预期**: `partitions` 列仅显示相关分区。

### 2. 索引使用验证
验证查询是否使用了正确的索引：

```sql
EXPLAIN SELECT * FROM metrics 
WHERE node_id='node-001' AND type='cpu' 
AND timestamp BETWEEN '2025-12-01' AND '2025-12-04';
```

**预期**: `key` 列显示 `idx_node_type_time`。

### 3. 查询性能对比
对比迁移前后的查询性能：

```sql
-- 记录查询时间
SET profiling = 1;
SELECT COUNT(*) FROM metrics 
WHERE node_id='node-001' AND type='cpu' 
AND timestamp BETWEEN '2025-12-01' AND '2025-12-04';
SHOW PROFILES;
```

---

## 故障排查

### 问题 1: 分区创建失败
**错误**: `Error Code: 1503. A PRIMARY KEY must include all columns in the table's partitioning function`

**解决**: 确保主键包含分区键（timestamp），已包含在迁移脚本中。

### 问题 2: 数据迁移失败
**错误**: 数据量过大，迁移超时

**解决**: 使用分批迁移方式，按日期分批插入。

### 问题 3: 查询性能未提升
**可能原因**:
- 分区裁剪未生效（WHERE 条件不包含分区键）
- 索引未正确使用

**解决**: 使用 `EXPLAIN` 分析查询计划，确保使用分区和索引。

---

## 参考文档

- [MySQL Partitioning](https://dev.mysql.com/doc/refman/8.0/en/partitioning.html)
- [RANGE Partitioning](https://dev.mysql.com/doc/refman/8.0/en/partitioning-range.html)
- [Partition Pruning](https://dev.mysql.com/doc/refman/8.0/en/partitioning-pruning.html)

---

**版本**: v1.0  
**更新日期**: 2025-12-04  
**维护者**: Ops Scaffold Framework Team

