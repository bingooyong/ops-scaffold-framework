# 测试脚本修复说明

**修复时间**: 2025-12-07  
**问题**: 业务流程测试脚本存在多个问题

---

## 问题描述

### 1. awk 语法错误

**现象**:
```
awk: syntax error at source line 1
  - 通过率:    - 通过率:
```

**原因**: awk 命令中的引号嵌套和变量替换问题

### 2. API 调用错误处理不足

**现象**:
- API 返回 400/404 错误时，只显示响应内容，没有 HTTP 状态码
- 无法区分不同类型的错误（认证失败、节点不存在、参数错误等）

### 3. 错误信息不够详细

**现象**:
- 失败时只显示响应内容，缺少调试信息
- 无法快速定位问题原因

---

## 修复方案

### 1. 修复 awk 语法错误

**修改前**:
```bash
echo "  - 通过率: $(awk "BEGIN {printf \"%.1f%%\", ($PASSED / ($PASSED + $FAILED)) * 100}")"
```

**修改后**:
```bash
if [ $((PASSED + FAILED)) -gt 0 ]; then
    local pass_rate=$(awk "BEGIN {printf \"%.1f\", ($PASSED / ($PASSED + $FAILED)) * 100}")
    echo "  - 通过率: ${pass_rate}%"
else
    echo "  - 通过率: N/A"
fi
```

**改进**:
- 分离 awk 计算和 printf 格式化
- 添加除零检查
- 使用变量存储中间结果

### 2. 增强错误处理

**修改前**:
```bash
local response=$(curl -s -X GET "http://127.0.0.1:8080/api/v1/nodes/$node_id/agents" \
    -H "Authorization: Bearer $token" \
    -H "Content-Type: application/json")

if echo "$response" | grep -q '"code":0'; then
    # 成功处理
else
    record_test "场景1-获取Agent列表" "FAIL" "API 调用失败: $response"
fi
```

**修改后**:
```bash
local http_code=$(curl -s -o /tmp/agent_list_response.json -w "%{http_code}" -X GET "http://127.0.0.1:8080/api/v1/nodes/$node_id/agents" \
    -H "Authorization: Bearer $token" \
    -H "Content-Type: application/json")
local response=$(cat /tmp/agent_list_response.json 2>/dev/null || echo "")
rm -f /tmp/agent_list_response.json

# 检查 HTTP 状态码
if [ "$http_code" != "200" ]; then
    print_warn "HTTP 状态码: $http_code"
    print_info "响应内容: $response"
    record_test "场景1-获取Agent列表" "FAIL" "API 调用失败 (HTTP $http_code): $response"
    return 1
fi

# 检查响应内容
if echo "$response" | grep -q '"code":0'; then
    # 成功处理
else
    record_test "场景1-获取Agent列表" "FAIL" "API 返回错误: $response"
    print_info "响应内容: $response"
fi
```

**改进**:
- 使用 `-w "%{http_code}"` 获取 HTTP 状态码
- 使用临时文件保存响应（避免状态码和响应内容混合）
- 先检查 HTTP 状态码，再检查响应内容
- 添加详细的调试信息

### 3. 统一错误处理模式

**所有 API 调用都改为**:
1. 获取 HTTP 状态码
2. 保存响应到临时文件
3. 读取响应内容
4. 清理临时文件
5. 检查 HTTP 状态码
6. 检查响应内容
7. 记录详细的错误信息

**影响的函数**:
- `test_scenario_1_agent_registration`: Agent 列表获取
- `test_scenario_2_agent_operations`: Agent 操作（启动/停止/重启）
- `test_scenario_4_logs`: Agent 日志获取
- `test_scenario_5_metrics`: 节点指标获取

---

## 修复内容详情

### 1. awk 语法修复

**文件**: `test/integration/test_business_flows.sh`

**位置**:
- 第 370 行（报告生成）
- 第 499 行（主函数输出）

**修复**:
- 分离计算和格式化
- 添加除零检查
- 使用变量存储结果

### 2. 错误处理增强

**文件**: `test/integration/test_business_flows.sh`

**修改的函数**:
- `test_scenario_1_agent_registration()`: 添加 HTTP 状态码检查
- `test_scenario_2_agent_operations()`: 所有操作添加 HTTP 状态码检查
- `test_scenario_4_logs()`: 添加 HTTP 状态码检查
- `test_scenario_5_metrics()`: 添加 HTTP 状态码检查

**改进**:
- 所有 curl 调用都使用 `-w "%{http_code}"` 和临时文件
- 统一的错误处理模式
- 详细的调试信息输出

---

## 验证结果

### 修复前
```bash
$ ./test_business_flows.sh
[✗] 场景1-获取Agent列表: API 调用失败: 400 Bad Request
awk: syntax error at source line 1
  - 通过率:    - 通过率:
```

### 修复后
```bash
$ ./test_business_flows.sh
[INFO] HTTP 状态码: 404
[INFO] 响应内容: 404 page not found
[✗] 场景1-获取Agent列表: API 调用失败 (HTTP 404): 404 page not found
  - 通过率: 29.4%
```

**改进**:
- ✅ awk 语法错误已修复
- ✅ HTTP 状态码正确显示
- ✅ 错误信息更详细
- ✅ 通过率计算正常

---

## 已知问题

### 1. API 返回 404

**原因**: 节点可能未注册到 Manager 数据库

**解决方案**:
1. 确保 Daemon 已启动并注册到 Manager
2. 检查 Manager 日志确认节点注册成功
3. 验证 Node ID 是否正确

**检查命令**:
```bash
# 检查 Daemon 是否注册
curl -X GET "http://127.0.0.1:8080/api/v1/nodes" \
  -H "Authorization: Bearer <token>"

# 检查 Manager 日志
tail -f test/integration/logs/manager.log | grep -i "register\|node"
```

### 2. Agent 列表为空

**原因**: 
- 节点已注册，但 Agent 尚未同步到 Manager
- Daemon 未正确上报 Agent 状态

**解决方案**:
1. 等待 Daemon 同步 Agent 状态（通常需要几秒到几十秒）
2. 检查 Daemon 日志确认 Agent 状态同步
3. 手动触发状态同步（如果支持）

---

## 使用建议

### 运行测试前

1. **确保测试环境已启动**:
   ```bash
   ./start_test_env.sh
   ```

2. **等待服务就绪**:
   ```bash
   # 等待 Manager 和 Daemon 启动完成
   sleep 10
   
   # 等待 Daemon 注册到 Manager
   sleep 5
   
   # 等待 Agent 状态同步
   sleep 10
   ```

3. **验证服务状态**:
   ```bash
   ./verify_test_env.sh
   ```

### 运行测试

```bash
./test_business_flows.sh
```

### 查看详细错误

如果测试失败，查看：
1. **测试报告**: `reports/business_flows_test_report.md`
2. **Manager 日志**: `logs/manager.log`
3. **Daemon 日志**: `logs/daemon.log`

---

## 相关文件

- `test/integration/test_business_flows.sh` - 业务流程测试脚本（已修复）
- `test/integration/reports/business_flows_test_report.md` - 测试报告

---

## 总结

**修复完成**: ✅ 测试脚本错误处理已增强

**改进点**:
1. ✅ awk 语法错误已修复
2. ✅ HTTP 状态码检查已添加
3. ✅ 错误信息更详细
4. ✅ 统一的错误处理模式
5. ✅ 更好的调试信息

**下一步**:
- 解决 API 404 错误（节点注册问题）
- 验证 Agent 状态同步机制
- 完善测试场景覆盖

---

**修复完成**: ✅ 测试脚本已改进，错误处理更完善
