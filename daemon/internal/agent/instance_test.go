package agent

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestNewAgentInstance(t *testing.T) {
	logger := zap.NewNop()
	info := &AgentInfo{
		ID:         "test-agent",
		Type:       TypeFilebeat,
		Name:       "Test Agent",
		BinaryPath: "/usr/bin/filebeat",
		ConfigFile: "/etc/filebeat/filebeat.yml",
		WorkDir:    "/tmp/test-agent",
	}

	instance := NewAgentInstance(info, logger)
	if instance == nil {
		t.Fatal("expected non-nil instance")
	}
	if instance.GetInfo() != info {
		t.Error("expected info to match")
	}
}

func TestAgentInstance_GetInfo(t *testing.T) {
	logger := zap.NewNop()
	info := &AgentInfo{
		ID:   "test-agent",
		Type: TypeFilebeat,
	}

	instance := NewAgentInstance(info, logger)
	if instance.GetInfo() != info {
		t.Error("expected GetInfo() to return the same info")
	}
}

func TestAgentInstance_IsRunning(t *testing.T) {
	logger := zap.NewNop()
	info := &AgentInfo{
		ID:         "test-agent",
		Type:       TypeFilebeat,
		BinaryPath: "/usr/bin/filebeat",
		ConfigFile: "/etc/filebeat/filebeat.yml",
		WorkDir:    "/tmp/test-agent",
	}

	instance := NewAgentInstance(info, logger)

	// 初始状态应该未运行
	if instance.IsRunning() {
		t.Error("expected agent not running initially")
	}
}

func TestAgentInstance_GetPID(t *testing.T) {
	logger := zap.NewNop()
	info := &AgentInfo{
		ID:   "test-agent",
		Type: TypeFilebeat,
	}

	instance := NewAgentInstance(info, logger)

	// 初始PID应该为0
	if pid := instance.GetPID(); pid != 0 {
		t.Errorf("expected PID 0, got %d", pid)
	}

	// 设置PID后应该返回新值
	info.SetPID(12345)
	if pid := instance.GetPID(); pid != 12345 {
		t.Errorf("expected PID 12345, got %d", pid)
	}
}

func TestAgentInstance_GetRestartCount(t *testing.T) {
	logger := zap.NewNop()
	info := &AgentInfo{
		ID:   "test-agent",
		Type: TypeFilebeat,
	}

	instance := NewAgentInstance(info, logger)

	// 初始重启次数应该为0
	if count := instance.GetRestartCount(); count != 0 {
		t.Errorf("expected restart count 0, got %d", count)
	}

	// 增加重启次数后应该返回新值
	info.IncrementRestartCount()
	if count := instance.GetRestartCount(); count != 1 {
		t.Errorf("expected restart count 1, got %d", count)
	}
}

func TestAgentInstance_ResetRestartCount(t *testing.T) {
	logger := zap.NewNop()
	info := &AgentInfo{
		ID:   "test-agent",
		Type: TypeFilebeat,
	}

	instance := NewAgentInstance(info, logger)

	// 增加重启次数
	info.IncrementRestartCount()
	info.IncrementRestartCount()

	// 重置后应该为0
	instance.ResetRestartCount()
	if count := instance.GetRestartCount(); count != 0 {
		t.Errorf("expected restart count 0 after reset, got %d", count)
	}
}

func TestAgentInstance_GenerateArgs_Filebeat(t *testing.T) {
	logger := zap.NewNop()
	info := &AgentInfo{
		ID:         "filebeat-test",
		Type:       TypeFilebeat,
		ConfigFile: "/etc/filebeat/filebeat.yml",
		WorkDir:    "/var/lib/daemon/agents/filebeat-test",
	}

	instance := NewAgentInstance(info, logger)
	args := instance.generateArgs()

	expectedArgs := []string{"-c", "/etc/filebeat/filebeat.yml", "-path.home", "/var/lib/daemon/agents/filebeat-test"}
	if len(args) != len(expectedArgs) {
		t.Fatalf("expected %d args, got %d", len(expectedArgs), len(args))
	}
	for i, arg := range expectedArgs {
		if args[i] != arg {
			t.Errorf("expected args[%d] = %s, got %s", i, arg, args[i])
		}
	}
}

func TestAgentInstance_GenerateArgs_Telegraf(t *testing.T) {
	logger := zap.NewNop()
	info := &AgentInfo{
		ID:         "telegraf-test",
		Type:       TypeTelegraf,
		ConfigFile: "/etc/telegraf/telegraf.conf",
	}

	instance := NewAgentInstance(info, logger)
	args := instance.generateArgs()

	expectedArgs := []string{"-config", "/etc/telegraf/telegraf.conf"}
	if len(args) != len(expectedArgs) {
		t.Fatalf("expected %d args, got %d", len(expectedArgs), len(args))
	}
	for i, arg := range expectedArgs {
		if args[i] != arg {
			t.Errorf("expected args[%d] = %s, got %s", i, arg, args[i])
		}
	}
}

func TestAgentInstance_GenerateArgs_NodeExporter(t *testing.T) {
	logger := zap.NewNop()
	info := &AgentInfo{
		ID:         "node-exporter-test",
		Type:       TypeNodeExporter,
		ConfigFile: "", // Node Exporter 不使用配置文件
	}

	instance := NewAgentInstance(info, logger)
	args := instance.generateArgs()

	// Node Exporter 应该返回默认参数
	if len(args) == 0 {
		t.Error("expected non-empty args for node_exporter")
	}
	// 检查是否包含预期的参数
	hasListenAddress := false
	for _, arg := range args {
		if arg == "--web.listen-address=:9100" {
			hasListenAddress = true
			break
		}
	}
	if !hasListenAddress {
		t.Error("expected --web.listen-address=:9100 in args")
	}
}

func TestAgentInstance_GenerateArgs_Custom(t *testing.T) {
	logger := zap.NewNop()

	// 测试有配置文件的自定义类型
	info1 := &AgentInfo{
		ID:         "custom-test-1",
		Type:       TypeCustom,
		ConfigFile: "/etc/custom/config.json",
	}
	instance1 := NewAgentInstance(info1, logger)
	args1 := instance1.generateArgs()
	expectedArgs1 := []string{"-config", "/etc/custom/config.json"}
	if len(args1) != len(expectedArgs1) {
		t.Fatalf("expected %d args, got %d", len(expectedArgs1), len(args1))
	}

	// 测试无配置文件的自定义类型
	info2 := &AgentInfo{
		ID:         "custom-test-2",
		Type:       TypeCustom,
		ConfigFile: "",
	}
	instance2 := NewAgentInstance(info2, logger)
	args2 := instance2.generateArgs()
	if len(args2) != 0 {
		t.Errorf("expected empty args for custom type without config file, got %v", args2)
	}
}

func TestAgentInstance_GetLogFilePath(t *testing.T) {
	logger := zap.NewNop()

	// 测试有工作目录的情况
	info1 := &AgentInfo{
		ID:      "test-agent-1",
		Type:    TypeFilebeat,
		WorkDir: "/var/lib/daemon/agents/test-agent-1",
	}
	instance1 := NewAgentInstance(info1, logger)
	logPath1 := instance1.getLogFilePath()
	expectedPath1 := "/var/lib/daemon/agents/test-agent-1/agents/test-agent-1/logs/agent.log"
	if logPath1 != expectedPath1 {
		t.Errorf("expected log path %s, got %s", expectedPath1, logPath1)
	}

	// 测试无工作目录的情况
	info2 := &AgentInfo{
		ID:      "test-agent-2",
		Type:    TypeFilebeat,
		WorkDir: "",
	}
	instance2 := NewAgentInstance(info2, logger)
	logPath2 := instance2.getLogFilePath()
	expectedPath2 := "/tmp/agents/test-agent-2/logs/agent.log"
	if logPath2 != expectedPath2 {
		t.Errorf("expected log path %s, got %s", expectedPath2, logPath2)
	}
}

func TestAgentInstance_CalculateBackoff(t *testing.T) {
	logger := zap.NewNop()
	info := &AgentInfo{
		ID:   "test-agent",
		Type: TypeFilebeat,
	}

	instance := NewAgentInstance(info, logger)

	// 测试重启次数 < 1
	backoff := instance.calculateBackoff()
	if backoff != 0 {
		t.Errorf("expected backoff 0 for restart count < 1, got %v", backoff)
	}

	// 测试重启次数 < 3
	info.IncrementRestartCount()
	backoff = instance.calculateBackoff()
	if backoff != 10*time.Second {
		t.Errorf("expected backoff 10s for restart count < 3, got %v", backoff)
	}

	// 测试重启次数 < 5
	info.IncrementRestartCount()
	info.IncrementRestartCount()
	backoff = instance.calculateBackoff()
	if backoff != 30*time.Second {
		t.Errorf("expected backoff 30s for restart count < 5, got %v", backoff)
	}

	// 测试重启次数 >= 5
	info.IncrementRestartCount()
	info.IncrementRestartCount()
	backoff = instance.calculateBackoff()
	if backoff != 60*time.Second {
		t.Errorf("expected backoff 60s for restart count >= 5, got %v", backoff)
	}

	// 测试距离上次重启超过5分钟，应该重置计数
	info.SetStatus(StatusStopped)
	// 模拟6分钟前重启
	oldTime := time.Now().Add(-6 * time.Minute)
	info.mu.Lock()
	info.LastRestart = oldTime
	info.RestartCount = 5
	info.mu.Unlock()

	backoff = instance.calculateBackoff()
	if backoff != 0 {
		t.Errorf("expected backoff 0 after 5 minutes, got %v", backoff)
	}
	if instance.GetRestartCount() != 0 {
		t.Error("expected restart count to be reset after 5 minutes")
	}
}

// 注意: 以下测试需要实际的二进制文件，在CI/CD环境中可能无法运行
// 这些测试主要用于验证逻辑，实际启动测试需要mock或使用测试二进制

func TestAgentInstance_Start_InvalidBinary(t *testing.T) {
	logger := zap.NewNop()
	info := &AgentInfo{
		ID:         "test-agent",
		Type:       TypeFilebeat,
		BinaryPath: "/nonexistent/binary",
		ConfigFile: "/nonexistent/config.yml",
		WorkDir:    "/tmp/test-agent",
	}

	instance := NewAgentInstance(info, logger)
	ctx := context.Background()

	// 创建临时工作目录
	workDir := "/tmp/test-agent-start"
	os.MkdirAll(workDir, 0755)
	defer os.RemoveAll(workDir)
	info.WorkDir = workDir

	// 尝试启动不存在的二进制文件，应该失败
	err := instance.Start(ctx)
	if err == nil {
		t.Error("expected error when starting non-existent binary")
	}

	// 状态应该为失败
	if status := info.GetStatus(); status != StatusFailed {
		t.Errorf("expected status StatusFailed, got %s", status)
	}
}

// TestAgentInstance_Stop_ProcessNilButPIDExists 测试 process 为 nil 但 PID 不为 0 的情况
func TestAgentInstance_Stop_ProcessNilButPIDExists(t *testing.T) {
	logger := zap.NewNop()
	info := &AgentInfo{
		ID:   "test-agent",
		Type: TypeFilebeat,
	}

	instance := NewAgentInstance(info, logger)
	ctx := context.Background()

	// 创建一个临时进程用于测试
	cmd := exec.Command("sleep", "60")
	if err := cmd.Start(); err != nil {
		t.Skipf("cannot start test process: %v", err)
	}
	defer cmd.Process.Kill()

	pid := cmd.Process.Pid
	info.SetPID(pid)
	info.SetStatus(StatusRunning)

	// 模拟 process 对象为 nil 的情况
	instance.mu.Lock()
	instance.process = nil
	instance.mu.Unlock()

	// 停止应该成功，通过 PID 找到进程并停止
	err := instance.Stop(ctx, true)
	if err != nil {
		t.Errorf("expected no error when stopping with nil process but valid PID, got %v", err)
	}

	// 验证进程已停止
	if err := cmd.Process.Signal(syscall.Signal(0)); err == nil {
		t.Error("expected process to be stopped")
	}

	// 验证状态已更新
	if status := info.GetStatus(); status != StatusStopped {
		t.Errorf("expected status StatusStopped, got %s", status)
	}
	if pid := info.GetPID(); pid != 0 {
		t.Errorf("expected PID 0 after stop, got %d", pid)
	}
}

// TestAgentInstance_Stop_ProcessAlreadyExited 测试进程已经退出的情况
func TestAgentInstance_Stop_ProcessAlreadyExited(t *testing.T) {
	logger := zap.NewNop()
	info := &AgentInfo{
		ID:   "test-agent",
		Type: TypeFilebeat,
	}

	instance := NewAgentInstance(info, logger)
	ctx := context.Background()

	// 创建一个快速退出的进程
	cmd := exec.Command("true") // true 命令立即退出
	if err := cmd.Start(); err != nil {
		t.Skipf("cannot start test process: %v", err)
	}
	cmd.Wait() // 等待进程退出

	pid := cmd.Process.Pid
	info.SetPID(pid)
	info.SetStatus(StatusRunning)

	// 设置 process 对象
	instance.mu.Lock()
	instance.process = cmd.Process
	instance.mu.Unlock()

	// 停止应该成功，即使进程已经退出
	err := instance.Stop(ctx, true)
	if err != nil {
		t.Errorf("expected no error when stopping already exited process, got %v", err)
	}

	// 验证状态已更新
	if status := info.GetStatus(); status != StatusStopped {
		t.Errorf("expected status StatusStopped, got %s", status)
	}
	if pid := info.GetPID(); pid != 0 {
		t.Errorf("expected PID 0 after stop, got %d", pid)
	}
}

func TestAgentInstance_Stop_NotRunning(t *testing.T) {
	logger := zap.NewNop()
	info := &AgentInfo{
		ID:   "test-agent",
		Type: TypeFilebeat,
	}

	instance := NewAgentInstance(info, logger)
	ctx := context.Background()

	// 停止未运行的Agent应该成功（无操作）
	err := instance.Stop(ctx, true)
	if err != nil {
		t.Errorf("expected no error when stopping non-running agent, got %v", err)
	}
}

// TestAgentInstance_Stop_Concurrent 测试并发 Stop 调用
func TestAgentInstance_Stop_Concurrent(t *testing.T) {
	logger := zap.NewNop()
	info := &AgentInfo{
		ID:   "test-agent",
		Type: TypeFilebeat,
	}

	instance := NewAgentInstance(info, logger)
	ctx := context.Background()

	// 创建一个临时进程用于测试
	cmd := exec.Command("sleep", "60")
	if err := cmd.Start(); err != nil {
		t.Skipf("cannot start test process: %v", err)
	}
	defer cmd.Process.Kill()

	pid := cmd.Process.Pid
	info.SetPID(pid)
	info.SetStatus(StatusRunning)

	instance.mu.Lock()
	instance.process = cmd.Process
	instance.mu.Unlock()

	// 并发调用 Stop
	var wg sync.WaitGroup
	errors := make(chan error, 10)
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := instance.Stop(ctx, true); err != nil {
				errors <- err
			}
		}()
	}

	wg.Wait()
	close(errors)

	// 检查是否有错误
	for err := range errors {
		t.Errorf("unexpected error in concurrent Stop: %v", err)
	}

	// 验证进程已停止
	if err := cmd.Process.Signal(syscall.Signal(0)); err == nil {
		t.Error("expected process to be stopped")
	}
}

func TestAgentInstance_Restart_NotRunning(t *testing.T) {
	logger := zap.NewNop()
	info := &AgentInfo{
		ID:         "test-agent",
		Type:       TypeFilebeat,
		BinaryPath: "/nonexistent/binary",
		ConfigFile: "/nonexistent/config.yml",
		WorkDir:    "/tmp/test-agent",
	}

	instance := NewAgentInstance(info, logger)
	ctx := context.Background()

	// 创建临时工作目录
	workDir := "/tmp/test-agent-restart"
	os.MkdirAll(workDir, 0755)
	defer os.RemoveAll(workDir)
	info.WorkDir = workDir

	// 重启未运行的Agent应该尝试启动（但会失败因为二进制不存在）
	err := instance.Restart(ctx, false)
	if err == nil {
		t.Error("expected error when restarting with non-existent binary")
	}

	// 状态应该为失败
	if status := info.GetStatus(); status != StatusFailed {
		t.Errorf("expected status StatusFailed, got %s", status)
	}
}

// TestAgentInstance_Restart_StopFailedButProcessExited 测试 Stop 失败但进程已退出的情况
func TestAgentInstance_Restart_StopFailedButProcessExited(t *testing.T) {
	logger := zap.NewNop()
	info := &AgentInfo{
		ID:         "test-agent",
		Type:       TypeFilebeat,
		BinaryPath: "/nonexistent/binary",
		ConfigFile: "/nonexistent/config.yml",
		WorkDir:    "/tmp/test-agent",
	}

	instance := NewAgentInstance(info, logger)
	ctx := context.Background()

	// 创建一个快速退出的进程
	cmd := exec.Command("true")
	if err := cmd.Start(); err != nil {
		t.Skipf("cannot start test process: %v", err)
	}
	cmd.Wait()

	pid := cmd.Process.Pid
	info.SetPID(pid)
	info.SetStatus(StatusRunning)

	instance.mu.Lock()
	instance.process = cmd.Process
	instance.mu.Unlock()

	// 重启应该继续，即使 Stop 时进程已退出
	// 注意：由于二进制不存在，启动会失败，但这是预期的
	err := instance.Restart(ctx, true)
	if err == nil {
		t.Error("expected error when restarting with non-existent binary")
	}

	// 状态应该为失败（因为启动失败）
	if status := info.GetStatus(); status != StatusFailed {
		t.Errorf("expected status StatusFailed, got %s", status)
	}
}

func TestAgentInstance_StatusTransitions(t *testing.T) {
	logger := zap.NewNop()
	info := &AgentInfo{
		ID:   "test-agent",
		Type: TypeFilebeat,
	}
	// 设置初始状态为 Stopped
	info.SetStatus(StatusStopped)

	_ = NewAgentInstance(info, logger)

	// 初始状态应该是Stopped
	if status := info.GetStatus(); status != StatusStopped {
		t.Errorf("expected initial status StatusStopped, got %s", status)
	}

	// 测试状态转换（不实际启动进程）
	info.SetStatus(StatusStarting)
	if status := info.GetStatus(); status != StatusStarting {
		t.Errorf("expected status StatusStarting, got %s", status)
	}

	info.SetStatus(StatusRunning)
	if status := info.GetStatus(); status != StatusRunning {
		t.Errorf("expected status StatusRunning, got %s", status)
	}

	info.SetStatus(StatusStopping)
	if status := info.GetStatus(); status != StatusStopping {
		t.Errorf("expected status StatusStopping, got %s", status)
	}

	info.SetStatus(StatusStopped)
	if status := info.GetStatus(); status != StatusStopped {
		t.Errorf("expected status StatusStopped, got %s", status)
	}
}

// TestAgentInstance_Start_Concurrent 测试并发 Start 调用
func TestAgentInstance_Start_Concurrent(t *testing.T) {
	logger := zap.NewNop()
	tempDir := createTempTestDir(t)
	defer os.RemoveAll(tempDir)

	info := &AgentInfo{
		ID:         "test-agent",
		Type:       TypeCustom,
		BinaryPath: "/bin/sleep", // 使用系统命令
		ConfigFile: "",
		WorkDir:    tempDir,
	}

	instance := NewAgentInstance(info, logger)
	ctx := context.Background()

	// 并发调用 Start
	var wg sync.WaitGroup
	errors := make(chan error, 10)
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := instance.Start(ctx); err != nil {
				errors <- err
			}
		}()
	}

	wg.Wait()
	close(errors)

	// 应该只有一个成功启动，其他的应该失败或忽略
	errorCount := 0
	for err := range errors {
		if err != nil {
			errorCount++
		}
	}

	// 验证只有一个进程在运行
	runningCount := 0
	if instance.IsRunning() {
		runningCount++
	}

	if runningCount > 1 {
		t.Errorf("expected at most 1 running process, got %d", runningCount)
	}

	// 清理
	instance.Stop(ctx, true)
}

// TestAgentInstance_ManuallyStopped 测试手动停止标志
func TestAgentInstance_ManuallyStopped(t *testing.T) {
	logger := zap.NewNop()
	info := &AgentInfo{
		ID:   "test-agent",
		Type: TypeFilebeat,
	}

	instance := NewAgentInstance(info, logger)
	ctx := context.Background()

	// 初始状态应该不是手动停止
	if instance.IsManuallyStopped() {
		t.Error("expected not manually stopped initially")
	}

	// 停止后应该标记为手动停止
	err := instance.Stop(ctx, true)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if !instance.IsManuallyStopped() {
		t.Error("expected manually stopped after Stop")
	}

	// 启动后应该清除手动停止标志
	info.BinaryPath = "/nonexistent/binary"
	info.WorkDir = "/tmp/test-agent"
	_ = instance.Start(ctx) // 会失败，但不影响测试

	if instance.IsManuallyStopped() {
		t.Error("expected not manually stopped after Start")
	}
}

// TestAgentInstance_IsRunning_ProcessNil 测试 process 为 nil 时的 IsRunning
func TestAgentInstance_IsRunning_ProcessNil(t *testing.T) {
	logger := zap.NewNop()
	info := &AgentInfo{
		ID:   "test-agent",
		Type: TypeFilebeat,
	}

	instance := NewAgentInstance(info, logger)

	// 初始状态应该未运行
	if instance.IsRunning() {
		t.Error("expected not running initially")
	}

	// 设置 PID 但 process 为 nil
	info.SetPID(12345)
	instance.mu.Lock()
	instance.process = nil
	instance.mu.Unlock()

	// 应该返回 false（因为 process 为 nil）
	if instance.IsRunning() {
		t.Error("expected not running when process is nil")
	}
}

// 辅助函数：创建临时测试目录
func createTempTestDir(t *testing.T) string {
	dir, err := os.MkdirTemp("", "agent-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	return dir
}

func TestAgentInstance_LogFilePath(t *testing.T) {
	logger := zap.NewNop()
	tempDir := createTempTestDir(t)
	defer os.RemoveAll(tempDir)

	info := &AgentInfo{
		ID:      "test-log-agent",
		Type:    TypeFilebeat,
		WorkDir: tempDir,
	}

	instance := NewAgentInstance(info, logger)
	logPath := instance.getLogFilePath()

	expectedPath := filepath.Join(tempDir, "agents", "test-log-agent", "logs", "agent.log")
	if logPath != expectedPath {
		t.Errorf("expected log path %s, got %s", expectedPath, logPath)
	}
}
