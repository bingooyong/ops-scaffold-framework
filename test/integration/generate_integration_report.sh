#!/bin/bash
# 生成完整集成测试报告
# 功能: 汇总所有测试结果，生成完整的集成测试报告

set -e

# 颜色定义
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPORT_DIR="$SCRIPT_DIR/reports"

print_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
print_step() { echo -e "${BLUE}[STEP]${NC} $1"; }

# 解析测试报告
parse_report() {
    local report_file=$1
    if [ ! -f "$report_file" ]; then
        echo "0|0|N/A"
        return
    fi
    
    local total=$(grep -o "总测试数.*[0-9]*" "$report_file" | grep -o "[0-9]*" | head -1 || echo "0")
    local passed=$(grep -o "通过.*[0-9]*" "$report_file" | grep -o "[0-9]*" | head -1 || echo "0")
    local failed=$(grep -o "失败.*[0-9]*" "$report_file" | grep -o "[0-9]*" | head -1 || echo "0")
    local pass_rate=$(grep -o "通过率.*[0-9.]*%" "$report_file" | grep -o "[0-9.]*%" | head -1 || echo "N/A")
    
    echo "${total}|${passed}|${failed}|${pass_rate}"
}

# 生成集成测试报告
generate_report() {
    local report_file="$REPORT_DIR/integration_test_report.md"
    
    print_step "生成完整集成测试报告: $report_file"
    
    # 解析各测试报告
    local business_flows=$(parse_report "$REPORT_DIR/business_flows_test_report.md")
    local error_scenarios=$(parse_report "$REPORT_DIR/error_scenarios_test_report.md")
    local performance=$(parse_report "$REPORT_DIR/performance_test_report.md")
    
    IFS='|' read -r bf_total bf_passed bf_failed bf_rate <<< "$business_flows"
    IFS='|' read -r es_total es_passed es_failed es_rate <<< "$error_scenarios"
    IFS='|' read -r perf_total perf_passed perf_failed perf_rate <<< "$performance"
    
    # 计算总计
    local total_total=$((bf_total + es_total + perf_total))
    local total_passed=$((bf_passed + es_passed + perf_passed))
    local total_failed=$((bf_failed + es_failed + perf_failed))
    local total_rate="N/A"
    if [ $total_total -gt 0 ]; then
        total_rate=$(awk "BEGIN {printf \"%.1f%%\", ($total_passed / $total_total) * 100}")
    fi
    
    cat > "$report_file" <<EOF
# 完整系统集成测试报告

**生成时间**: $(date '+%Y-%m-%d %H:%M:%S')
**测试环境**: 完整系统集成测试环境
**测试版本**: v0.3.0

---

## 执行摘要

### 总体测试结果

| 测试类型 | 总测试数 | 通过 | 失败 | 通过率 |
|---------|---------|------|------|--------|
| 业务流程测试 | $bf_total | $bf_passed | $bf_failed | $bf_rate |
| 异常场景测试 | $es_total | $es_passed | $es_failed | $es_rate |
| 性能测试 | $perf_total | $perf_passed | $perf_failed | $perf_rate |
| **总计** | **$total_total** | **$total_passed** | **$total_failed** | **$total_rate** |

### 测试环境信息

- **Manager**: HTTP API (8080), gRPC (9090)
- **Daemon**: gRPC (9091)
- **Agent 实例**: agent-001, agent-002, agent-003
- **数据库**: MySQL 8.0+

---

## 测试场景详情

### Step 2: 业务流程测试

**测试文件**: \`test/integration/test_business_flows.sh\`

**测试场景**:
1. Agent 注册和发现
2. Agent 操作流程（启动/停止/重启）
3. 状态同步流程
4. 日志查看流程
5. 监控图表流程

**结果**: 
- 通过: $bf_passed
- 失败: $bf_failed
- 通过率: $bf_rate

**详细报告**: [业务流程测试报告](./business_flows_test_report.md)

---

### Step 3: 异常场景测试

**测试文件**: \`test/integration/test_error_scenarios.sh\`

**测试场景**:
1. Daemon 断线重连
2. Agent 异常退出自动重启
3. 网络延迟和超时
4. 并发操作冲突
5. 数据库连接失败（模拟）

**结果**: 
- 通过: $es_passed
- 失败: $es_failed
- 通过率: $es_rate

**详细报告**: [异常场景测试报告](./error_scenarios_test_report.md)

---

### Step 4: 性能测试

**测试文件**: \`test/integration/test_performance.sh\`

**测试场景**:
1. 多 Agent 并发运行
2. 高频心跳处理
3. 批量操作性能
4. Web 前端性能

**结果**: 
- 通过: $perf_passed
- 失败: $perf_failed
- 通过率: $perf_rate

**详细报告**: [性能测试报告](./performance_test_report.md)

---

## 发现的问题

### 严重问题

$(if [ $total_failed -gt 0 ]; then
    echo "1. **测试失败项**: 共 $total_failed 个测试项失败"
    echo "   - 业务流程测试失败: $bf_failed 项"
    echo "   - 异常场景测试失败: $es_failed 项"
    echo "   - 性能测试失败: $perf_failed 项"
    echo ""
    echo "   详细问题请查看各测试报告的问题记录部分。"
else
    echo "无严重问题"
fi)

### 已知问题

1. **API 404 错误**: 
   - **现象**: 部分 API 调用返回 404
   - **原因**: 节点可能未注册到 Manager 数据库，或 Agent 状态未同步
   - **影响**: 业务流程测试部分失败
   - **状态**: 已创建诊断脚本 \`diagnose_and_fix.sh\`

2. **Daemon 状态同步失败**:
   - **现象**: \`unknown service proto.DaemonService\`
   - **原因**: Manager gRPC 服务器可能未正确注册 DaemonService
   - **影响**: Agent 状态无法同步到 Manager
   - **状态**: 需要重启 Manager 服务

3. **数据库字段缺失**:
   - **现象**: \`Unknown column 'last_heartbeat_at'\`
   - **原因**: 数据库表结构可能不完整
   - **影响**: 心跳更新失败
   - **状态**: 需要运行数据库迁移

---

## 建议和改进方向

### 短期改进

1. **修复 API 404 错误**:
   - 确保 Daemon 正确注册到 Manager
   - 验证 Agent 状态同步机制
   - 检查数据库表结构

2. **修复 gRPC 服务注册**:
   - 重启 Manager 服务
   - 验证 DaemonService 正确注册
   - 检查 proto 包路径一致性

3. **完善数据库迁移**:
   - 运行数据库迁移脚本
   - 添加缺失的字段
   - 验证表结构完整性

### 长期改进

1. **增强错误处理**:
   - 改进 API 错误响应
   - 添加更详细的错误信息
   - 实现错误重试机制

2. **性能优化**:
   - 优化数据库查询
   - 实现连接池
   - 添加缓存机制

3. **测试覆盖**:
   - 增加单元测试覆盖
   - 添加端到端测试
   - 实现自动化测试流水线

---

## 测试环境配置

### 服务配置

- **Manager 配置**: \`test/integration/config/manager.test.yaml\`
- **Daemon 配置**: \`test/integration/config/daemon.test.yaml\`
- **Agent 配置**: \`test/integration/config/agent-*.test.yaml\`

### 启动和清理

- **启动脚本**: \`test/integration/start_test_env.sh\`
- **清理脚本**: \`test/integration/cleanup_test_env.sh\`
- **验证脚本**: \`test/integration/verify_test_env.sh\`
- **诊断脚本**: \`test/integration/diagnose_and_fix.sh\`

---

## 附录

### 测试脚本清单

- \`test/integration/test_business_flows.sh\` - 业务流程测试
- \`test/integration/test_error_scenarios.sh\` - 异常场景测试
- \`test/integration/test_performance.sh\` - 性能测试
- \`test/integration/generate_integration_report.sh\` - 集成测试报告生成

### 相关文档

- [Step 1 完成总结](./STEP1_COMPLETION_SUMMARY.md)
- [Step 2 完成总结](./STEP2_COMPLETION_SUMMARY.md)
- [测试环境验证报告](./test_env_verification_report.md)
- [Agent 清理修复说明](./AGENT_CLEANUP_FIX.md)
- [端口清理修复说明](./PORT_CLEANUP_FIX.md)
- [测试脚本修复说明](./TEST_SCRIPT_FIXES.md)

---

**报告生成完成**

**生成时间**: $(date '+%Y-%m-%d %H:%M:%S')
**报告版本**: v1.0
EOF

    print_info "完整集成测试报告已生成: $report_file"
}

# 主函数
main() {
    echo -e "${GREEN}========================================${NC}"
    echo -e "${GREEN}生成完整集成测试报告${NC}"
    echo -e "${GREEN}========================================${NC}"
    echo ""
    
    generate_report
    echo ""
    
    echo -e "${GREEN}========================================${NC}"
    echo -e "${GREEN}报告生成完成${NC}"
    echo -e "${GREEN}========================================${NC}"
    echo ""
    echo "报告位置: $REPORT_DIR/integration_test_report.md"
    echo ""
}

main "$@"
