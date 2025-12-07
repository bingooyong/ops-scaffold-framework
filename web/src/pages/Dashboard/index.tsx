/**
 * Dashboard 页面
 */

import { Box, Container, Grid, Typography, Alert, Button } from '@mui/material';
import {
  Memory as CpuIcon,
  Storage as MemoryIcon,
  Dns as DiskIcon,
  Devices as DevicesIcon,
} from '@mui/icons-material';
import { useTheme } from '@mui/material/styles';
import { format } from 'date-fns';
import { zhCN } from 'date-fns/locale';
import { useClusterOverview } from '../../hooks';
import MetricCard from '../../components/Metrics/MetricCard';
import { TopNodesCard, AlertsPanel } from '../../components/Dashboard';
import { RefreshControl } from '../../components/Metrics';
import { getUsageColor, getDiskUsageColor, formatBytesToGB } from '../../utils/metricsUtils';
import { useMetricsStore } from '../../stores';

export default function Dashboard() {
  const theme = useTheme();
  const { data, isLoading, error, refetch, dataUpdatedAt } = useClusterOverview();
  const { refreshInterval, setRefreshInterval } = useMetricsStore();

  const aggregate = data?.data?.aggregate;
  const nodes = data?.data?.nodes || [];

  // 计算节点在线率
  const onlineRate = aggregate?.node_counts.total
    ? (aggregate.node_counts.online / aggregate.node_counts.total) * 100
    : 0;

  // 节点状态颜色
  const nodeStatusColor =
    onlineRate > 90
      ? theme.palette.success.main
      : onlineRate > 70
      ? theme.palette.warning.main
      : theme.palette.error.main;

  // 处理刷新
  const handleRefresh = () => {
    refetch();
  };

  // 格式化最后更新时间
  const formatLastUpdateTime = (timestamp: number) => {
    if (!timestamp) return '-';
    return format(new Date(timestamp), 'yyyy-MM-dd HH:mm:ss', { locale: zhCN });
  };

  return (
    <Container maxWidth={false} sx={{ maxWidth: '1400px', px: 3 }}>
      {/* 页面标题和刷新控制 */}
      <Box
        sx={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          mb: 3,
          flexWrap: 'wrap',
          gap: 2,
        }}
      >
        <Typography variant="h4">集群监控 Dashboard</Typography>
        <RefreshControl
          value={refreshInterval}
          onChange={setRefreshInterval}
          onRefresh={handleRefresh}
        />
      </Box>

      {/* 错误提示 */}
      {error && (
        <Alert
          severity="error"
          action={
            <Button color="inherit" size="small" onClick={() => refetch()}>
              重试
            </Button>
          }
          sx={{ mb: 3 }}
        >
          加载集群数据失败: {error instanceof Error ? error.message : '未知错误'}
        </Alert>
      )}

      {/* 集群资源概览卡片 */}
      <Grid container spacing={2.5} sx={{ mt: 0.5 }}>
        {/* 卡片1 - 集群平均 CPU 使用率 */}
        <Grid size={{ xs: 12, sm: 6, md: 3 }}>
          <MetricCard
            title="集群平均 CPU"
            value={isLoading ? '-' : aggregate?.avg_cpu?.toFixed(1) || '0.0'}
            unit="%"
            percentage={aggregate?.avg_cpu || 0}
            icon={<CpuIcon fontSize="large" />}
            color={aggregate?.avg_cpu ? getUsageColor(aggregate.avg_cpu, theme) : undefined}
            loading={isLoading}
            error={error?.message}
          />
        </Grid>

        {/* 卡片2 - 集群总内存使用 */}
        <Grid size={{ xs: 12, sm: 6, md: 3 }}>
          <MetricCard
            title="集群总内存"
            value={
              isLoading
                ? '-'
                : aggregate?.total_memory_gb
                ? `${aggregate.total_memory_gb.toFixed(2)}`
                : '0.00'
            }
            unit="GB"
            percentage={0} // 总内存不显示百分比
            icon={<MemoryIcon fontSize="large" />}
            color={theme.palette.info.main}
            loading={isLoading}
            error={error?.message}
          />
        </Grid>

        {/* 卡片3 - 集群总磁盘使用 */}
        <Grid size={{ xs: 12, sm: 6, md: 3 }}>
          <MetricCard
            title="集群总磁盘"
            value={
              isLoading
                ? '-'
                : aggregate?.total_disk_gb
                ? `${aggregate.total_disk_gb.toFixed(2)}`
                : '0.00'
            }
            unit="GB"
            percentage={0} // 总磁盘不显示百分比
            icon={<DiskIcon fontSize="large" />}
            color={theme.palette.info.main}
            loading={isLoading}
            error={error?.message}
          />
        </Grid>

        {/* 卡片4 - 节点状态统计 */}
        <Grid size={{ xs: 12, sm: 6, md: 3 }}>
          <MetricCard
            title="节点状态"
            value={
              isLoading
                ? '-'
                : `${aggregate?.node_counts.online || 0} / ${aggregate?.node_counts.total || 0}`
            }
            unit=""
            percentage={onlineRate}
            icon={<DevicesIcon fontSize="large" />}
            color={nodeStatusColor}
            loading={isLoading}
            error={error?.message}
            extraInfo={
              aggregate?.node_counts.offline ? (
                <Typography variant="caption" color="text.secondary">
                  离线: {aggregate.node_counts.offline}
                </Typography>
              ) : undefined
            }
          />
        </Grid>
      </Grid>

      {/* Top 节点排名和告警面板 */}
      <Grid container spacing={2.5} sx={{ mt: 0.5 }}>
        <Grid size={{ xs: 12, lg: 6 }}>
          <TopNodesCard nodes={nodes} loading={isLoading} />
        </Grid>
        <Grid size={{ xs: 12, lg: 6 }}>
          <AlertsPanel nodes={nodes} loading={isLoading} />
        </Grid>
      </Grid>

      {/* 最后更新时间 */}
      {dataUpdatedAt && !isLoading && (
        <Box sx={{ mt: 3, textAlign: 'right' }}>
          <Typography variant="caption" color="text.secondary">
            最后更新: {formatLastUpdateTime(dataUpdatedAt)}
          </Typography>
        </Box>
      )}
    </Container>
  );
}
