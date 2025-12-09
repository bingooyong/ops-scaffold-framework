/**
 * 时间范围选择器组件
 */

import { useState, useEffect, useMemo, memo } from 'react';
import {
  ToggleButton,
  ToggleButtonGroup,
  Button,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Alert,
  Box,
} from '@mui/material';
import { LocalizationProvider } from '@mui/x-date-pickers/LocalizationProvider';
import { DateTimePicker } from '@mui/x-date-pickers/DateTimePicker';
import { AdapterDateFns } from '@mui/x-date-pickers/AdapterDateFns';
import { zhCN } from 'date-fns/locale';
import type { TimeRange } from '../../types';

export interface TimeRangeSelectorProps {
  value: TimeRange;
  onChange: (range: TimeRange) => void;
}

type PresetValue = '15m' | '30m' | '1h' | '1d' | '7d' | '30d' | null;

function TimeRangeSelector({
  value,
  onChange,
}: TimeRangeSelectorProps) {
  const [open, setOpen] = useState(false);
  const [customStartTime, setCustomStartTime] = useState<Date | null>(value.startTime);
  const [customEndTime, setCustomEndTime] = useState<Date | null>(value.endTime);
  const [error, setError] = useState<string | null>(null);

  // 计算当前选中的预设值（仅基于时间差，不检查当前时间）
  const calculatePreset = useMemo((): PresetValue => {
    const start = value.startTime.getTime();
    const end = value.endTime.getTime();
    const diff = end - start;

    // 检查时间差是否匹配预设值（允许 1 秒的误差）
    if (Math.abs(diff - 15 * 60 * 1000) < 1000) return '15m';
    if (Math.abs(diff - 30 * 60 * 1000) < 1000) return '30m';
    if (Math.abs(diff - 60 * 60 * 1000) < 1000) return '1h';
    if (Math.abs(diff - 24 * 60 * 60 * 1000) < 1000) return '1d';
    if (Math.abs(diff - 7 * 24 * 60 * 60 * 1000) < 1000) return '7d';
    if (Math.abs(diff - 30 * 24 * 60 * 60 * 1000) < 1000) return '30d';

    return null;
  }, [value.startTime, value.endTime]);

  const [selectedPreset, setSelectedPreset] = useState<PresetValue>(calculatePreset);

  // 当 value 变化时，更新预设值
  useEffect(() => {
    setSelectedPreset(calculatePreset);
  }, [calculatePreset]);

  // 处理预设值选择
  const handlePresetChange = (
    _event: React.MouseEvent<HTMLElement>,
    newPreset: PresetValue
  ) => {
    if (newPreset === null) return;

    setSelectedPreset(newPreset);
    setError(null);

    const now = new Date();
    let startTime: Date;

    switch (newPreset) {
      case '15m':
        startTime = new Date(now.getTime() - 15 * 60 * 1000);
        break;
      case '30m':
        startTime = new Date(now.getTime() - 30 * 60 * 1000);
        break;
      case '1h':
        startTime = new Date(now.getTime() - 60 * 60 * 1000);
        break;
      case '1d':
        startTime = new Date(now.getTime() - 24 * 60 * 60 * 1000);
        break;
      case '7d':
        startTime = new Date(now.getTime() - 7 * 24 * 60 * 60 * 1000);
        break;
      case '30d':
        startTime = new Date(now.getTime() - 30 * 24 * 60 * 60 * 1000);
        break;
      default:
        return;
    }

    onChange({
      startTime,
      endTime: now,
    });
  };

  // 打开自定义时间对话框
  const handleOpenCustom = () => {
    setCustomStartTime(value.startTime);
    setCustomEndTime(value.endTime);
    setError(null);
    setOpen(true);
  };

  // 关闭对话框
  const handleCloseCustom = () => {
    setOpen(false);
    setError(null);
  };

  // 确认自定义时间
  const handleConfirmCustom = () => {
    if (!customStartTime || !customEndTime) {
      setError('请选择开始时间和结束时间');
      return;
    }

    if (customEndTime <= customStartTime) {
      setError('结束时间必须晚于开始时间');
      return;
    }

    const diffDays = (customEndTime.getTime() - customStartTime.getTime()) / (1000 * 60 * 60 * 24);
    if (diffDays > 30) {
      setError('时间范围不能超过 30 天');
      return;
    }

    setSelectedPreset(null);
    onChange({
      startTime: customStartTime,
      endTime: customEndTime,
    });
    handleCloseCustom();
  };

  return (
    <Box>
      <ToggleButtonGroup
        value={selectedPreset}
        exclusive
        onChange={handlePresetChange}
        aria-label="时间范围选择"
        size="small"
      >
        <ToggleButton value="15m" aria-label="15分钟">
          15分钟
        </ToggleButton>
        <ToggleButton value="30m" aria-label="30分钟">
          30分钟
        </ToggleButton>
        <ToggleButton value="1h" aria-label="1小时">
          1小时
        </ToggleButton>
        <ToggleButton value="1d" aria-label="1天">
          1天
        </ToggleButton>
        <ToggleButton value="7d" aria-label="7天">
          7天
        </ToggleButton>
        <ToggleButton value="30d" aria-label="30天">
          30天
        </ToggleButton>
      </ToggleButtonGroup>
      <Button
        variant="outlined"
        size="small"
        onClick={handleOpenCustom}
        sx={{ ml: 1 }}
      >
        自定义时间
      </Button>

      {/* 自定义时间对话框 */}
      <Dialog open={open} onClose={handleCloseCustom} maxWidth="sm" fullWidth>
        <DialogTitle>选择自定义时间范围</DialogTitle>
        <DialogContent>
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2, mt: 2 }}>
            <LocalizationProvider dateAdapter={AdapterDateFns} adapterLocale={zhCN}>
              <DateTimePicker
                label="开始时间"
                value={customStartTime}
                onChange={(newValue) => {
                  setCustomStartTime(newValue);
                  setError(null);
                }}
                slotProps={{
                  textField: {
                    fullWidth: true,
                  },
                }}
              />
              <DateTimePicker
                label="结束时间"
                value={customEndTime}
                onChange={(newValue) => {
                  setCustomEndTime(newValue);
                  setError(null);
                }}
                slotProps={{
                  textField: {
                    fullWidth: true,
                  },
                }}
              />
            </LocalizationProvider>
            {error && (
              <Alert severity="error" sx={{ mt: 1 }}>
                {error}
              </Alert>
            )}
          </Box>
        </DialogContent>
        <DialogActions>
          <Button onClick={handleCloseCustom}>取消</Button>
          <Button onClick={handleConfirmCustom} variant="contained">
            确认
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
}

export default memo(TimeRangeSelector);

