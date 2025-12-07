/**
 * 监控指标数据格式化工具
 */

import type { MetricData } from '../types';

/**
 * 格式化历史指标数据为 Recharts 所需格式
 */
export function formatMetricsHistory(
  metrics: MetricData[],
  valueKey: string
): { timestamp: number; value: number }[] {
  if (!metrics || metrics.length === 0) {
    return [];
  }

  return metrics.map((item) => {
    const timestamp = new Date(item.timestamp).getTime();
    const value = item.values[valueKey];
    
    // 处理缺失值，使用 0
    const numericValue = typeof value === 'number' ? value : 0;

    return {
      timestamp,
      value: numericValue,
    };
  });
}

