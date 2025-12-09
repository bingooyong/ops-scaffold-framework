/**
 * 刷新控制组件
 */

import { useState, useEffect, useMemo, memo } from 'react';
import {
  ToggleButton,
  ToggleButtonGroup,
  Box,
  Typography,
  CircularProgress,
} from '@mui/material';
import { Pause as PauseIcon, Refresh as RefreshIcon } from '@mui/icons-material';

export interface RefreshControlProps {
  value: number | null;
  onChange: (interval: number | null) => void;
  onRefresh?: () => void;
}

function RefreshControl({
  value,
  onChange,
  onRefresh,
}: RefreshControlProps) {
  // 计算初始倒计时值
  const initialCountdown = useMemo(() => {
    return value === null ? null : Math.floor(value / 1000);
  }, [value]);

  const [countdown, setCountdown] = useState<number | null>(initialCountdown);

  // 当 value 变化时，更新倒计时
  useEffect(() => {
    setCountdown(initialCountdown);
  }, [initialCountdown]);

  // 倒计时逻辑
  useEffect(() => {
    if (value === null || countdown === null) {
      return;
    }

    const interval = setInterval(() => {
      setCountdown((prev) => {
        if (prev === null || prev <= 1) {
          // 倒计时结束，触发刷新
          if (onRefresh) {
            onRefresh();
          }
          // 重置倒计时
          return Math.floor(value / 1000);
        }
        return prev - 1;
      });
    }, 1000);

    return () => clearInterval(interval);
  }, [value, countdown, onRefresh]);

  const handleChange = (
    _event: React.MouseEvent<HTMLElement>,
    newValue: number | null
  ) => {
    onChange(newValue === 0 ? null : newValue);
  };

  return (
    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
      <ToggleButtonGroup
        value={value === null ? 0 : value}
        exclusive
        onChange={handleChange}
        aria-label="刷新间隔选择"
        size="small"
      >
        <ToggleButton value={0} aria-label="暂停">
          <PauseIcon sx={{ mr: 0.5 }} fontSize="small" />
          暂停
        </ToggleButton>
        <ToggleButton value={30000} aria-label="30秒">
          <RefreshIcon sx={{ mr: 0.5 }} fontSize="small" />
          30秒
        </ToggleButton>
        <ToggleButton value={60000} aria-label="1分钟">
          <RefreshIcon sx={{ mr: 0.5 }} fontSize="small" />
          1分钟
        </ToggleButton>
      </ToggleButtonGroup>
      {countdown !== null && countdown > 0 && (
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, ml: 1 }}>
          <CircularProgress
            size={20}
            variant="determinate"
            value={(countdown / Math.floor((value || 0) / 1000)) * 100}
          />
          <Typography variant="caption" color="text.secondary">
            {countdown}s
          </Typography>
        </Box>
      )}
    </Box>
  );
}

export default memo(RefreshControl);
