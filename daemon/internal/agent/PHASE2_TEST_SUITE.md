# Phase 2 测试套件文档

## 概述

本文档描述了 Phase 2 Agent 管理核心功能的完整测试套件，包括集成测试、边界场景测试和性能测试。

## 测试文件列表

### 集成测试
- `phase2_integration_test.go`: Phase 2 所有功能的集成测试

### 性能测试
- `phase2_performance_test.go`: 关键操作的性能基准测试

### 组件测试文件（已补充边界测试）
- `config_manager_test.go`: 配置管理器测试（包含并发、边界场景测试）
- `metadata_store_test.go`: 元数据存储测试（包含并发、大规模数据测试）
- `heartbeat_receiver_test.go`: 心跳接收器测试（包含突发流量、并发测试）
- `resource_monitor_test.go`: 资源监控器测试（包含大规模、阈值测试）
- `log_manager_test.go`: 日志管理器测试（包含大文件、并发测试）

## 集成测试场景

### 1. TestPhase2_CompleteAgentLifecycle
测试完整的 Agent 生命周期管理流程：
- 创建所有 Phase 2 组件（MultiAgentManager、ConfigManager、MetadataStore、HeartbeatReceiver、ResourceMonitor、LogManager）
- 注册 Agent 并启动
- 验证元数据创建（Status、StartTime）
- 发送心跳，验证元数据更新（LastHeartbeat、ResourceUsage）
- 更新配置，验证配置热重载
- 检查资源监控数据采集
- 检查日志文件创建
- 停止 Agent，验证元数据更新（Status = stopped）
- 验证所有组件正常工作

### 2. TestPhase2_ConfigAndMetadataIntegration
测试配置管理和元数据跟踪的协同：
- 创建 ConfigManager 和 MetadataStore
- 读取 Agent 配置
- 更新配置并验证
- 验证元数据中的配置信息
- 测试配置更新后元数据的一致性

### 3. TestPhase2_HeartbeatAndResourceMonitoring
测试心跳接收和资源监控的协同：
- 创建 HeartbeatReceiver 和 ResourceMonitor
- 启动 Agent
- 发送心跳（包含 CPU、Memory 数据）
- 验证心跳数据更新到元数据
- 等待资源监控采集
- 验证资源监控数据也更新到元数据
- 验证两种数据源的一致性

### 4. TestPhase2_LogManagementAndCleanup
测试日志管理和清理：
- 创建 LogManager
- 启动 Agent，生成日志
- 测试日志查询接口（GetAgentLogs、SearchLogs）
- 测试日志轮转（模拟大文件）
- 测试日志清理（模拟过期文件）
- 验证日志文件管理正确

## 边界场景测试

### ConfigManager 边界测试
- `TestReadConfig_ConcurrentRead`: 测试并发读取配置
- `TestUpdateConfig_ConcurrentUpdate`: 测试并发更新配置（验证原子性）
- `TestValidateConfig_EdgeCases`: 测试边界值验证（空配置、超大配置等）
- `TestStartWatching_MultipleAgents`: 测试监听多个 Agent 配置文件

### MetadataStore 边界测试
- `TestSaveMetadata_ConcurrentSave`: 测试并发保存元数据
- `TestListAllMetadata_LargeDataset`: 测试大量 Agent 的元数据列举（100个Agent）
- `TestUpdateMetadata_RaceCondition`: 测试并发更新元数据的竞态条件

### HeartbeatReceiver 边界测试
- `TestHandleHeartbeat_BurstTraffic`: 测试突发流量（短时间内100个心跳）
- `TestWorkerPool_WorkerFailure`: 测试 worker 失败时的处理
- `TestStats_ConcurrentUpdate`: 测试统计信息的并发更新

### ResourceMonitor 边界测试
- `TestCollectAllAgents_LargeScale`: 测试大规模 Agent 的并发采集（50个Agent）
- `TestCheckResourceThresholds_MultipleAgents`: 测试多个 Agent 同时超过阈值
- `TestGetResourceHistory_LargeHistory`: 测试大量历史数据的查询性能（1440个数据点）

### LogManager 边界测试
- `TestGetAgentLogs_VeryLargeFile`: 测试超大日志文件的读取（>10MB，15MB文件）
- `TestSearchLogs_ConcurrentSearch`: 测试并发搜索日志（10个goroutine）
- `TestCleanupOldLogs_ManyFiles`: 测试大量日志文件的清理性能（200个文件）

## 性能基准测试

### 配置管理性能
- `BenchmarkConfigManager_ReadConfig`: 配置读取性能
- `BenchmarkConfigManager_UpdateConfig`: 配置更新性能
- `BenchmarkConfigManager_ConcurrentRead`: 并发读取配置性能

### 元数据存储性能
- `BenchmarkMetadataStore_SaveMetadata`: 元数据保存性能
- `BenchmarkMetadataStore_ConcurrentSave`: 并发保存元数据性能
- `BenchmarkMetadataStore_GetMetadata`: 获取元数据性能

### 心跳处理性能
- `BenchmarkHeartbeatReceiver_ProcessHeartbeat`: 心跳处理性能
- `BenchmarkHeartbeatReceiver_ConcurrentHeartbeat`: 并发心跳处理性能

### 资源监控性能
- `BenchmarkResourceMonitor_CollectResources`: 资源采集性能

### 日志管理性能
- `BenchmarkLogManager_GetAgentLogs`: 日志查询性能

### 资源使用历史性能
- `BenchmarkResourceUsageHistory_AddResourceData`: 资源使用历史添加数据性能
- `BenchmarkResourceUsageHistory_GetRecent`: 资源使用历史查询性能

### Agent 管理性能
- `BenchmarkMultiAgentManager_RegisterAgent`: 注册 Agent 性能

## 运行测试

### 运行所有 Phase 2 相关测试
```bash
go test -v ./daemon/internal/agent/... -run "Phase2|Config|Metadata|Heartbeat|Resource|Log"
```

### 运行集成测试
```bash
go test -v ./daemon/internal/agent/... -run "Phase2"
```

### 运行性能基准测试
```bash
go test -bench=. -benchmem ./daemon/internal/agent/... -run "^$"
```

### 生成覆盖率报告
```bash
cd daemon
go test -coverprofile=coverage.out ./internal/agent/...
go tool cover -html=coverage.out -o coverage.html
```

### 运行 Race Detector 测试
```bash
go test -race ./daemon/internal/agent/...
```

## 测试统计

### 测试用例数量
- **集成测试**: 4 个测试场景
- **边界场景测试**: 15 个测试用例
- **性能基准测试**: 13 个基准测试

### 各组件测试用例数量
- **ConfigManager**: 原有测试 + 4 个边界测试
- **MetadataStore**: 原有测试 + 3 个边界测试
- **HeartbeatReceiver**: 原有测试 + 3 个边界测试
- **ResourceMonitor**: 原有测试 + 3 个边界测试
- **LogManager**: 原有测试 + 3 个边界测试

### 总体测试覆盖率目标
- **目标覆盖率**: > 80%
- **各组件覆盖率**:
  - `config_manager.go`: > 80%
  - `metadata_store.go`: > 80%
  - `heartbeat_receiver.go`: > 80%
  - `resource_monitor.go`: > 80%
  - `log_manager.go`: > 80%

## 测试结果

### 集成测试结果
所有集成测试场景验证了 Phase 2 所有功能的协同工作，确保：
- Agent 生命周期管理完整
- 配置管理和元数据跟踪协同工作
- 心跳接收和资源监控数据一致
- 日志管理和清理正常工作

### 边界场景测试结果
边界场景测试覆盖了：
- 并发访问安全性
- 大规模数据场景
- 异常情况处理
- 极端条件验证

### 性能测试结果
性能基准测试验证了关键操作的性能，确保：
- 配置读取/更新性能满足要求
- 元数据保存/查询性能满足要求
- 心跳处理性能满足要求
- 资源采集性能满足要求
- 日志查询性能满足要求

## 注意事项

1. **并发测试**: 所有并发测试都验证了线程安全性，确保没有 data race
2. **大规模测试**: 大规模数据测试验证了系统在处理大量数据时的性能
3. **边界值测试**: 边界值测试验证了系统在极端条件下的稳定性
4. **集成测试**: 集成测试验证了各组件协同工作的正确性

## 持续改进

测试套件应该随着功能的发展而持续改进：
- 添加新的测试场景
- 提高测试覆盖率
- 优化性能基准测试
- 补充更多边界场景测试
