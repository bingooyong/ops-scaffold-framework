# CONSENSUS: Agent 监控功能增强

## 1. 需求概述
在 Agent 详情页增加 "文件描述符" 和 "磁盘 I/O" 的监控展示，利用后端已有的数据指标。

## 2. 技术实现方案

### 2.1 前端修改

#### 2.1.1 `useAgentMetricsHistory` Hook 增强
修改 `web/src/hooks/useAgentMetrics.ts`，扩展 `type` 参数支持：
- `open_files`: 直接使用 `dp.open_files`。
- `disk_io`: 返回 `disk_read_bytes` 和 `disk_write_bytes`。

#### 2.1.2 Agent 详情页更新
修改 `web/src/pages/Nodes/Detail.tsx` 中的 `AgentsTabContent` 组件：
1.  新增 `openFilesHistory` 数据获取。
2.  新增 `diskIOHistory` 数据获取。
3.  新增两个 `MetricsChart` 组件实例：
    - **文件描述符**: 展示 `open_files` 曲线。
    - **磁盘 I/O**: 展示 `read_bytes` 和 `write_bytes` 曲线 (双线图)。

### 2.2 数据处理策略
- **Open Files**: 直接展示数量 (Gauge)。
- **Disk I/O**: 展示累计字节数 (Counter)。虽然速率更直观，但考虑到前端计算复杂度和数据点可能的不均匀性，第一版先展示原始累计值，或者如果可能，在 Hook 中进行简单差分计算转换为速率。
  - *决策*: 为了保持实现简单且稳健，先展示累计值。如果曲线过于陡峭或难以阅读，后续可优化为速率。或者，鉴于这是 "补充已有数据"，展示原始数据是最忠实的实现。
  - *修正*: 再次查看 `useAgentMetricsHistory`，它目前只返回单值 `usage_percent`。
  - 为了支持 Disk I/O 的双线 (Read/Write)，需要修改 Hook 返回的数据结构，或者让 Hook 返回多条线的数据。
  - 当前 Hook 返回 `{ values: { usage_percent: number } }`。
  - 我将修改 Hook 返回 `{ values: { [key: string]: number } }` 以支持多条线。

## 3. 验收标准
1.  `web/src/hooks/useAgentMetrics.ts` 支持 `open_files` 和 `disk_io` 类型。
2.  `web/src/pages/Nodes/Detail.tsx` 成功展示 "文件描述符" 和 "磁盘 I/O" 图表。
3.  图表样式与现有 CPU/内存图表保持一致。
4.  代码通过编译，无类型错误。

## 4. 任务列表
1.  **Refactor Hook**: 修改 `useAgentMetricsHistory` 支持多值和新类型。
2.  **Update UI**: 在 `Detail.tsx` 中添加新图表。
