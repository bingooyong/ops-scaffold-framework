package agent

import (
	"fmt"
	"testing"
	"time"

	"go.uber.org/zap/zaptest"
)

func TestNewResourceMonitor(t *testing.T) {
	logger := zaptest.NewLogger(t)
	workDir := t.TempDir()
	multiManager, err := NewMultiAgentManager(workDir, logger)
	if err != nil {
		t.Fatalf("failed to create multi agent manager: %v", err)
	}

	registry := multiManager.GetRegistry()
	monitor := NewResourceMonitor(multiManager, registry, logger)

	if monitor == nil {
		t.Fatal("NewResourceMonitor returned nil")
	}
	if monitor.multiManager != multiManager {
		t.Error("multiManager not set correctly")
	}
	if monitor.registry != registry {
		t.Error("registry not set correctly")
	}
	if monitor.interval != 60*time.Second {
		t.Errorf("expected default interval 60s, got %v", monitor.interval)
	}
}

func TestSetInterval(t *testing.T) {
	logger := zaptest.NewLogger(t)
	workDir := t.TempDir()
	multiManager, err := NewMultiAgentManager(workDir, logger)
	if err != nil {
		t.Fatalf("failed to create multi agent manager: %v", err)
	}

	registry := multiManager.GetRegistry()
	monitor := NewResourceMonitor(multiManager, registry, logger)

	newInterval := 30 * time.Second
	monitor.SetInterval(newInterval)

	if monitor.interval != newInterval {
		t.Errorf("expected interval %v, got %v", newInterval, monitor.interval)
	}
}

func TestSetThreshold(t *testing.T) {
	logger := zaptest.NewLogger(t)
	workDir := t.TempDir()
	multiManager, err := NewMultiAgentManager(workDir, logger)
	if err != nil {
		t.Fatalf("failed to create multi agent manager: %v", err)
	}

	registry := multiManager.GetRegistry()
	monitor := NewResourceMonitor(multiManager, registry, logger)

	agentID := "test-agent"
	threshold := &ResourceThreshold{
		CPUThreshold:      80.0,
		MemoryThreshold:   1024 * 1024 * 100, // 100MB
		ThresholdDuration: 5 * time.Minute,
	}

	monitor.SetThreshold(agentID, threshold)

	monitor.mu.RLock()
	defer monitor.mu.RUnlock()

	if monitor.thresholds[agentID] == nil {
		t.Fatal("threshold not set")
	}
	if monitor.thresholds[agentID].CPUThreshold != 80.0 {
		t.Errorf("expected CPU threshold 80.0, got %f", monitor.thresholds[agentID].CPUThreshold)
	}
}

func TestCollectAgentResources_NotRunning(t *testing.T) {
	logger := zaptest.NewLogger(t)
	workDir := t.TempDir()
	multiManager, err := NewMultiAgentManager(workDir, logger)
	if err != nil {
		t.Fatalf("failed to create multi agent manager: %v", err)
	}

	registry := multiManager.GetRegistry()
	monitor := NewResourceMonitor(multiManager, registry, logger)

	// 注册一个Agent但不启动
	_, err = registry.Register(
		"test-agent",
		TypeCustom,
		"Test Agent",
		"/bin/echo",
		"",
		workDir,
		"",
	)
	if err != nil {
		t.Fatalf("failed to register agent: %v", err)
	}

	// 尝试采集资源(Agent未运行,PID=0)
	_, err = monitor.collectAgentResources("test-agent")
	if err == nil {
		t.Error("expected error when agent not running")
	}
}

func TestCollectAgentResources_InvalidPID(t *testing.T) {
	logger := zaptest.NewLogger(t)
	workDir := t.TempDir()
	multiManager, err := NewMultiAgentManager(workDir, logger)
	if err != nil {
		t.Fatalf("failed to create multi agent manager: %v", err)
	}

	registry := multiManager.GetRegistry()
	monitor := NewResourceMonitor(multiManager, registry, logger)

	// 注册一个Agent
	info, err := registry.Register(
		"test-agent",
		TypeCustom,
		"Test Agent",
		"/bin/echo",
		"",
		workDir,
		"",
	)
	if err != nil {
		t.Fatalf("failed to register agent: %v", err)
	}

	// 设置一个无效的PID
	info.SetPID(999999)

	// 尝试采集资源
	_, err = monitor.collectAgentResources("test-agent")
	if err == nil {
		t.Error("expected error with invalid PID")
	}
}

func TestUpdateAgentResourceData(t *testing.T) {
	logger := zaptest.NewLogger(t)
	workDir := t.TempDir()
	multiManager, err := NewMultiAgentManager(workDir, logger)
	if err != nil {
		t.Fatalf("failed to create multi agent manager: %v", err)
	}

	registry := multiManager.GetRegistry()
	monitor := NewResourceMonitor(multiManager, registry, logger)

	// 注册并创建Agent实例
	info, err := registry.Register(
		"test-agent",
		TypeCustom,
		"Test Agent",
		"/bin/echo",
		"",
		workDir,
		"",
	)
	if err != nil {
		t.Fatalf("failed to register agent: %v", err)
	}

	instance := NewAgentInstance(info, logger)
	multiManager.mu.Lock()
	multiManager.instances["test-agent"] = instance
	multiManager.mu.Unlock()

	// 创建测试数据点
	dataPoint := &ResourceDataPoint{
		Timestamp:      time.Now(),
		CPU:            50.0,
		MemoryRSS:      1024 * 1024 * 50, // 50MB
		MemoryVMS:      1024 * 1024 * 100,
		DiskReadBytes:  1024,
		DiskWriteBytes: 2048,
		OpenFiles:      10,
		NumThreads:     5,
	}

	// 更新资源数据
	monitor.updateAgentResourceData("test-agent", dataPoint)

	// 验证元数据已保存
	metadata, err := multiManager.GetAgentMetadata("test-agent")
	if err != nil {
		t.Fatalf("failed to get metadata: %v", err)
	}

	if len(metadata.ResourceUsage.CPU) == 0 {
		t.Error("CPU data not added to history")
	}
	if len(metadata.ResourceUsage.Memory) == 0 {
		t.Error("Memory data not added to history")
	}
	if metadata.ResourceUsage.CPU[0] != 50.0 {
		t.Errorf("expected CPU 50.0, got %f", metadata.ResourceUsage.CPU[0])
	}
}

func TestCheckResourceThresholds_CPUExceeded(t *testing.T) {
	logger := zaptest.NewLogger(t)
	workDir := t.TempDir()
	multiManager, err := NewMultiAgentManager(workDir, logger)
	if err != nil {
		t.Fatalf("failed to create multi agent manager: %v", err)
	}

	registry := multiManager.GetRegistry()
	monitor := NewResourceMonitor(multiManager, registry, logger)

	agentID := "test-agent"
	threshold := &ResourceThreshold{
		CPUThreshold:      50.0,
		MemoryThreshold:   1024 * 1024 * 100,
		ThresholdDuration: 1 * time.Minute,
	}
	monitor.SetThreshold(agentID, threshold)

	// 创建超过CPU阈值的数据点
	dataPoint := &ResourceDataPoint{
		Timestamp: time.Now(),
		CPU:       80.0,             // 超过50.0阈值
		MemoryRSS: 1024 * 1024 * 50, // 未超过阈值
	}

	// 检查阈值(第一次超过,应该记录警告)
	monitor.checkResourceThresholds(agentID, dataPoint)

	// 验证超阈值开始时间已记录
	monitor.mu.RLock()
	exceededMap := monitor.exceededSince[agentID]
	monitor.mu.RUnlock()

	if exceededMap == nil {
		t.Error("exceededSince map not initialized")
	} else if exceededMap["cpu"].IsZero() {
		t.Error("CPU exceeded time not recorded")
	}
}

func TestCheckResourceThresholds_MemoryExceeded(t *testing.T) {
	logger := zaptest.NewLogger(t)
	workDir := t.TempDir()
	multiManager, err := NewMultiAgentManager(workDir, logger)
	if err != nil {
		t.Fatalf("failed to create multi agent manager: %v", err)
	}

	registry := multiManager.GetRegistry()
	monitor := NewResourceMonitor(multiManager, registry, logger)

	agentID := "test-agent"
	threshold := &ResourceThreshold{
		CPUThreshold:      50.0,
		MemoryThreshold:   1024 * 1024 * 100, // 100MB
		ThresholdDuration: 1 * time.Minute,
	}
	monitor.SetThreshold(agentID, threshold)

	// 创建超过内存阈值的数据点
	dataPoint := &ResourceDataPoint{
		Timestamp: time.Now(),
		CPU:       30.0,              // 未超过阈值
		MemoryRSS: 1024 * 1024 * 200, // 超过100MB阈值
	}

	// 检查阈值
	monitor.checkResourceThresholds(agentID, dataPoint)

	// 验证超阈值开始时间已记录
	monitor.mu.RLock()
	exceededMap := monitor.exceededSince[agentID]
	monitor.mu.RUnlock()

	if exceededMap == nil {
		t.Error("exceededSince map not initialized")
	} else if exceededMap["memory"].IsZero() {
		t.Error("Memory exceeded time not recorded")
	}
}

func TestCheckResourceThresholds_DurationCheck(t *testing.T) {
	logger := zaptest.NewLogger(t)
	workDir := t.TempDir()
	multiManager, err := NewMultiAgentManager(workDir, logger)
	if err != nil {
		t.Fatalf("failed to create multi agent manager: %v", err)
	}

	registry := multiManager.GetRegistry()
	monitor := NewResourceMonitor(multiManager, registry, logger)

	agentID := "test-agent"
	threshold := &ResourceThreshold{
		CPUThreshold:      50.0,
		MemoryThreshold:   1024 * 1024 * 100,
		ThresholdDuration: 1 * time.Second, // 很短的持续时间用于测试
	}
	monitor.SetThreshold(agentID, threshold)

	// 第一次超过阈值
	dataPoint1 := &ResourceDataPoint{
		Timestamp: time.Now(),
		CPU:       80.0,
		MemoryRSS: 1024 * 1024 * 50,
	}
	monitor.checkResourceThresholds(agentID, dataPoint1)

	// 等待超过阈值持续时间
	time.Sleep(2 * time.Second)

	// 再次检查(应该触发ERROR日志)
	dataPoint2 := &ResourceDataPoint{
		Timestamp: time.Now(),
		CPU:       80.0,
		MemoryRSS: 1024 * 1024 * 50,
	}
	monitor.checkResourceThresholds(agentID, dataPoint2)

	// 验证持续时间计算
	duration := monitor.getExceededDuration(agentID, "cpu", time.Now())
	if duration < 1*time.Second {
		t.Errorf("expected duration >= 1s, got %v", duration)
	}
}

func TestGetResourceHistory(t *testing.T) {
	logger := zaptest.NewLogger(t)
	workDir := t.TempDir()
	multiManager, err := NewMultiAgentManager(workDir, logger)
	if err != nil {
		t.Fatalf("failed to create multi agent manager: %v", err)
	}

	registry := multiManager.GetRegistry()
	monitor := NewResourceMonitor(multiManager, registry, logger)

	// 注册并创建Agent实例
	info, err := registry.Register(
		"test-agent",
		TypeCustom,
		"Test Agent",
		"/bin/echo",
		"",
		workDir,
		"",
	)
	if err != nil {
		t.Fatalf("failed to register agent: %v", err)
	}

	instance := NewAgentInstance(info, logger)
	multiManager.mu.Lock()
	multiManager.instances["test-agent"] = instance
	multiManager.mu.Unlock()

	// 创建元数据并添加一些历史数据
	metadata := &AgentMetadata{
		ID:            "test-agent",
		Type:          "custom",
		Status:        "running",
		StartTime:     time.Now(),
		RestartCount:  0,
		ResourceUsage: *NewResourceUsageHistory(1440),
	}

	// 添加多个数据点
	for i := 0; i < 5; i++ {
		metadata.ResourceUsage.AddResourceData(float64(10+i*10), uint64(1024*1024*(10+i*10)))
		time.Sleep(10 * time.Millisecond) // 确保时间戳不同
	}

	// 保存元数据
	metadataStore := multiManager.metadataStore
	if err := metadataStore.SaveMetadata("test-agent", metadata); err != nil {
		t.Fatalf("failed to save metadata: %v", err)
	}

	// 获取历史数据
	history, err := monitor.GetResourceHistory("test-agent", 1*time.Hour)
	if err != nil {
		t.Fatalf("failed to get resource history: %v", err)
	}

	if len(history) == 0 {
		t.Error("expected history data, got empty")
	}

	// 验证数据点
	if history[0].CPU != 10.0 {
		t.Errorf("expected first CPU 10.0, got %f", history[0].CPU)
	}
}

func TestGetResourceHistoryAggregated(t *testing.T) {
	logger := zaptest.NewLogger(t)
	workDir := t.TempDir()
	multiManager, err := NewMultiAgentManager(workDir, logger)
	if err != nil {
		t.Fatalf("failed to create multi agent manager: %v", err)
	}

	registry := multiManager.GetRegistry()
	monitor := NewResourceMonitor(multiManager, registry, logger)

	// 注册并创建Agent实例
	info, err := registry.Register(
		"test-agent",
		TypeCustom,
		"Test Agent",
		"/bin/echo",
		"",
		workDir,
		"",
	)
	if err != nil {
		t.Fatalf("failed to register agent: %v", err)
	}

	instance := NewAgentInstance(info, logger)
	multiManager.mu.Lock()
	multiManager.instances["test-agent"] = instance
	multiManager.mu.Unlock()

	// 创建元数据并添加一些历史数据
	metadata := &AgentMetadata{
		ID:            "test-agent",
		Type:          "custom",
		Status:        "running",
		StartTime:     time.Now(),
		RestartCount:  0,
		ResourceUsage: *NewResourceUsageHistory(1440),
	}

	// 添加多个数据点(在同一分钟内)
	for i := 0; i < 3; i++ {
		metadata.ResourceUsage.AddResourceData(10.0+float64(i)*10.0, uint64(1024*1024*(10+i*10)))
		time.Sleep(10 * time.Millisecond)
	}

	// 保存元数据
	metadataStore := multiManager.metadataStore
	if err := metadataStore.SaveMetadata("test-agent", metadata); err != nil {
		t.Fatalf("failed to save metadata: %v", err)
	}

	// 获取聚合历史数据(按1分钟聚合)
	aggregated, err := monitor.GetResourceHistoryAggregated("test-agent", 1*time.Hour, 1*time.Minute)
	if err != nil {
		t.Fatalf("failed to get aggregated history: %v", err)
	}

	if len(aggregated) == 0 {
		t.Error("expected aggregated data, got empty")
	}

	// 验证聚合结果(平均值应该是20.0,因为10+20+30的平均值)
	if aggregated[0].CPU < 19.0 || aggregated[0].CPU > 21.0 {
		t.Errorf("expected aggregated CPU around 20.0, got %f", aggregated[0].CPU)
	}
}

func TestMonitorLoop_StartStop(t *testing.T) {
	logger := zaptest.NewLogger(t)
	workDir := t.TempDir()
	multiManager, err := NewMultiAgentManager(workDir, logger)
	if err != nil {
		t.Fatalf("failed to create multi agent manager: %v", err)
	}

	registry := multiManager.GetRegistry()
	monitor := NewResourceMonitor(multiManager, registry, logger)

	// 设置较短的采集间隔用于测试
	monitor.SetInterval(100 * time.Millisecond)

	// 启动监控
	monitor.Start()

	// 等待一段时间
	time.Sleep(250 * time.Millisecond)

	// 停止监控
	monitor.Stop()

	// 验证监控已停止(通过检查context是否已取消)
	select {
	case <-monitor.ctx.Done():
		// 正常,context已取消
	default:
		t.Error("context not cancelled after stop")
	}
}

func TestGetExceededDuration(t *testing.T) {
	logger := zaptest.NewLogger(t)
	workDir := t.TempDir()
	multiManager, err := NewMultiAgentManager(workDir, logger)
	if err != nil {
		t.Fatalf("failed to create multi agent manager: %v", err)
	}

	registry := multiManager.GetRegistry()
	monitor := NewResourceMonitor(multiManager, registry, logger)

	agentID := "test-agent"
	resourceType := "cpu"

	// 第一次调用应该返回0(还没有记录超阈值开始时间)
	now := time.Now()
	duration := monitor.getExceededDuration(agentID, resourceType, now)
	if duration != 0 {
		t.Errorf("expected duration 0 for first call, got %v", duration)
	}

	// 等待一段时间后再次调用
	time.Sleep(100 * time.Millisecond)
	now2 := time.Now()
	duration2 := monitor.getExceededDuration(agentID, resourceType, now2)
	if duration2 < 100*time.Millisecond {
		t.Errorf("expected duration >= 100ms, got %v", duration2)
	}
}

func TestClearExceededSince(t *testing.T) {
	logger := zaptest.NewLogger(t)
	workDir := t.TempDir()
	multiManager, err := NewMultiAgentManager(workDir, logger)
	if err != nil {
		t.Fatalf("failed to create multi agent manager: %v", err)
	}

	registry := multiManager.GetRegistry()
	monitor := NewResourceMonitor(multiManager, registry, logger)

	agentID := "test-agent"
	resourceType := "cpu"

	// 先记录超阈值开始时间
	now := time.Now()
	monitor.getExceededDuration(agentID, resourceType, now)

	// 清除
	monitor.clearExceededSince(agentID, resourceType)

	// 验证已清除
	monitor.mu.RLock()
	exceededMap := monitor.exceededSince[agentID]
	monitor.mu.RUnlock()

	if exceededMap != nil && !exceededMap[resourceType].IsZero() {
		t.Error("exceeded time not cleared")
	}
}

// TestConcurrentCollection 测试并发采集多个Agent
func TestConcurrentCollection(t *testing.T) {
	logger := zaptest.NewLogger(t)
	workDir := t.TempDir()
	multiManager, err := NewMultiAgentManager(workDir, logger)
	if err != nil {
		t.Fatalf("failed to create multi agent manager: %v", err)
	}

	registry := multiManager.GetRegistry()

	// 注册多个Agent
	for i := 0; i < 3; i++ {
		agentID := "test-agent-" + string(rune('a'+i))
		info, err := registry.Register(
			agentID,
			TypeCustom,
			"Test Agent "+string(rune('a'+i)),
			"/bin/echo",
			"",
			workDir,
			"",
		)
		if err != nil {
			t.Fatalf("failed to register agent %s: %v", agentID, err)
		}

		instance := NewAgentInstance(info, logger)
		multiManager.mu.Lock()
		multiManager.instances[agentID] = instance
		multiManager.mu.Unlock()
	}

	monitor := NewResourceMonitor(multiManager, registry, logger)

	// 并发采集(collectAllAgents内部已经使用goroutine)
	// 这里主要测试不会panic
	monitor.collectAllAgents()

	// 验证没有panic即成功
}

// TestCollectAgentResources_RealProcess 测试采集真实进程资源(如果可能)
func TestCollectAgentResources_RealProcess(t *testing.T) {
	// 跳过此测试,因为需要真实的运行进程
	t.Skip("skipping test that requires real running process")
}

// TestCollectAllAgents_LargeScale 测试大规模Agent的并发采集
func TestCollectAllAgents_LargeScale(t *testing.T) {
	logger := zaptest.NewLogger(t)
	workDir := t.TempDir()
	multiManager, err := NewMultiAgentManager(workDir, logger)
	if err != nil {
		t.Fatalf("failed to create multi agent manager: %v", err)
	}

	registry := multiManager.GetRegistry()

	// 注册大量Agent
	const numAgents = 50
	for i := 0; i < numAgents; i++ {
		agentID := fmt.Sprintf("test-agent-%d", i)
		info, err := registry.Register(
			agentID,
			TypeCustom,
			fmt.Sprintf("Test Agent %d", i),
			"/bin/echo",
			"",
			workDir,
			"",
		)
		if err != nil {
			t.Fatalf("failed to register agent %s: %v", agentID, err)
		}

		instance := NewAgentInstance(info, logger)
		multiManager.mu.Lock()
		multiManager.instances[agentID] = instance
		multiManager.mu.Unlock()
	}

	monitor := NewResourceMonitor(multiManager, registry, logger)

	// 测试大规模并发采集
	startTime := time.Now()
	monitor.collectAllAgents()
	duration := time.Since(startTime)

	// 验证没有panic且完成时间合理
	if duration > 10*time.Second {
		t.Errorf("collection took too long: %v", duration)
	}

	t.Logf("collected resources for %d agents in %v", numAgents, duration)
}

// TestCheckResourceThresholds_MultipleAgents 测试多个Agent同时超过阈值
func TestCheckResourceThresholds_MultipleAgents(t *testing.T) {
	logger := zaptest.NewLogger(t)
	workDir := t.TempDir()
	multiManager, err := NewMultiAgentManager(workDir, logger)
	if err != nil {
		t.Fatalf("failed to create multi agent manager: %v", err)
	}

	registry := multiManager.GetRegistry()

	// 注册多个Agent并设置阈值
	const numAgents = 5
	agentIDs := make([]string, numAgents)
	for i := 0; i < numAgents; i++ {
		agentID := fmt.Sprintf("test-agent-%d", i)
		agentIDs[i] = agentID

		info, err := registry.Register(
			agentID,
			TypeCustom,
			fmt.Sprintf("Test Agent %d", i),
			"/bin/echo",
			"",
			workDir,
			"",
		)
		if err != nil {
			t.Fatalf("failed to register agent %s: %v", agentID, err)
		}

		instance := NewAgentInstance(info, logger)
		multiManager.mu.Lock()
		multiManager.instances[agentID] = instance
		multiManager.mu.Unlock()
	}

	monitor := NewResourceMonitor(multiManager, registry, logger)

	// 为每个Agent设置阈值
	for _, agentID := range agentIDs {
		threshold := &ResourceThreshold{
			CPUThreshold:      50.0,
			MemoryThreshold:   1024 * 1024 * 100, // 100MB
			ThresholdDuration: 1 * time.Minute,
		}
		monitor.SetThreshold(agentID, threshold)
	}

	// 模拟资源数据点（超过阈值）
	for _, agentID := range agentIDs {
		dataPoint := &ResourceDataPoint{
			Timestamp: time.Now(),
			CPU:       75.0,              // 超过50%阈值
			MemoryRSS: 150 * 1024 * 1024, // 超过100MB阈值
		}
		monitor.checkResourceThresholds(agentID, dataPoint)
	}

	// 验证所有Agent的阈值检查都执行了（不会panic）
	t.Logf("checked thresholds for %d agents", numAgents)
}

// TestGetResourceHistory_LargeHistory 测试大量历史数据的查询性能
func TestGetResourceHistory_LargeHistory(t *testing.T) {
	logger := zaptest.NewLogger(t)
	workDir := t.TempDir()
	multiManager, err := NewMultiAgentManager(workDir, logger)
	if err != nil {
		t.Fatalf("failed to create multi agent manager: %v", err)
	}

	registry := multiManager.GetRegistry()

	agentID := "test-agent"
	info, err := registry.Register(
		agentID,
		TypeCustom,
		"Test Agent",
		"/bin/echo",
		"",
		workDir,
		"",
	)
	if err != nil {
		t.Fatalf("failed to register agent: %v", err)
	}

	agentInfo := &AgentInfo{
		ID:         agentID,
		Type:       TypeCustom,
		Name:       "Test Agent",
		BinaryPath: "/bin/echo",
		WorkDir:    workDir,
	}
	_, err = multiManager.RegisterAgent(agentInfo)
	if err != nil {
		t.Fatalf("failed to register agent to manager: %v", err)
	}

	// 创建元数据并添加大量历史数据
	metadata := &AgentMetadata{
		ID:            agentID,
		Type:          string(TypeCustom),
		Status:        "running",
		StartTime:     time.Now(),
		RestartCount:  0,
		ResourceUsage: *NewResourceUsageHistory(1440),
	}

	// 添加大量数据点（模拟24小时的数据，每分钟一个点）
	for i := 0; i < 1440; i++ {
		cpu := float64(i % 100)
		memory := uint64(1024*1024*100 + i*1024)
		metadata.ResourceUsage.AddResourceData(cpu, memory)
	}

	// 保存元数据
	metadataStore := multiManager.metadataStore
	if err := metadataStore.SaveMetadata(agentID, metadata); err != nil {
		t.Fatalf("failed to save metadata: %v", err)
	}

	monitor := NewResourceMonitor(multiManager, registry, logger)

	// 测试查询性能
	startTime := time.Now()
	history, err := monitor.GetResourceHistory(agentID, 24*time.Hour)
	duration := time.Since(startTime)

	if err != nil {
		t.Fatalf("failed to get resource history: %v", err)
	}

	if len(history) == 0 {
		t.Error("expected non-empty history")
	}

	// 验证查询时间合理（应该很快）
	if duration > 1*time.Second {
		t.Errorf("query took too long: %v", duration)
	}

	t.Logf("queried %d data points in %v", len(history), duration)
}
