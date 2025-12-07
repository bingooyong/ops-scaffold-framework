/**
 * 监控指标工具函数
 */

import type { Theme } from '@mui/material/styles';

/**
 * 根据使用率获取颜色（CPU/内存）
 * < 60% 绿色 (success)
 * 60-80% 黄色 (warning)
 * > 80% 红色 (error)
 */
export function getUsageColor(usage: number, theme?: Theme): string {
  if (theme) {
    // 如果提供了 theme，返回实际颜色值
    if (usage < 60) {
      return theme.palette.success.main;
    } else if (usage < 80) {
      return theme.palette.warning.main;
    } else {
      return theme.palette.error.main;
    }
  } else {
    // 如果没有提供 theme，返回主题键（用于 MetricCard 组件）
    if (usage < 60) {
      return 'success.main';
    } else if (usage < 80) {
      return 'warning.main';
    } else {
      return 'error.main';
    }
  }
}

/**
 * 根据磁盘使用率获取颜色
 * < 85% 绿色 (success)
 * 85-95% 黄色 (warning)
 * > 95% 红色 (error)
 */
export function getDiskUsageColor(usage: number, theme?: Theme): string {
  if (theme) {
    // 如果提供了 theme，返回实际颜色值
    if (usage < 85) {
      return theme.palette.success.main;
    } else if (usage < 95) {
      return theme.palette.warning.main;
    } else {
      return theme.palette.error.main;
    }
  } else {
    // 如果没有提供 theme，返回主题键（用于 MetricCard 组件）
    if (usage < 85) {
      return 'success.main';
    } else if (usage < 95) {
      return 'warning.main';
    } else {
      return 'error.main';
    }
  }
}

/**
 * 格式化字节数为 GB
 */
export function formatBytesToGB(bytes: number): number {
  return Number((bytes / (1024 * 1024 * 1024)).toFixed(2));
}

/**
 * 格式化字节数为 MB
 */
export function formatBytesToMB(bytes: number): number {
  return Number((bytes / (1024 * 1024)).toFixed(2));
}

