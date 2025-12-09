/**
 * 监控指标相关 Hook
 */

import { useEffect } from 'react';
import { useQuery } from '@tanstack/react-query';
import { useNavigate } from 'react-router-dom';
import { getLatestMetrics, getMetricsHistory, getMetricsSummary, getClusterOverview } from '../api';
import { ErrorCode } from '../types';
import type { TimeRange, APIResponse, MetricsHistoryResponse } from '../types';
import { useMetricsStore } from '../stores';

/**
 * 使用最新指标
 */
export function useLatestMetrics(nodeId: string) {
  return useQuery({
    queryKey: ['metrics', 'latest', nodeId],
    queryFn: () => getLatestMetrics(nodeId),
    enabled: !!nodeId,
    refetchInterval: 30000, // 30 秒自动刷新
    staleTime: 25000, // 缓存 25 秒
  });
}

/**
 * 使用历史指标数据
 */
export function useMetricsHistory(
  nodeId: string,
  type: string,
  timeRange: TimeRange
) {
  return useQuery<APIResponse<MetricsHistoryResponse>>({
    queryKey: [
      'metrics',
      'history',
      nodeId,
      type,
      timeRange.startTime.getTime(),
      timeRange.endTime.getTime(),
    ],
    queryFn: () =>
      getMetricsHistory(nodeId, type, {
        start_time: timeRange.startTime.toISOString(),
        end_time: timeRange.endTime.toISOString(),
      }),
    enabled: !!nodeId && !!type,
    refetchOnWindowFocus: false, // 禁用窗口聚焦时自动刷新
    staleTime: 5 * 60 * 1000, // 历史数据缓存 5 分钟
  });
}

/**
 * 使用指标统计摘要
 */
export function useMetricsSummary(nodeId: string, timeRange?: TimeRange) {
  return useQuery({
    queryKey: [
      'metrics',
      'summary',
      nodeId,
      timeRange?.startTime.getTime(),
      timeRange?.endTime.getTime(),
    ],
    queryFn: () => getMetricsSummary(nodeId, timeRange),
    enabled: !!nodeId,
    staleTime: 5 * 60 * 1000, // 缓存 5 分钟
  });
}

/**
 * 使用集群资源概览
 */
export function useClusterOverview() {
  const navigate = useNavigate();
  const { refreshInterval } = useMetricsStore();
  
  const query = useQuery({
    queryKey: ['metrics', 'cluster', 'overview'],
    queryFn: () => getClusterOverview(),
    refetchInterval: refreshInterval || false, // 根据 refreshInterval 决定是否自动刷新
    staleTime: 25000, // 缓存 25 秒
  });

  useEffect(() => {
    if (query.error) {
      const error = query.error;
      // 检查是否是 Axios 错误
      if (error && typeof error === 'object' && 'response' in error) {
        const axiosError = error as { response?: { status?: number; data?: { code?: number } } };
        if (axiosError.response?.status === 401 || axiosError.response?.data?.code === ErrorCode.Unauthorized) {
          navigate('/login');
        }
      }
    }
  }, [query.error, navigate]);

  return query;
}

