/**
 * 节点详情页面
 */

import { useState, useEffect, useMemo, useCallback, lazy, Suspense } from 'react';
import { useParams, Link } from 'react-router-dom';
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
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  IconButton,
  Tooltip,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  ToggleButton,
  ToggleButtonGroup,
} from '@mui/material';
import {
  Home as HomeIcon,
  Storage as StorageIcon,
  Refresh as RefreshIcon,
  Sync as SyncIcon,
  PlayArrow as PlayArrowIcon,
  Stop as StopIcon,
  RestartAlt as RestartAltIcon,
  Description as DescriptionIcon,
  Download as DownloadIcon,
} from '@mui/icons-material';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useNode, useLatestMetrics, useMetricsHistory } from '../../hooks';
import { useAgentMetricsHistory } from '../../hooks/useAgentMetrics';
import { CPUCard, MemoryCard, DiskCard, NetworkCard } from '../../components/Metrics';
import TimeRangeSelector from '../../components/Metrics/TimeRangeSelector';
import RefreshControl from '../../components/Metrics/RefreshControl';
import { formatMetricsHistory } from '../../utils/metricsFormat';
import { useMetricsStore } from '../../stores';
import { listAgents, operateAgent, getAgentLogs, syncAgents } from '../../api';
import type { NodeStatus, Agent, AgentOperation, TimeRange } from '../../types';

// 懒加载 Recharts 图表组件
const MetricsChart = lazy(() => import('../../components/Metrics/MetricsChart'));
const NetworkChart = lazy(() => import('../../components/Metrics/NetworkChart'));

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
          <Grid size={{ xs: 12, sm: 6 }}>
            <Typography variant="body2" color="text.secondary">
              节点 ID
            </Typography>
            <Typography variant="body1">{node.node_id}</Typography>
          </Grid>
          <Grid size={{ xs: 12, sm: 6 }}>
            <Typography variant="body2" color="text.secondary">
              IP 地址
            </Typography>
            <Typography variant="body1">{node.ip}</Typography>
          </Grid>
          <Grid size={{ xs: 12, sm: 6 }}>
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
          <Grid size={{ xs: 12, sm: 6 }}>
            <Typography variant="body2" color="text.secondary">
              操作系统
            </Typography>
            <Typography variant="body1">
              {node.os} / {node.arch}
            </Typography>
          </Grid>
          <Grid size={{ xs: 12, sm: 6 }}>
            <Typography variant="body2" color="text.secondary">
              Daemon 版本
            </Typography>
            <Typography variant="body1">{node.daemon_version || '-'}</Typography>
          </Grid>
          <Grid size={{ xs: 12, sm: 6 }}>
            <Typography variant="body2" color="text.secondary">
              Agent 版本
            </Typography>
            <Typography variant="body1">{node.agent_version || '-'}</Typography>
          </Grid>
        </Grid>
      </Paper>

      {/* Tab 导航 */}
      <Box sx={{ borderBottom: 1, borderColor: 'divider', mb: 3 }}>
        <Tabs value={tabValue} onChange={handleTabChange} aria-label="节点详情标签页">
          <Tab label="概览" />
          <Tab label="监控" />
          <Tab label="日志" />
          <Tab label="Agents" />
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

      <TabPanel value={tabValue} index={3}>
        <AgentsTabContent nodeId={node.node_id} />
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

  // 网络数据需要特殊处理：区分入口流量（rx_bytes）和出口流量（tx_bytes）
  const networkChartData = useMemo(() => {
    return (networkHistory.data?.data || []).map((item) => {
      const rxBytes = (item.values?.rx_bytes as number) || 0;
      const txBytes = (item.values?.tx_bytes as number) || 0;
      return {
        timestamp: new Date(item.timestamp).getTime(),
        rxBytes, // 接收字节数（入口流量）
        txBytes, // 发送字节数（出口流量）
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
      <Grid container spacing={2} sx={{ mb: 3 }} alignItems="stretch">
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
            <NetworkChart
              data={networkChartData}
              title="网络流量趋势"
              loading={networkHistory.isLoading}
              height={350}
            />
          </Suspense>
        </Grid>
      </Grid>
    </Box>
  );
}

/**
 * Agents Tab 内容组件
 */
export function AgentsTabContent({ nodeId }: { nodeId: string }) {
  const queryClient = useQueryClient();
  const [successMessage, setSuccessMessage] = useState<string | null>(null);
  const [logDialogOpen, setLogDialogOpen] = useState(false);
  const [viewingAgentId, setViewingAgentId] = useState<string | null>(null);
  const [autoRefresh, setAutoRefresh] = useState(true);
  const [selectedAgentId, setSelectedAgentId] = useState<string | null>(null);
  const [timeRangePreset, setTimeRangePreset] = useState<'1h' | '6h' | '24h'>('1h');

  // 获取 Agent 列表
  const {
    data,
    isLoading,
    error,
    refetch,
    isRefetching,
  } = useQuery({
    queryKey: ['agents', nodeId],
    queryFn: () => listAgents(nodeId),
    refetchInterval: 30000, // 每 30 秒自动刷新
  });

  // 计算时间范围
  const getTimeRange = useCallback((preset: '1h' | '6h' | '24h'): TimeRange => {
    const now = new Date();
    const hours = preset === '1h' ? 1 : preset === '6h' ? 6 : 24;
    const startTime = new Date(now.getTime() - hours * 60 * 60 * 1000);
    return { startTime, endTime: now };
  }, []);

  const agentTimeRange = useMemo(() => getTimeRange(timeRangePreset), [timeRangePreset, getTimeRange]);

  // 获取 Agent 指标数据
  const cpuHistory = useAgentMetricsHistory(nodeId, selectedAgentId, agentTimeRange, 'cpu');
  const memoryHistory = useAgentMetricsHistory(nodeId, selectedAgentId, agentTimeRange, 'memory');
  const openFilesHistory = useAgentMetricsHistory(nodeId, selectedAgentId, agentTimeRange, 'open_files');
  const diskIOHistory = useAgentMetricsHistory(nodeId, selectedAgentId, agentTimeRange, 'disk_io');

  // 格式化图表数据
  const cpuChartData = useMemo(() => {
    if (!cpuHistory.data?.data?.data_points) return [];
    return cpuHistory.data.data.data_points.map((dp) => ({
      timestamp: new Date(dp.timestamp).getTime(),
      value: dp.values?.usage_percent || 0,
    }));
  }, [cpuHistory.data?.data?.data_points]);

  const memoryChartData = useMemo(() => {
    if (!memoryHistory.data?.data?.data_points) return [];
    return memoryHistory.data.data.data_points.map((dp) => ({
      timestamp: new Date(dp.timestamp).getTime(),
      value: dp.values?.usage_percent || 0,
    }));
  }, [memoryHistory.data?.data?.data_points]);

  const openFilesChartData = useMemo(() => {
    if (!openFilesHistory.data?.data?.data_points) return [];
    return openFilesHistory.data.data.data_points.map((dp) => ({
      timestamp: new Date(dp.timestamp).getTime(),
      value: dp.values?.count || 0,
    }));
  }, [openFilesHistory.data?.data?.data_points]);

  const diskReadChartData = useMemo(() => {
    if (!diskIOHistory.data?.data?.data_points) return [];
    return diskIOHistory.data.data.data_points.map((dp) => ({
      timestamp: new Date(dp.timestamp).getTime(),
      value: dp.values?.read_mb || 0,
    }));
  }, [diskIOHistory.data?.data?.data_points]);

  const diskWriteChartData = useMemo(() => {
    if (!diskIOHistory.data?.data?.data_points) return [];
    return diskIOHistory.data.data.data_points.map((dp) => ({
      timestamp: new Date(dp.timestamp).getTime(),
      value: dp.values?.write_mb || 0,
    }));
  }, [diskIOHistory.data?.data?.data_points]);

  // 当 Agent 列表加载完成时，默认选中第一个 Agent
  useEffect(() => {
    if (data?.data?.agents && data.data.agents.length > 0 && !selectedAgentId) {
      setSelectedAgentId(data.data.agents[0].agent_id);
    }
  }, [data?.data?.agents, selectedAgentId]);

  // 时间范围变化时，重新获取数据
  useEffect(() => {
    if (selectedAgentId) {
      cpuHistory.refetch();
      memoryHistory.refetch();
      openFilesHistory.refetch();
      diskIOHistory.refetch();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [agentTimeRange.startTime.getTime(), agentTimeRange.endTime.getTime()]);

  // 操作 Agent 的 mutation
  const operateMutation = useMutation({
    mutationFn: ({ agentId, operation }: { agentId: string; operation: AgentOperation }) =>
      operateAgent(nodeId, agentId, operation),
    onSuccess: (_data, variables) => {
      // 操作成功后刷新列表
      queryClient.invalidateQueries({ queryKey: ['agents', nodeId] });
      // 显示成功提示
      const operationText = {
        start: '启动',
        stop: '停止',
        restart: '重启',
      }[variables.operation];
      setSuccessMessage(`Agent ${operationText}成功`);
      // 3 秒后清除成功消息
      setTimeout(() => setSuccessMessage(null), 3000);
    },
    onError: (error) => {
      // 错误处理在 UI 中通过 mutation.error 显示
      console.error('操作 Agent 失败:', error);
    },
  });

  // 同步 Agent 状态的 mutation
  const syncMutation = useMutation({
    mutationFn: () => syncAgents(nodeId),
    onSuccess: (response) => {
      // 同步成功后刷新列表
      queryClient.invalidateQueries({ queryKey: ['agents', nodeId] });
      // 显示成功提示
      const syncedCount = response.data?.synced_count || 0;
      setSuccessMessage(`同步成功，已更新 ${syncedCount} 个 Agent 状态`);
      // 3 秒后清除成功消息
      setTimeout(() => setSuccessMessage(null), 3000);
    },
    onError: (error) => {
      console.error('同步 Agent 状态失败:', error);
    },
  });

  // 处理操作按钮点击
  const handleOperate = (agentId: string, operation: AgentOperation) => {
    operateMutation.mutate({ agentId, operation });
  };

  // 处理查看日志按钮点击
  const handleViewLogs = (agentId: string) => {
    setViewingAgentId(agentId);
    setLogDialogOpen(true);
  };

  // 处理关闭日志 Dialog
  const handleCloseLogDialog = () => {
    setLogDialogOpen(false);
    setViewingAgentId(null);
  };

  // 获取状态颜色
  const getStatusColor = (status: string) => {
    switch (status) {
      case 'running':
        return 'success';
      case 'stopped':
        return 'default';
      case 'error':
      case 'failed':
        return 'error';
      case 'starting':
      case 'stopping':
      case 'restarting':
        return 'warning';
      default:
        return 'default';
    }
  };

  // 获取状态文本
  const getStatusText = (status: string) => {
    switch (status) {
      case 'running':
        return '运行中';
      case 'stopped':
        return '已停止';
      case 'error':
      case 'failed':
        return '错误';
      case 'starting':
        return '启动中';
      case 'stopping':
        return '停止中';
      case 'restarting':
        return '重启中';
      default:
        return status;
    }
  };

  // 格式化时间
  const formatTime = (timeStr?: string) => {
    if (!timeStr) return '-';
    try {
      const date = new Date(timeStr);
      return date.toLocaleString('zh-CN', {
        year: 'numeric',
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit',
      });
    } catch {
      return timeStr;
    }
  };

  // 加载状态
  if (isLoading && !data) {
    return (
      <Box display="flex" justifyContent="center" py={4}>
        <CircularProgress />
      </Box>
    );
  }

  // 错误状态
  if (error) {
    return (
      <Alert
        severity="error"
        action={
          <Button color="inherit" size="small" onClick={() => refetch()}>
            重试
          </Button>
        }
        sx={{ mb: 3 }}
      >
        {error instanceof Error ? error.message : '加载 Agent 列表失败'}
      </Alert>
    );
  }

  const agents = data?.data?.agents || [];

  return (
    <Box>
      {/* 成功提示 */}
      {successMessage && (
        <Alert severity="success" sx={{ mb: 2 }} onClose={() => setSuccessMessage(null)}>
          {successMessage}
        </Alert>
      )}

      {/* 操作错误提示 */}
      {operateMutation.error && (
        <Alert
          severity="error"
          sx={{ mb: 2 }}
          onClose={() => operateMutation.reset()}
        >
          {operateMutation.error instanceof Error
            ? operateMutation.error.message
            : '操作失败'}
        </Alert>
      )}

      {/* 同步错误提示 */}
      {syncMutation.error && (
        <Alert
          severity="error"
          sx={{ mb: 2 }}
          onClose={() => syncMutation.reset()}
        >
          {syncMutation.error instanceof Error
            ? syncMutation.error.message
            : '同步失败，请检查 Daemon 是否在线'}
        </Alert>
      )}

      {/* 工具栏 */}
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
        <Typography variant="h6">Agent 列表</Typography>
        <Box sx={{ display: 'flex', gap: 1 }}>
          <Tooltip title="刷新列表（从数据库查询）">
            <IconButton onClick={() => refetch()} disabled={isRefetching} size="small">
              <RefreshIcon />
            </IconButton>
          </Tooltip>
          <Tooltip title="同步状态（从 Daemon 获取最新状态）">
            <IconButton 
              onClick={() => syncMutation.mutate()} 
              disabled={syncMutation.isPending} 
              size="small"
              color={syncMutation.isPending ? 'default' : 'primary'}
            >
              {syncMutation.isPending ? (
                <CircularProgress size={20} />
              ) : (
                <SyncIcon />
              )}
            </IconButton>
          </Tooltip>
        </Box>
      </Box>

      {/* Agent 列表表格 */}
      {agents.length === 0 ? (
        <Paper sx={{ p: 4, textAlign: 'center' }}>
          <Typography variant="body2" color="text.secondary">
            该节点下暂无 Agent
          </Typography>
        </Paper>
      ) : (
        <TableContainer component={Paper} sx={{ overflow: 'auto' }}>
          <Table>
            <TableHead>
              <TableRow>
                <TableCell>Agent ID</TableCell>
                <TableCell>类型</TableCell>
                <TableCell>版本</TableCell>
                <TableCell>状态</TableCell>
                <TableCell>PID</TableCell>
                <TableCell>最后心跳</TableCell>
                <TableCell align="right">操作</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {agents.map((agent: Agent) => {
                const isStartDisabled = agent.status === 'running' || operateMutation.isPending;
                const isStopDisabled = agent.status === 'stopped' || operateMutation.isPending;
                const isRestartDisabled = agent.status === 'stopped' || operateMutation.isPending;
                const isSelected = selectedAgentId === agent.agent_id;

                return (
                  <TableRow
                    key={agent.id}
                    hover
                    selected={isSelected}
                    onClick={() => setSelectedAgentId(agent.agent_id)}
                    sx={{ cursor: 'pointer' }}
                  >
                    <TableCell>{agent.agent_id}</TableCell>
                    <TableCell>{agent.type}</TableCell>
                    <TableCell>{agent.version || '-'}</TableCell>
                    <TableCell>
                      <Chip
                        label={getStatusText(agent.status)}
                        color={getStatusColor(agent.status) as 'success' | 'error' | 'warning' | 'default'}
                        size="small"
                      />
                    </TableCell>
                    <TableCell>{agent.pid || '-'}</TableCell>
                    <TableCell>{formatTime(agent.last_heartbeat)}</TableCell>
                    <TableCell align="right">
                      <Tooltip title="启动">
                        <span>
                          <IconButton
                            size="small"
                            color="primary"
                            disabled={isStartDisabled}
                            onClick={() => handleOperate(agent.agent_id, 'start')}
                          >
                            <PlayArrowIcon />
                          </IconButton>
                        </span>
                      </Tooltip>
                      <Tooltip title="停止">
                        <span>
                          <IconButton
                            size="small"
                            color="error"
                            disabled={isStopDisabled}
                            onClick={() => handleOperate(agent.agent_id, 'stop')}
                          >
                            <StopIcon />
                          </IconButton>
                        </span>
                      </Tooltip>
                      <Tooltip title="重启">
                        <span>
                          <IconButton
                            size="small"
                            color="warning"
                            disabled={isRestartDisabled}
                            onClick={() => handleOperate(agent.agent_id, 'restart')}
                          >
                            <RestartAltIcon />
                          </IconButton>
                        </span>
                      </Tooltip>
                      <Tooltip title="查看日志">
                        <IconButton
                          size="small"
                          color="default"
                          onClick={() => handleViewLogs(agent.agent_id)}
                        >
                          <DescriptionIcon />
                        </IconButton>
                      </Tooltip>
                    </TableCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
        </TableContainer>
      )}

      {/* Agent 监控图表区域 */}
      {selectedAgentId && agents.length > 0 && (
        <Box sx={{ mt: 4 }}>
          <Paper elevation={2} sx={{ p: 2, mb: 2 }}>
            <Box
              sx={{
                display: 'flex',
                justifyContent: 'space-between',
                alignItems: 'center',
                flexWrap: 'wrap',
                gap: 2,
              }}
            >
              <Typography variant="h6">
                Agent 监控 - {agents.find((a) => a.agent_id === selectedAgentId)?.agent_id}
              </Typography>
              <ToggleButtonGroup
                value={timeRangePreset}
                exclusive
                onChange={(_event, newPreset: '1h' | '6h' | '24h' | null) => {
                  if (newPreset) {
                    setTimeRangePreset(newPreset);
                  }
                }}
                aria-label="时间范围选择"
                size="small"
              >
                <ToggleButton value="1h" aria-label="1小时">
                  1小时
                </ToggleButton>
                <ToggleButton value="6h" aria-label="6小时">
                  6小时
                </ToggleButton>
                <ToggleButton value="24h" aria-label="24小时">
                  24小时
                </ToggleButton>
              </ToggleButtonGroup>
            </Box>
          </Paper>

          {/* 图表区域 */}
          <Grid container spacing={2.5}>
            <Grid size={{ xs: 12, md: 6 }}>
              <Suspense fallback={<CircularProgress />}>
                <MetricsChart
                  data={cpuChartData}
                  title="CPU 使用率"
                  unit="%"
                  color="#1976d2"
                  loading={cpuHistory.isLoading}
                  height={300}
                />
              </Suspense>
            </Grid>
            <Grid size={{ xs: 12, md: 6 }}>
              <Suspense fallback={<CircularProgress />}>
                <MetricsChart
                  data={memoryChartData}
                  title="内存使用率"
                  unit="%"
                  color="#d32f2f"
                  loading={memoryHistory.isLoading}
                  height={300}
                />
              </Suspense>
            </Grid>
            <Grid size={{ xs: 12, md: 6 }}>
              <Suspense fallback={<CircularProgress />}>
                <MetricsChart
                  data={openFilesChartData}
                  title="文件描述符"
                  unit="个"
                  color="#ed6c02"
                  loading={openFilesHistory.isLoading}
                  height={300}
                />
              </Suspense>
            </Grid>
            <Grid size={{ xs: 12, md: 6 }}>
              <Suspense fallback={<CircularProgress />}>
                <MetricsChart
                  data={diskReadChartData}
                  title="磁盘读取 (累计)"
                  unit="MB"
                  color="#9c27b0"
                  loading={diskIOHistory.isLoading}
                  height={300}
                />
              </Suspense>
            </Grid>
            <Grid size={{ xs: 12, md: 6 }}>
              <Suspense fallback={<CircularProgress />}>
                <MetricsChart
                  data={diskWriteChartData}
                  title="磁盘写入 (累计)"
                  unit="MB"
                  color="#2e7d32"
                  loading={diskIOHistory.isLoading}
                  height={300}
                />
              </Suspense>
            </Grid>
          </Grid>

          {/* 图表错误提示 */}
          {(cpuHistory.error || memoryHistory.error || openFilesHistory.error || diskIOHistory.error) && (
            <Alert
              severity="error"
              sx={{ mt: 2 }}
              action={
                <Button
                  color="inherit"
                  size="small"
                  onClick={() => {
                    cpuHistory.refetch();
                    memoryHistory.refetch();
                    openFilesHistory.refetch();
                    diskIOHistory.refetch();
                  }}
                >
                  重试
                </Button>
              }
            >
              {cpuHistory.error instanceof Error
                ? cpuHistory.error.message
                : memoryHistory.error instanceof Error
                ? memoryHistory.error.message
                : openFilesHistory.error instanceof Error
                ? openFilesHistory.error.message
                : diskIOHistory.error instanceof Error
                ? diskIOHistory.error.message
                : '加载监控数据失败'}
            </Alert>
          )}
        </Box>
      )}

      {/* 没有选中 Agent 时的提示 */}
      {!selectedAgentId && agents.length > 0 && (
        <Paper sx={{ p: 4, textAlign: 'center', mt: 4 }}>
          <Typography variant="body2" color="text.secondary">
            请选择一个 Agent 查看监控数据
          </Typography>
        </Paper>
      )}

      {/* Agent 日志查看 Dialog */}
      <AgentLogsDialog
        open={logDialogOpen}
        onClose={handleCloseLogDialog}
        nodeId={nodeId}
        agentId={viewingAgentId}
        autoRefresh={autoRefresh}
        onAutoRefreshChange={setAutoRefresh}
      />
    </Box>
  );
}

/**
 * Agent 日志查看 Dialog 组件
 */
function AgentLogsDialog({
  open,
  onClose,
  nodeId,
  agentId,
  autoRefresh,
  onAutoRefreshChange,
}: {
  open: boolean;
  onClose: () => void;
  nodeId: string;
  agentId: string | null;
  autoRefresh: boolean;
  onAutoRefreshChange: (value: boolean) => void;
}) {
  // 获取 Agent 日志
  const {
    data: logData,
    isLoading: isLoadingLogs,
    error: logError,
    refetch: refetchLogs,
    isRefetching: isRefetchingLogs,
  } = useQuery({
    queryKey: ['agent-logs', nodeId, agentId],
    queryFn: () => getAgentLogs(nodeId, agentId!, 100),
    enabled: open && !!agentId, // 仅在 Dialog 打开时获取
    refetchInterval: autoRefresh && open ? 5000 : false, // 每 5 秒自动刷新
  });

  // 处理导出日志
  const handleExportLogs = () => {
    if (!logData?.data?.logs) return;

    // 将日志数组转换为文本
    const logText = logData.data.logs.join('\n');

    // 创建 Blob 对象
    const blob = new Blob([logText], { type: 'text/plain;charset=utf-8' });

    // 创建下载链接
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = `agent-${agentId}-logs-${new Date().toISOString()}.txt`;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    URL.revokeObjectURL(url);
  };

  // 处理错误信息
  const getErrorMessage = () => {
    if (!logError) return null;
    if (logError instanceof Error) {
      // 检查是否是 501 错误
      if (logError.message.includes('501') || logError.message.includes('Not Implemented')) {
        return '日志功能暂未实现，请稍后重试';
      }
      return logError.message;
    }
    return '加载日志失败';
  };

  return (
    <Dialog open={open} onClose={onClose} maxWidth="md" fullWidth>
      <DialogTitle>
        Agent 日志 - {agentId}
        {autoRefresh && (
          <Typography variant="caption" color="text.secondary" sx={{ ml: 1 }}>
            (自动刷新中...)
          </Typography>
        )}
      </DialogTitle>
      <DialogContent>
        {isLoadingLogs && !logData ? (
          <Box display="flex" justifyContent="center" py={4}>
            <CircularProgress />
          </Box>
        ) : logError ? (
          <Alert
            severity="error"
            action={
              <Button color="inherit" size="small" onClick={() => refetchLogs()}>
                重试
              </Button>
            }
            sx={{ mb: 2 }}
          >
            {getErrorMessage()}
          </Alert>
        ) : !logData?.data?.logs || logData.data.logs.length === 0 ? (
          <Box py={4} textAlign="center">
            <Typography variant="body2" color="text.secondary">
              暂无日志
            </Typography>
          </Box>
        ) : (
          <Box
            sx={{
              maxHeight: '400px',
              overflow: 'auto',
              bgcolor: 'grey.900',
              color: 'grey.100',
              p: 2,
              borderRadius: 1,
              fontFamily: 'monospace',
              fontSize: '0.875rem',
              position: 'relative',
            }}
          >
            {isRefetchingLogs && (
              <Box
                sx={{
                  position: 'absolute',
                  top: 8,
                  right: 8,
                  display: 'flex',
                  alignItems: 'center',
                  gap: 1,
                  bgcolor: 'rgba(0, 0, 0, 0.7)',
                  px: 1,
                  py: 0.5,
                  borderRadius: 1,
                }}
              >
                <CircularProgress size={16} sx={{ color: 'grey.100' }} />
                <Typography variant="caption" sx={{ color: 'grey.100' }}>
                  刷新中...
                </Typography>
              </Box>
            )}
            {logData.data.logs.map((log, index) => (
              <Typography
                key={index}
                component="pre"
                sx={{
                  m: 0,
                  whiteSpace: 'pre-wrap',
                  wordBreak: 'break-word',
                  lineHeight: 1.5,
                }}
              >
                {log}
              </Typography>
            ))}
          </Box>
        )}
      </DialogContent>
      <DialogActions>
        <Button
          onClick={() => onAutoRefreshChange(!autoRefresh)}
          variant={autoRefresh ? 'outlined' : 'text'}
          size="small"
        >
          {autoRefresh ? '停止自动刷新' : '开启自动刷新'}
        </Button>
        <Button
          onClick={() => refetchLogs()}
          disabled={isRefetchingLogs || isLoadingLogs}
          startIcon={<RefreshIcon />}
          size="small"
        >
          刷新
        </Button>
        <Button
          onClick={handleExportLogs}
          disabled={!logData?.data?.logs || logData.data.logs.length === 0}
          startIcon={<DownloadIcon />}
          variant="outlined"
          size="small"
        >
          导出
        </Button>
        <Button onClick={onClose} variant="contained" size="small">
          关闭
        </Button>
      </DialogActions>
    </Dialog>
  );
}

