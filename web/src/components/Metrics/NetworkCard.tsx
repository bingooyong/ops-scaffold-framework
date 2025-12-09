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

  // 使用 useMemo 缓存计算结果
  const { displayValue, displayUnit, extraInfo, hasData } = useMemo(() => {
    // 类型安全地获取网络数据
    const networkData = data?.data?.network;
    const values = networkData?.values;

    if (isLoading || error || !networkData) {
      return {
        displayValue: '-',
        displayUnit: undefined,
        extraInfo: undefined,
        hasData: false,
      };
    }

    const rxBytes = (values?.rx_bytes as number) || 0;
    const txBytes = (values?.tx_bytes as number) || 0;
    const totalBytes = rxBytes + txBytes;
    const intervalSeconds = values?.interval_seconds as number | undefined;

    // 检查是否有有效数据
    const hasValidData = totalBytes > 0 || rxBytes > 0 || txBytes > 0;

    const rxMB = formatBytesToMB(rxBytes);
    const txMB = formatBytesToMB(txBytes);
    const total = formatBytesToMB(totalBytes);

    // 格式化时间间隔
    let timeRangeText = '';
    if (intervalSeconds && intervalSeconds > 0) {
      if (intervalSeconds < 60) {
        timeRangeText = `${Math.round(intervalSeconds)}秒`;
      } else if (intervalSeconds < 3600) {
        timeRangeText = `${Math.round(intervalSeconds / 60)}分钟`;
      } else {
        timeRangeText = `${(intervalSeconds / 3600).toFixed(1)}小时`;
      }
    }

    // 计算速率（如果知道采集间隔）
    let rateText = '';
    if (intervalSeconds && intervalSeconds > 0) {
      const rxRate = rxBytes / intervalSeconds;
      const txRate = txBytes / intervalSeconds;
      const rxRateMB = formatBytesToMB(rxRate);
      const txRateMB = formatBytesToMB(txRate);
      rateText = `速率: ${rxRateMB.toFixed(2)} MB/s ↓ / ${txRateMB.toFixed(2)} MB/s ↑`;
    }

    // 构建额外信息：优先显示速率，然后显示累计值和时间范围
    // 格式：速率: XXX MB/s ↓ / XXX MB/s ↑ • 接收: XXX MB / 发送: XXX MB [过去XX分钟]
    let displayValue: string;
    let displayUnit: string | undefined;
    const extraInfoParts: string[] = [];

    if (intervalSeconds && intervalSeconds > 0 && rateText) {
      // 如果有速率信息，优先显示速率作为主值
      const rxRate = rxBytes / intervalSeconds;
      const txRate = txBytes / intervalSeconds;
      const totalRate = rxRate + txRate;
      const totalRateMB = formatBytesToMB(totalRate);
      displayValue = totalRateMB.toFixed(2);
      displayUnit = 'MB/s';
      extraInfoParts.push(rateText);
      extraInfoParts.push(`接收: ${rxMB.toFixed(2)} MB • 发送: ${txMB.toFixed(2)} MB`);
      if (timeRangeText) {
        extraInfoParts.push(`[过去${timeRangeText}]`);
      }
    } else {
      // 如果没有速率信息，显示累计值
      displayValue = total.toFixed(2);
      displayUnit = 'MB';
      extraInfoParts.push(`接收: ${rxMB.toFixed(2)} MB • 发送: ${txMB.toFixed(2)} MB`);
      if (timeRangeText) {
        extraInfoParts.push(`[过去${timeRangeText}]`);
      }
    }

    return {
      totalMB: total,
      displayValue,
      displayUnit,
      extraInfo: extraInfoParts.join(' • '),
      hasData: hasValidData,
    };
  }, [isLoading, error, data]);

  const networkDataForCheck = data?.data?.network;
  if (!isLoading && !error && !networkDataForCheck) {
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
      value={displayValue}
      unit={displayUnit}
      icon={<NetworkCheckIcon />}
      color={color}
      loading={isLoading}
      error={error?.message}
      extraInfo={extraInfo}
    />
  );
}

