/**
 * 用户相关类型定义
 */

export interface User {
  id: number;
  username: string;
  email: string;
  role: UserRole;
  status: UserStatus;
  created_at: string;
  updated_at: string;
  last_login_at?: string;
}

export const UserRole = {
  Admin: 'admin',
  User: 'user',
  Guest: 'guest',
} as const;

export type UserRole = typeof UserRole[keyof typeof UserRole];

export const UserStatus = {
  Active: 'active',
  Disabled: 'disabled',
  Locked: 'locked',
} as const;

export type UserStatus = typeof UserStatus[keyof typeof UserStatus];

export interface LoginRequest {
  username: string;
  password: string;
}

export interface LoginResponse {
  token: string;
  user: User;
}

export interface RegisterRequest {
  username: string;
  password: string;
  email: string;
}

export interface RegisterResponse {
  user: User;
}

export interface ChangePasswordRequest {
  old_password: string;
  new_password: string;
}
