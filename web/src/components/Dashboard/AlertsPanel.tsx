/**
 * 告警面板组件
 */

import { useMemo } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  Card,
  CardHeader,
  CardContent,
  List,
  ListItem,
  Alert,
  Typography,
  Box,
  Divider,
  Badge,
  Skeleton,
} from '@mui/material';
import { Notifications as NotificationsIcon } from '@mui/icons-material';
import type { NodeMetrics } from '../../types';
import { checkNodeAlerts } from '../../utils/alertRules';

interface AlertsPanelProps {
  nodes: NodeMetrics[];
  loading?: boolean;
}

export default function AlertsPanel({ nodes, loading = false }: AlertsPanelProps) {
  const navigate = useNavigate();

  // 计算所有节点的告警列表
  const alerts = useMemo(() => {
    if (!nodes || nodes.length === 0) {
      return [];
    }

    // 遍历所有节点，提取告警
    const allAlerts = nodes.flatMap((node) => checkNodeAlerts(node));

    // 按严重级别排序：critical 在前，warning 在后
    const levelOrder: Record<string, number> = { critical: 0, warning: 1, normal: 2 };
    return allAlerts.sort((a, b) => levelOrder[a.level] - levelOrder[b.level]);
  }, [nodes]);

  // 过滤出实际告警（排除 normal）
  const activeAlerts = useMemo(() => {
    return alerts.filter((a) => a.level !== 'normal');
  }, [alerts]);

  // 判断是否有 critical 告警
  const hasCritical = activeAlerts.some((a) => a.level === 'critical');

  // 告警数量徽章颜色
  const badgeColor = hasCritical ? 'error' : activeAlerts.length > 0 ? 'warning' : 'default';

  // 显示前 10 个告警
  const displayedAlerts = activeAlerts.slice(0, 10);
  const hasMore = activeAlerts.length > 10;

  return (
    <Card elevation={2} sx={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
      <CardHeader
        title="告警信息"
        action={
          <Badge badgeContent={activeAlerts.length} color={badgeColor}>
            <NotificationsIcon />
          </Badge>
        }
      />
      <CardContent sx={{ flex: 1, display: 'flex', flexDirection: 'column', minHeight: 200 }}>
        {loading ? (
          // Loading 状态：显示 Skeleton 占位符
          <List>
            {[1, 2, 3].map((index) => (
              <Box key={index}>
                <ListItem>
                  <Skeleton variant="rectangular" width="100%" height={60} />
                </ListItem>
                {index < 3 && <Divider />}
              </Box>
            ))}
          </List>
        ) : activeAlerts.length === 0 ? (
          // 无告警状态
          <Box sx={{ flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
            <Alert severity="success" sx={{ width: '100%' }}>
              所有节点运行正常
            </Alert>
          </Box>
        ) : (
          // 告警列表
          <List>
            {displayedAlerts.map((alert, index) => (
              <Box key={`${alert.node_id}-${alert.metric_type}-${index}`}>
                <ListItem
                  onClick={() => navigate(`/nodes/${alert.node_id}`)}
                  sx={{
                    p: 0,
                    cursor: 'pointer',
                    '&:hover': {
                      backgroundColor: 'action.hover',
                    },
                  }}
                >
                  <Alert
                    severity={alert.level === 'critical' ? 'error' : 'warning'}
                    variant="outlined"
                    sx={{ width: '100%', cursor: 'pointer' }}
                  >
                    <Typography variant="body2" component="div">
                      <Typography component="span" fontWeight="bold">
                        {alert.hostname}
                      </Typography>
                      {' - '}
                      <Typography component="span" color="text.secondary">
                        {alert.metric_type} 使用率 {alert.current_value.toFixed(1)}%
                      </Typography>
                      {' - '}
                      <Typography component="span" color="text.secondary">
                        {alert.message}
                      </Typography>
                    </Typography>
                  </Alert>
                </ListItem>
                {index < displayedAlerts.length - 1 && <Divider sx={{ my: 1 }} />}
              </Box>
            ))}
            {hasMore && (
              <Typography variant="body2" color="text.secondary" align="center" sx={{ mt: 2 }}>
                还有 {activeAlerts.length - 10} 个告警...
              </Typography>
            )}
          </List>
        )}
      </CardContent>
    </Card>
  );
}

