/**
 * Agent 管理相关 API
 */

import client from './interceptors';
import type {
  APIResponse,
  AgentOperation,
  AgentLogsResponse,
  AgentListResponse,
  AgentMetricsHistoryResponse,
} from '../types';

/**
 * 获取节点下的所有 Agent 列表
 * @param nodeId 节点ID
 */
export function listAgents(nodeId: string): Promise<APIResponse<AgentListResponse>> {
  return client
    .get(`/api/v1/nodes/${nodeId}/agents`)
    .then((res) => res.data);
}

/**
 * 操作 Agent(启动/停止/重启)
 * @param nodeId 节点ID
 * @param agentId Agent ID
 * @param operation 操作类型: "start" | "stop" | "restart"
 */
export function operateAgent(
  nodeId: string,
  agentId: string,
  operation: AgentOperation
): Promise<APIResponse> {
  return client
    .post(`/api/v1/nodes/${nodeId}/agents/${agentId}/operate`, {
      operation,
    })
    .then((res) => res.data);
}

/**
 * 获取 Agent 日志
 * @param nodeId 节点ID
 * @param agentId Agent ID
 * @param lines 日志行数,默认 100,最大 1000
 */
export function getAgentLogs(
  nodeId: string,
  agentId: string,
  lines?: number
): Promise<APIResponse<AgentLogsResponse>> {
  const params = lines ? { lines } : {};
  return client
    .get(`/api/v1/nodes/${nodeId}/agents/${agentId}/logs`, { params })
    .then((res) => res.data);
}

/**
 * 获取 Agent 资源使用指标历史数据
 * @param nodeId 节点ID
 * @param agentId Agent ID
 * @param duration 查询时间范围（秒），默认 3600（1小时），最大 7天
 */
export function getAgentMetrics(
  nodeId: string,
  agentId: string,
  duration?: number
): Promise<APIResponse<AgentMetricsHistoryResponse>> {
  const params = duration ? { duration } : {};
  return client
    .get(`/api/v1/nodes/${nodeId}/agents/${agentId}/metrics`, { params })
    .then((res) => res.data);
}

/**
 * 同步响应类型
 */
export interface SyncAgentsResponse {
  message: string;
  synced_count: number;
}

/**
 * 手动同步 Agent 状态
 * 从 Daemon 获取最新的 Agent 状态并更新到数据库
 * @param nodeId 节点ID
 */
export function syncAgents(nodeId: string): Promise<APIResponse<SyncAgentsResponse>> {
  return client
    .post(`/api/v1/nodes/${nodeId}/agents/sync`)
    .then((res) => res.data);
}
