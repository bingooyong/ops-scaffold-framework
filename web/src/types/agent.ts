/**
 * Agent 相关类型定义
 */

export interface Agent {
  id: number;
  node_id: string;
  agent_id: string;
  type: string; // 如 "filebeat", "telegraf", "node_exporter"
  version?: string;
  status: string; // "running", "stopped", "error", "starting", "stopping"
  config?: string; // JSON 格式的配置
  pid?: number;
  last_heartbeat?: string; // ISO 8601 格式时间戳
  last_sync_time?: string; // ISO 8601 格式时间戳
  created_at: string; // ISO 8601 格式时间戳
  updated_at: string; // ISO 8601 格式时间戳
}

export type AgentOperation = 'start' | 'stop' | 'restart';

export interface OperateAgentRequest {
  operation: AgentOperation;
}

export interface AgentLogsResponse {
  logs: string[];
  count: number;
}

export interface AgentListResponse {
  agents: Agent[];
  count: number;
}

/**
 * Agent 资源数据点（与后端 ResourceDataPoint 对应）
 */
export interface AgentResourceDataPoint {
  timestamp: number; // Unix 时间戳（秒）
  cpu: number; // CPU 使用率（百分比）
  memory_rss: number; // 内存占用 RSS（字节）
  memory_vms: number; // 内存占用 VMS（字节）
  disk_read_bytes: number; // 磁盘读取字节数
  disk_write_bytes: number; // 磁盘写入字节数
  open_files: number; // 打开文件数
}

/**
 * Agent 历史指标响应
 */
export interface AgentMetricsHistoryResponse {
  agent_id: string;
  data_points: AgentResourceDataPoint[];
  count: number;
}
