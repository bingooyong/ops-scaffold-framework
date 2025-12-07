/**
 * 监控指标状态管理 Store
 */

import { create } from 'zustand';
import type { TimeRange } from '../types';

interface MetricsState {
  timeRange: TimeRange;
  refreshInterval: number | null;

  // Actions
  setTimeRange: (range: TimeRange) => void;
  setRefreshInterval: (interval: number | null) => void;
}

export const useMetricsStore = create<MetricsState>((set) => ({
  // 默认时间范围：最近 1 小时
  timeRange: {
    startTime: new Date(Date.now() - 3600000),
    endTime: new Date(),
  },
  // 默认刷新间隔：30 秒
  refreshInterval: 30000,

  // 设置时间范围
  setTimeRange: (range) => {
    set({ timeRange: range });
  },

  // 设置刷新间隔
  setRefreshInterval: (interval) => {
    set({ refreshInterval: interval });
  },
}));

