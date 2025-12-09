# 测试运行指南

## 安装依赖

首先确保已安装所有测试依赖：

```bash
cd web
npm install
```

如果遇到依赖冲突，可以尝试：

```bash
npm install --legacy-peer-deps
```

## 运行测试

### 基本命令

```bash
# 运行所有测试
npm test

# 运行测试（watch 模式）
npm test -- --watch

# 运行测试 UI
npm run test:ui

# 运行测试并生成覆盖率
npm run test:coverage
```

### 运行特定测试文件

```bash
# 运行单个测试文件
npm test -- src/api/__tests__/agents.test.ts

# 运行特定目录的测试
npm test -- src/api/__tests__/
```

### 调试模式

```bash
# 详细输出
npm test -- --reporter=verbose

# 只运行失败的测试
npm test -- --onlyFailures
```

## 常见问题

### 1. 依赖未安装

**错误**: `Cannot find module 'vitest'`

**解决**: 
```bash
npm install
```

### 2. TypeScript 类型错误

**错误**: `Type error: ...`

**解决**: 检查 `tsconfig.json` 配置，确保包含测试文件：

```json
{
  "include": ["src", "src/**/*.test.ts", "src/**/*.test.tsx"]
}
```

### 3. Mock 问题

**错误**: `Cannot find module '...'` 或 mock 不生效

**解决**: 
- 确保 mock 在 `vi.mock()` 调用之前
- 检查导入路径是否正确
- 使用 `vi.hoisted()` 处理循环依赖

### 4. 测试环境问题

**错误**: `ReferenceError: document is not defined`

**解决**: 
- 确保 `vitest.config.ts` 中设置了 `environment: 'jsdom'`
- 检查 `src/test/setup.ts` 是否正确配置

## 测试文件结构

```
web/
├── src/
│   ├── api/
│   │   └── __tests__/
│   │       └── agents.test.ts
│   ├── pages/
│   │   └── Nodes/
│   │       └── __tests__/
│   │           ├── AgentsTabContent.test.tsx
│   │           ├── AgentLogsDialog.test.tsx
│   │           └── AgentsIntegration.test.tsx
│   └── test/
│       ├── setup.ts          # 测试环境配置
│       └── testUtils.tsx      # 测试工具函数
├── vitest.config.ts          # Vitest 配置
└── package.json
```

## 测试覆盖率目标

- API 客户端: > 80%
- 组件: > 70%
- Hooks: > 80%
- Utils: > 90%

## 参考文档

- [Vitest 文档](https://vitest.dev/)
- [React Testing Library 文档](https://testing-library.com/react)
- [项目测试规范](../docs/前端开发规范.md#10-测试规范)
