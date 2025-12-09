/**
 * 通用指标卡片组件
 */

import { type ReactNode, memo } from 'react';
import {
  Card,
  CardContent,
  Typography,
  Box,
  LinearProgress,
  Skeleton,
  Alert,
} from '@mui/material';

export interface MetricCardProps {
  title: string;
  value: number | string;
  unit?: string;
  percentage?: number;
  icon?: ReactNode;
  color?: string;
  loading?: boolean;
  error?: string;
  extraInfo?: ReactNode;
}

function MetricCard({
  title,
  value,
  unit,
  percentage,
  icon,
  color,
  loading,
  error,
  extraInfo,
}: MetricCardProps) {
  if (loading) {
    return (
      <Card elevation={2} sx={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
        <CardContent sx={{ flex: 1 }}>
          <Skeleton variant="text" width="60%" height={32} />
          <Skeleton variant="text" width="40%" height={48} sx={{ mt: 1 }} />
          <Skeleton variant="rectangular" height={8} sx={{ mt: 2, borderRadius: 1 }} />
        </CardContent>
      </Card>
    );
  }

  if (error) {
    return (
      <Card elevation={2} sx={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
        <CardContent sx={{ flex: 1 }}>
          <Alert severity="error">{error}</Alert>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card elevation={2} sx={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
      <CardContent sx={{ flex: 1, display: 'flex', flexDirection: 'column' }}>
        {/* 标题和图标 */}
        <Box display="flex" justifyContent="space-between" alignItems="center" mb={2}>
          <Typography variant="h6" component="div">
            {title}
          </Typography>
          {icon && <Box sx={{ color: 'text.secondary' }}>{icon}</Box>}
        </Box>

        {/* 数值和单位 */}
        <Box mb={percentage !== undefined ? 2 : 0}>
          <Typography variant="h4" component="div" sx={{ fontWeight: 'bold' }}>
            {value}
            {unit && (
              <Typography component="span" variant="body2" sx={{ ml: 0.5, color: 'text.secondary' }}>
                {unit}
              </Typography>
            )}
          </Typography>
        </Box>

        {/* 进度条 */}
        {percentage !== undefined && (
          <Box sx={{ mb: 1 }}>
            <LinearProgress
              variant="determinate"
              value={Math.min(percentage, 100)}
              sx={{
                height: 8,
                borderRadius: 1,
                backgroundColor: 'grey.200',
                '& .MuiLinearProgress-bar': {
                  backgroundColor: color || 'primary.main',
                },
              }}
            />
          </Box>
        )}

        {/* 附加信息 */}
        <Box sx={{ mt: 'auto', pt: extraInfo ? 1 : 0 }}>
          {extraInfo && (
            <Typography variant="caption" color="text.secondary">
              {extraInfo}
            </Typography>
          )}
        </Box>
      </CardContent>
    </Card>
  );
}

export default memo(MetricCard);

