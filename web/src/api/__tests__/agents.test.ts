/**
 * Agent API 客户端测试
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { listAgents, operateAgent, getAgentLogs } from '../agents';
import type { APIResponse, AgentListResponse, AgentLogsResponse } from '../../types';

// Mock axios client
const mockClient = {
  get: vi.fn(),
  post: vi.fn(),
};

// Mock interceptors to return our mock client
vi.mock('../interceptors', () => ({
  default: mockClient,
}));

describe('Agent API', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe('listAgents', () => {
    it('应该成功获取 Agent 列表', async () => {
      const mockResponse: APIResponse<AgentListResponse> = {
        code: 0,
        message: 'success',
        data: {
          agents: [
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
          ],
          count: 1,
        },
      };

      mockClient.get.mockResolvedValue({ data: mockResponse });

      const result = await listAgents('node-1');

      expect(mockClient.get).toHaveBeenCalledWith('/api/v1/nodes/node-1/agents');
      expect(result).toEqual(mockResponse);
      expect(result.data.agents).toHaveLength(1);
      expect(result.data.agents[0].agent_id).toBe('agent-1');
    });

    it('应该处理 API 错误', async () => {
      const mockError = new Error('Network error');
      mockClient.get.mockRejectedValue(mockError);

      await expect(listAgents('node-1')).rejects.toThrow('Network error');
      expect(mockClient.get).toHaveBeenCalledWith('/api/v1/nodes/node-1/agents');
    });

    it('应该处理空列表', async () => {
      const mockResponse: APIResponse<AgentListResponse> = {
        code: 0,
        message: 'success',
        data: {
          agents: [],
          count: 0,
        },
      };

      mockClient.get.mockResolvedValue({ data: mockResponse });

      const result = await listAgents('node-1');

      expect(result.data.agents).toHaveLength(0);
      expect(result.data.count).toBe(0);
    });
  });

  describe('operateAgent', () => {
    it('应该成功启动 Agent', async () => {
      const mockResponse: APIResponse = {
        code: 0,
        message: 'Agent started successfully',
        data: null,
      };

      mockClient.post.mockResolvedValue({ data: mockResponse });

      const result = await operateAgent('node-1', 'agent-1', 'start');

      expect(mockClient.post).toHaveBeenCalledWith(
        '/api/v1/nodes/node-1/agents/agent-1/operate',
        { operation: 'start' }
      );
      expect(result).toEqual(mockResponse);
    });

    it('应该成功停止 Agent', async () => {
      const mockResponse: APIResponse = {
        code: 0,
        message: 'Agent stopped successfully',
        data: null,
      };

      mockClient.post.mockResolvedValue({ data: mockResponse });

      const result = await operateAgent('node-1', 'agent-1', 'stop');

      expect(mockClient.post).toHaveBeenCalledWith(
        '/api/v1/nodes/node-1/agents/agent-1/operate',
        { operation: 'stop' }
      );
      expect(result).toEqual(mockResponse);
    });

    it('应该成功重启 Agent', async () => {
      const mockResponse: APIResponse = {
        code: 0,
        message: 'Agent restarted successfully',
        data: null,
      };

      mockClient.post.mockResolvedValue({ data: mockResponse });

      const result = await operateAgent('node-1', 'agent-1', 'restart');

      expect(mockClient.post).toHaveBeenCalledWith(
        '/api/v1/nodes/node-1/agents/agent-1/operate',
        { operation: 'restart' }
      );
      expect(result).toEqual(mockResponse);
    });

    it('应该处理操作失败', async () => {
      const mockError = new Error('Operation failed');
      mockClient.post.mockRejectedValue(mockError);

      await expect(operateAgent('node-1', 'agent-1', 'start')).rejects.toThrow(
        'Operation failed'
      );
    });
  });

  describe('getAgentLogs', () => {
    it('应该成功获取 Agent 日志（默认行数）', async () => {
      const mockResponse: APIResponse<AgentLogsResponse> = {
        code: 0,
        message: 'success',
        data: {
          logs: ['Log line 1', 'Log line 2', 'Log line 3'],
          count: 3,
        },
      };

      mockClient.get.mockResolvedValue({ data: mockResponse });

      const result = await getAgentLogs('node-1', 'agent-1');

      expect(mockClient.get).toHaveBeenCalledWith(
        '/api/v1/nodes/node-1/agents/agent-1/logs',
        { params: {} }
      );
      expect(result).toEqual(mockResponse);
      expect(result.data.logs).toHaveLength(3);
    });

    it('应该成功获取指定行数的日志', async () => {
      const mockResponse: APIResponse<AgentLogsResponse> = {
        code: 0,
        message: 'success',
        data: {
          logs: Array.from({ length: 50 }, (_, i) => `Log line ${i + 1}`),
          count: 50,
        },
      };

      mockClient.get.mockResolvedValue({ data: mockResponse });

      const result = await getAgentLogs('node-1', 'agent-1', 50);

      expect(mockClient.get).toHaveBeenCalledWith(
        '/api/v1/nodes/node-1/agents/agent-1/logs',
        { params: { lines: 50 } }
      );
      expect(result.data.logs).toHaveLength(50);
    });

    it('应该处理空日志', async () => {
      const mockResponse: APIResponse<AgentLogsResponse> = {
        code: 0,
        message: 'success',
        data: {
          logs: [],
          count: 0,
        },
      };

      mockClient.get.mockResolvedValue({ data: mockResponse });

      const result = await getAgentLogs('node-1', 'agent-1');

      expect(result.data.logs).toHaveLength(0);
      expect(result.data.count).toBe(0);
    });

    it('应该处理获取日志失败', async () => {
      const mockError = new Error('Failed to fetch logs');
      mockClient.get.mockRejectedValue(mockError);

      await expect(getAgentLogs('node-1', 'agent-1')).rejects.toThrow(
        'Failed to fetch logs'
      );
    });
  });
});
