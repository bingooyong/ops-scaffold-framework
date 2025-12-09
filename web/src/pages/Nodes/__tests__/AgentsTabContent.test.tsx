/**
 * AgentsTabContent 组件测试
 */

import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '../../../test/testUtils';
import userEvent from '@testing-library/user-event';
// import { QueryClient } from '@tanstack/react-query';
import { AgentsTabContent } from '../Detail';
import { NodeDetail } from '../index';
import { listAgents, operateAgent } from '../../../api';
import type { Agent } from '../../../types';

// Mock API functions
vi.mock('../../../api', () => ({
  listAgents: vi.fn(),
  operateAgent: vi.fn(),
  getAgentLogs: vi.fn(),
}));

// Mock useAgentMetricsHistory hook
vi.mock('../../../hooks/useAgentMetrics', () => ({
  useAgentMetricsHistory: vi.fn(() => ({
    data: null,
    isLoading: false,
    error: null,
    refetch: vi.fn(),
  })),
}));

// Mock React.lazy for MetricsChart
vi.mock('react', async () => {
  const actual = await vi.importActual('react');
  return {
    ...actual,
    lazy: (_fn: () => Promise<any>) => {
      const Component = ({ title }: { title: string }) => <div data-testid="metrics-chart">{title}</div>;
      Component.displayName = 'LazyComponent';
      return Component;
    },
  };
});

// Mock MetricsChart component (lazy loaded)
vi.mock('../../../components/Metrics/MetricsChart', () => ({
  default: ({ title }: { title: string }) => <div data-testid="metrics-chart">{title}</div>,
}));

// Mock other Metrics components
vi.mock('../../../components/Metrics', () => ({
  CPUCard: ({ nodeId }: { nodeId: string }) => <div data-testid="cpu-card">{nodeId}</div>,
  MemoryCard: ({ nodeId }: { nodeId: string }) => <div data-testid="memory-card">{nodeId}</div>,
  DiskCard: ({ nodeId }: { nodeId: string }) => <div data-testid="disk-card">{nodeId}</div>,
  NetworkCard: ({ nodeId }: { nodeId: string }) => <div data-testid="network-card">{nodeId}</div>,
  TimeRangeSelector: () => (
    <div data-testid="time-range-selector">TimeRangeSelector</div>
  ),
  RefreshControl: () => (
    <div data-testid="refresh-control">RefreshControl</div>
  ),
}));

// Mock formatMetricsHistory
vi.mock('../../../utils/metricsFormat', () => ({
  formatMetricsHistory: vi.fn((data) => data || []),
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
  {
    id: 2,
    node_id: 'node-1',
    agent_id: 'agent-2',
    type: 'telegraf',
    version: '2.0.0',
    status: 'stopped',
    pid: undefined,
    last_heartbeat: undefined,
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
  },
];

// Mock useParams and useNavigate
vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual('react-router-dom');
  return {
    ...actual,
    useParams: () => ({ id: 'node-1' }),
    useNavigate: () => vi.fn(),
    Link: ({ children, to }: any) => <a href={to}>{children}</a>,
  };
});

// Mock useNode hook
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

// Mock useMetricsStore
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

describe('AgentsTabContent', () => {
  // let queryClient: QueryClient;

  beforeEach(() => {
    /*
    queryClient = new QueryClient({
      defaultOptions: {
        queries: { retry: false },
        mutations: { retry: false },
      },
    });
    */
    vi.clearAllMocks();
  });

  it('应该显示加载状态', async () => {
    const user = userEvent.setup();
    (listAgents as any).mockImplementation(() => new Promise(() => {})); // Never resolves

    render(<NodeDetail />);
    
    // 切换到 Agents Tab
    const agentsTab = screen.getByRole('tab', { name: /agents/i });
    await user.click(agentsTab);

    expect(screen.getByRole('progressbar')).toBeInTheDocument();
  });

  it('应该显示错误状态', async () => {
    const user = userEvent.setup();
    const error = new Error('Failed to load agents');
    (listAgents as any).mockRejectedValue(error);

    render(<NodeDetail />);
    
    const agentsTab = screen.getByRole('tab', { name: /agents/i });
    await user.click(agentsTab);

    await waitFor(() => {
      expect(screen.getByText(/加载 Agent 列表失败/i)).toBeInTheDocument();
    });
  });

  it('应该显示空状态', async () => {
    const user = userEvent.setup();
    (listAgents as any).mockResolvedValue({
      code: 0,
      message: 'success',
      data: { agents: [], count: 0 },
    });

    render(<NodeDetail />);
    
    const agentsTab = screen.getByRole('tab', { name: /agents/i });
    await user.click(agentsTab);

    await waitFor(() => {
      expect(screen.getByText(/该节点下暂无 Agent/i)).toBeInTheDocument();
    });
  });

  it('应该正确显示 Agent 列表', async () => {
    (listAgents as any).mockResolvedValue({
      code: 0,
      message: 'success',
      data: { agents: mockAgents, count: 2 },
    });

    render(<AgentsTabContent nodeId="node-1" />);

    await waitFor(() => {
      expect(screen.getByText('agent-1')).toBeInTheDocument();
      expect(screen.getByText('agent-2')).toBeInTheDocument();
    });

    // 验证表格列
    expect(screen.getByText('Agent ID')).toBeInTheDocument();
    expect(screen.getByText('类型')).toBeInTheDocument();
    expect(screen.getByText('版本')).toBeInTheDocument();
    expect(screen.getByText('状态')).toBeInTheDocument();
    expect(screen.getByText('PID')).toBeInTheDocument();
    expect(screen.getByText('最后心跳')).toBeInTheDocument();
  });

  it('应该正确显示 Agent 状态 Chip', async () => {
    const user = userEvent.setup();
    (listAgents as any).mockResolvedValue({
      code: 0,
      message: 'success',
      data: { agents: mockAgents, count: 2 },
    });

    render(<NodeDetail />);
    
    const agentsTab = screen.getByRole('tab', { name: /agents/i });
    await user.click(agentsTab);

    await waitFor(() => {
      expect(screen.getByText('运行中')).toBeInTheDocument();
      expect(screen.getByText('已停止')).toBeInTheDocument();
    });
  });

  it('应该能够启动 Agent', async () => {
    const user = userEvent.setup();
    (listAgents as any).mockResolvedValue({
      code: 0,
      message: 'success',
      data: { agents: [mockAgents[1]], count: 1 }, // stopped agent
    });
    (operateAgent as any).mockResolvedValue({
      code: 0,
      message: 'Agent started successfully',
      data: null,
    });

    render(<NodeDetail />);
    
    const agentsTab = screen.getByRole('tab', { name: /agents/i });
    await user.click(agentsTab);

    await waitFor(() => {
      expect(screen.getByText('agent-2')).toBeInTheDocument();
    });

    // 找到启动按钮并点击
    const startButtons = screen.getAllByLabelText(/启动/i);
    const startButton = startButtons.find((btn) => !btn.closest('span')?.hasAttribute('aria-disabled'));
    
    if (startButton) {
      await user.click(startButton);

      await waitFor(() => {
        expect(operateAgent).toHaveBeenCalledWith('node-1', 'agent-2', 'start');
      });
    }
  });

  it('应该能够停止 Agent', async () => {
    const user = userEvent.setup();
    (listAgents as any).mockResolvedValue({
      code: 0,
      message: 'success',
      data: { agents: [mockAgents[0]], count: 1 }, // running agent
    });
    (operateAgent as any).mockResolvedValue({
      code: 0,
      message: 'Agent stopped successfully',
      data: null,
    });

    render(<NodeDetail />);
    
    const agentsTab = screen.getByRole('tab', { name: /agents/i });
    await user.click(agentsTab);

    await waitFor(() => {
      expect(screen.getByText('agent-1')).toBeInTheDocument();
    });

    // 找到停止按钮并点击
    const stopButtons = screen.getAllByLabelText(/停止/i);
    const stopButton = stopButtons.find((btn) => !btn.closest('span')?.hasAttribute('aria-disabled'));
    
    if (stopButton) {
      await user.click(stopButton);

      await waitFor(() => {
        expect(operateAgent).toHaveBeenCalledWith('node-1', 'agent-1', 'stop');
      });
    }
  });

  it('应该能够重启 Agent', async () => {
    const user = userEvent.setup();
    (listAgents as any).mockResolvedValue({
      code: 0,
      message: 'success',
      data: { agents: [mockAgents[0]], count: 1 }, // running agent
    });
    (operateAgent as any).mockResolvedValue({
      code: 0,
      message: 'Agent restarted successfully',
      data: null,
    });

    render(<NodeDetail />);
    
    const agentsTab = screen.getByRole('tab', { name: /agents/i });
    await user.click(agentsTab);

    await waitFor(() => {
      expect(screen.getByText('agent-1')).toBeInTheDocument();
    });

    // 找到重启按钮并点击
    const restartButtons = screen.getAllByLabelText(/重启/i);
    const restartButton = restartButtons.find((btn) => !btn.closest('span')?.hasAttribute('aria-disabled'));
    
    if (restartButton) {
      await user.click(restartButton);

      await waitFor(() => {
        expect(operateAgent).toHaveBeenCalledWith('node-1', 'agent-1', 'restart');
      });
    }
  });

  it('应该禁用已运行 Agent 的启动按钮', async () => {
    const user = userEvent.setup();
    (listAgents as any).mockResolvedValue({
      code: 0,
      message: 'success',
      data: { agents: [mockAgents[0]], count: 1 }, // running agent
    });

    render(<NodeDetail />);
    
    const agentsTab = screen.getByRole('tab', { name: /agents/i });
    await user.click(agentsTab);

    await waitFor(() => {
      expect(screen.getByText('agent-1')).toBeInTheDocument();
    });

    // 启动按钮应该被禁用
    const startButtons = screen.getAllByLabelText(/启动/i);
    const startButton = startButtons[0];
    expect(startButton.closest('span')).toHaveAttribute('aria-disabled', 'true');
  });

  it('应该禁用已停止 Agent 的停止和重启按钮', async () => {
    const user = userEvent.setup();
    (listAgents as any).mockResolvedValue({
      code: 0,
      message: 'success',
      data: { agents: [mockAgents[1]], count: 1 }, // stopped agent
    });

    render(<NodeDetail />);
    
    const agentsTab = screen.getByRole('tab', { name: /agents/i });
    await user.click(agentsTab);

    await waitFor(() => {
      expect(screen.getByText('agent-2')).toBeInTheDocument();
    });

    // 停止和重启按钮应该被禁用
    const stopButtons = screen.getAllByLabelText(/停止/i);
    const restartButtons = screen.getAllByLabelText(/重启/i);
    
    if (stopButtons[0]) {
      expect(stopButtons[0].closest('span')).toHaveAttribute('aria-disabled', 'true');
    }
    if (restartButtons[0]) {
      expect(restartButtons[0].closest('span')).toHaveAttribute('aria-disabled', 'true');
    }
  });

  it('应该能够选择 Agent', async () => {
    const user = userEvent.setup();
    (listAgents as any).mockResolvedValue({
      code: 0,
      message: 'success',
      data: { agents: mockAgents, count: 2 },
    });

    render(<NodeDetail />);
    
    const agentsTab = screen.getByRole('tab', { name: /agents/i });
    await user.click(agentsTab);

    await waitFor(() => {
      expect(screen.getByText('agent-1')).toBeInTheDocument();
    });

    // 点击 Agent 行
    const agentRow = screen.getByText('agent-1').closest('tr');
    if (agentRow) {
      await user.click(agentRow);
      
      // 验证 Agent 被选中（通过检查监控图表区域是否显示）
      await waitFor(() => {
        expect(screen.getByText(/Agent 监控/i)).toBeInTheDocument();
      });
    }
  });

  it('应该显示操作成功提示', async () => {
    const user = userEvent.setup();
    (listAgents as any).mockResolvedValue({
      code: 0,
      message: 'success',
      data: { agents: [mockAgents[1]], count: 1 },
    });
    (operateAgent as any).mockResolvedValue({
      code: 0,
      message: 'Agent started successfully',
      data: null,
    });

    render(<NodeDetail />);
    
    const agentsTab = screen.getByRole('tab', { name: /agents/i });
    await user.click(agentsTab);

    await waitFor(() => {
      expect(screen.getByText('agent-2')).toBeInTheDocument();
    });

    const startButtons = screen.getAllByLabelText(/启动/i);
    const startButton = startButtons.find((btn) => !btn.closest('span')?.hasAttribute('aria-disabled'));
    
    if (startButton) {
      await user.click(startButton);

      await waitFor(() => {
        expect(screen.getByText(/启动成功/i)).toBeInTheDocument();
      });
    }
  });
});
