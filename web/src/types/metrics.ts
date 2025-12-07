/**
 * 监控指标相关类型定义
 */

/**
 * 指标数据
 */
export interface MetricData {
  id: number;
  node_id: string;
  type: 'cpu' | 'memory' | 'disk' | 'network';
  timestamp: string;
  values: Record<string, any>;
}

/**
 * 最新指标响应
 */
export interface MetricsLatestResponse {
  cpu?: MetricData;
  memory?: MetricData;
  disk?: MetricData;
  network?: MetricData;
}

/**
 * 历史指标响应
 */
export type MetricsHistoryResponse = MetricData[];

/**
 * 指标统计摘要响应
 */
export interface MetricsSummaryResponse {
  cpu?: MetricSummary;
  memory?: MetricSummary;
  disk?: MetricSummary;
  network?: MetricSummary;
}

/**
 * 指标统计摘要
 */
export interface MetricSummary {
  min: number;
  max: number;
  avg: number;
  latest: number;
}

/**
 * 时间范围
 */
export interface TimeRange {
  startTime: Date;
  endTime: Date;
}

/**
 * 节点指标数据（用于集群概览）
 */
export interface NodeMetrics {
  node_id: string;
  hostname: string;
  ip: string;
  status: string;
  cpu_usage: number;
  memory_usage: number;
  disk_usage: number;
  network_rx: number;
  network_tx: number;
}

/**
 * 集群概览响应
 */
export interface ClusterOverviewResponse {
  aggregate: {
    avg_cpu: number;
    avg_memory: number;
    avg_disk: number;
    total_memory_gb: number;
    total_disk_gb: number;
    node_counts: {
      total: number;
      online: number;
      offline: number;
    };
  };
  nodes: NodeMetrics[];
}

