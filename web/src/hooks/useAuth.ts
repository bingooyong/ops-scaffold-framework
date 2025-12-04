/**
 * 认证相关 Hook
 */

import { useMutation } from '@tanstack/react-query';
import { useAuthStore } from '../stores';
import { login, register, changePassword, getProfile } from '../api';
import type { LoginRequest, RegisterRequest, ChangePasswordRequest } from '../types';

/**
 * 使用认证 Hook
 */
export function useAuth() {
  const { setAuth, clearAuth, updateUser, isAuthenticated, user } = useAuthStore();

  // 登录
  const loginMutation = useMutation({
    mutationFn: (data: LoginRequest) => login(data),
    onSuccess: (response) => {
      const { token, user } = response.data;
      setAuth(user, token);
    },
  });

  // 注册
  const registerMutation = useMutation({
    mutationFn: (data: RegisterRequest) => register(data),
  });

  // 修改密码
  const changePasswordMutation = useMutation({
    mutationFn: (data: ChangePasswordRequest) => changePassword(data),
  });

  // 获取用户资料
  const getProfileMutation = useMutation({
    mutationFn: () => getProfile(),
    onSuccess: (response) => {
      updateUser(response.data.user);
    },
  });

  // 登出
  const logout = () => {
    clearAuth();
  };

  return {
    // 状态
    isAuthenticated,
    user,

    // 操作
    login: loginMutation.mutateAsync,
    register: registerMutation.mutateAsync,
    changePassword: changePasswordMutation.mutateAsync,
    getProfile: getProfileMutation.mutateAsync,
    logout,

    // 加载状态
    isLoggingIn: loginMutation.isPending,
    isRegistering: registerMutation.isPending,
    isChangingPassword: changePasswordMutation.isPending,
    isLoadingProfile: getProfileMutation.isPending,

    // 错误信息
    loginError: loginMutation.error,
    registerError: registerMutation.error,
    changePasswordError: changePasswordMutation.error,
    profileError: getProfileMutation.error,
  };
}
