# 测试修复说明

## 已修复的问题

### 1. Vitest 配置
- ✅ 修复了 `__dirname` 在 ESM 模式下的问题
- ✅ 添加了测试文件匹配模式 `include`
- ✅ 配置了正确的路径别名

### 2. 测试文件修复

#### `agents.test.ts`
- ✅ 移除了未使用的 `axios` 导入
- ✅ 修复了 mock client 的使用方式

#### `AgentsTabContent.test.tsx`
- ✅ 修复了组件导入路径（使用 `NodeDetail` 而非 `NodeDetailPage`）
- ✅ 添加了所有必要的组件 mock：
  - MetricsChart (lazy loaded)
  - CPUCard, MemoryCard, DiskCard, NetworkCard
  - TimeRangeSelector, RefreshControl
- ✅ 添加了 `formatMetricsHistory` utils mock
- ✅ 修复了 `react-router-dom` mock（使用 `vi.importActual`）

### 3. 测试工具
- ✅ 创建了 `testUtils.tsx` 提供自定义 render 函数
- ✅ 创建了 `setup.ts` 配置测试环境
- ✅ 添加了必要的全局 mock（matchMedia, IntersectionObserver, ResizeObserver）

## 可能遇到的问题及解决方案

### 问题 1: 依赖未安装

**症状**: `Cannot find module 'vitest'` 或类似错误

**解决**:
```bash
cd web
npm install
```

如果遇到 peer dependency 冲突：
```bash
npm install --legacy-peer-deps
```

### 问题 2: TypeScript 类型错误

**症状**: 类型检查失败

**解决**: 检查 `tsconfig.json` 和 `tsconfig.app.json`，确保包含测试文件。

### 问题 3: Mock 不生效

**症状**: Mock 函数没有被调用或返回错误值

**解决**:
1. 确保 `vi.mock()` 在导入之前调用
2. 检查 mock 路径是否正确
3. 使用 `vi.hoisted()` 处理循环依赖

### 问题 4: 组件渲染失败

**症状**: `render()` 失败或组件找不到

**解决**:
1. 确保所有依赖都已 mock
2. 检查组件导入路径
3. 使用 `testUtils.tsx` 中的 `render` 函数（包含必要的 Provider）

### 问题 5: React Query 相关错误

**症状**: `useQuery` 或 `useMutation` 相关错误

**解决**:
- 测试工具已自动包含 `QueryClientProvider`
- 确保 mock 了所有使用 React Query 的 hooks

## 运行测试

### 基本命令
```bash
npm test
```

### 查看详细错误
```bash
npm test -- --reporter=verbose
```

### 运行单个测试文件
```bash
npm test -- src/api/__tests__/agents.test.ts
```

### 调试模式
```bash
npm test -- --reporter=verbose --no-coverage
```

## 测试文件清单

1. ✅ `src/api/__tests__/agents.test.ts` - Agent API 测试
2. ✅ `src/pages/Nodes/__tests__/AgentsTabContent.test.tsx` - AgentsTabContent 组件测试
3. ✅ `src/pages/Nodes/__tests__/AgentLogsDialog.test.tsx` - AgentLogsDialog 测试
4. ✅ `src/pages/Nodes/__tests__/AgentsIntegration.test.tsx` - 集成测试

## 下一步

如果测试仍然失败，请：

1. **查看具体错误信息**:
   ```bash
   npm test 2>&1 | tee test-output.log
   ```

2. **检查依赖**:
   ```bash
   npm list vitest @testing-library/react
   ```

3. **清理并重新安装**:
   ```bash
   rm -rf node_modules package-lock.json
   npm install
   ```

4. **运行单个简单测试验证环境**:
   ```bash
   npm test -- src/api/__tests__/agents.test.ts
   ```

## 联系支持

如果问题持续存在，请提供：
- 完整的错误输出
- `package.json` 内容
- `vitest.config.ts` 内容
- Node.js 和 npm 版本
