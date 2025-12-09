# ALIGNMENT: Agent 监控功能增强

## 1. 项目上下文分析

### 1.1 现有架构
- **Backend**: Daemon 和 Manager 已经支持采集和存储 Agent 的 `open_files` (文件描述符), `disk_read_bytes` (磁盘读取), `disk_write_bytes` (磁盘写入) 等指标。
- **Frontend**: 
  - `AgentResourceDataPoint` 类型定义已包含相关字段。
  - `useAgentMetricsHistory` Hook 目前仅硬编码支持 `cpu` 和 `memory`。
  - `AgentsTabContent` 页面仅展示 CPU 和内存监控图表。

### 1.2 需求理解
用户请求 "`div` agent 监控，补充文件描述，等其他已经有的数据监控"，解读为：
1.  **补充文件描述**: 在 Agent 监控页面增加 "文件描述符 (Open Files)" 的监控图表。
2.  **其他已有的数据监控**: 在 Agent 监控页面增加 "磁盘 I/O (Disk I/O)" 等后端已有但前端未展示的监控图表。

## 2. 需求边界与验收标准

### 2.1 需求边界
- 修改前端代码，不涉及后端变更（后端已支持）。
- 重点修改 `useAgentMetricsHistory` Hook 和 `AgentsTabContent` 组件。

### 2.2 验收标准
1.  **文件描述符监控**:
    - Agent 详情页展示 "文件描述符数量" 趋势图。
    - 单位为 "个" (Count)。
2.  **磁盘 I/O 监控**:
    - Agent 详情页展示 "磁盘读/写速率" 或 "累计读写量" 趋势图。
    - 单位自动格式化 (KB/s, MB/s 等)。
3.  **UI 一致性**:
    - 新增图表风格与现有 CPU/内存图表保持一致。

## 3. 疑问与决策

- **Q1**: `div` 是什么意思？
  - **A1**: 推测为 "Detailed" (详情) 或 "Dev" 的误触/简写，结合上下文 "agent 监控" 和 "文件描述"，明确是指 Agent 监控详情页。
- **Q2**: 磁盘监控展示累计值还是速率？
  - **A2**: `ResourceDataPoint` 提供的是 `disk_read_bytes` (累计值)。通常监控图表展示速率 (Bytes/sec) 更直观，需要前端计算差值或直接展示累计变化。鉴于后端 API 似乎直接返回点数据，如果后端未做速率计算，前端直接展示累计值会导致曲线一直上升，意义不大。
  - **决策**: 先查看后端数据特性。通常 Prometheus/Telegraf 采集的是 Counter (累计值)。如果 Daemon 采集的是 Counter，前端展示需要计算 Rate。
  - **修正**: 查看 `daemon/docs/FILE_DESCRIPTOR_MONITORING.md`，`OpenFiles` 是 Gauge (瞬时值)。
  - `disk_read_bytes` 通常是 Counter。
  - 既然 `useAgentMetricsHistory` 只是简单映射，且 CPU 是 Gauge，Memory 是 Gauge。Disk Bytes 可能是 Counter。
  - 如果是 Counter，前端直接画图会是单调递增曲线。用户可能更想看 "Disk Usage" 或 "I/O Rate"。
  - 考虑到 "简单补充"，直接展示 "累计读写量" 可能不是用户想要的。
  - **检查后端实现**: `daemon` 如何采集 `disk_read_bytes`。

### 3.1 检查后端采集逻辑

我需要检查 `daemon` 采集 `disk` 指标的代码，确认是 Counter 还是 Gauge (例如当前进程的 I/O 速率)。
如果是 `gopsutil` 的 `IOCounters`，通常是累计值。

## 4. 下一步计划
1. 检查后端磁盘指标采集逻辑。
2. 生成 CONSENSUS 文档。
