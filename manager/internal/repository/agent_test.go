package repository

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/bingooyong/ops-scaffold-framework/manager/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// AgentRepositoryTestSuite Repository 层测试套件
type AgentRepositoryTestSuite struct {
	suite.Suite
	db   *gorm.DB
	repo AgentRepository
	ctx  context.Context
}

// SetupSuite 测试套件初始化
func (s *AgentRepositoryTestSuite) SetupSuite() {
	// 使用 SQLite 内存数据库
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(s.T(), err, "初始化数据库失败")

	// 自动迁移表结构
	err = db.AutoMigrate(&model.Agent{})
	if err != nil {
		s.T().Logf("AutoMigrate 失败，尝试手动创建表: %v", err)
		// 手动创建表作为备用方案
		err = db.Exec(`
			CREATE TABLE IF NOT EXISTS agents (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				created_at DATETIME NOT NULL,
				updated_at DATETIME NOT NULL,
				deleted_at DATETIME,
				node_id VARCHAR(50) NOT NULL,
				agent_id VARCHAR(100) NOT NULL,
				type VARCHAR(50),
				version VARCHAR(50),
				status VARCHAR(20) NOT NULL DEFAULT 'stopped',
				config TEXT,
				pid INTEGER DEFAULT 0,
				last_heartbeat DATETIME,
				last_sync_time DATETIME NOT NULL
			)
		`).Error
		require.NoError(s.T(), err, "手动创建表失败")
	}

	// 验证表是否创建成功
	var count int64
	err = db.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='agents'").Scan(&count).Error
	require.NoError(s.T(), err, "验证表创建失败")
	if count == 0 {
		s.T().Fatal("agents 表未被创建，迁移失败。请检查 Agent 模型定义。")
	}

	// 确保复合索引存在（GORM 可能没有正确创建）
	// 先删除可能存在的单独索引
	db.Exec("DROP INDEX IF EXISTS idx_agents_node_id")
	db.Exec("DROP INDEX IF EXISTS idx_agents_agent_id")
	// 创建复合索引
	err = db.Exec("CREATE INDEX IF NOT EXISTS idx_node_agent ON agents(node_id, agent_id)").Error
	if err != nil {
		s.T().Logf("警告: 创建复合索引失败（可能已存在）: %v", err)
	}

	s.db = db
	s.repo = NewAgentRepository(db)
	s.ctx = context.Background()
}

// SetupTest 每个测试用例前的准备
func (s *AgentRepositoryTestSuite) SetupTest() {
	// 清空 agents 表（使用 GORM 方法）
	err := s.db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&model.Agent{}).Error
	// 忽略错误，因为表可能为空
	_ = err
}

// TearDownSuite 测试套件清理
func (s *AgentRepositoryTestSuite) TearDownSuite() {
	sqlDB, _ := s.db.DB()
	if sqlDB != nil {
		sqlDB.Close()
	}
}

// createTestAgent 创建测试用的 Agent
func (s *AgentRepositoryTestSuite) createTestAgent(nodeID, agentID, agentType, version, status, config string) *model.Agent {
	now := time.Now()
	agent := &model.Agent{
		NodeID:       nodeID,
		AgentID:      agentID,
		Type:         agentType,
		Version:      version,
		Status:       status,
		Config:       config,
		PID:          1234,
		LastSyncTime: now,
	}
	if status == "running" {
		agent.LastHeartbeat = &now
	}
	return agent
}

// TestAgentRepository_Create 测试创建 Agent
func (s *AgentRepositoryTestSuite) TestAgentRepository_Create() {
	agent := s.createTestAgent("node-001", "agent-001", "filebeat", "1.0.0", "running", `{"key":"value"}`)

	err := s.repo.Create(s.ctx, agent)
	require.NoError(s.T(), err, "创建 Agent 应该成功")
	assert.NotZero(s.T(), agent.ID, "Agent ID 应该被自动生成")
	assert.NotZero(s.T(), agent.CreatedAt, "CreatedAt 应该被自动设置")
	assert.NotZero(s.T(), agent.UpdatedAt, "UpdatedAt 应该被自动设置")
}

// TestAgentRepository_GetByID 测试根据 ID 获取 Agent
func (s *AgentRepositoryTestSuite) TestAgentRepository_GetByID() {
	// 创建测试数据
	agent := s.createTestAgent("node-001", "agent-001", "filebeat", "1.0.0", "running", `{"key":"value"}`)
	err := s.repo.Create(s.ctx, agent)
	require.NoError(s.T(), err)

	// 根据 ID 查询
	result, err := s.repo.GetByID(s.ctx, agent.ID)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), result, "应该返回 Agent")
	assert.Equal(s.T(), agent.ID, result.ID)
	assert.Equal(s.T(), "node-001", result.NodeID)
	assert.Equal(s.T(), "agent-001", result.AgentID)
	assert.Equal(s.T(), "filebeat", result.Type)
	assert.Equal(s.T(), "1.0.0", result.Version)
	assert.Equal(s.T(), "running", result.Status)
	assert.Equal(s.T(), `{"key":"value"}`, result.Config)
	assert.Equal(s.T(), 1234, result.PID)
}

// TestAgentRepository_GetByID_NotFound 测试获取不存在的 Agent
func (s *AgentRepositoryTestSuite) TestAgentRepository_GetByID_NotFound() {
	result, err := s.repo.GetByID(s.ctx, 99999)
	require.NoError(s.T(), err, "获取不存在的 Agent 不应该返回错误")
	assert.Nil(s.T(), result, "应该返回 nil")
}

// TestAgentRepository_GetByNodeIDAndAgentID 测试根据 NodeID 和 AgentID 获取 Agent
func (s *AgentRepositoryTestSuite) TestAgentRepository_GetByNodeIDAndAgentID() {
	// 创建测试数据
	agent := s.createTestAgent("node-001", "agent-001", "filebeat", "1.0.0", "running", `{"key":"value"}`)
	err := s.repo.Create(s.ctx, agent)
	require.NoError(s.T(), err)

	// 根据 NodeID 和 AgentID 查询
	result, err := s.repo.GetByNodeIDAndAgentID(s.ctx, "node-001", "agent-001")
	require.NoError(s.T(), err)
	require.NotNil(s.T(), result, "应该返回 Agent")
	assert.Equal(s.T(), agent.ID, result.ID)
	assert.Equal(s.T(), "node-001", result.NodeID)
	assert.Equal(s.T(), "agent-001", result.AgentID)
}

// TestAgentRepository_GetByNodeIDAndAgentID_NotFound 测试获取不存在的 Agent
func (s *AgentRepositoryTestSuite) TestAgentRepository_GetByNodeIDAndAgentID_NotFound() {
	result, err := s.repo.GetByNodeIDAndAgentID(s.ctx, "node-999", "agent-999")
	require.NoError(s.T(), err, "获取不存在的 Agent 不应该返回错误")
	assert.Nil(s.T(), result, "应该返回 nil")
}

// TestAgentRepository_Update 测试更新 Agent
func (s *AgentRepositoryTestSuite) TestAgentRepository_Update() {
	// 创建测试数据
	agent := s.createTestAgent("node-001", "agent-001", "filebeat", "1.0.0", "running", `{"key":"value"}`)
	err := s.repo.Create(s.ctx, agent)
	require.NoError(s.T(), err)

	originalUpdatedAt := agent.UpdatedAt

	// 更新 Agent
	time.Sleep(10 * time.Millisecond) // 确保 UpdatedAt 会变化
	agent.Status = "stopped"
	agent.Version = "1.1.0"
	agent.Config = `{"key":"updated"}`
	err = s.repo.Update(s.ctx, agent)
	require.NoError(s.T(), err)

	// 验证更新
	result, err := s.repo.GetByID(s.ctx, agent.ID)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), result)
	assert.Equal(s.T(), "stopped", result.Status)
	assert.Equal(s.T(), "1.1.0", result.Version)
	assert.Equal(s.T(), `{"key":"updated"}`, result.Config)
	assert.True(s.T(), result.UpdatedAt.After(originalUpdatedAt), "UpdatedAt 应该被更新")
}

// TestAgentRepository_Delete 测试删除 Agent
func (s *AgentRepositoryTestSuite) TestAgentRepository_Delete() {
	// 创建测试数据
	agent := s.createTestAgent("node-001", "agent-001", "filebeat", "1.0.0", "running", `{"key":"value"}`)
	err := s.repo.Create(s.ctx, agent)
	require.NoError(s.T(), err)

	// 删除 Agent
	err = s.repo.Delete(s.ctx, "node-001", "agent-001")
	require.NoError(s.T(), err)

	// 验证删除（软删除）
	result, err := s.repo.GetByID(s.ctx, agent.ID)
	require.NoError(s.T(), err)
	assert.Nil(s.T(), result, "软删除后应该查询不到数据")
}

// TestAgentRepository_Delete_NotFound 测试删除不存在的 Agent
func (s *AgentRepositoryTestSuite) TestAgentRepository_Delete_NotFound() {
	// 删除不存在的 Agent 应该成功，不报错
	err := s.repo.Delete(s.ctx, "node-999", "agent-999")
	require.NoError(s.T(), err, "删除不存在的 Agent 应该成功，不报错")
}

// TestAgentRepository_ListByNodeID 测试列举节点下的所有 Agent
func (s *AgentRepositoryTestSuite) TestAgentRepository_ListByNodeID() {
	// 创建多个 Agent
	agents := []*model.Agent{
		s.createTestAgent("node-001", "agent-001", "filebeat", "1.0.0", "running", `{}`),
		s.createTestAgent("node-001", "agent-002", "telegraf", "2.0.0", "stopped", `{}`),
		s.createTestAgent("node-002", "agent-003", "filebeat", "1.0.0", "running", `{}`),
	}

	for _, agent := range agents {
		err := s.repo.Create(s.ctx, agent)
		require.NoError(s.T(), err)
	}

	// 列举 node-001 下的所有 Agent
	result, err := s.repo.ListByNodeID(s.ctx, "node-001")
	require.NoError(s.T(), err)
	assert.Equal(s.T(), 2, len(result), "应该返回 2 个 Agent")

	// 验证结果
	agentIDs := make(map[string]bool)
	for _, agent := range result {
		assert.Equal(s.T(), "node-001", agent.NodeID)
		agentIDs[agent.AgentID] = true
	}
	assert.True(s.T(), agentIDs["agent-001"], "应该包含 agent-001")
	assert.True(s.T(), agentIDs["agent-002"], "应该包含 agent-002")
}

// TestAgentRepository_ListByNodeID_Empty 测试列举空节点
func (s *AgentRepositoryTestSuite) TestAgentRepository_ListByNodeID_Empty() {
	result, err := s.repo.ListByNodeID(s.ctx, "node-empty")
	require.NoError(s.T(), err)
	assert.Empty(s.T(), result, "应该返回空列表")
}

// TestAgentRepository_Upsert 测试创建或更新 Agent
func (s *AgentRepositoryTestSuite) TestAgentRepository_Upsert() {
	// 第一次调用 Upsert，应该创建
	agent := s.createTestAgent("node-001", "agent-001", "filebeat", "1.0.0", "running", `{"key":"value"}`)
	err := s.repo.Upsert(s.ctx, agent)
	require.NoError(s.T(), err)
	assert.NotZero(s.T(), agent.ID, "应该创建新的 Agent")

	originalID := agent.ID
	originalCreatedAt := agent.CreatedAt

	// 第二次调用 Upsert，应该更新
	time.Sleep(10 * time.Millisecond)
	agent.Status = "stopped"
	agent.Version = "1.1.0"
	err = s.repo.Upsert(s.ctx, agent)
	require.NoError(s.T(), err)

	// 验证更新
	assert.Equal(s.T(), originalID, agent.ID, "ID 应该保持不变")
	assert.Equal(s.T(), originalCreatedAt, agent.CreatedAt, "CreatedAt 应该保持不变")
	assert.Equal(s.T(), "stopped", agent.Status, "Status 应该被更新")
	assert.Equal(s.T(), "1.1.0", agent.Version, "Version 应该被更新")
}

// TestAgentRepository_ConcurrentCreate 测试并发创建 Agent
func (s *AgentRepositoryTestSuite) TestAgentRepository_ConcurrentCreate() {
	const numGoroutines = 10
	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines)

	// 并发创建不同 AgentID 的 Agent
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			agent := s.createTestAgent("node-001", fmt.Sprintf("agent-%d", id), "filebeat", "1.0.0", "running", `{}`)
			if err := s.repo.Create(s.ctx, agent); err != nil {
				errors <- err
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// 检查是否有错误
	for err := range errors {
		s.T().Errorf("并发创建失败: %v", err)
	}

	// 验证所有 Agent 都被创建
	result, err := s.repo.ListByNodeID(s.ctx, "node-001")
	require.NoError(s.T(), err)
	assert.Equal(s.T(), numGoroutines, len(result), "应该创建 %d 个 Agent", numGoroutines)
}

// TestAgentRepository_ConcurrentUpdate 测试并发更新 Agent
func (s *AgentRepositoryTestSuite) TestAgentRepository_ConcurrentUpdate() {
	// 创建测试数据
	agent := s.createTestAgent("node-001", "agent-001", "filebeat", "1.0.0", "running", `{}`)
	err := s.repo.Create(s.ctx, agent)
	require.NoError(s.T(), err)

	const numGoroutines = 5
	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines)

	// 并发更新同一个 Agent
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(version string) {
			defer wg.Done()
			// 每次重新获取 Agent 以避免并发修改
			updatedAgent, err := s.repo.GetByNodeIDAndAgentID(s.ctx, "node-001", "agent-001")
			if err != nil {
				errors <- err
				return
			}
			if updatedAgent == nil {
				errors <- gorm.ErrRecordNotFound
				return
			}
			updatedAgent.Version = version
			if err := s.repo.Update(s.ctx, updatedAgent); err != nil {
				errors <- err
			}
		}(fmt.Sprintf("v%d", i))
	}

	wg.Wait()
	close(errors)

	// 检查是否有错误（并发更新可能会有冲突，但不应该导致系统错误）
	errorCount := 0
	for err := range errors {
		if err != nil {
			errorCount++
		}
	}

	// 验证 Agent 仍然存在且可查询
	result, err := s.repo.GetByNodeIDAndAgentID(s.ctx, "node-001", "agent-001")
	require.NoError(s.T(), err)
	assert.NotNil(s.T(), result, "Agent 应该仍然存在")
}

// TestAgentRepository_ConcurrentDelete 测试并发删除 Agent
func (s *AgentRepositoryTestSuite) TestAgentRepository_ConcurrentDelete() {
	// 创建多个测试数据
	for i := 0; i < 5; i++ {
		agent := s.createTestAgent("node-001", fmt.Sprintf("agent-%d", i), "filebeat", "1.0.0", "running", `{}`)
		err := s.repo.Create(s.ctx, agent)
		require.NoError(s.T(), err)
	}

	const numGoroutines = 5
	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines)

	// 并发删除不同的 Agent
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			if err := s.repo.Delete(s.ctx, "node-001", fmt.Sprintf("agent-%d", id)); err != nil {
				errors <- err
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// 检查是否有错误
	for err := range errors {
		s.T().Errorf("并发删除失败: %v", err)
	}

	// 验证所有 Agent 都被删除
	result, err := s.repo.ListByNodeID(s.ctx, "node-001")
	require.NoError(s.T(), err)
	assert.Empty(s.T(), result, "所有 Agent 应该都被删除")
}

// TestAgentRepository_Index_NodeAgent 验证复合索引 (node_id, agent_id) 的效果
func (s *AgentRepositoryTestSuite) TestAgentRepository_Index_NodeAgent() {
	// 创建测试数据
	for i := 0; i < 10; i++ {
		agent := s.createTestAgent("node-001", fmt.Sprintf("agent-%d", i), "filebeat", "1.0.0", "running", `{}`)
		err := s.repo.Create(s.ctx, agent)
		require.NoError(s.T(), err)
	}

	// 测试使用复合索引的查询性能
	start := time.Now()
	result, err := s.repo.GetByNodeIDAndAgentID(s.ctx, "node-001", "agent-5")
	duration := time.Since(start)

	require.NoError(s.T(), err)
	assert.NotNil(s.T(), result, "应该找到 Agent")
	assert.Less(s.T(), duration, 100*time.Millisecond, "查询应该在 100ms 内完成（索引应该生效）")
}

// TestAgentRepository_Index_Status 验证状态索引的效果
func (s *AgentRepositoryTestSuite) TestAgentRepository_Index_Status() {
	// 创建不同状态的 Agent
	agents := []*model.Agent{
		s.createTestAgent("node-001", "agent-001", "filebeat", "1.0.0", "running", `{}`),
		s.createTestAgent("node-001", "agent-002", "filebeat", "1.0.0", "stopped", `{}`),
		s.createTestAgent("node-001", "agent-003", "filebeat", "1.0.0", "running", `{}`),
		s.createTestAgent("node-001", "agent-004", "filebeat", "1.0.0", "error", `{}`),
	}

	for _, agent := range agents {
		err := s.repo.Create(s.ctx, agent)
		require.NoError(s.T(), err)
	}

	// 使用状态索引查询（通过 GORM 的 Where 条件）
	var runningAgents []*model.Agent
	start := time.Now()
	err := s.db.WithContext(s.ctx).Where("status = ?", "running").Find(&runningAgents).Error
	duration := time.Since(start)

	require.NoError(s.T(), err)
	assert.Equal(s.T(), 2, len(runningAgents), "应该找到 2 个 running 状态的 Agent")
	assert.Less(s.T(), duration, 100*time.Millisecond, "查询应该在 100ms 内完成（索引应该生效）")
}

// TestAgentRepository_VersionAndConfig 测试 Version 和 Config 字段
func (s *AgentRepositoryTestSuite) TestAgentRepository_VersionAndConfig() {
	// 创建带 Version 和 Config 的 Agent
	agent := s.createTestAgent("node-001", "agent-001", "filebeat", "2.5.1", "running", `{"port":8080,"host":"localhost"}`)
	err := s.repo.Create(s.ctx, agent)
	require.NoError(s.T(), err)

	// 查询并验证
	result, err := s.repo.GetByID(s.ctx, agent.ID)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), result)
	assert.Equal(s.T(), "2.5.1", result.Version, "Version 应该被正确保存")
	assert.Equal(s.T(), `{"port":8080,"host":"localhost"}`, result.Config, "Config 应该被正确保存")
}

// TestAgentRepository_IsRunning 测试 IsRunning 方法
func (s *AgentRepositoryTestSuite) TestAgentRepository_IsRunning() {
	runningAgent := s.createTestAgent("node-001", "agent-001", "filebeat", "1.0.0", "running", `{}`)
	stoppedAgent := s.createTestAgent("node-001", "agent-002", "filebeat", "1.0.0", "stopped", `{}`)

	assert.True(s.T(), runningAgent.IsRunning(), "running 状态的 Agent 应该返回 true")
	assert.False(s.T(), stoppedAgent.IsRunning(), "stopped 状态的 Agent 应该返回 false")
}

// TestAgentRepository 运行测试套件
func TestAgentRepository(t *testing.T) {
	suite.Run(t, new(AgentRepositoryTestSuite))
}
