# DESIGN: Agent 监控功能增强

## 1. 模块设计

### 1.1 `useAgentMetricsHistory` Hook
**路径**: `web/src/hooks/useAgentMetrics.ts`

**变更**:
- 修改 `type` 参数定义: `'cpu' | 'memory' | 'open_files' | 'disk_io'`
- 修改返回值数据结构，支持动态 keys。

**逻辑**:
```typescript
// 伪代码
export function useAgentMetricsHistory(..., type: 'cpu' | 'memory' | 'open_files' | 'disk_io') {
  // ...
  const dataPoints = response.data?.data_points.map(dp => {
    const values: Record<string, number> = {};
    
    switch(type) {
      case 'cpu':
        values.cpu = dp.cpu;
        break;
      case 'memory':
        values.memory = dp.memory_rss / (1024 * 1024);
        break;
      case 'open_files':
        values.open_files = dp.open_files;
        break;
      case 'disk_io':
        values.read = dp.disk_read_bytes / (1024 * 1024); // MB
        values.write = dp.disk_write_bytes / (1024 * 1024); // MB
        break;
    }
    
    return {
      timestamp: ...,
      values
    };
  });
  // ...
}
```

### 1.2 `AgentsTabContent` Component
**路径**: `web/src/pages/Nodes/Detail.tsx`

**变更**:
- 调用 `useAgentMetricsHistory` 获取 `open_files` 和 `disk_io` 数据。
- 渲染新的 `MetricsChart`。

**布局**:
```jsx
<Grid container spacing={3}>
  <Grid item xs={12} md={6}>
    <MetricsChart title="CPU 使用率" ... />
  </Grid>
  <Grid item xs={12} md={6}>
    <MetricsChart title="内存使用" ... />
  </Grid>
  <Grid item xs={12} md={6}>
    <MetricsChart title="文件描述符" ... />
  </Grid>
  <Grid item xs={12} md={6}>
    <MetricsChart title="磁盘 I/O (累计)" ... />
  </Grid>
</Grid>
```

## 2. 接口定义

### 2.1 前端 Hook 接口
```typescript
export function useAgentMetricsHistory(
  nodeId: string,
  agentId: string | null,
  timeRange: TimeRange,
  type: 'cpu' | 'memory' | 'open_files' | 'disk_io'
): UseQueryResult<{
  agent_id: string;
  type: string;
  data_points: Array<{
    timestamp: string;
    values: Record<string, number>;
  }>;
}>
```

## 3. 数据流向
Backend (gRPC/HTTP) -> `getAgentMetrics` (API) -> `useAgentMetricsHistory` (Hook) -> `AgentsTabContent` (UI) -> `MetricsChart` (Render)
