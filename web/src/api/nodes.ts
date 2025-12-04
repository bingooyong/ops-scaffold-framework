/**
 * 节点管理相关 API
 */

import client from './interceptors';
import type { APIResponse, PageResponse, Node, NodeStatistics } from '../types';

/**
 * 获取节点列表
 */
export function getNodes(params: {
  page?: number;
  page_size?: number;
  status?: string;
}): Promise<APIResponse<PageResponse<Node>>> {
  return client
    .get('/api/v1/nodes', { params })
    .then((res) => res.data);
}

/**
 * 获取节点详情
 */
export function getNode(id: string): Promise<APIResponse<{ node: Node }>> {
  return client
    .get(`/api/v1/nodes/${id}`)
    .then((res) => res.data);
}

/**
 * 删除节点
 */
export function deleteNode(id: number): Promise<APIResponse> {
  return client
    .delete(`/api/v1/nodes/${id}`)
    .then((res) => res.data);
}

/**
 * 获取节点统计信息
 */
export function getNodeStatistics(): Promise<APIResponse<{ statistics: NodeStatistics }>> {
  return client
    .get('/api/v1/nodes/statistics')
    .then((res) => res.data);
}
