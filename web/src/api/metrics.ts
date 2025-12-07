/**
 * 监控指标相关 API
 */

import client from './interceptors';
import type { APIResponse, MetricsLatestResponse, MetricsHistoryResponse, MetricsSummaryResponse, TimeRange, ClusterOverviewResponse } from '../types';

/**
 * 获取节点最新指标
 */
export function getLatestMetrics(nodeId: string): Promise<APIResponse<MetricsLatestResponse>> {
  return client
    .get(`/api/v1/metrics/nodes/${nodeId}/latest`)
    .then((res) => res.data);
}

/**
 * 获取历史指标数据
 */
export function getMetricsHistory(
  nodeId: string,
  type: string,
  params: { start_time: string; end_time: string }
): Promise<APIResponse<MetricsHistoryResponse>> {
  return client
    .get(`/api/v1/metrics/nodes/${nodeId}/${type}/history`, { params })
    .then((res) => res.data);
}

/**
 * 获取指标统计摘要
 */
export function getMetricsSummary(
  nodeId: string,
  timeRange?: TimeRange
): Promise<APIResponse<MetricsSummaryResponse>> {
  const params: Record<string, string> = {};
  
  if (timeRange) {
    params.start_time = timeRange.startTime.toISOString();
    params.end_time = timeRange.endTime.toISOString();
  }

  return client
    .get(`/api/v1/metrics/nodes/${nodeId}/summary`, { params })
    .then((res) => res.data);
}

/**
 * 获取集群资源概览
 */
export function getClusterOverview(): Promise<APIResponse<ClusterOverviewResponse>> {
  return client
    .get('/api/v1/metrics/cluster/overview')
    .then((res) => res.data);
}

