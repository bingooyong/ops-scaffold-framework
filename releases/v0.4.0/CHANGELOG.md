# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.4.0] - 2025-01-27

### Added
- **Agent 管理功能**: 支持在单个节点上同时管理多个第三方 Agent 实例
  - 多 Agent 并发管理(Filebeat、Telegraf、Node Exporter 等)
  - Agent 生命周期管理(启动、停止、重启)
  - 健康检查和自动恢复
  - 资源监控和告警
  - 配置管理和热更新
  - 日志查看和管理
- **Manager 后端 API**: Agent 管理相关的 HTTP API 和 gRPC 接口
- **Web 前端界面**: Agent 管理 UI(列表、操作、日志查看、监控图表)
- **测试 Agent 框架**: 用于测试和演示的测试 Agent
- **完整文档**: 用户使用指南、管理员手册、开发者文档、配置示例

### Changed
- Daemon 架构重构: 从单 Agent 管理升级为多 Agent 管理
- 配置文件格式: 支持 `agents` 数组配置多个 Agent
- 向后兼容: 支持旧格式配置文件自动转换

### Fixed
- 修复 Agent 操作超时问题
- 修复手动停止后自动重启的问题
- 修复重启操作的回退时间问题

### Security
- 加强输入验证(Agent ID、Node ID 格式验证)
- 路径验证(防止路径注入)
- 错误信息处理改进(避免信息泄露)
- 日志脱敏工具函数

### Performance
- 添加 pprof 性能分析支持
- 性能指标收集器
- 性能基准测试
- 优化心跳处理和状态同步性能

### Documentation
- 新增 Agent 管理功能使用指南
- 新增 Agent 管理管理员手册
- 新增 Agent 管理开发者文档
- 新增 Agent 管理配置示例
- 更新项目 README.md
- 更新 Daemon 模块设计文档

## [0.3.0] - 2025-11-30

### Added
- Manager 后端基础功能
- Daemon 守护进程基础功能
- Web 前端基础功能
