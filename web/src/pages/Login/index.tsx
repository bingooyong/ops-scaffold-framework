/**
 * 登录页面
 */

import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  Box,
  Card,
  CardContent,
  TextField,
  Button,
  Typography,
  Alert,
  Container,
} from '@mui/material';
import { useForm } from 'react-hook-form';
import { useAuth } from '../../hooks';
import type { LoginRequest } from '../../types';

export default function Login() {
  const navigate = useNavigate();
  const { login, isLoggingIn, loginError } = useAuth();
  const [error, setError] = useState<string>('');

  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<LoginRequest>();

  const onSubmit = async (data: LoginRequest) => {
    try {
      setError('');
      await login(data);
      navigate('/dashboard');
    } catch (err: unknown) {
      const errorMessage = err instanceof Error ? err.message : '登录失败，请重试';
      setError(errorMessage);
    }
  };

  return (
    <Container maxWidth="sm">
      <Box
        sx={{
          minHeight: '100vh',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
        }}
      >
        <Card sx={{ width: '100%', maxWidth: 500 }}>
          <CardContent sx={{ p: 4 }}>
            <Typography variant="h4" component="h1" gutterBottom align="center">
              Ops Scaffold Framework
            </Typography>
            <Typography variant="body2" color="text.secondary" align="center" mb={4}>
              分布式运维管理平台
            </Typography>

            {(error || loginError) && (
              <Alert severity="error" sx={{ mb: 2 }}>
                {error || loginError?.message}
              </Alert>
            )}

            <form onSubmit={handleSubmit(onSubmit)}>
              <TextField
                fullWidth
                label="用户名"
                margin="normal"
                {...register('username', {
                  required: '请输入用户名',
                })}
                error={!!errors.username}
                helperText={errors.username?.message}
                disabled={isLoggingIn}
              />

              <TextField
                fullWidth
                label="密码"
                type="password"
                margin="normal"
                {...register('password', {
                  required: '请输入密码',
                })}
                error={!!errors.password}
                helperText={errors.password?.message}
                disabled={isLoggingIn}
              />

              <Button
                fullWidth
                type="submit"
                variant="contained"
                size="large"
                sx={{ mt: 3 }}
                disabled={isLoggingIn}
              >
                {isLoggingIn ? '登录中...' : '登录'}
              </Button>
            </form>
          </CardContent>
        </Card>
      </Box>
    </Container>
  );
}
