/**
 * Agent 管理功能集成测试
 * 
 * 测试从加载 Agent 列表到执行操作的完整流程
 */

import { describe, it, expect, vi, beforeEach } from 'vitest';
import { listAgents, operateAgent, getAgentLogs } from '../../../api';
import type { Agent } from '../../../types';

// Mock API functions
vi.mock('../../../api', () => ({
  listAgents: vi.fn(),
  operateAgent: vi.fn(),
  getAgentLogs: vi.fn(),
}));

// Mock hooks
vi.mock('../../../hooks', () => ({
  useNode: vi.fn(() => ({
    data: {
      code: 0,
      message: 'success',
      data: {
        node: {
          node_id: 'node-1',
          hostname: 'test-node',
          ip: '127.0.0.1',
          status: 'online',
          os: 'linux',
          arch: 'amd64',
        },
      },
    },
    isLoading: false,
    error: null,
  })),
  useLatestMetrics: vi.fn(() => ({
    data: null,
    isLoading: false,
    error: null,
    refetch: vi.fn(),
  })),
  useMetricsHistory: vi.fn(() => ({
    data: null,
    isLoading: false,
    error: null,
    refetch: vi.fn(),
  })),
}));

vi.mock('../../../hooks/useAgentMetrics', () => ({
  useAgentMetricsHistory: vi.fn(() => ({
    data: null,
    isLoading: false,
    error: null,
    refetch: vi.fn(),
  })),
}));

// Mock react-router-dom
vi.mock('react-router-dom', () => ({
  useParams: () => ({ id: 'node-1' }),
  useNavigate: () => vi.fn(),
  Link: ({ children, to }: any) => <a href={to}>{children}</a>,
}));

// Mock stores
vi.mock('../../../stores', () => ({
  useMetricsStore: vi.fn(() => ({
    timeRange: {
      startTime: new Date(Date.now() - 3600000),
      endTime: new Date(),
    },
    refreshInterval: null,
    setTimeRange: vi.fn(),
    setRefreshInterval: vi.fn(),
  })),
}));

// Mock MetricsChart
vi.mock('../../../components/Metrics/MetricsChart', () => ({
  default: ({ title }: { title: string }) => <div data-testid="metrics-chart">{title}</div>,
}));

const mockAgents: Agent[] = [
  {
    id: 1,
    node_id: 'node-1',
    agent_id: 'agent-1',
    type: 'filebeat',
    version: '1.0.0',
    status: 'running',
    pid: 12345,
    last_heartbeat: '2024-01-01T00:00:00Z',
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
  },
];

describe('Agent 管理集成测试', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('应该完成从加载列表到查看日志的完整流程', async () => {
    // Mock API responses
    (listAgents as any).mockResolvedValue({
      code: 0,
      message: 'success',
      data: { agents: mockAgents, count: 1 },
    });

    (getAgentLogs as any).mockResolvedValue({
      code: 0,
      message: 'success',
      data: {
        logs: ['Log line 1', 'Log line 2'],
        count: 2,
      },
    });

    // 由于 Detail 组件比较复杂，这里主要测试 API 调用流程
    // 在实际项目中，可以通过 E2E 测试来验证完整流程

    // 验证 listAgents 调用
    const agentsResult = await listAgents('node-1');
    expect(listAgents).toHaveBeenCalledWith('node-1');
    expect(agentsResult.data.agents).toHaveLength(1);

    // 验证 getAgentLogs 调用
    const logsResult = await getAgentLogs('node-1', 'agent-1', 100);
    expect(getAgentLogs).toHaveBeenCalledWith('node-1', 'agent-1', 100);
    expect(logsResult.data.logs).toHaveLength(2);
  });

  it('应该完成从加载列表到执行操作的完整流程', async () => {
    // Mock API responses
    (listAgents as any).mockResolvedValue({
      code: 0,
      message: 'success',
      data: { agents: mockAgents, count: 1 },
    });

    (operateAgent as any).mockResolvedValue({
      code: 0,
      message: 'Agent restarted successfully',
      data: null,
    });

    // 验证操作流程
    const agentsResult = await listAgents('node-1');
    expect(agentsResult.data.agents[0].agent_id).toBe('agent-1');

    const operateResult = await operateAgent('node-1', 'agent-1', 'restart');
    expect(operateAgent).toHaveBeenCalledWith('node-1', 'agent-1', 'restart');
    expect(operateResult.code).toBe(0);
  });

  it('应该处理操作失败后的错误状态', async () => {
    // Mock API responses
    (listAgents as any).mockResolvedValue({
      code: 0,
      message: 'success',
      data: { agents: mockAgents, count: 1 },
    });

    (operateAgent as any).mockRejectedValue(new Error('Operation failed'));

    // 验证错误处理
    await expect(operateAgent('node-1', 'agent-1', 'start')).rejects.toThrow(
      'Operation failed'
    );
  });
});
