package agent

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap"
)

// newTestMultiAgentManager 创建用于测试的MultiAgentManager
func newTestMultiAgentManager(t *testing.T) *MultiAgentManager {
	tmpDir := t.TempDir()
	logger := zap.NewNop()
	mam, err := NewMultiAgentManager(tmpDir, logger)
	if err != nil {
		t.Fatalf("failed to create MultiAgentManager: %v", err)
	}
	return mam
}

func TestNewMultiAgentManager(t *testing.T) {
	tmpDir := t.TempDir()
	logger := zap.NewNop()
	mam, err := NewMultiAgentManager(tmpDir, logger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mam == nil {
		t.Fatal("expected non-nil MultiAgentManager")
	}
	if mam.Count() != 0 {
		t.Errorf("expected 0 agents, got %d", mam.Count())
	}
	if mam.GetRegistry() == nil {
		t.Error("expected non-nil registry")
	}
}

func TestMultiAgentManager_RegisterAgent(t *testing.T) {
	mam := newTestMultiAgentManager(t)

	// 创建AgentInfo
	info := &AgentInfo{
		ID:         "test-agent",
		Type:       TypeFilebeat,
		Name:       "Test Agent",
		BinaryPath: "/usr/bin/filebeat",
		ConfigFile: "/etc/filebeat/filebeat.yml",
		WorkDir:    "/tmp/test-agent",
	}

	// 注册Agent
	instance, err := mam.RegisterAgent(info)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if instance == nil {
		t.Fatal("expected non-nil instance")
	}
	if mam.Count() != 1 {
		t.Errorf("expected 1 agent, got %d", mam.Count())
	}

	// 测试重复注册
	_, err = mam.RegisterAgent(info)
	if err == nil {
		t.Fatal("expected error for duplicate registration")
	}
	if _, ok := err.(*AgentExistsError); !ok {
		t.Errorf("expected AgentExistsError, got %T", err)
	}
}

func TestMultiAgentManager_GetAgent(t *testing.T) {
	mam := newTestMultiAgentManager(t)

	// 注册Agent
	info := &AgentInfo{
		ID:   "test-agent",
		Type: TypeFilebeat,
	}
	_, err := mam.RegisterAgent(info)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 获取Agent
	instance := mam.GetAgent("test-agent")
	if instance == nil {
		t.Fatal("expected non-nil instance")
	}
	if instance.GetInfo().ID != "test-agent" {
		t.Errorf("expected agent ID 'test-agent', got '%s'", instance.GetInfo().ID)
	}

	// 获取不存在的Agent
	instance = mam.GetAgent("non-existent")
	if instance != nil {
		t.Error("expected nil for non-existent agent")
	}
}

func TestMultiAgentManager_UnregisterAgent(t *testing.T) {
	mam := newTestMultiAgentManager(t)

	// 注册Agent
	info := &AgentInfo{
		ID:   "test-agent",
		Type: TypeFilebeat,
	}
	_, err := mam.RegisterAgent(info)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 注销Agent（未运行状态）
	err = mam.UnregisterAgent("test-agent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mam.Count() != 0 {
		t.Errorf("expected 0 agents after unregister, got %d", mam.Count())
	}

	// 注销不存在的Agent
	err = mam.UnregisterAgent("non-existent")
	if err == nil {
		t.Fatal("expected error for non-existent agent")
	}
	if _, ok := err.(*AgentNotFoundError); !ok {
		t.Errorf("expected AgentNotFoundError, got %T", err)
	}
}

func TestMultiAgentManager_UnregisterAgent_Running(t *testing.T) {
	mam := newTestMultiAgentManager(t)

	// 注册Agent
	info := &AgentInfo{
		ID:         "test-agent",
		Type:       TypeFilebeat,
		BinaryPath: "/bin/sleep", // 使用系统命令用于测试
		WorkDir:    t.TempDir(),
	}
	instance, err := mam.RegisterAgent(info)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 启动Agent使其真正运行
	ctx := context.Background()
	err = instance.Start(ctx)
	if err != nil {
		t.Skipf("cannot start test process: %v", err)
	}

	// 等待进程启动并验证
	var isRunning bool
	for i := 0; i < 10; i++ {
		time.Sleep(50 * time.Millisecond)
		if instance.IsRunning() {
			isRunning = true
			break
		}
	}

	if !isRunning {
		// 如果进程没有运行，清理并跳过测试
		instance.Stop(ctx, true)
		t.Skip("agent is not running, skipping test")
	}

	// 尝试注销运行中的Agent（应该失败）
	err = mam.UnregisterAgent("test-agent")
	if err == nil {
		instance.Stop(ctx, true)
		t.Fatal("expected error for running agent")
	}
	if _, ok := err.(*AgentRunningError); !ok {
		instance.Stop(ctx, true)
		t.Errorf("expected AgentRunningError, got %T", err)
	}

	// 停止Agent后再次注销
	err = instance.Stop(ctx, true)
	if err != nil {
		t.Fatalf("unexpected error stopping agent: %v", err)
	}

	// 等待进程停止
	for i := 0; i < 10; i++ {
		time.Sleep(50 * time.Millisecond)
		if !instance.IsRunning() {
			break
		}
	}

	err = mam.UnregisterAgent("test-agent")
	if err != nil {
		t.Fatalf("unexpected error after stopping: %v", err)
	}
}

func TestMultiAgentManager_ListAgents(t *testing.T) {
	mam := newTestMultiAgentManager(t)

	// 注册多个Agent
	agents := []struct {
		id   string
		typ  AgentType
		name string
	}{
		{"agent-1", TypeFilebeat, "Filebeat Agent"},
		{"agent-2", TypeTelegraf, "Telegraf Agent"},
		{"agent-3", TypeNodeExporter, "Node Exporter"},
	}

	for _, a := range agents {
		info := &AgentInfo{
			ID:   a.id,
			Type: a.typ,
			Name: a.name,
		}
		_, err := mam.RegisterAgent(info)
		if err != nil {
			t.Fatalf("unexpected error registering %s: %v", a.id, err)
		}
	}

	// 列举所有Agent
	list := mam.ListAgents()
	if len(list) != 3 {
		t.Errorf("expected 3 agents, got %d", len(list))
	}

	// 验证所有Agent都在列表中
	found := make(map[string]bool)
	for _, instance := range list {
		found[instance.GetInfo().ID] = true
	}
	for _, a := range agents {
		if !found[a.id] {
			t.Errorf("agent %s not found in list", a.id)
		}
	}
}

func TestMultiAgentManager_StartAgent(t *testing.T) {
	mam := newTestMultiAgentManager(t)

	// 注册Agent
	info := &AgentInfo{
		ID:         "test-agent",
		Type:       TypeFilebeat,
		BinaryPath: "/nonexistent/binary",
		ConfigFile: "/nonexistent/config.yml",
		WorkDir:    "/tmp/test-agent",
	}
	_, err := mam.RegisterAgent(info)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctx := context.Background()

	// 启动不存在的Agent
	err = mam.StartAgent(ctx, "non-existent")
	if err == nil {
		t.Fatal("expected error for non-existent agent")
	}
	if _, ok := err.(*AgentNotFoundError); !ok {
		t.Errorf("expected AgentNotFoundError, got %T", err)
	}

	// 启动存在的Agent（会失败因为二进制不存在，但这是预期的）
	err = mam.StartAgent(ctx, "test-agent")
	// 允许错误，因为二进制文件不存在
	if err == nil {
		t.Log("Note: binary exists (unexpected in test environment)")
	}
}

func TestMultiAgentManager_StopAgent(t *testing.T) {
	mam := newTestMultiAgentManager(t)

	// 注册Agent
	info := &AgentInfo{
		ID:   "test-agent",
		Type: TypeFilebeat,
	}
	_, err := mam.RegisterAgent(info)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctx := context.Background()

	// 停止不存在的Agent
	err = mam.StopAgent(ctx, "non-existent", true)
	if err == nil {
		t.Fatal("expected error for non-existent agent")
	}

	// 停止存在的Agent（未运行状态，应该成功）
	err = mam.StopAgent(ctx, "test-agent", true)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestMultiAgentManager_RestartAgent(t *testing.T) {
	mam := newTestMultiAgentManager(t)

	// 注册Agent
	info := &AgentInfo{
		ID:         "test-agent",
		Type:       TypeFilebeat,
		BinaryPath: "/nonexistent/binary",
		ConfigFile: "/nonexistent/config.yml",
		WorkDir:    "/tmp/test-agent",
	}
	_, err := mam.RegisterAgent(info)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctx := context.Background()

	// 重启不存在的Agent
	err = mam.RestartAgent(ctx, "non-existent", false)
	if err == nil {
		t.Fatal("expected error for non-existent agent")
	}

	// 重启存在的Agent（会失败因为二进制不存在，但这是预期的）
	err = mam.RestartAgent(ctx, "test-agent", false)
	// 允许错误，因为二进制文件不存在
	if err == nil {
		t.Log("Note: binary exists (unexpected in test environment)")
	}
}

func TestMultiAgentManager_StartAll(t *testing.T) {
	mam := newTestMultiAgentManager(t)

	// 注册多个Agent
	agents := []string{"agent-1", "agent-2", "agent-3"}
	for _, id := range agents {
		info := &AgentInfo{
			ID:         id,
			Type:       TypeFilebeat,
			BinaryPath: "/nonexistent/binary",
			ConfigFile: "/nonexistent/config.yml",
			WorkDir:    "/tmp/" + id,
		}
		_, err := mam.RegisterAgent(info)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	ctx := context.Background()

	// 启动所有Agent
	results := mam.StartAll(ctx)

	// 验证结果
	if len(results) != len(agents) {
		t.Errorf("expected %d results, got %d", len(agents), len(results))
	}

	// 所有Agent都应该有结果（即使失败）
	for _, id := range agents {
		if _, exists := results[id]; !exists {
			t.Errorf("expected result for agent %s", id)
		}
	}
}

func TestMultiAgentManager_StopAll(t *testing.T) {
	mam := newTestMultiAgentManager(t)

	// 注册多个Agent
	agents := []string{"agent-1", "agent-2", "agent-3"}
	for _, id := range agents {
		info := &AgentInfo{
			ID:   id,
			Type: TypeFilebeat,
		}
		_, err := mam.RegisterAgent(info)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	ctx := context.Background()

	// 停止所有Agent
	results := mam.StopAll(ctx, true)

	// 验证结果
	if len(results) != len(agents) {
		t.Errorf("expected %d results, got %d", len(agents), len(results))
	}
}

func TestMultiAgentManager_RestartAll(t *testing.T) {
	mam := newTestMultiAgentManager(t)

	// 注册多个Agent
	agents := []string{"agent-1", "agent-2"}
	for _, id := range agents {
		info := &AgentInfo{
			ID:         id,
			Type:       TypeFilebeat,
			BinaryPath: "/nonexistent/binary",
			ConfigFile: "/nonexistent/config.yml",
			WorkDir:    "/tmp/" + id,
		}
		_, err := mam.RegisterAgent(info)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	ctx := context.Background()

	// 重启所有Agent
	results := mam.RestartAll(ctx)

	// 验证结果
	if len(results) != len(agents) {
		t.Errorf("expected %d results, got %d", len(agents), len(results))
	}
}

func TestMultiAgentManager_Count(t *testing.T) {
	mam := newTestMultiAgentManager(t)

	// 初始计数应该为0
	if count := mam.Count(); count != 0 {
		t.Errorf("expected 0 agents, got %d", count)
	}

	// 注册Agent后计数应该增加
	info1 := &AgentInfo{ID: "agent-1", Type: TypeFilebeat}
	_, err := mam.RegisterAgent(info1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count := mam.Count(); count != 1 {
		t.Errorf("expected 1 agent, got %d", count)
	}

	info2 := &AgentInfo{ID: "agent-2", Type: TypeTelegraf}
	_, err = mam.RegisterAgent(info2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count := mam.Count(); count != 2 {
		t.Errorf("expected 2 agents, got %d", count)
	}

	// 注销后计数应该减少
	err = mam.UnregisterAgent("agent-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count := mam.Count(); count != 1 {
		t.Errorf("expected 1 agent after unregister, got %d", count)
	}
}

func TestMultiAgentManager_LoadAgentsFromRegistry(t *testing.T) {
	mam := newTestMultiAgentManager(t)

	// 在注册表中注册Agent
	_, err := mam.GetRegistry().Register(
		"registry-agent",
		TypeFilebeat,
		"Registry Agent",
		"/usr/bin/filebeat",
		"/etc/filebeat/filebeat.yml",
		"/tmp/registry-agent",
		"",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 从注册表加载Agent
	err = mam.LoadAgentsFromRegistry()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 验证Agent实例已创建
	instance := mam.GetAgent("registry-agent")
	if instance == nil {
		t.Fatal("expected agent instance to be created")
	}
	if instance.GetInfo().ID != "registry-agent" {
		t.Errorf("expected agent ID 'registry-agent', got '%s'", instance.GetInfo().ID)
	}
}

func TestMultiAgentManager_GetAgentStatus(t *testing.T) {
	mam := newTestMultiAgentManager(t)

	// 注册Agent
	info := &AgentInfo{
		ID:   "test-agent",
		Type: TypeFilebeat,
		Name: "Test Agent",
	}
	// 设置初始状态为 Stopped
	info.SetStatus(StatusStopped)
	
	_, err := mam.RegisterAgent(info)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 获取状态
	status, err := mam.GetAgentStatus("test-agent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status == nil {
		t.Fatal("expected non-nil status")
	}
	if status.ID != "test-agent" {
		t.Errorf("expected ID 'test-agent', got '%s'", status.ID)
	}
	if status.Type != TypeFilebeat {
		t.Errorf("expected Type TypeFilebeat, got %s", status.Type)
	}
	if status.Status != StatusStopped {
		t.Errorf("expected Status StatusStopped, got %s", status.Status)
	}
	if status.IsRunning {
		t.Error("expected IsRunning false")
	}

	// 获取不存在的Agent状态
	_, err = mam.GetAgentStatus("non-existent")
	if err == nil {
		t.Fatal("expected error for non-existent agent")
	}
}

func TestMultiAgentManager_GetAllAgentStatus(t *testing.T) {
	mam := newTestMultiAgentManager(t)

	// 注册多个Agent
	agents := []struct {
		id   string
		typ  AgentType
		name string
	}{
		{"agent-1", TypeFilebeat, "Filebeat Agent"},
		{"agent-2", TypeTelegraf, "Telegraf Agent"},
	}

	for _, a := range agents {
		info := &AgentInfo{
			ID:   a.id,
			Type: a.typ,
			Name: a.name,
		}
		_, err := mam.RegisterAgent(info)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	// 获取所有Agent状态
	statuses := mam.GetAllAgentStatus()
	if len(statuses) != 2 {
		t.Errorf("expected 2 statuses, got %d", len(statuses))
	}

	// 验证所有Agent都在状态列表中
	found := make(map[string]bool)
	for _, status := range statuses {
		found[status.ID] = true
	}
	for _, a := range agents {
		if !found[a.id] {
			t.Errorf("agent %s not found in status list", a.id)
		}
	}
}

func TestMultiAgentManager_GetRegistry(t *testing.T) {
	mam := newTestMultiAgentManager(t)

	registry := mam.GetRegistry()
	if registry == nil {
		t.Fatal("expected non-nil registry")
	}

	// 验证可以通过registry注册Agent
	_, err := registry.Register(
		"registry-test",
		TypeFilebeat,
		"Registry Test",
		"/usr/bin/filebeat",
		"/etc/filebeat/filebeat.yml",
		"/tmp/registry-test",
		"",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 验证Agent在注册表中
	if !registry.Exists("registry-test") {
		t.Error("expected agent to exist in registry")
	}
}
