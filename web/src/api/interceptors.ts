/**
 * Axios 拦截器配置
 */

import type { AxiosError, InternalAxiosRequestConfig, AxiosResponse } from 'axios';
import client from './client';
import { useAuthStore } from '../stores';
import type { APIResponse } from '../types';
import { ErrorCode } from '../types';

/**
 * 请求拦截器
 */
client.interceptors.request.use(
  (config: InternalAxiosRequestConfig) => {
    // 添加 Token（从 zustand store 获取）
    const token = useAuthStore.getState().token;
    if (token && config.headers) {
      config.headers.Authorization = `Bearer ${token}`;
    }

    return config;
  },
  (error: AxiosError) => {
    return Promise.reject(error);
  }
);

/**
 * 响应拦截器
 */
client.interceptors.response.use(
  (response: AxiosResponse<APIResponse>) => {
    const { data } = response;

    // 检查业务错误码
    if (data.code !== ErrorCode.Success) {
      // Token 相关错误，清除登录状态
      if (
        data.code === ErrorCode.Unauthorized ||
        data.code === ErrorCode.TokenExpired ||
        data.code === ErrorCode.TokenInvalid ||
        data.code === ErrorCode.InvalidToken
      ) {
        useAuthStore.getState().clearAuth();
        window.location.href = '/login';
      }

      // 抛出业务错误
      return Promise.reject(new Error(data.message || '请求失败'));
    }

    return response;
  },
  (error: AxiosError<APIResponse>) => {
    // 网络错误或服务器错误
    if (error.response) {
      const { status, data } = error.response;

      // 根据 HTTP 状态码处理
      switch (status) {
        case 401:
          useAuthStore.getState().clearAuth();
          window.location.href = '/login';
          return Promise.reject(new Error('未授权，请重新登录'));

        case 403:
          return Promise.reject(new Error('没有权限访问'));

        case 404:
          return Promise.reject(new Error('请求的资源不存在'));

        case 500:
          return Promise.reject(new Error('服务器内部错误'));

        default:
          return Promise.reject(new Error(data?.message || '请求失败'));
      }
    } else if (error.request) {
      // 网络错误 - 提供更详细的错误信息
      const baseURL = error.config?.baseURL || '未知';
      const url = error.config?.url || '未知';
      const message = `网络连接失败，无法连接到服务器 ${baseURL}${url}。请检查：
1. Manager 服务是否已启动（运行 make run-dev）
2. API 地址配置是否正确（检查 .env.development 文件）
3. 防火墙或网络设置`;
      return Promise.reject(new Error(message));
    } else {
      return Promise.reject(new Error(error.message || '请求失败'));
    }
  }
);

export default client;
