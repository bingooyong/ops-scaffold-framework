/**
 * 节点相关类型定义
 */

export interface Node {
  id: number;
  node_id: string;
  hostname: string;
  ip: string;
  os: string;
  arch: string;
  status: NodeStatus;
  labels?: Record<string, string>;
  daemon_version?: string;
  agent_version?: string;
  created_at: string;
  updated_at: string;
  last_seen_at?: string;
}

export const NodeStatus = {
  Online: 'online',
  Offline: 'offline',
  Unknown: 'unknown',
} as const;

export type NodeStatus = typeof NodeStatus[keyof typeof NodeStatus];

export interface NodeStatistics {
  total: number;
  online: number;
  offline: number;
}

export interface NodeMetricsData {
  node_id: string;
  timestamp: string;
  cpu: CPUMetrics;
  memory: MemoryMetrics;
  disk: DiskMetrics[];
  network: NetworkMetrics[];
}

export interface CPUMetrics {
  usage_percent: number;
  cores: number;
  model?: string;
}

export interface MemoryMetrics {
  total_bytes: number;
  used_bytes: number;
  available_bytes: number;
  usage_percent: number;
}

export interface DiskMetrics {
  device: string;
  mount_point: string;
  total_bytes: number;
  used_bytes: number;
  available_bytes: number;
  usage_percent: number;
}

export interface NetworkMetrics {
  interface: string;
  rx_bytes: number;
  tx_bytes: number;
  rx_packets: number;
  tx_packets: number;
  rx_errors: number;
  tx_errors: number;
}
