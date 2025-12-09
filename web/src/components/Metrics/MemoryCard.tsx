/**
 * 内存指标卡片组件
 */

import { useMemo } from 'react';
import { Storage as StorageIcon } from '@mui/icons-material';
import { useLatestMetrics } from '../../hooks';
import { getUsageColor, formatBytesToGB } from '../../utils/metricsUtils';
import MetricCard from './MetricCard';

interface MemoryCardProps {
  nodeId: string;
}

export default function MemoryCard({ nodeId }: MemoryCardProps) {
  const { data, isLoading, error } = useLatestMetrics(nodeId);

  const memoryData = data?.data?.memory;
  const values = memoryData?.values;

  // 使用 useMemo 缓存计算结果
  const { usedGB, totalGB, usagePercent, color, hasData } = useMemo(() => {
    if (!isLoading && !error && !memoryData) {
      return {
        usedGB: 0,
        totalGB: 0,
        usagePercent: 0,
        color: undefined,
        hasData: false,
      };
    }

    const usedBytes = (values?.used_bytes as number) || 0;
    const totalBytes = (values?.total_bytes as number) || 0;
    const usage = (values?.usage_percent as number) || 0;

    // 检查是否有有效数据
    const hasValidData = totalBytes > 0 || usedBytes > 0 || usage > 0;

    return {
      usedGB: formatBytesToGB(usedBytes),
      totalGB: formatBytesToGB(totalBytes),
      usagePercent: usage,
      color: getUsageColor(usage),
      hasData: hasValidData,
    };
  }, [isLoading, error, memoryData, values]);

  if (!isLoading && !error && !memoryData) {
    return (
      <MetricCard
        title="内存使用"
        value="-"
        icon={<StorageIcon />}
        loading={false}
      />
    );
  }

  // 如果没有有效数据，显示提示
  if (!isLoading && !error && !hasData) {
    return (
      <MetricCard
        title="内存使用"
        value="暂无数据"
        icon={<StorageIcon />}
        loading={false}
      />
    );
  }

  return (
    <MetricCard
      title="内存使用"
      value={`${usedGB} / ${totalGB}`}
      unit="GB"
      percentage={usagePercent}
      icon={<StorageIcon />}
      color={color}
      loading={isLoading}
      error={error?.message}
    />
  );
}

