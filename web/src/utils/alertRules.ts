/**
 * 告警规则工具函数
 */

import type { NodeMetrics } from '../types';

/**
 * 告警级别
 */
export type AlertLevel = 'normal' | 'warning' | 'critical';

/**
 * 告警对象
 */
export interface Alert {
  node_id: string;
  hostname: string;
  metric_type: string;
  current_value: number;
  level: AlertLevel;
  message: string;
}

/**
 * 检查单个指标的告警级别
 * @param metricType 指标类型 (cpu/memory/disk/network)
 * @param value 指标值（百分比）
 * @returns 告警级别和消息
 */
export function checkAlert(metricType: string, value: number): { level: AlertLevel; message: string } {
  // CPU/Memory 类型
  if (metricType === 'cpu' || metricType === 'memory') {
    if (value > 90) {
      return { level: 'critical', message: '资源使用严重' };
    } else if (value > 80) {
      return { level: 'warning', message: '资源使用偏高' };
    } else {
      return { level: 'normal', message: '正常' };
    }
  }

  // Disk 类型（更严格的阈值）
  if (metricType === 'disk') {
    if (value > 95) {
      return { level: 'critical', message: '磁盘使用严重' };
    } else if (value > 85) {
      return { level: 'warning', message: '磁盘使用偏高' };
    } else {
      return { level: 'normal', message: '正常' };
    }
  }

  // Network 类型（暂时返回 normal，后续可扩展）
  if (metricType === 'network') {
    // 可以基于流量阈值判断，这里暂时返回 normal
    return { level: 'normal', message: '正常' };
  }

  // 未知类型返回 normal
  return { level: 'normal', message: '正常' };
}

/**
 * 检查节点的所有指标告警
 * @param node 节点指标数据
 * @returns 告警数组（只包含非 normal 级别的告警）
 */
export function checkNodeAlerts(node: NodeMetrics): Alert[] {
  const alerts: Alert[] = [];

  // 检查 CPU 使用率
  if (node.cpu_usage !== undefined && node.cpu_usage !== null) {
    const { level, message } = checkAlert('cpu', node.cpu_usage);
    if (level !== 'normal') {
      alerts.push({
        node_id: node.node_id,
        hostname: node.hostname,
        metric_type: 'CPU',
        current_value: node.cpu_usage,
        level,
        message,
      });
    }
  }

  // 检查内存使用率
  if (node.memory_usage !== undefined && node.memory_usage !== null) {
    const { level, message } = checkAlert('memory', node.memory_usage);
    if (level !== 'normal') {
      alerts.push({
        node_id: node.node_id,
        hostname: node.hostname,
        metric_type: '内存',
        current_value: node.memory_usage,
        level,
        message,
      });
    }
  }

  // 检查磁盘使用率
  if (node.disk_usage !== undefined && node.disk_usage !== null) {
    const { level, message } = checkAlert('disk', node.disk_usage);
    if (level !== 'normal') {
      alerts.push({
        node_id: node.node_id,
        hostname: node.hostname,
        metric_type: '磁盘',
        current_value: node.disk_usage,
        level,
        message,
      });
    }
  }

  // Network 类型暂时不检查，后续可扩展

  return alerts;
}

