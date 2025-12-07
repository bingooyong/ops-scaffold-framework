/**
 * CPU 指标卡片组件
 */

import { useMemo } from 'react';
import { Memory as CpuIcon } from '@mui/icons-material';
import { useLatestMetrics } from '../../hooks';
import { getUsageColor } from '../../utils/metricsUtils';
import MetricCard from './MetricCard';

interface CPUCardProps {
  nodeId: string;
}

export default function CPUCard({ nodeId }: CPUCardProps) {
  const { data, isLoading, error } = useLatestMetrics(nodeId);

  const cpuData = data?.data?.cpu;
  const values = cpuData?.values;

  // 使用 useMemo 缓存计算结果
  const { usagePercent, color, extraInfo } = useMemo(() => {
    if (!isLoading && !error && !cpuData) {
      return {
        usagePercent: 0,
        color: undefined,
        extraInfo: undefined,
      };
    }

    const usage = values?.usage_percent || 0;
    const cores = values?.cores;
    const model = values?.model;
    const colorValue = getUsageColor(usage);

    const extra = (
      <>
        {cores && `核心数: ${cores}`}
        {cores && model && ' • '}
        {model && `型号: ${model}`}
      </>
    );

    return {
      usagePercent: usage,
      color: colorValue,
      extraInfo: extra,
    };
  }, [isLoading, error, cpuData, values]);

  if (!isLoading && !error && !cpuData) {
    return (
      <MetricCard
        title="CPU 使用率"
        value="-"
        icon={<CpuIcon />}
        loading={false}
      />
    );
  }

  return (
    <MetricCard
      title="CPU 使用率"
      value={usagePercent.toFixed(1)}
      unit="%"
      percentage={usagePercent}
      icon={<CpuIcon />}
      color={color}
      loading={isLoading}
      error={error?.message}
      extraInfo={extraInfo || undefined}
    />
  );
}

