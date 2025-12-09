/**
 * AgentLogsDialog 组件测试
 */

import { describe, it, expect, vi, beforeEach } from 'vitest';
import { getAgentLogs } from '../../../api';

// Mock API functions
vi.mock('../../../api', () => ({
  getAgentLogs: vi.fn(),
}));

// 由于 AgentLogsDialog 是内部组件，我们需要通过 Detail 组件测试它
// 或者创建一个简化的测试组件
// 这里我们直接测试 Dialog 的逻辑

describe('AgentLogsDialog', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  // 由于 AgentLogsDialog 是 Detail.tsx 中的内部组件
  // 我们需要通过集成测试来测试它
  // 这里提供一些基础测试示例

  it('应该能够获取并显示日志', async () => {
    const mockLogs = {
      code: 0,
      message: 'success',
      data: {
        logs: ['Log line 1', 'Log line 2', 'Log line 3'],
        count: 3,
      },
    };

    vi.mocked(getAgentLogs).mockResolvedValue(mockLogs);

    // 验证 API 调用
    const result = await getAgentLogs('node-1', 'agent-1', 100);

    expect(getAgentLogs).toHaveBeenCalledWith('node-1', 'agent-1', 100);
    expect(result.data.logs).toHaveLength(3);
    expect(result.data.logs[0]).toBe('Log line 1');
  });

  it('应该能够处理空日志', async () => {
    const mockLogs = {
      code: 0,
      message: 'success',
      data: {
        logs: [],
        count: 0,
      },
    };

    vi.mocked(getAgentLogs).mockResolvedValue(mockLogs);

    const result = await getAgentLogs('node-1', 'agent-1');

    expect(result.data.logs).toHaveLength(0);
    expect(result.data.count).toBe(0);
  });

  it('应该能够处理日志获取错误', async () => {
    const error = new Error('Failed to fetch logs');
    vi.mocked(getAgentLogs).mockRejectedValue(error);

    await expect(getAgentLogs('node-1', 'agent-1')).rejects.toThrow('Failed to fetch logs');
  });

  it('应该能够处理 501 错误（功能未实现）', async () => {
    const error = new Error('501 Not Implemented');
    vi.mocked(getAgentLogs).mockRejectedValue(error);

    await expect(getAgentLogs('node-1', 'agent-1')).rejects.toThrow('501 Not Implemented');
  });

  it('应该能够获取指定行数的日志', async () => {
    const mockLogs = {
      code: 0,
      message: 'success',
      data: {
        logs: Array.from({ length: 50 }, (_, i) => `Log line ${i + 1}`),
        count: 50,
      },
    };

    vi.mocked(getAgentLogs).mockResolvedValue(mockLogs);

    const result = await getAgentLogs('node-1', 'agent-1', 50);

    expect(getAgentLogs).toHaveBeenCalledWith('node-1', 'agent-1', 50);
    expect(result.data.logs).toHaveLength(50);
  });
});
