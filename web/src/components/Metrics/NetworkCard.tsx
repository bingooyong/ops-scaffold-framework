/**
 * 网络指标卡片组件
 */

import { useMemo } from 'react';
import { NetworkCheck as NetworkCheckIcon } from '@mui/icons-material';
import { useLatestMetrics } from '../../hooks';
import { formatBytesToMB } from '../../utils/metricsUtils';
import MetricCard from './MetricCard';

interface NetworkCardProps {
  nodeId: string;
}

export default function NetworkCard({ nodeId }: NetworkCardProps) {
  const { data, isLoading, error } = useLatestMetrics(nodeId);

  const networkData = data?.data?.network;
  const values = networkData?.values;

  // 使用 useMemo 缓存计算结果
  const { totalMB, extraInfo, hasData } = useMemo(() => {
    if (!isLoading && !error && !networkData) {
      return {
        totalMB: 0,
        extraInfo: undefined,
        hasData: false,
      };
    }

    const rxBytes = values?.rx_bytes || 0;
    const txBytes = values?.tx_bytes || 0;
    const totalBytes = rxBytes + txBytes;

    // 检查是否有有效数据
    const hasValidData = totalBytes > 0 || rxBytes > 0 || txBytes > 0;

    const rxMB = formatBytesToMB(rxBytes);
    const txMB = formatBytesToMB(txBytes);
    const total = formatBytesToMB(totalBytes);

    return {
      totalMB: total,
      extraInfo: `接收: ${rxMB} MB • 发送: ${txMB} MB`,
      hasData: hasValidData,
    };
  }, [isLoading, error, networkData, values]);

  if (!isLoading && !error && !networkData) {
    return (
      <MetricCard
        title="网络流量"
        value="-"
        icon={<NetworkCheckIcon />}
        loading={false}
      />
    );
  }

  // 如果没有有效数据，显示提示
  if (!isLoading && !error && !hasData) {
    return (
      <MetricCard
        title="网络流量"
        value="暂无数据"
        icon={<NetworkCheckIcon />}
        loading={false}
      />
    );
  }

  // 网络流量使用固定颜色（蓝色）
  const color = 'primary.main';

  return (
    <MetricCard
      title="网络流量"
      value={totalMB.toFixed(2)}
      unit="MB"
      icon={<NetworkCheckIcon />}
      color={color}
      loading={isLoading}
      error={error?.message}
      extraInfo={extraInfo}
    />
  );
}

