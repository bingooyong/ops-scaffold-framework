package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestSaveMetadata_Success(t *testing.T) {
	tmpDir := t.TempDir()
	logger := zap.NewNop()
	store, err := NewFileMetadataStore(tmpDir, logger)
	if err != nil {
		t.Fatalf("failed to create metadata store: %v", err)
	}

	metadata := &AgentMetadata{
		ID:            "test-agent",
		Type:          "filebeat",
		Version:       "7.14.0",
		Status:        "running",
		StartTime:     time.Now(),
		RestartCount:  0,
		LastHeartbeat: time.Now(),
		ResourceUsage: *NewResourceUsageHistory(1440),
	}

	err = store.SaveMetadata("test-agent", metadata)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 验证文件已创建
	metadataPath := filepath.Join(tmpDir, "metadata", "test-agent.json")
	if _, err := os.Stat(metadataPath); os.IsNotExist(err) {
		t.Fatal("metadata file was not created")
	}
}

func TestSaveMetadata_AtomicWrite(t *testing.T) {
	tmpDir := t.TempDir()
	logger := zap.NewNop()
	store, err := NewFileMetadataStore(tmpDir, logger)
	if err != nil {
		t.Fatalf("failed to create metadata store: %v", err)
	}

	metadata := &AgentMetadata{
		ID:            "test-agent",
		Type:          "filebeat",
		Status:        "running",
		StartTime:     time.Now(),
		RestartCount:  0,
		ResourceUsage: *NewResourceUsageHistory(1440),
	}

	err = store.SaveMetadata("test-agent", metadata)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 验证临时文件不存在(原子性写入后应该被删除)
	tmpPath := filepath.Join(tmpDir, "metadata", "test-agent.json.tmp")
	if _, err := os.Stat(tmpPath); !os.IsNotExist(err) {
		t.Error("temporary file should not exist after atomic write")
	}

	// 验证原文件存在
	metadataPath := filepath.Join(tmpDir, "metadata", "test-agent.json")
	if _, err := os.Stat(metadataPath); os.IsNotExist(err) {
		t.Fatal("metadata file was not created")
	}
}

func TestGetMetadata_Success(t *testing.T) {
	tmpDir := t.TempDir()
	logger := zap.NewNop()
	store, err := NewFileMetadataStore(tmpDir, logger)
	if err != nil {
		t.Fatalf("failed to create metadata store: %v", err)
	}

	expectedMetadata := &AgentMetadata{
		ID:            "test-agent",
		Type:          "filebeat",
		Version:       "7.14.0",
		Status:        "running",
		StartTime:     time.Now(),
		RestartCount:  5,
		LastHeartbeat: time.Now(),
		ResourceUsage: *NewResourceUsageHistory(1440),
	}

	err = store.SaveMetadata("test-agent", expectedMetadata)
	if err != nil {
		t.Fatalf("failed to save metadata: %v", err)
	}

	// 读取元数据
	metadata, err := store.GetMetadata("test-agent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if metadata.ID != expectedMetadata.ID {
		t.Errorf("expected ID %s, got %s", expectedMetadata.ID, metadata.ID)
	}
	if metadata.Type != expectedMetadata.Type {
		t.Errorf("expected Type %s, got %s", expectedMetadata.Type, metadata.Type)
	}
	if metadata.Version != expectedMetadata.Version {
		t.Errorf("expected Version %s, got %s", expectedMetadata.Version, metadata.Version)
	}
	if metadata.Status != expectedMetadata.Status {
		t.Errorf("expected Status %s, got %s", expectedMetadata.Status, metadata.Status)
	}
	if metadata.RestartCount != expectedMetadata.RestartCount {
		t.Errorf("expected RestartCount %d, got %d", expectedMetadata.RestartCount, metadata.RestartCount)
	}
}

func TestGetMetadata_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	logger := zap.NewNop()
	store, err := NewFileMetadataStore(tmpDir, logger)
	if err != nil {
		t.Fatalf("failed to create metadata store: %v", err)
	}

	_, err = store.GetMetadata("non-existent")
	if err == nil {
		t.Fatal("expected error for non-existent metadata")
	}
	if err != os.ErrNotExist {
		t.Errorf("expected os.ErrNotExist, got %v", err)
	}
}

func TestUpdateMetadata_Success(t *testing.T) {
	tmpDir := t.TempDir()
	logger := zap.NewNop()
	store, err := NewFileMetadataStore(tmpDir, logger)
	if err != nil {
		t.Fatalf("failed to create metadata store: %v", err)
	}

	// 创建初始元数据
	initialMetadata := &AgentMetadata{
		ID:            "test-agent",
		Type:          "filebeat",
		Status:        "stopped",
		StartTime:     time.Now(),
		RestartCount:  0,
		ResourceUsage: *NewResourceUsageHistory(1440),
	}

	err = store.SaveMetadata("test-agent", initialMetadata)
	if err != nil {
		t.Fatalf("failed to save initial metadata: %v", err)
	}

	// 更新元数据
	updates := &AgentMetadata{
		Status: "running",
	}

	err = store.UpdateMetadata("test-agent", updates)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 验证更新
	metadata, err := store.GetMetadata("test-agent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if metadata.Status != "running" {
		t.Errorf("expected Status 'running', got '%s'", metadata.Status)
	}
	// 其他字段应该保持不变
	if metadata.ID != initialMetadata.ID {
		t.Errorf("expected ID %s, got %s", initialMetadata.ID, metadata.ID)
	}
	if metadata.Type != initialMetadata.Type {
		t.Errorf("expected Type %s, got %s", initialMetadata.Type, metadata.Type)
	}
}

func TestUpdateMetadata_PartialUpdate(t *testing.T) {
	tmpDir := t.TempDir()
	logger := zap.NewNop()
	store, err := NewFileMetadataStore(tmpDir, logger)
	if err != nil {
		t.Fatalf("failed to create metadata store: %v", err)
	}

	// 创建初始元数据
	initialMetadata := &AgentMetadata{
		ID:            "test-agent",
		Type:          "filebeat",
		Version:       "7.14.0",
		Status:        "stopped",
		StartTime:     time.Now(),
		RestartCount:  3,
		ResourceUsage: *NewResourceUsageHistory(1440),
	}

	err = store.SaveMetadata("test-agent", initialMetadata)
	if err != nil {
		t.Fatalf("failed to save initial metadata: %v", err)
	}

	// 部分更新(仅更新Status和RestartCount)
	newStartTime := time.Now().Add(1 * time.Hour)
	updates := &AgentMetadata{
		Status:       "running",
		StartTime:    newStartTime,
		RestartCount: 4,
	}

	err = store.UpdateMetadata("test-agent", updates)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 验证更新
	metadata, err := store.GetMetadata("test-agent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if metadata.Status != "running" {
		t.Errorf("expected Status 'running', got '%s'", metadata.Status)
	}
	if metadata.RestartCount != 4 {
		t.Errorf("expected RestartCount 4, got %d", metadata.RestartCount)
	}
	// Version应该保持不变(未在updates中设置)
	if metadata.Version != initialMetadata.Version {
		t.Errorf("expected Version %s to remain unchanged, got %s", initialMetadata.Version, metadata.Version)
	}
}

func TestListAllMetadata(t *testing.T) {
	tmpDir := t.TempDir()
	logger := zap.NewNop()
	store, err := NewFileMetadataStore(tmpDir, logger)
	if err != nil {
		t.Fatalf("failed to create metadata store: %v", err)
	}

	// 创建多个元数据
	agents := []string{"agent-1", "agent-2", "agent-3"}
	for _, id := range agents {
		metadata := &AgentMetadata{
			ID:            id,
			Type:          "filebeat",
			Status:        "running",
			StartTime:     time.Now(),
			RestartCount:  0,
			ResourceUsage: *NewResourceUsageHistory(1440),
		}
		err := store.SaveMetadata(id, metadata)
		if err != nil {
			t.Fatalf("failed to save metadata for %s: %v", id, err)
		}
	}

	// 列举所有元数据
	allMetadata, err := store.ListAllMetadata()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(allMetadata) != len(agents) {
		t.Errorf("expected %d metadata entries, got %d", len(agents), len(allMetadata))
	}

	// 验证所有Agent都在列表中
	found := make(map[string]bool)
	for _, metadata := range allMetadata {
		found[metadata.ID] = true
	}
	for _, id := range agents {
		if !found[id] {
			t.Errorf("agent %s not found in list", id)
		}
	}
}

func TestListAllMetadata_IgnoreInvalid(t *testing.T) {
	tmpDir := t.TempDir()
	logger := zap.NewNop()
	store, err := NewFileMetadataStore(tmpDir, logger)
	if err != nil {
		t.Fatalf("failed to create metadata store: %v", err)
	}

	// 创建有效的元数据
	validMetadata := &AgentMetadata{
		ID:            "valid-agent",
		Type:          "filebeat",
		Status:        "running",
		StartTime:     time.Now(),
		RestartCount:  0,
		ResourceUsage: *NewResourceUsageHistory(1440),
	}
	err = store.SaveMetadata("valid-agent", validMetadata)
	if err != nil {
		t.Fatalf("failed to save valid metadata: %v", err)
	}

	// 创建无效的JSON文件
	invalidPath := filepath.Join(tmpDir, "metadata", "invalid-agent.json")
	err = os.WriteFile(invalidPath, []byte("invalid json content"), 0644)
	if err != nil {
		t.Fatalf("failed to create invalid file: %v", err)
	}

	// 列举所有元数据(应该忽略无效文件)
	allMetadata, err := store.ListAllMetadata()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 应该只返回有效的元数据
	if len(allMetadata) != 1 {
		t.Errorf("expected 1 metadata entry, got %d", len(allMetadata))
	}
	if allMetadata[0].ID != "valid-agent" {
		t.Errorf("expected valid-agent, got %s", allMetadata[0].ID)
	}
}

func TestDeleteMetadata(t *testing.T) {
	tmpDir := t.TempDir()
	logger := zap.NewNop()
	store, err := NewFileMetadataStore(tmpDir, logger)
	if err != nil {
		t.Fatalf("failed to create metadata store: %v", err)
	}

	// 创建元数据
	metadata := &AgentMetadata{
		ID:            "test-agent",
		Type:          "filebeat",
		Status:        "running",
		StartTime:     time.Now(),
		RestartCount:  0,
		ResourceUsage: *NewResourceUsageHistory(1440),
	}

	err = store.SaveMetadata("test-agent", metadata)
	if err != nil {
		t.Fatalf("failed to save metadata: %v", err)
	}

	// 删除元数据
	err = store.DeleteMetadata("test-agent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 验证文件已删除
	metadataPath := filepath.Join(tmpDir, "metadata", "test-agent.json")
	if _, err := os.Stat(metadataPath); !os.IsNotExist(err) {
		t.Error("metadata file should be deleted")
	}

	// 验证GetMetadata返回错误
	_, err = store.GetMetadata("test-agent")
	if err == nil {
		t.Fatal("expected error for deleted metadata")
	}
}

func TestDeleteMetadata_NotExists(t *testing.T) {
	tmpDir := t.TempDir()
	logger := zap.NewNop()
	store, err := NewFileMetadataStore(tmpDir, logger)
	if err != nil {
		t.Fatalf("failed to create metadata store: %v", err)
	}

	// 删除不存在的元数据(应该成功,幂等操作)
	err = store.DeleteMetadata("non-existent")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestConcurrentAccess(t *testing.T) {
	tmpDir := t.TempDir()
	logger := zap.NewNop()
	store, err := NewFileMetadataStore(tmpDir, logger)
	if err != nil {
		t.Fatalf("failed to create metadata store: %v", err)
	}

	// 并发写入
	var wg sync.WaitGroup
	numGoroutines := 10
	numWrites := 10

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numWrites; j++ {
				metadata := &AgentMetadata{
					ID:            "test-agent",
					Type:          "filebeat",
					Status:        "running",
					StartTime:     time.Now(),
					RestartCount:  id*numWrites + j,
					ResourceUsage: *NewResourceUsageHistory(1440),
				}
				err := store.SaveMetadata("test-agent", metadata)
				if err != nil {
					t.Errorf("goroutine %d write %d failed: %v", id, j, err)
				}
			}
		}(i)
	}

	wg.Wait()

	// 验证最终状态
	metadata, err := store.GetMetadata("test-agent")
	if err != nil {
		t.Fatalf("failed to get metadata: %v", err)
	}

	if metadata.ID != "test-agent" {
		t.Errorf("expected ID 'test-agent', got '%s'", metadata.ID)
	}
}

func TestResourceUsageHistory_AddData(t *testing.T) {
	history := NewResourceUsageHistory(10)

	// 添加数据点
	for i := 0; i < 5; i++ {
		history.AddResourceData(float64(i*10), uint64(i*1000))
	}

	if len(history.CPU) != 5 {
		t.Errorf("expected 5 CPU data points, got %d", len(history.CPU))
	}
	if len(history.Memory) != 5 {
		t.Errorf("expected 5 Memory data points, got %d", len(history.Memory))
	}
	if len(history.Timestamps) != 5 {
		t.Errorf("expected 5 Timestamps, got %d", len(history.Timestamps))
	}

	// 验证数据
	for i := 0; i < 5; i++ {
		if history.CPU[i] != float64(i*10) {
			t.Errorf("expected CPU[%d] = %f, got %f", i, float64(i*10), history.CPU[i])
		}
		if history.Memory[i] != uint64(i*1000) {
			t.Errorf("expected Memory[%d] = %d, got %d", i, uint64(i*1000), history.Memory[i])
		}
	}
}

func TestResourceUsageHistory_MaxSize(t *testing.T) {
	maxSize := 5
	history := NewResourceUsageHistory(maxSize)

	// 添加超过最大大小的数据点
	for i := 0; i < 10; i++ {
		history.AddResourceData(float64(i), uint64(i))
	}

	// 应该只保留最后maxSize个数据点
	if len(history.CPU) != maxSize {
		t.Errorf("expected %d CPU data points, got %d", maxSize, len(history.CPU))
	}
	if len(history.Memory) != maxSize {
		t.Errorf("expected %d Memory data points, got %d", maxSize, len(history.Memory))
	}

	// 验证保留的是最后的数据点
	expectedStart := 5
	for i := 0; i < maxSize; i++ {
		expectedValue := expectedStart + i
		if history.CPU[i] != float64(expectedValue) {
			t.Errorf("expected CPU[%d] = %f, got %f", i, float64(expectedValue), history.CPU[i])
		}
		if history.Memory[i] != uint64(expectedValue) {
			t.Errorf("expected Memory[%d] = %d, got %d", i, uint64(expectedValue), history.Memory[i])
		}
	}
}

func TestResourceUsageHistory_GetRecent(t *testing.T) {
	history := NewResourceUsageHistory(100)

	now := time.Now()
	// 添加不同时间的数据点
	for i := 0; i < 10; i++ {
		// 手动设置时间戳(因为AddResourceData使用当前时间)
		history.mu.Lock()
		history.CPU = append(history.CPU, float64(i))
		history.Memory = append(history.Memory, uint64(i*1000))
		history.Timestamps = append(history.Timestamps, now.Add(time.Duration(i-9)*time.Minute))
		history.mu.Unlock()
	}

	// 获取最近5分钟的数据
	recentCPU := history.GetRecentCPU(5 * time.Minute)
	recentMemory := history.GetRecentMemory(5 * time.Minute)

	// 应该返回最近5分钟内的数据点(大约5个)
	if len(recentCPU) < 5 {
		t.Errorf("expected at least 5 recent CPU data points, got %d", len(recentCPU))
	}
	if len(recentMemory) < 5 {
		t.Errorf("expected at least 5 recent Memory data points, got %d", len(recentMemory))
	}

	// 验证数据一致性
	if len(recentCPU) != len(recentMemory) {
		t.Errorf("CPU and Memory should have same length, got %d and %d", len(recentCPU), len(recentMemory))
	}
}

func TestResourceUsageHistory_GetRecent_Empty(t *testing.T) {
	history := NewResourceUsageHistory(100)

	recentCPU := history.GetRecentCPU(5 * time.Minute)
	recentMemory := history.GetRecentMemory(5 * time.Minute)

	if len(recentCPU) != 0 {
		t.Errorf("expected empty CPU data, got %d", len(recentCPU))
	}
	if len(recentMemory) != 0 {
		t.Errorf("expected empty Memory data, got %d", len(recentMemory))
	}
}

// TestSaveMetadata_ConcurrentSave 测试并发保存元数据
func TestSaveMetadata_ConcurrentSave(t *testing.T) {
	tmpDir := t.TempDir()
	logger := zap.NewNop()
	store, err := NewFileMetadataStore(tmpDir, logger)
	if err != nil {
		t.Fatalf("failed to create metadata store: %v", err)
	}

	agentID := "test-concurrent-agent"
	const numGoroutines = 10
	done := make(chan bool, numGoroutines)
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			metadata := &AgentMetadata{
				ID:            agentID,
				Type:          "filebeat",
				Version:       fmt.Sprintf("1.0.%d", id),
				Status:        "running",
				StartTime:     time.Now(),
				RestartCount:  id,
				ResourceUsage: *NewResourceUsageHistory(1440),
			}
			err := store.SaveMetadata(agentID, metadata)
			if err != nil {
				errors <- err
				return
			}
			done <- true
		}(i)
	}

	// 等待所有goroutine完成
	successCount := 0
	for i := 0; i < numGoroutines; i++ {
		select {
		case <-done:
			successCount++
		case err := <-errors:
			t.Logf("concurrent save error (may be expected): %v", err)
		case <-time.After(5 * time.Second):
			t.Fatal("concurrent save timeout")
		}
	}

	// 验证最终元数据存在且可读
	finalMetadata, err := store.GetMetadata(agentID)
	if err != nil {
		t.Fatalf("failed to get final metadata: %v", err)
	}
	if finalMetadata.ID != agentID {
		t.Errorf("expected metadata ID %s, got %s", agentID, finalMetadata.ID)
	}

	t.Logf("concurrent saves completed: %d successful", successCount)
}

// TestListAllMetadata_LargeDataset 测试大量Agent的元数据列举
func TestListAllMetadata_LargeDataset(t *testing.T) {
	tmpDir := t.TempDir()
	logger := zap.NewNop()
	store, err := NewFileMetadataStore(tmpDir, logger)
	if err != nil {
		t.Fatalf("failed to create metadata store: %v", err)
	}

	// 创建大量Agent元数据
	const numAgents = 100
	for i := 0; i < numAgents; i++ {
		agentID := fmt.Sprintf("agent-%d", i)
		metadata := &AgentMetadata{
			ID:            agentID,
			Type:          "filebeat",
			Status:        "running",
			StartTime:     time.Now(),
			RestartCount:  0,
			ResourceUsage: *NewResourceUsageHistory(1440),
		}
		if err := store.SaveMetadata(agentID, metadata); err != nil {
			t.Fatalf("failed to save metadata for %s: %v", agentID, err)
		}
	}

	// 列举所有元数据
	allMetadata, err := store.ListAllMetadata()
	if err != nil {
		t.Fatalf("failed to list all metadata: %v", err)
	}

	if len(allMetadata) != numAgents {
		t.Errorf("expected %d metadata entries, got %d", numAgents, len(allMetadata))
	}

	// 验证每个元数据都正确
	for i := 0; i < numAgents; i++ {
		agentID := fmt.Sprintf("agent-%d", i)
		found := false
		for _, metadata := range allMetadata {
			if metadata.ID == agentID {
				found = true
				if metadata.Type != "filebeat" {
					t.Errorf("expected type 'filebeat' for %s, got %s", agentID, metadata.Type)
				}
				break
			}
		}
		if !found {
			t.Errorf("metadata for %s not found in list", agentID)
		}
	}
}

// TestUpdateMetadata_RaceCondition 测试并发更新元数据的竞态条件
func TestUpdateMetadata_RaceCondition(t *testing.T) {
	tmpDir := t.TempDir()
	logger := zap.NewNop()
	store, err := NewFileMetadataStore(tmpDir, logger)
	if err != nil {
		t.Fatalf("failed to create metadata store: %v", err)
	}

	agentID := "test-race-agent"
	// 创建初始元数据
	initialMetadata := &AgentMetadata{
		ID:            agentID,
		Type:          "filebeat",
		Status:        "running",
		StartTime:     time.Now(),
		RestartCount:  0,
		ResourceUsage: *NewResourceUsageHistory(1440),
	}
	if err := store.SaveMetadata(agentID, initialMetadata); err != nil {
		t.Fatalf("failed to save initial metadata: %v", err)
	}

	// 并发更新元数据
	const numGoroutines = 10
	done := make(chan bool, numGoroutines)
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			updates := &AgentMetadata{
				RestartCount: id,
				Status:       "running",
			}
			err := store.UpdateMetadata(agentID, updates)
			if err != nil {
				errors <- err
				return
			}
			done <- true
		}(i)
	}

	// 等待所有goroutine完成
	successCount := 0
	for i := 0; i < numGoroutines; i++ {
		select {
		case <-done:
			successCount++
		case err := <-errors:
			t.Logf("concurrent update error (may be expected): %v", err)
		case <-time.After(5 * time.Second):
			t.Fatal("concurrent update timeout")
		}
	}

	// 验证最终元数据存在且可读
	finalMetadata, err := store.GetMetadata(agentID)
	if err != nil {
		t.Fatalf("failed to get final metadata: %v", err)
	}
	if finalMetadata.ID != agentID {
		t.Errorf("expected metadata ID %s, got %s", agentID, finalMetadata.ID)
	}

	t.Logf("concurrent updates completed: %d successful, final restart count: %d", successCount, finalMetadata.RestartCount)
}
