/**
 * 节点管理相关 Hook
 */

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { getNodes, getNode, deleteNode, getNodeStatistics } from '../api';

/**
 * 使用节点列表
 */
export function useNodes(params: {
  page?: number;
  page_size?: number;
  status?: string;
}) {
  return useQuery({
    queryKey: ['nodes', params],
    queryFn: () => getNodes(params),
  });
}

/**
 * 使用节点详情
 */
export function useNode(id: string) {
  return useQuery({
    queryKey: ['node', id],
    queryFn: () => getNode(id),
    enabled: !!id,
  });
}

/**
 * 使用节点统计
 */
export function useNodeStatistics() {
  return useQuery({
    queryKey: ['node-statistics'],
    queryFn: () => getNodeStatistics(),
  });
}

/**
 * 使用删除节点
 */
export function useDeleteNode() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: number) => deleteNode(id),
    onSuccess: () => {
      // 重新获取节点列表
      queryClient.invalidateQueries({ queryKey: ['nodes'] });
      queryClient.invalidateQueries({ queryKey: ['node-statistics'] });
    },
  });
}
