/**
 * Agent 监控指标相关 Hook
 * 
 * 后端 API 端点: GET /api/v1/nodes/:nodeId/agents/:agentId/metrics?duration=3600
 */

import { useQuery } from '@tanstack/react-query';
import { getAgentMetrics } from '../api/agents';
import type { TimeRange } from '../types';

interface AgentMetricsHistoryData {
  agent_id: string;
  type: string;
  data_points: {
    timestamp: string;
    values: Record<string, number>;
  }[];
}

/**
 * 使用 Agent 历史指标数据
 * @param nodeId 节点ID
 * @param agentId Agent ID
 * @param timeRange 时间范围
 * @param type 指标类型: 'cpu' | 'memory' | 'open_files' | 'disk_io'
 */
export function useAgentMetricsHistory(
  nodeId: string,
  agentId: string | null,
  timeRange: TimeRange,
  type: 'cpu' | 'memory' | 'open_files' | 'disk_io'
) {
  // 计算查询时间范围（秒）
  const durationSeconds = Math.ceil(
    (timeRange.endTime.getTime() - timeRange.startTime.getTime()) / 1000
  );

  return useQuery<{ data: AgentMetricsHistoryData }>({
    queryKey: [
      'agent-metrics',
      'history',
      nodeId,
      agentId,
      type,
      timeRange.startTime.getTime(),
      timeRange.endTime.getTime(),
    ],
    queryFn: async () => {
      if (!nodeId || !agentId) {
        throw new Error('nodeId and agentId are required');
      }

      // 调用真实 API
      const response = await getAgentMetrics(nodeId, agentId, durationSeconds);

      if (response.code !== 0) {
        throw new Error(response.message || '获取 Agent 指标失败');
      }

      // 转换数据格式以匹配前端使用的格式
      const dataPoints = (response.data?.data_points || []).map((dp) => {
        // 根据类型选择对应的值
        const values: Record<string, number> = {};
        
        if (type === 'cpu') {
          // CPU 使用率：直接使用后端的百分比值
          values.usage_percent = dp.cpu;
        } else if (type === 'memory') {
          // 内存使用：后端返回的是字节数，转换为 MB
          values.usage_percent = dp.memory_rss / (1024 * 1024); // 转换为 MB
        } else if (type === 'open_files') {
          // 文件描述符数量
          values.count = dp.open_files;
        } else if (type === 'disk_io') {
          // 磁盘 I/O：转换为 MB
          values.read_mb = dp.disk_read_bytes / (1024 * 1024);
          values.write_mb = dp.disk_write_bytes / (1024 * 1024);
        }

        return {
          timestamp: new Date(dp.timestamp * 1000).toISOString(), // Unix 时间戳转换为 ISO 字符串
          values,
        };
      });

      return {
        data: {
          agent_id: agentId,
          type,
          data_points: dataPoints,
        },
      };
    },
    enabled: !!nodeId && !!agentId && durationSeconds > 0,
    refetchOnWindowFocus: false,
    staleTime: 5 * 60 * 1000, // 历史数据缓存 5 分钟
  });
}
