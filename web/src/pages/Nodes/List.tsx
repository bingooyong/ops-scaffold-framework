/**
 * 节点列表页面
 */

import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  Box,
  Card,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  TablePagination,
  Typography,
  Chip,
  IconButton,
  CircularProgress,
} from '@mui/material';
import { Delete as DeleteIcon, Refresh as RefreshIcon } from '@mui/icons-material';
import { useNodes, useDeleteNode } from '../../hooks';
import { formatDateTime } from '../../utils';
import type { NodeStatus } from '../../types';

export default function NodeList() {
  const navigate = useNavigate();
  const [page, setPage] = useState(0);
  const [pageSize, setPageSize] = useState(20);

  const { data, isLoading, refetch } = useNodes({
    page: page + 1,
    page_size: pageSize,
  });

  const { mutate: deleteNodeMutation } = useDeleteNode();

  const nodes = data?.data?.list || [];
  const total = data?.data?.page_info?.total || 0;

  const handleChangePage = (_event: unknown, newPage: number) => {
    setPage(newPage);
  };

  const handleChangeRowsPerPage = (event: React.ChangeEvent<HTMLInputElement>) => {
    setPageSize(parseInt(event.target.value, 10));
    setPage(0);
  };

  const handleDelete = (id: number) => {
    if (window.confirm('确定要删除这个节点吗？')) {
      deleteNodeMutation(id);
    }
  };

  const getStatusColor = (status: NodeStatus) => {
    switch (status) {
      case 'online':
        return 'success';
      case 'offline':
        return 'error';
      default:
        return 'default';
    }
  };

  const getStatusText = (status: NodeStatus) => {
    switch (status) {
      case 'online':
        return '在线';
      case 'offline':
        return '离线';
      default:
        return '未知';
    }
  };

  return (
    <Box>
      <Box display="flex" justifyContent="space-between" alignItems="center" mb={3}>
        <Typography variant="h4">节点管理</Typography>
        <IconButton onClick={() => refetch()}>
          <RefreshIcon />
        </IconButton>
      </Box>

      <Card>
        <TableContainer>
          <Table>
            <TableHead>
              <TableRow>
                <TableCell>节点ID</TableCell>
                <TableCell>主机名</TableCell>
                <TableCell>IP地址</TableCell>
                <TableCell>操作系统</TableCell>
                <TableCell>架构</TableCell>
                <TableCell>状态</TableCell>
                <TableCell>最后心跳</TableCell>
                <TableCell>操作</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {isLoading ? (
                <TableRow>
                  <TableCell colSpan={8} align="center">
                    <CircularProgress />
                  </TableCell>
                </TableRow>
              ) : nodes.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={8} align="center">
                    暂无数据
                  </TableCell>
                </TableRow>
              ) : (
                nodes.map((node) => (
                  <TableRow key={node.id} hover>
                    <TableCell>{node.node_id}</TableCell>
                    <TableCell>
                      <Typography
                        component="a"
                        onClick={() => navigate(`/nodes/${node.node_id}`)}
                        sx={{
                          cursor: 'pointer',
                          color: 'primary.main',
                          textDecoration: 'none',
                          '&:hover': {
                            textDecoration: 'underline',
                          },
                        }}
                      >
                        {node.hostname}
                      </Typography>
                    </TableCell>
                    <TableCell>{node.ip}</TableCell>
                    <TableCell>{node.os}</TableCell>
                    <TableCell>{node.arch}</TableCell>
                    <TableCell>
                      <Chip
                        label={getStatusText(node.status)}
                        color={getStatusColor(node.status)}
                        size="small"
                      />
                    </TableCell>
                    <TableCell>
                      {node.last_heartbeat_at
                        ? formatDateTime(node.last_heartbeat_at)
                        : '-'}
                    </TableCell>
                    <TableCell>
                      <IconButton
                        size="small"
                        color="error"
                        onClick={() => handleDelete(node.id)}
                      >
                        <DeleteIcon />
                      </IconButton>
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </TableContainer>
        <TablePagination
          component="div"
          count={total}
          page={page}
          onPageChange={handleChangePage}
          rowsPerPage={pageSize}
          onRowsPerPageChange={handleChangeRowsPerPage}
          rowsPerPageOptions={[10, 20, 50, 100]}
          labelRowsPerPage="每页行数:"
          labelDisplayedRows={({ from, to, count }) =>
            `${from}-${to} / 共 ${count !== -1 ? count : `超过 ${to}`}`
          }
        />
      </Card>
    </Box>
  );
}
