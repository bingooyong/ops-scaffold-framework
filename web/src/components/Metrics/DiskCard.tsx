/**
 * 磁盘指标卡片组件
 */

import { useMemo } from 'react';
import { Storage as HardDriveIcon } from '@mui/icons-material';
import { useLatestMetrics } from '../../hooks';
import { getDiskUsageColor, formatBytesToGB } from '../../utils/metricsUtils';
import MetricCard from './MetricCard';

interface DiskCardProps {
  nodeId: string;
}

export default function DiskCard({ nodeId }: DiskCardProps) {
  const { data, isLoading, error } = useLatestMetrics(nodeId);

  const diskData = data?.data?.disk;
  const values = diskData?.values;

  // 使用 useMemo 缓存计算结果
  const { usedGB, totalGB, usagePercent, color, hasData } = useMemo(() => {
    if (!isLoading && !error && !diskData) {
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
      color: getDiskUsageColor(usage),
      hasData: hasValidData,
    };
  }, [isLoading, error, diskData, values]);

  if (!isLoading && !error && !diskData) {
    return (
      <MetricCard
        title="磁盘使用"
        value="-"
        icon={<HardDriveIcon />}
        loading={false}
      />
    );
  }

  // 如果没有有效数据，显示提示
  if (!isLoading && !error && !hasData) {
    return (
      <MetricCard
        title="磁盘使用"
        value="暂无数据"
        icon={<HardDriveIcon />}
        loading={false}
      />
    );
  }

  return (
    <MetricCard
      title="磁盘使用"
      value={`${usedGB} / ${totalGB}`}
      unit="GB"
      percentage={usagePercent}
      icon={<HardDriveIcon />}
      color={color}
      loading={isLoading}
      error={error?.message}
    />
  );
}

