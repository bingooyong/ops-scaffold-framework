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

export enum UserRole {
  Admin = 'admin',
  User = 'user',
  Guest = 'guest',
}

export enum UserStatus {
  Active = 'active',
  Disabled = 'disabled',
  Locked = 'locked',
}

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
