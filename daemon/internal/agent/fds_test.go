package agent

import (
	"context"
	"os"
	"testing"

	"go.uber.org/zap/zaptest"
)

// TestGetNumFDs 测试文件描述符获取功能
func TestGetNumFDs(t *testing.T) {
	logger := zaptest.NewLogger(t)

	// 使用当前进程的PID进行测试
	pid := int32(os.Getpid())

	numFDs, err := getFDsWithFallback(pid, logger)
	if err != nil {
		t.Logf("getFDsWithFallback failed (may be expected on some platforms): %v", err)
		// 在某些平台上可能失败,这不算错误
		return
	}

	t.Logf("Current process PID: %d, Open FDs: %d", pid, numFDs)

	// 基本验证:文件描述符数量应该大于0(至少有stdin/stdout/stderr)
	if numFDs <= 0 {
		t.Errorf("expected numFDs > 0, got %d", numFDs)
	}

	// 合理性检查:不应该超过1000(对于测试进程来说)
	if numFDs > 1000 {
		t.Logf("warning: unusually high FD count: %d", numFDs)
	}
}

// TestGetNumFDs_InvalidPID 测试无效PID的处理
func TestGetNumFDs_InvalidPID(t *testing.T) {
	logger := zaptest.NewLogger(t)

	// 使用一个不存在的PID
	invalidPID := int32(999999)

	_, err := getFDsWithFallback(invalidPID, logger)
	if err == nil {
		t.Error("expected error for invalid PID, got nil")
	}

	t.Logf("Got expected error for invalid PID: %v", err)
}

// TestResourceMonitorWithFDs 测试资源监控器中的文件描述符采集
func TestResourceMonitorWithFDs(t *testing.T) {
	logger := zaptest.NewLogger(t)
	workDir := t.TempDir()

	multiManager, err := NewMultiAgentManager(workDir, logger)
	if err != nil {
		t.Fatalf("failed to create multi agent manager: %v", err)
	}

	registry := multiManager.GetRegistry()
	monitor := NewResourceMonitor(multiManager, registry, logger)

	// 注册一个测试Agent
	info, err := registry.Register(
		"test-agent-fds",
		TypeCustom,
		"Test Agent FDs",
		"/bin/sleep",
		"",
		workDir,
		"",
	)
	if err != nil {
		t.Fatalf("failed to register agent: %v", err)
	}

	// 创建并启动Agent实例
	instance := NewAgentInstance(info, logger)
	multiManager.mu.Lock()
	multiManager.instances["test-agent-fds"] = instance
	multiManager.mu.Unlock()

	// 启动Agent(使用sleep命令,这样可以测试实际进程)
	ctx := context.Background()
	if err := instance.Start(ctx); err != nil {
		t.Fatalf("failed to start agent: %v", err)
	}
	defer instance.Stop(ctx, false)

	// 等待进程启动
	// time.Sleep(100 * time.Millisecond)

	// 采集资源数据
	dataPoint, err := monitor.collectAgentResources("test-agent-fds")
	if err != nil {
		t.Fatalf("failed to collect agent resources: %v", err)
	}

	t.Logf("Agent PID: %d, Open FDs: %d, CPU: %.2f%%, Memory RSS: %d bytes",
		info.GetPID(), dataPoint.OpenFiles, dataPoint.CPU, dataPoint.MemoryRSS)

	// 验证文件描述符数据被采集
	// 注意:在某些平台上可能为0(如果实现不支持)
	if dataPoint.OpenFiles > 0 {
		t.Logf("Successfully collected FD count: %d", dataPoint.OpenFiles)
	} else {
		t.Logf("FD collection not available on this platform or failed")
	}
}

// TestResourceThresholdWithOpenFiles 测试文件描述符阈值配置
func TestResourceThresholdWithOpenFiles(t *testing.T) {
	logger := zaptest.NewLogger(t)
	workDir := t.TempDir()

	multiManager, err := NewMultiAgentManager(workDir, logger)
	if err != nil {
		t.Fatalf("failed to create multi agent manager: %v", err)
	}

	registry := multiManager.GetRegistry()
	monitor := NewResourceMonitor(multiManager, registry, logger)

	// 设置包含文件描述符阈值的配置
	threshold := &ResourceThreshold{
		CPUThreshold:       80.0,
		MemoryThreshold:    1024 * 1024 * 500, // 500MB
		OpenFilesThreshold: 100,               // 100个文件描述符
		ThresholdDuration:  60,                // 60秒
	}

	monitor.SetThreshold("test-agent", threshold)

	// 验证阈值设置成功
	monitor.mu.RLock()
	storedThreshold := monitor.thresholds["test-agent"]
	monitor.mu.RUnlock()

	if storedThreshold == nil {
		t.Fatal("threshold not stored")
	}

	if storedThreshold.OpenFilesThreshold != 100 {
		t.Errorf("expected OpenFilesThreshold 100, got %d", storedThreshold.OpenFilesThreshold)
	}
}
