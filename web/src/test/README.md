# 测试说明

## 运行测试

```bash
# 运行所有测试
npm test

# 运行测试 UI
npm run test:ui

# 运行测试并生成覆盖率报告
npm run test:coverage
```

## 测试文件结构

- `src/api/__tests__/`: API 客户端测试
- `src/pages/__tests__/`: 页面组件测试
- `src/test/setup.ts`: 测试环境配置
- `src/test/testUtils.tsx`: 测试工具函数

## 注意事项

1. 测试使用 Vitest + React Testing Library
2. 所有测试文件需要 mock 外部依赖（API、hooks、stores 等）
3. 组件测试通过 `testUtils.tsx` 中的 `render` 函数，自动包含必要的 Provider
