/**
 * API 响应基础类型定义
 */

// 统一响应格式
export interface APIResponse<T = unknown> {
  code: number;
  message: string;
  data: T;
  timestamp?: string;
}

// 分页信息
export interface PageInfo {
  page: number;
  page_size: number;
  total: number;
  total_pages?: number;
}

// 分页响应
export interface PageResponse<T = unknown> {
  list: T[];
  page_info: PageInfo;
}

// 错误码常量
export const ErrorCode = {
  Success: 0,

  // 客户端错误 1xxx
  InvalidRequest: 1001,
  MissingParameter: 1002,
  InvalidParameter: 1003,
  Unauthorized: 1004,
  TokenExpired: 1005,
  TokenInvalid: 1006,
  PermissionDenied: 1007,
  InvalidToken: 1008,
  InvalidCredentials: 1009,

  // 业务错误 2xxx
  UserNotFound: 2001,
  NodeNotFound: 2002,
  TaskNotFound: 2003,
  VersionNotFound: 2004,
  UserExists: 2009,
  NodeOffline: 2101,
  NodeBusy: 2102,

  // 服务器错误 5xxx
  InternalError: 5001,
  DatabaseError: 5002,
} as const;

export type ErrorCode = typeof ErrorCode[keyof typeof ErrorCode];
