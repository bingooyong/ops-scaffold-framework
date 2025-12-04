# Manager 集成测试

## 概述

本目录包含 Manager 服务的自动化集成测试套件，基于 Go 原生测试框架和 testify 库实现。

## 测试文件结构

```
test/integration/
├── manager_integration_test.go  # Manager API集成测试
└── README.md                     # 本文档
```

## 测试覆盖的功能模块

### Phase 0: 环境检查
- ✅ 健康检查

### Phase 1: 认证模块
- ✅ 用户注册
- ✅ 用户登录
- ✅ 获取用户资料
- ✅ 修改密码

### Phase 2: 节点管理
- ✅ 查询节点列表
- ✅ 获取统计信息

### Phase 3: 错误场景
- ✅ 无效凭证登录
- ✅ 无Token访问
- ✅ 注册重复用户

## 运行测试

### 前置条件

1. **启动 Manager 服务**
   ```bash
   cd manager
   ./bin/manager -config configs/manager.dev.yaml
   ```

2. **确保数据库可用**
   ```bash
   mysql -h 127.0.0.1 -P 3306 -uroot -prootpassword -e "SHOW DATABASES;"
   ```

### 方式1：使用自动化脚本（推荐）⭐⭐⭐⭐⭐

```bash
cd manager
./test/run_tests.sh
```

### 方式2：直接运行 Go 测试

```bash
cd manager

# 运行所有集成测试
go test -v ./test/integration/

# 运行特定测试阶段
go test -v ./test/integration/ -run "Phase1"

# 运行单个测试
go test -v ./test/integration/ -run Test_02_Phase1_Login

# 生成覆盖率报告
go test -v ./test/integration/ -coverprofile=coverage.out
go tool cover -html=coverage.out

# 并行测试（快速）
go test -v ./test/integration/ -parallel 4

# 设置超时
go test -v ./test/integration/ -timeout 5m
```

## 环境变量

```bash
export MANAGER_BASE_URL=http://localhost:8080  # 默认: http://127.0.0.1:8080
```

## 测试套件特性

### 1. 测试顺序保证

使用 `Test_XX_` 前缀确保测试按顺序执行：

```go
func (s *ManagerTestSuite) Test_01_Phase0_HealthCheck() { ... }
func (s *ManagerTestSuite) Test_02_Phase1_Register() { ... }
```

### 2. 测试数据共享

使用 `s.testData` 在测试之间共享数据（如 token）：

```go
// Phase1 登录后保存 token
token := result["token"].(string)
s.testData["token"] = token

// Phase2 使用保存的 token
token := s.testData["token"].(string)
```

### 3. Setup/TearDown

```go
// 整个套件开始前执行一次
func (s *ManagerTestSuite) SetupSuite() { ... }

// 整个套件结束后执行一次
func (s *ManagerTestSuite) TearDownSuite() { ... }
```

### 4. 丰富的断言

```go
// 基本断言
assert.Equal(t, expected, actual)
assert.NotNil(t, value)
assert.Contains(t, haystack, needle)

// 必须断言（失败立即停止）
require.NoError(t, err, "必须成功")
require.True(t, condition, "必须为真")
```

## CI/CD 集成示例

### GitHub Actions

```yaml
name: Manager Tests
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    services:
      mysql:
        image: mysql:8.0
        env:
          MYSQL_ROOT_PASSWORD: rootpassword
          MYSQL_DATABASE: ops_manager_dev
        ports:
          - 3306:3306
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.24'
      - name: Build and Test
        run: |
          cd manager
          make build
          ./bin/manager &
          sleep 5
          go test -v ./test/integration/
```

### GitLab CI

```yaml
test:
  stage: test
  image: golang:1.24
  services:
    - mysql:8.0
  variables:
    MYSQL_ROOT_PASSWORD: rootpassword
    MYSQL_DATABASE: ops_manager_dev
  script:
    - cd manager
    - go build -o bin/manager ./cmd/manager/
    - ./bin/manager &
    - sleep 5
    - go test -v ./test/integration/
```

## 添加新测试

```go
func (s *ManagerTestSuite) Test_99_MyNewFeature() {
    s.T().Log("=== 我的新功能测试 ===")

    // 1. 准备测试数据
    data := map[string]interface{}{
        "field": "value",
    }

    // 2. 调用API
    resp, body := s.request("POST", "/api/v1/my-endpoint", data, true)

    // 3. 断言结果
    assert.Equal(s.T(), http.StatusOK, resp.StatusCode)

    var result map[string]interface{}
    json.Unmarshal(body, &result)
    assert.Equal(s.T(), 0, int(result["code"].(float64)))

    s.T().Log("✅ 测试通过")
}
```

## 最佳实践

### ✅ DO

1. **使用Go原生测试** - 类型安全、易维护
2. **使用testify/suite** - 结构化测试
3. **编写可复用的helper方法** - `s.request()`
4. **测试隔离** - 每个测试独立，不依赖外部状态
5. **清理资源** - 测试后清理创建的数据
6. **有意义的测试名称** - `Test_02_Phase1_Login`
7. **详细的日志** - `s.T().Log()` 记录关键步骤

### ❌ DON'T

1. **不要依赖测试执行顺序** - 除非使用suite和命名约定
2. **不要使用硬编码数据** - 动态创建测试数据
3. **不要跳过错误检查** - 使用 `require` 确保前置条件
4. **不要在测试中使用 `time.Sleep` 过多** - 使用重试机制

## 故障排查

### 测试失败？

```bash
# 1. 检查服务是否启动
curl http://127.0.0.1:8080/health

# 2. 检查数据库连接
mysql -h 127.0.0.1 -P 3306 -uroot -prootpassword -e "USE ops_manager_dev;"

# 3. 查看服务日志
tail -f manager/logs/manager.log

# 4. 运行单个测试
go test -v ./test/integration/ -run Test_02_Phase1_Login

# 5. 增加超时时间
go test -v ./test/integration/ -timeout 10m
```

### 依赖问题？

```bash
# 更新依赖
cd manager
go mod tidy

# 安装testify
go get github.com/stretchr/testify@latest
```

## 参考文档

- [Manager API 文档](../../../docs/api/Manager_API.md)
- [testify 文档](https://github.com/stretchr/testify)

---

**版本**: v1.0
**更新**: 2025-12-03
**维护**: Ops Scaffold Framework Team
