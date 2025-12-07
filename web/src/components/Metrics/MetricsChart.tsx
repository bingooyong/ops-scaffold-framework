/**
 * 监控指标趋势图表组件
 */

import { useMemo, memo } from 'react';
import {
  AreaChart,
  Area,
  XAxis,
  YAxis,
  Tooltip,
  ResponsiveContainer,
  Legend,
} from 'recharts';
import { format } from 'date-fns';
import { Paper, Typography, Box, Skeleton } from '@mui/material';

export interface MetricsChartProps {
  data: { timestamp: number; value: number }[];
  title: string;
  unit: string;
  color: string;
  loading?: boolean;
  height?: number;
}

/**
 * 自定义 Tooltip 组件
 */
function CustomTooltip({ active, payload, label }: any) {
  if (active && payload && payload.length) {
    const timestamp = label as number;
    const value = payload[0].value as number;
    const unit = payload[0].payload?.unit || '';

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
        <Typography variant="body1" sx={{ fontWeight: 'bold', mt: 0.5 }}>
          {value.toFixed(2)} {unit}
        </Typography>
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
 * 返回一个接受 timestamp 并返回格式化字符串的函数
 */
function createXAxisFormatter(timeRange: number) {
  return (timestamp: number) => formatXAxisTick(timestamp, timeRange);
}

/**
 * 格式化 Y 轴刻度
 */
function formatYAxisTick(value: number, unit: string): string {
  if (unit === '%') {
    return `${value}%`;
  }
  return `${value} ${unit}`;
}

function MetricsChart({
  data,
  title,
  unit,
  color,
  loading,
  height = 300,
}: MetricsChartProps) {
  // 计算时间范围（用于决定 X 轴格式）
  const timeRange = useMemo(() => {
    if (data.length < 2) return 0;
    const first = data[0].timestamp;
    const last = data[data.length - 1].timestamp;
    return last - first;
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

  // 为数据添加 unit 属性（用于 Tooltip）
  const chartData = data.map((item) => ({
    ...item,
    unit,
  }));

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
            <linearGradient id={`gradient-${color}`} x1="0" y1="0" x2="0" y2="1">
              <stop offset="5%" stopColor={color} stopOpacity={0.8} />
              <stop offset="95%" stopColor={color} stopOpacity={0.1} />
            </linearGradient>
          </defs>
          <XAxis
            dataKey="timestamp"
            tickFormatter={createXAxisFormatter(timeRange)}
            style={{ fontSize: '12px' }}
          />
          <YAxis
            tickFormatter={(value) => formatYAxisTick(value, unit)}
            style={{ fontSize: '12px' }}
          />
          <Tooltip content={<CustomTooltip />} />
          <Area
            type="monotone"
            dataKey="value"
            stroke={color}
            fill={`url(#gradient-${color})`}
            fillOpacity={0.6}
          />
        </AreaChart>
      </ResponsiveContainer>
    </Paper>
  );
}

export default memo(MetricsChart);

