package agent

import (
	"fmt"
	"testing"
	"time"
)

func TestNewAgentRegistry(t *testing.T) {
	registry := NewAgentRegistry()
	if registry == nil {
		t.Fatal("expected non-nil registry")
	}
	if registry.Count() != 0 {
		t.Errorf("expected empty registry, got %d agents", registry.Count())
	}
}

func TestAgentRegistry_Register(t *testing.T) {
	registry := NewAgentRegistry()

	// 测试成功注册
	info, err := registry.Register(
		"test-agent",
		TypeFilebeat,
		"Test Agent",
		"/usr/bin/filebeat",
		"/etc/filebeat/filebeat.yml",
		"/var/lib/daemon/agents/test-agent",
		"",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info == nil {
		t.Fatal("expected non-nil AgentInfo")
	}
	if info.ID != "test-agent" {
		t.Errorf("expected ID 'test-agent', got '%s'", info.ID)
	}
	if info.Type != TypeFilebeat {
		t.Errorf("expected Type '%s', got '%s'", TypeFilebeat, info.Type)
	}
	if info.Status != StatusStopped {
		t.Errorf("expected Status '%s', got '%s'", StatusStopped, info.Status)
	}
	if registry.Count() != 1 {
		t.Errorf("expected 1 agent, got %d", registry.Count())
	}

	// 测试重复注册
	_, err = registry.Register(
		"test-agent",
		TypeTelegraf,
		"Another Test Agent",
		"/usr/bin/telegraf",
		"/etc/telegraf/telegraf.conf",
		"/var/lib/daemon/agents/test-agent",
		"",
	)
	if err == nil {
		t.Fatal("expected error for duplicate ID")
	}
	if _, ok := err.(*AgentExistsError); !ok {
		t.Errorf("expected AgentExistsError, got %T", err)
	}
}

func TestAgentRegistry_Get(t *testing.T) {
	registry := NewAgentRegistry()

	// 注册一个Agent
	_, err := registry.Register(
		"test-agent",
		TypeFilebeat,
		"Test Agent",
		"/usr/bin/filebeat",
		"/etc/filebeat/filebeat.yml",
		"/var/lib/daemon/agents/test-agent",
		"",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 测试获取存在的Agent
	info := registry.Get("test-agent")
	if info == nil {
		t.Fatal("expected non-nil AgentInfo")
	}
	if info.ID != "test-agent" {
		t.Errorf("expected ID 'test-agent', got '%s'", info.ID)
	}

	// 测试获取不存在的Agent
	info = registry.Get("non-existent")
	if info != nil {
		t.Errorf("expected nil for non-existent agent, got %v", info)
	}
}

func TestAgentRegistry_Unregister(t *testing.T) {
	registry := NewAgentRegistry()

	// 注册一个Agent
	_, err := registry.Register(
		"test-agent",
		TypeFilebeat,
		"Test Agent",
		"/usr/bin/filebeat",
		"/etc/filebeat/filebeat.yml",
		"/var/lib/daemon/agents/test-agent",
		"",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 测试注销存在的Agent（未运行状态）
	err = registry.Unregister("test-agent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if registry.Count() != 0 {
		t.Errorf("expected 0 agents after unregister, got %d", registry.Count())
	}
	if registry.Exists("test-agent") {
		t.Error("agent should not exist after unregister")
	}

	// 测试注销不存在的Agent
	err = registry.Unregister("non-existent")
	if err == nil {
		t.Fatal("expected error for non-existent agent")
	}
	if _, ok := err.(*AgentNotFoundError); !ok {
		t.Errorf("expected AgentNotFoundError, got %T", err)
	}
}

func TestAgentRegistry_Unregister_RunningAgent(t *testing.T) {
	registry := NewAgentRegistry()

	// 注册一个Agent
	info, err := registry.Register(
		"test-agent",
		TypeFilebeat,
		"Test Agent",
		"/usr/bin/filebeat",
		"/etc/filebeat/filebeat.yml",
		"/var/lib/daemon/agents/test-agent",
		"",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 设置Agent为运行状态
	info.SetStatus(StatusRunning)

	// 测试注销运行中的Agent（应该失败）
	err = registry.Unregister("test-agent")
	if err == nil {
		t.Fatal("expected error for running agent")
	}
	if _, ok := err.(*AgentRunningError); !ok {
		t.Errorf("expected AgentRunningError, got %T", err)
	}
	if registry.Count() != 1 {
		t.Errorf("expected 1 agent (still registered), got %d", registry.Count())
	}

	// 设置Agent为停止状态后再次注销
	info.SetStatus(StatusStopped)
	err = registry.Unregister("test-agent")
	if err != nil {
		t.Fatalf("unexpected error after stopping: %v", err)
	}
}

func TestAgentRegistry_List(t *testing.T) {
	registry := NewAgentRegistry()

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
		_, err := registry.Register(
			a.id,
			a.typ,
			a.name,
			"/usr/bin/"+string(a.typ),
			"",
			"/var/lib/daemon/agents/"+a.id,
			"",
		)
		if err != nil {
			t.Fatalf("unexpected error registering %s: %v", a.id, err)
		}
	}

	// 测试列举所有Agent
	list := registry.List()
	if len(list) != 3 {
		t.Errorf("expected 3 agents, got %d", len(list))
	}

	// 验证所有Agent都在列表中
	found := make(map[string]bool)
	for _, info := range list {
		found[info.ID] = true
	}
	for _, a := range agents {
		if !found[a.id] {
			t.Errorf("agent %s not found in list", a.id)
		}
	}
}

func TestAgentRegistry_ListByType(t *testing.T) {
	registry := NewAgentRegistry()

	// 注册多个不同类型的Agent
	_, err := registry.Register("filebeat-1", TypeFilebeat, "Filebeat 1", "/usr/bin/filebeat", "", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, err = registry.Register("filebeat-2", TypeFilebeat, "Filebeat 2", "/usr/bin/filebeat", "", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, err = registry.Register("telegraf-1", TypeTelegraf, "Telegraf 1", "/usr/bin/telegraf", "", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 测试按类型列举
	filebeats := registry.ListByType(TypeFilebeat)
	if len(filebeats) != 2 {
		t.Errorf("expected 2 filebeat agents, got %d", len(filebeats))
	}
	for _, info := range filebeats {
		if info.Type != TypeFilebeat {
			t.Errorf("expected TypeFilebeat, got %s", info.Type)
		}
	}

	telegrafs := registry.ListByType(TypeTelegraf)
	if len(telegrafs) != 1 {
		t.Errorf("expected 1 telegraf agent, got %d", len(telegrafs))
	}

	// 测试不存在的类型
	nodeExporters := registry.ListByType(TypeNodeExporter)
	if len(nodeExporters) != 0 {
		t.Errorf("expected 0 node_exporter agents, got %d", len(nodeExporters))
	}
}

func TestAgentRegistry_Count(t *testing.T) {
	registry := NewAgentRegistry()

	// 初始计数应该为0
	if registry.Count() != 0 {
		t.Errorf("expected 0 agents, got %d", registry.Count())
	}

	// 注册Agent后计数应该增加
	_, err := registry.Register("agent-1", TypeFilebeat, "Agent 1", "/usr/bin/filebeat", "", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if registry.Count() != 1 {
		t.Errorf("expected 1 agent, got %d", registry.Count())
	}

	_, err = registry.Register("agent-2", TypeTelegraf, "Agent 2", "/usr/bin/telegraf", "", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if registry.Count() != 2 {
		t.Errorf("expected 2 agents, got %d", registry.Count())
	}

	// 注销后计数应该减少
	err = registry.Unregister("agent-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if registry.Count() != 1 {
		t.Errorf("expected 1 agent after unregister, got %d", registry.Count())
	}
}

func TestAgentRegistry_Exists(t *testing.T) {
	registry := NewAgentRegistry()

	// 测试不存在的Agent
	if registry.Exists("non-existent") {
		t.Error("expected non-existent agent to not exist")
	}

	// 注册Agent后应该存在
	_, err := registry.Register("test-agent", TypeFilebeat, "Test Agent", "/usr/bin/filebeat", "", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !registry.Exists("test-agent") {
		t.Error("expected agent to exist after registration")
	}

	// 注销后应该不存在
	err = registry.Unregister("test-agent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if registry.Exists("test-agent") {
		t.Error("expected agent to not exist after unregister")
	}
}

func TestAgentInfo_GetSetPID(t *testing.T) {
	info := &AgentInfo{
		ID:   "test-agent",
		Type: TypeFilebeat,
	}

	// 初始PID应该为0
	if pid := info.GetPID(); pid != 0 {
		t.Errorf("expected PID 0, got %d", pid)
	}

	// 设置PID
	info.SetPID(12345)
	if pid := info.GetPID(); pid != 12345 {
		t.Errorf("expected PID 12345, got %d", pid)
	}
}

func TestAgentInfo_GetSetStatus(t *testing.T) {
	info := &AgentInfo{
		ID:   "test-agent",
		Type: TypeFilebeat,
	}

	// 初始状态应该为Stopped
	if status := info.GetStatus(); status != StatusStopped {
		t.Errorf("expected StatusStopped, got %s", status)
	}

	// 设置状态
	info.SetStatus(StatusRunning)
	if status := info.GetStatus(); status != StatusRunning {
		t.Errorf("expected StatusRunning, got %s", status)
	}

	info.SetStatus(StatusStopping)
	if status := info.GetStatus(); status != StatusStopping {
		t.Errorf("expected StatusStopping, got %s", status)
	}
}

func TestAgentInfo_RestartCount(t *testing.T) {
	info := &AgentInfo{
		ID:   "test-agent",
		Type: TypeFilebeat,
	}

	// 初始重启次数应该为0
	if count := info.GetRestartCount(); count != 0 {
		t.Errorf("expected restart count 0, got %d", count)
	}

	// 增加重启次数
	info.IncrementRestartCount()
	if count := info.GetRestartCount(); count != 1 {
		t.Errorf("expected restart count 1, got %d", count)
	}

	info.IncrementRestartCount()
	info.IncrementRestartCount()
	if count := info.GetRestartCount(); count != 3 {
		t.Errorf("expected restart count 3, got %d", count)
	}

	// 重置重启次数
	info.ResetRestartCount()
	if count := info.GetRestartCount(); count != 0 {
		t.Errorf("expected restart count 0 after reset, got %d", count)
	}
}

func TestAgentInfo_LastRestart(t *testing.T) {
	info := &AgentInfo{
		ID:   "test-agent",
		Type: TypeFilebeat,
	}

	// 初始LastRestart应该为零值
	initialTime := info.GetLastRestart()
	if !initialTime.IsZero() {
		t.Error("expected zero time for initial LastRestart")
	}

	// 增加重启次数会更新LastRestart
	beforeIncrement := time.Now()
	info.IncrementRestartCount()
	afterIncrement := time.Now()
	lastRestart := info.GetLastRestart()

	if lastRestart.Before(beforeIncrement) || lastRestart.After(afterIncrement) {
		t.Errorf("LastRestart should be between %v and %v, got %v", beforeIncrement, afterIncrement, lastRestart)
	}
}

func TestAgentInfo_UpdateTimestamp(t *testing.T) {
	info := &AgentInfo{
		ID:        "test-agent",
		Type:      TypeFilebeat,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	initialUpdatedAt := info.UpdatedAt

	// 等待一小段时间确保时间戳不同
	time.Sleep(10 * time.Millisecond)

	// 更新时间戳
	info.UpdateTimestamp()

	if !info.UpdatedAt.After(initialUpdatedAt) {
		t.Error("UpdatedAt should be updated after UpdateTimestamp()")
	}
}

// 并发安全测试
func TestAgentRegistry_ConcurrentAccess(t *testing.T) {
	registry := NewAgentRegistry()
	const numGoroutines = 100
	const numAgents = 10

	// 并发注册
	done := make(chan bool, numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer func() { done <- true }()
			agentID := fmt.Sprintf("agent-%d", id%numAgents)
			_, err := registry.Register(
				agentID,
				TypeFilebeat,
				"Test Agent",
				"/usr/bin/filebeat",
				"",
				"",
				"",
			)
			// 允许重复注册错误
			if err != nil {
				if _, ok := err.(*AgentExistsError); !ok {
					t.Errorf("unexpected error: %v", err)
				}
			}
		}(i)
	}

	// 等待所有goroutine完成
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// 验证最终状态
	count := registry.Count()
	if count < 1 || count > numAgents {
		t.Errorf("expected count between 1 and %d, got %d", numAgents, count)
	}
}

func TestAgentInfo_ConcurrentAccess(t *testing.T) {
	info := &AgentInfo{
		ID:   "test-agent",
		Type: TypeFilebeat,
	}

	const numGoroutines = 100

	// 并发读写
	done := make(chan bool, numGoroutines*2)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer func() { done <- true }()
			info.SetPID(12345)
			_ = info.GetPID()
		}()
		go func() {
			defer func() { done <- true }()
			info.SetStatus(StatusRunning)
			_ = info.GetStatus()
		}()
	}

	// 等待所有goroutine完成
	for i := 0; i < numGoroutines*2; i++ {
		<-done
	}

	// 验证最终状态（应该没有panic或数据竞争）
	if info.GetPID() != 12345 {
		t.Errorf("expected PID 12345, got %d", info.GetPID())
	}
	if info.GetStatus() != StatusRunning {
		t.Errorf("expected StatusRunning, got %s", info.GetStatus())
	}
}

// 错误类型测试
func TestAgentExistsError(t *testing.T) {
	err := &AgentExistsError{ID: "test-agent"}
	expectedMsg := "agent already exists: test-agent"
	if err.Error() != expectedMsg {
		t.Errorf("expected error message '%s', got '%s'", expectedMsg, err.Error())
	}
}

func TestAgentNotFoundError(t *testing.T) {
	err := &AgentNotFoundError{ID: "test-agent"}
	expectedMsg := "agent not found: test-agent"
	if err.Error() != expectedMsg {
		t.Errorf("expected error message '%s', got '%s'", expectedMsg, err.Error())
	}
}

func TestAgentRunningError(t *testing.T) {
	err := &AgentRunningError{ID: "test-agent"}
	expectedMsg := "agent is running, cannot unregister: test-agent"
	if err.Error() != expectedMsg {
		t.Errorf("expected error message '%s', got '%s'", expectedMsg, err.Error())
	}
}
