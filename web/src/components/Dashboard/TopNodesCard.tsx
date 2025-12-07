/**
 * Top 节点排名卡片组件
 */

import { useState, useMemo } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  Card,
  CardHeader,
  CardContent,
  ToggleButtonGroup,
  ToggleButton,
  List,
  ListItem,
  ListItemText,
  Typography,
  Box,
  LinearProgress,
  Divider,
  Skeleton,
  useTheme,
} from '@mui/material';
import {
  Memory as CpuIcon,
  Storage as MemoryIcon,
} from '@mui/icons-material';
import type { NodeMetrics } from '../../types';
import { getUsageColor } from '../../utils/metricsUtils';

interface TopNodesCardProps {
  nodes: NodeMetrics[];
  loading?: boolean;
}

type SortBy = 'cpu' | 'memory';

export default function TopNodesCard({ nodes, loading = false }: TopNodesCardProps) {
  const theme = useTheme();
  const navigate = useNavigate();
  const [sortBy, setSortBy] = useState<SortBy>('cpu');

  // 排序逻辑：按选定指标降序排序，取前 5 个
  const topNodes = useMemo(() => {
    if (!nodes || nodes.length === 0) {
      return [];
    }

    const sorted = [...nodes].sort((a, b) => {
      if (sortBy === 'cpu') {
        return b.cpu_usage - a.cpu_usage;
      } else {
        return b.memory_usage - a.memory_usage;
      }
    });

    return sorted.slice(0, 5);
  }, [nodes, sortBy]);

  // 获取排名颜色
  const getRankColor = (rank: number) => {
    switch (rank) {
      case 1:
        return '#FFD700'; // 金色
      case 2:
        return '#C0C0C0'; // 银色
      case 3:
        return '#CD7F32'; // 铜色
      default:
        return theme.palette.grey[500]; // 灰色
    }
  };

  // 处理排序维度切换
  const handleSortChange = (_event: React.MouseEvent<HTMLElement>, newSortBy: SortBy | null) => {
    if (newSortBy !== null) {
      setSortBy(newSortBy);
    }
  };

  return (
    <Card elevation={2}>
      <CardHeader
        title="Top 5 资源使用节点"
        action={
          <ToggleButtonGroup
            value={sortBy}
            exclusive
            onChange={handleSortChange}
            size="small"
            aria-label="排序维度"
          >
            <ToggleButton value="cpu" aria-label="按 CPU 排序">
              <CpuIcon sx={{ mr: 0.5 }} fontSize="small" />
              CPU
            </ToggleButton>
            <ToggleButton value="memory" aria-label="按内存排序">
              <MemoryIcon sx={{ mr: 0.5 }} fontSize="small" />
              内存
            </ToggleButton>
          </ToggleButtonGroup>
        }
      />
      <CardContent>
        {loading ? (
          // Loading 状态：显示 5 个 Skeleton
          <List>
            {[1, 2, 3, 4, 5].map((index) => (
              <Box key={index}>
                <ListItem>
                  <Skeleton variant="circular" width={32} height={32} sx={{ mr: 2 }} />
                  <ListItemText
                    primary={<Skeleton variant="text" width={120} />}
                    secondary={<Skeleton variant="text" width={80} />}
                  />
                  <Skeleton variant="text" width={60} sx={{ ml: 'auto' }} />
                </ListItem>
                {index < 5 && <Divider />}
              </Box>
            ))}
          </List>
        ) : topNodes.length === 0 ? (
          // 空数据状态
          <Typography variant="body2" color="text.secondary" align="center" sx={{ py: 4 }}>
            暂无节点数据
          </Typography>
        ) : (
          // 节点列表
          <List>
            {topNodes.map((node, index) => {
              const rank = index + 1;
              const usage = sortBy === 'cpu' ? node.cpu_usage : node.memory_usage;
              const usageColor = getUsageColor(usage, theme);

              return (
                <Box key={node.node_id}>
                  <ListItem
                    button
                    onClick={() => navigate(`/nodes/${node.node_id}`)}
                    sx={{
                      '&:hover': {
                        backgroundColor: 'action.hover',
                      },
                    }}
                  >
                    {/* 排名序号 */}
                    <Box
                      sx={{
                        width: 32,
                        height: 32,
                        borderRadius: '50%',
                        backgroundColor: getRankColor(rank),
                        color: 'white',
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                        mr: 2,
                        fontWeight: 'bold',
                      }}
                    >
                      <Typography variant="body2" fontWeight="bold">
                        {rank}
                      </Typography>
                    </Box>

                    {/* 节点名称和 IP */}
                    <ListItemText
                      primary={node.hostname}
                      secondary={node.ip}
                      sx={{ flex: 1 }}
                    />

                    {/* 使用率百分比和进度条 */}
                    <Box sx={{ minWidth: 120, ml: 2 }}>
                      <Typography variant="body2" align="right" sx={{ mb: 0.5 }}>
                        {usage.toFixed(1)}%
                      </Typography>
                      <LinearProgress
                        variant="determinate"
                        value={Math.min(usage, 100)}
                        sx={{
                          height: 6,
                          borderRadius: 3,
                          backgroundColor: 'grey.200',
                          '& .MuiLinearProgress-bar': {
                            backgroundColor: usageColor,
                          },
                        }}
                      />
                    </Box>
                  </ListItem>
                  {rank < 5 && <Divider />}
                </Box>
              );
            })}
          </List>
        )}
      </CardContent>
    </Card>
  );
}

