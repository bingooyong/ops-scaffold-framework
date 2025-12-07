/**
 * 节点详情页面
 */

import { useState, useEffect, useMemo, useCallback, lazy, Suspense } from 'react';
import { useParams, Link, useNavigate } from 'react-router-dom';
import {
  Box,
  Container,
  Breadcrumbs,
  Paper,
  Grid,
  Typography,
  Chip,
  Tabs,
  Tab,
  CircularProgress,
  Alert,
  Button,
} from '@mui/material';
import {
  Home as HomeIcon,
  Storage as StorageIcon,
} from '@mui/icons-material';
import { useNode, useLatestMetrics, useMetricsHistory } from '../../hooks';
import { CPUCard, MemoryCard, DiskCard, NetworkCard } from '../../components/Metrics';
import TimeRangeSelector from '../../components/Metrics/TimeRangeSelector';
import RefreshControl from '../../components/Metrics/RefreshControl';
import { formatMetricsHistory } from '../../utils/metricsFormat';
import { useMetricsStore } from '../../stores';
import type { NodeStatus } from '../../types';

// 懒加载 Recharts 图表组件
const MetricsChart = lazy(() => import('../../components/Metrics/MetricsChart'));

interface TabPanelProps {
  children?: React.ReactNode;
  index: number;
  value: number;
}

function TabPanel(props: TabPanelProps) {
  const { children, value, index, ...other } = props;

  return (
    <div
      role="tabpanel"
      hidden={value !== index}
      id={`node-tabpanel-${index}`}
      aria-labelledby={`node-tab-${index}`}
      {...other}
    >
      {value === index && <Box sx={{ pt: 3 }}>{children}</Box>}
    </div>
  );
}

export default function NodeDetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [tabValue, setTabValue] = useState(1); // 默认选中"监控" Tab

  const { data, isLoading, error } = useNode(id || '');

  // 设置页面标题
  useEffect(() => {
    document.title = '节点详情 | Ops Manager';
  }, []);

  // 处理 Tab 切换
  const handleTabChange = (_event: React.SyntheticEvent, newValue: number) => {
    setTabValue(newValue);
  };

  // 如果节点不存在或加载失败
  if (error) {
    return (
      <Container maxWidth="lg">
        <Box sx={{ py: 4, textAlign: 'center' }}>
          <Typography variant="h6" color="error">
            加载节点信息失败
          </Typography>
          <Typography variant="body2" color="text.secondary" sx={{ mt: 2 }}>
            {error instanceof Error ? error.message : '未知错误'}
          </Typography>
        </Box>
      </Container>
    );
  }

  if (isLoading) {
    return (
      <Container maxWidth="lg">
        <Box sx={{ display: 'flex', justifyContent: 'center', py: 4 }}>
          <CircularProgress />
        </Box>
      </Container>
    );
  }

  const node = data?.data?.node;
  if (!node) {
    return (
      <Container maxWidth="lg">
        <Box sx={{ py: 4, textAlign: 'center' }}>
          <Typography variant="h6">节点不存在</Typography>
        </Box>
      </Container>
    );
  }

  // 获取状态颜色
  const getStatusColor = (status: NodeStatus) => {
    switch (status) {
      case 'online':
        return 'success';
      case 'offline':
        return 'error';
      default:
        return 'default';
    }
  };

  // 获取状态文本
  const getStatusText = (status: NodeStatus) => {
    switch (status) {
      case 'online':
        return '在线';
      case 'offline':
        return '离线';
      default:
        return '未知';
    }
  };

  return (
    <Container maxWidth="xl">
      {/* 面包屑导航 */}
      <Breadcrumbs aria-label="breadcrumb" sx={{ mb: 3 }}>
        <Link
          to="/dashboard"
          style={{
            textDecoration: 'none',
            color: 'inherit',
            display: 'flex',
            alignItems: 'center',
          }}
        >
          <HomeIcon sx={{ mr: 0.5 }} fontSize="inherit" />
          首页
        </Link>
        <Link
          to="/nodes"
          style={{
            textDecoration: 'none',
            color: 'inherit',
            display: 'flex',
            alignItems: 'center',
          }}
        >
          <StorageIcon sx={{ mr: 0.5 }} fontSize="inherit" />
          节点列表
        </Link>
        <Typography color="text.primary">节点详情</Typography>
      </Breadcrumbs>

      {/* 基本信息卡片 */}
      <Paper elevation={2} sx={{ p: 3, mb: 3 }}>
        <Typography variant="h5" gutterBottom>
          {node.hostname || node.node_id}
        </Typography>
        <Grid container spacing={2} sx={{ mt: 1 }}>
          <Grid item xs={12} sm={6}>
            <Typography variant="body2" color="text.secondary">
              节点 ID
            </Typography>
            <Typography variant="body1">{node.node_id}</Typography>
          </Grid>
          <Grid item xs={12} sm={6}>
            <Typography variant="body2" color="text.secondary">
              IP 地址
            </Typography>
            <Typography variant="body1">{node.ip}</Typography>
          </Grid>
          <Grid item xs={12} sm={6}>
            <Typography variant="body2" color="text.secondary">
              状态
            </Typography>
            <Chip
              label={getStatusText(node.status)}
              color={getStatusColor(node.status) as 'success' | 'error' | 'default'}
              size="small"
              sx={{ mt: 0.5 }}
            />
          </Grid>
          <Grid item xs={12} sm={6}>
            <Typography variant="body2" color="text.secondary">
              操作系统
            </Typography>
            <Typography variant="body1">
              {node.os} / {node.arch}
            </Typography>
          </Grid>
          <Grid item xs={12} sm={6}>
            <Typography variant="body2" color="text.secondary">
              Daemon 版本
            </Typography>
            <Typography variant="body1">-</Typography>
          </Grid>
          <Grid item xs={12} sm={6}>
            <Typography variant="body2" color="text.secondary">
              Agent 版本
            </Typography>
            <Typography variant="body1">-</Typography>
          </Grid>
        </Grid>
      </Paper>

      {/* Tab 导航 */}
      <Box sx={{ borderBottom: 1, borderColor: 'divider', mb: 3 }}>
        <Tabs value={tabValue} onChange={handleTabChange} aria-label="节点详情标签页">
          <Tab label="概览" />
          <Tab label="监控" />
          <Tab label="日志" />
        </Tabs>
      </Box>

      {/* Tab 内容 */}
      <TabPanel value={tabValue} index={0}>
        <Typography variant="body2" color="text.secondary">
          功能开发中
        </Typography>
      </TabPanel>

      <TabPanel value={tabValue} index={1}>
        <MetricsTabContent nodeId={node.node_id} />
      </TabPanel>

      <TabPanel value={tabValue} index={2}>
        <Typography variant="body2" color="text.secondary">
          功能开发中
        </Typography>
      </TabPanel>
    </Container>
  );
}

/**
 * 监控 Tab 内容组件
 */
function MetricsTabContent({ nodeId }: { nodeId: string }) {
  // 从 Zustand Store 获取时间范围和刷新间隔
  const { timeRange, refreshInterval, setTimeRange, setRefreshInterval } = useMetricsStore();

  const { data, isLoading, error, refetch: refetchLatest } = useLatestMetrics(nodeId);

  // 历史数据查询
  const cpuHistory = useMetricsHistory(nodeId, 'cpu', timeRange);
  const memoryHistory = useMetricsHistory(nodeId, 'memory', timeRange);
  const diskHistory = useMetricsHistory(nodeId, 'disk', timeRange);
  const networkHistory = useMetricsHistory(nodeId, 'network', timeRange);

  // 时间范围变化时，重新查询历史数据
  useEffect(() => {
    cpuHistory.refetch();
    memoryHistory.refetch();
    diskHistory.refetch();
    networkHistory.refetch();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [timeRange.startTime.getTime(), timeRange.endTime.getTime()]);

  // 定时刷新逻辑
  useEffect(() => {
    if (!refreshInterval) return;

    const timer = setInterval(() => {
      refetchLatest();
      cpuHistory.refetch();
      memoryHistory.refetch();
      diskHistory.refetch();
      networkHistory.refetch();
    }, refreshInterval);

    return () => clearInterval(timer);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [refreshInterval]);

  // 使用 useMemo 缓存格式化后的历史数据
  const cpuChartData = useMemo(
    () => formatMetricsHistory(cpuHistory.data?.data || [], 'usage_percent'),
    [cpuHistory.data?.data]
  );

  const memoryChartData = useMemo(
    () => formatMetricsHistory(memoryHistory.data?.data || [], 'usage_percent'),
    [memoryHistory.data?.data]
  );

  const diskChartData = useMemo(
    () => formatMetricsHistory(diskHistory.data?.data || [], 'usage_percent'),
    [diskHistory.data?.data]
  );

  // 网络数据需要特殊处理：计算 rx_bytes + tx_bytes
  const networkChartData = useMemo(() => {
    return (networkHistory.data?.data || []).map((item) => {
      const rxBytes = item.values?.rx_bytes || 0;
      const txBytes = item.values?.tx_bytes || 0;
      const totalBytes = rxBytes + txBytes;
      return {
        timestamp: new Date(item.timestamp).getTime(),
        value: totalBytes / (1024 * 1024), // 转换为 MB
      };
    });
  }, [networkHistory.data?.data]);

  // 刷新所有数据（使用 useCallback 避免引用变化）
  // 注意：必须在所有条件返回之前调用 hooks
  const handleRefresh = useCallback(() => {
    refetchLatest();
    cpuHistory.refetch();
    memoryHistory.refetch();
    diskHistory.refetch();
    networkHistory.refetch();
  }, [refetchLatest, cpuHistory, memoryHistory, diskHistory, networkHistory]);

  // 整体 Loading 状态
  if (isLoading && !data) {
    return (
      <Box display="flex" justifyContent="center" py={4}>
        <CircularProgress />
      </Box>
    );
  }

  // 整体 Error 状态
  if (error) {
    return (
      <Alert
        severity="error"
        action={
          <Button color="inherit" size="small" onClick={() => refetchLatest()}>
            重试
          </Button>
        }
        sx={{ mb: 3 }}
      >
        {error instanceof Error ? error.message : '加载监控数据失败'}
      </Alert>
    );
  }

  return (
    <Box>
      {/* 控制面板 */}
      <Paper
        elevation={1}
        sx={{
          p: 2,
          mb: 3,
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          flexWrap: 'wrap',
          gap: 2,
        }}
      >
        <TimeRangeSelector value={timeRange} onChange={setTimeRange} />
        <RefreshControl
          value={refreshInterval}
          onChange={setRefreshInterval}
          onRefresh={handleRefresh}
        />
      </Paper>

      {/* 实时指标卡片 */}
      <Grid container spacing={2} sx={{ mb: 3 }}>
        <Grid size={{ xs: 12, sm: 6, md: 3 }}>
          <CPUCard nodeId={nodeId} />
        </Grid>
        <Grid size={{ xs: 12, sm: 6, md: 3 }}>
          <MemoryCard nodeId={nodeId} />
        </Grid>
        <Grid size={{ xs: 12, sm: 6, md: 3 }}>
          <DiskCard nodeId={nodeId} />
        </Grid>
        <Grid size={{ xs: 12, sm: 6, md: 3 }}>
          <NetworkCard nodeId={nodeId} />
        </Grid>
      </Grid>

      {/* 历史趋势图表 */}
      <Grid container spacing={2.5}>
        <Grid size={{ xs: 12, lg: 6 }}>
          <Suspense fallback={<CircularProgress />}>
            <MetricsChart
              data={cpuChartData}
              title="CPU 使用率趋势"
              unit="%"
              color="#2196f3"
              loading={cpuHistory.isLoading}
              height={350}
            />
          </Suspense>
        </Grid>
        <Grid size={{ xs: 12, lg: 6 }}>
          <Suspense fallback={<CircularProgress />}>
            <MetricsChart
              data={memoryChartData}
              title="内存使用率趋势"
              unit="%"
              color="#4caf50"
              loading={memoryHistory.isLoading}
              height={350}
            />
          </Suspense>
        </Grid>
        <Grid size={{ xs: 12, lg: 6 }}>
          <Suspense fallback={<CircularProgress />}>
            <MetricsChart
              data={diskChartData}
              title="磁盘使用率趋势"
              unit="%"
              color="#ff9800"
              loading={diskHistory.isLoading}
              height={350}
            />
          </Suspense>
        </Grid>
        <Grid size={{ xs: 12, lg: 6 }}>
          <Suspense fallback={<CircularProgress />}>
            <MetricsChart
              data={networkChartData}
              title="网络流量趋势"
              unit="MB"
              color="#9c27b0"
              loading={networkHistory.isLoading}
              height={350}
            />
          </Suspense>
        </Grid>
      </Grid>
    </Box>
  );
}

