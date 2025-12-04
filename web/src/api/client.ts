/**
 * Axios 客户端配置
 */

import axios from 'axios';
import type { AxiosInstance } from 'axios';

// 创建 Axios 实例
export const client: AxiosInstance = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || 'http://127.0.0.1:8080',
  timeout: Number(import.meta.env.VITE_API_TIMEOUT) || 30000,
  headers: {
    'Content-Type': 'application/json',
  },
});

export default client;
