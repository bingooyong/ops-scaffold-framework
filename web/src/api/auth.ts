/**
 * 认证相关 API
 */

import client from './interceptors';
import type {
  APIResponse,
  LoginRequest,
  LoginResponse,
  RegisterRequest,
  RegisterResponse,
  ChangePasswordRequest,
  User,
} from '../types';

/**
 * 用户登录
 */
export function login(data: LoginRequest): Promise<APIResponse<LoginResponse>> {
  return client.post('/api/v1/auth/login', data).then((res) => res.data);
}

/**
 * 用户注册
 */
export function register(data: RegisterRequest): Promise<APIResponse<RegisterResponse>> {
  return client.post('/api/v1/auth/register', data).then((res) => res.data);
}

/**
 * 获取用户资料
 */
export function getProfile(): Promise<APIResponse<{ user: User }>> {
  return client.get('/api/v1/auth/profile').then((res) => res.data);
}

/**
 * 修改密码
 */
export function changePassword(data: ChangePasswordRequest): Promise<APIResponse> {
  return client.post('/api/v1/auth/change-password', data).then((res) => res.data);
}
