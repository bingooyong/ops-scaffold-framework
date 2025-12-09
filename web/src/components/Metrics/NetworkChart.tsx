/**
 * 网络流量趋势图表组件
 * 区分入口流量（接收）和出口流量（发送）
 */

import { useMemo, memo } from 'react';
import {
  AreaChart,
  Area,
  XAxis,
  YAxis,
  Tooltip,
  Legend,
  ResponsiveContainer,
} from 'recharts';
import { format } from 'date-fns';
import { Paper, Typography, Skeleton } from '@mui/material';

export interface NetworkChartDataPoint {
  timestamp: number;
  rxBytes: number; // 接收字节数
  txBytes: number; // 发送字节数
}

export interface NetworkChartProps {
  data: NetworkChartDataPoint[];
  title?: string;
  loading?: boolean;
  height?: number;
}

/**
 * 自定义 Tooltip 组件的 Props
 */
interface CustomTooltipProps {
  active?: boolean;
  payload?: Array<{
    name?: string;
    value?: number;
    color?: string;
    dataKey?: string;
  }>;
  label?: number | string;
}

/**
 * 自定义 Tooltip 组件
 */
function CustomTooltip({ active, payload, label }: CustomTooltipProps) {
  if (active && payload && payload.length) {
    const timestamp = label as number;
    const rxData = payload.find((p) => p.dataKey === 'rxBytes');
    const txData = payload.find((p) => p.dataKey === 'txBytes');

    return (
      <Paper
        elevation={3}
        sx={{
          p: 1.5,
          backgroundColor: 'rgba(255, 255, 255, 0.95)',
        }}
      >
        <Typography variant="body2" color="text.secondary">
          {format(new Date(timestamp), 'yyyy-MM-dd HH:mm:ss')}
        </Typography>
        {rxData && (
          <Typography variant="body2" sx={{ mt: 0.5, color: rxData.color }}>
            接收: {(rxData.value as number).toFixed(2)} MB
          </Typography>
        )}
        {txData && (
          <Typography variant="body2" sx={{ mt: 0.5, color: txData.color }}>
            发送: {(txData.value as number).toFixed(2)} MB
          </Typography>
        )}
        {rxData && txData && (
          <Typography variant="body2" sx={{ mt: 0.5, fontWeight: 'bold' }}>
            总计: {((rxData.value as number) + (txData.value as number)).toFixed(2)} MB
          </Typography>
        )}
      </Paper>
    );
  }
  return null;
}

/**
 * 格式化 X 轴时间刻度
 */
function formatXAxisTick(timestamp: number, timeRange: number): string {
  const date = new Date(timestamp);
  
  // 根据时间范围决定格式
  if (timeRange <= 15 * 60 * 1000) {
    // 15 分钟内：显示 HH:mm
    return format(date, 'HH:mm');
  } else if (timeRange <= 24 * 60 * 60 * 1000) {
    // 1 天内：显示 HH:mm
    return format(date, 'HH:mm');
  } else if (timeRange <= 7 * 24 * 60 * 60 * 1000) {
    // 7 天内：显示 MM-DD HH:mm
    return format(date, 'MM-dd HH:mm');
  } else {
    // 30 天内：显示 MM-DD
    return format(date, 'MM-dd');
  }
}

/**
 * 创建 X 轴格式化函数
 */
function createXAxisFormatter(timeRange: number) {
  return (timestamp: number) => formatXAxisTick(timestamp, timeRange);
}

/**
 * 格式化 Y 轴刻度
 */
function formatYAxisTick(value: number): string {
  if (value >= 1024) {
    return `${(value / 1024).toFixed(1)} GB`;
  }
  return `${value.toFixed(1)} MB`;
}

function NetworkChart({
  data,
  title = '网络流量趋势',
  loading,
  height = 300,
}: NetworkChartProps) {
  // 计算时间范围（用于决定 X 轴格式）
  const timeRange = useMemo(() => {
    if (data.length < 2) return 0;
    const first = data[0].timestamp;
    const last = data[data.length - 1].timestamp;
    return last - first;
  }, [data]);

  // 转换数据格式：将字节转换为 MB
  const chartData = useMemo(() => {
    return data.map((item) => ({
      timestamp: item.timestamp,
      rxBytes: item.rxBytes / (1024 * 1024), // 转换为 MB
      txBytes: item.txBytes / (1024 * 1024), // 转换为 MB
    }));
  }, [data]);

  if (loading) {
    return (
      <Paper elevation={2} sx={{ p: 2 }}>
        <Skeleton variant="text" width="40%" height={24} sx={{ mb: 2 }} />
        <Skeleton variant="rectangular" height={height} />
      </Paper>
    );
  }

  if (!data || data.length === 0) {
    return (
      <Paper elevation={2} sx={{ p: 2, textAlign: 'center' }}>
        <Typography variant="body2" color="text.secondary">
          暂无数据
        </Typography>
      </Paper>
    );
  }

  // 入口流量颜色（接收，通常用蓝色/绿色表示）
  const rxColor = '#2196F3'; // Material Blue
  // 出口流量颜色（发送，通常用橙色/红色表示）
  const txColor = '#FF9800'; // Material Orange

  return (
    <Paper elevation={2} sx={{ p: 2, height: '100%' }}>
      <Typography variant="h6" gutterBottom>
        {title}
      </Typography>
      <ResponsiveContainer width="100%" height={height}>
        <AreaChart
          data={chartData}
          margin={{ top: 10, right: 30, left: 20, bottom: 30 }}
        >
          <defs>
            {/* 入口流量渐变 */}
            <linearGradient id="gradient-rx" x1="0" y1="0" x2="0" y2="1">
              <stop offset="5%" stopColor={rxColor} stopOpacity={0.8} />
              <stop offset="95%" stopColor={rxColor} stopOpacity={0.1} />
            </linearGradient>
            {/* 出口流量渐变 */}
            <linearGradient id="gradient-tx" x1="0" y1="0" x2="0" y2="1">
              <stop offset="5%" stopColor={txColor} stopOpacity={0.8} />
              <stop offset="95%" stopColor={txColor} stopOpacity={0.1} />
            </linearGradient>
          </defs>
          <XAxis
            dataKey="timestamp"
            tickFormatter={createXAxisFormatter(timeRange)}
            style={{ fontSize: '12px' }}
          />
          <YAxis
            tickFormatter={formatYAxisTick}
            style={{ fontSize: '12px' }}
            label={{ value: '流量 (MB)', angle: -90, position: 'insideLeft' }}
          />
          <Tooltip content={<CustomTooltip />} />
          <Legend
            formatter={(value) => {
              if (value === 'rxBytes') return '接收流量';
              if (value === 'txBytes') return '发送流量';
              return value;
            }}
          />
          {/* 入口流量（接收） */}
          <Area
            type="monotone"
            dataKey="rxBytes"
            name="rxBytes"
            stroke={rxColor}
            strokeWidth={2}
            fill="url(#gradient-rx)"
            fillOpacity={0.4}
          />
          {/* 出口流量（发送） */}
          <Area
            type="monotone"
            dataKey="txBytes"
            name="txBytes"
            stroke={txColor}
            strokeWidth={2}
            fill="url(#gradient-tx)"
            fillOpacity={0.4}
          />
        </AreaChart>
      </ResponsiveContainer>
    </Paper>
  );
}

export default memo(NetworkChart);
