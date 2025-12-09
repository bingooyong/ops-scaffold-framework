/**
 * 本地存储工具
 */

import type { User } from '../types/user';

const TOKEN_KEY = 'ops_token';
const USER_KEY = 'ops_user';

export const storage = {
  // Token 相关
  getToken(): string | null {
    return localStorage.getItem(TOKEN_KEY);
  },

  setToken(token: string): void {
    localStorage.setItem(TOKEN_KEY, token);
  },

  removeToken(): void {
    localStorage.removeItem(TOKEN_KEY);
  },

  // 用户信息相关
  getUser(): User | null {
    const userStr = localStorage.getItem(USER_KEY);
    if (!userStr) return null;

    try {
      return JSON.parse(userStr) as User;
    } catch {
      return null;
    }
  },

  setUser(user: User): void {
    localStorage.setItem(USER_KEY, JSON.stringify(user));
  },

  removeUser(): void {
    localStorage.removeItem(USER_KEY);
  },

  // 清除所有数据
  clear(): void {
    this.removeToken();
    this.removeUser();
  },
};
