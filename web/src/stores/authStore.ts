/**
 * 认证状态管理 Store
 */

import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import type { User } from '../types';
import { storage } from '../utils';

interface AuthState {
  user: User | null;
  token: string | null;
  isAuthenticated: boolean;
  _hasHydrated: boolean; // 标记是否已完成水合（从持久化存储恢复）

  // Actions
  setAuth: (user: User, token: string) => void;
  clearAuth: () => void;
  updateUser: (user: User) => void;
  setHasHydrated: (state: boolean) => void;
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set) => ({
      user: null,
      token: null,
      isAuthenticated: false,
      _hasHydrated: false,

      // 设置认证信息
      setAuth: (user, token) => {
        storage.setUser(user);
        storage.setToken(token);
        set({
          user,
          token,
          isAuthenticated: true,
        });
      },

      // 清除认证信息
      clearAuth: () => {
        storage.clear();
        set({
          user: null,
          token: null,
          isAuthenticated: false,
        });
      },

      // 更新用户信息
      updateUser: (user) => {
        storage.setUser(user);
        set({ user });
      },

      // 设置水合状态
      setHasHydrated: (state) => {
        set({ _hasHydrated: state });
      },
    }),
    {
      name: 'ops-auth-storage', // localStorage key
      partialize: (state) => ({
        user: state.user,
        token: state.token,
        isAuthenticated: state.user !== null && state.token !== null,
      }),
      onRehydrateStorage: () => (state, error) => {
        // 水合完成后，根据存储的数据设置 isAuthenticated
        if (error) {
          console.error('Failed to rehydrate auth state:', error);
        }
        if (state) {
          // 根据恢复的数据重新计算 isAuthenticated
          state.isAuthenticated = state.user !== null && state.token !== null;
          // 标记水合完成
          state.setHasHydrated(true);
        }
      },
    }
  )
);
