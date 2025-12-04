/**
 * Dashboard 页面
 */

import { Box, Grid, Card, CardContent, Typography } from '@mui/material';
import { useNodeStatistics } from '../../hooks';

export default function Dashboard() {
  const { data, isLoading } = useNodeStatistics();

  const stats = data?.data?.statistics;

  return (
    <Box>
      <Typography variant="h4" gutterBottom>
        仪表盘
      </Typography>

      <Grid container spacing={3} mt={2}>
        <Grid item xs={12} md={4}>
          <Card>
            <CardContent>
              <Typography color="text.secondary" gutterBottom>
                总节点数
              </Typography>
              <Typography variant="h3">
                {isLoading ? '-' : stats?.total || 0}
              </Typography>
            </CardContent>
          </Card>
        </Grid>

        <Grid item xs={12} md={4}>
          <Card>
            <CardContent>
              <Typography color="text.secondary" gutterBottom>
                在线节点
              </Typography>
              <Typography variant="h3" color="success.main">
                {isLoading ? '-' : stats?.online || 0}
              </Typography>
            </CardContent>
          </Card>
        </Grid>

        <Grid item xs={12} md={4}>
          <Card>
            <CardContent>
              <Typography color="text.secondary" gutterBottom>
                离线节点
              </Typography>
              <Typography variant="h3" color="error.main">
                {isLoading ? '-' : stats?.offline || 0}
              </Typography>
            </CardContent>
          </Card>
        </Grid>
      </Grid>
    </Box>
  );
}
