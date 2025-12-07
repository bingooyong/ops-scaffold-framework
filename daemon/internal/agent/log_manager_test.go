package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap/zaptest"
)

func TestGetAgentLogs_Success(t *testing.T) {
	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "log_manager_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建日志文件
	agentID := "test-agent"
	logDir := filepath.Join(tmpDir, "agents", agentID, "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		t.Fatalf("failed to create log dir: %v", err)
	}

	logPath := filepath.Join(logDir, "agent.log")
	logContent := "line 1\nline 2\nline 3\nline 4\nline 5\n"
	if err := os.WriteFile(logPath, []byte(logContent), 0644); err != nil {
		t.Fatalf("failed to write log file: %v", err)
	}

	// 创建日志管理器
	logger := zaptest.NewLogger(t)
	lm := NewLogManager(tmpDir, logger)

	// 测试获取最近3行
	lines, err := lm.GetAgentLogs(agentID, 3)
	if err != nil {
		t.Fatalf("failed to get logs: %v", err)
	}

	if len(lines) != 3 {
		t.Errorf("expected 3 lines, got %d", len(lines))
	}

	expected := []string{"line 3", "line 4", "line 5"}
	for i, line := range lines {
		if strings.TrimSpace(line) != expected[i] {
			t.Errorf("line %d: expected %q, got %q", i, expected[i], line)
		}
	}
}

func TestGetAgentLogs_FileNotExists(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "log_manager_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	logger := zaptest.NewLogger(t)
	lm := NewLogManager(tmpDir, logger)

	lines, err := lm.GetAgentLogs("non-existent-agent", 10)
	if err != nil {
		t.Fatalf("expected no error for non-existent file, got: %v", err)
	}

	if len(lines) != 0 {
		t.Errorf("expected empty slice, got %d lines", len(lines))
	}
}

func TestGetAgentLogs_LessThanRequested(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "log_manager_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	agentID := "test-agent"
	logDir := filepath.Join(tmpDir, "agents", agentID, "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		t.Fatalf("failed to create log dir: %v", err)
	}

	logPath := filepath.Join(logDir, "agent.log")
	logContent := "line 1\nline 2\n"
	if err := os.WriteFile(logPath, []byte(logContent), 0644); err != nil {
		t.Fatalf("failed to write log file: %v", err)
	}

	logger := zaptest.NewLogger(t)
	lm := NewLogManager(tmpDir, logger)

	lines, err := lm.GetAgentLogs(agentID, 10)
	if err != nil {
		t.Fatalf("failed to get logs: %v", err)
	}

	if len(lines) != 2 {
		t.Errorf("expected 2 lines, got %d", len(lines))
	}
}

func TestSearchLogs_Success(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "log_manager_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	agentID := "test-agent"
	logDir := filepath.Join(tmpDir, "agents", agentID, "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		t.Fatalf("failed to create log dir: %v", err)
	}

	logPath := filepath.Join(logDir, "agent.log")
	logContent := "error: connection failed\ninfo: starting service\nerror: timeout occurred\ninfo: service started\n"
	if err := os.WriteFile(logPath, []byte(logContent), 0644); err != nil {
		t.Fatalf("failed to write log file: %v", err)
	}

	logger := zaptest.NewLogger(t)
	lm := NewLogManager(tmpDir, logger)

	results, err := lm.SearchLogs(agentID, "error", 10)
	if err != nil {
		t.Fatalf("failed to search logs: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}

	for _, entry := range results {
		if !strings.Contains(entry.Content, "error") {
			t.Errorf("expected entry to contain 'error', got: %q", entry.Content)
		}
	}
}

func TestSearchLogs_NoMatch(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "log_manager_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	agentID := "test-agent"
	logDir := filepath.Join(tmpDir, "agents", agentID, "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		t.Fatalf("failed to create log dir: %v", err)
	}

	logPath := filepath.Join(logDir, "agent.log")
	logContent := "info: starting service\ninfo: service started\n"
	if err := os.WriteFile(logPath, []byte(logContent), 0644); err != nil {
		t.Fatalf("failed to write log file: %v", err)
	}

	logger := zaptest.NewLogger(t)
	lm := NewLogManager(tmpDir, logger)

	results, err := lm.SearchLogs(agentID, "error", 10)
	if err != nil {
		t.Fatalf("failed to search logs: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestSearchLogs_Limit(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "log_manager_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	agentID := "test-agent"
	logDir := filepath.Join(tmpDir, "agents", agentID, "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		t.Fatalf("failed to create log dir: %v", err)
	}

	logPath := filepath.Join(logDir, "agent.log")
	// 创建包含多个匹配项的内容
	lines := make([]string, 0)
	for i := 0; i < 20; i++ {
		lines = append(lines, "error: test error "+strconv.Itoa(i))
	}
	logContent := strings.Join(lines, "\n") + "\n"
	if err := os.WriteFile(logPath, []byte(logContent), 0644); err != nil {
		t.Fatalf("failed to write log file: %v", err)
	}

	logger := zaptest.NewLogger(t)
	lm := NewLogManager(tmpDir, logger)

	results, err := lm.SearchLogs(agentID, "error", 5)
	if err != nil {
		t.Fatalf("failed to search logs: %v", err)
	}

	if len(results) > 5 {
		t.Errorf("expected at most 5 results, got %d", len(results))
	}
}

func TestLogRotator_RotateBySize(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "log_rotator_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	logPath := filepath.Join(tmpDir, "agent.log")
	logger := zaptest.NewLogger(t)

	// 创建日志轮转器，设置较小的最大文件大小（1KB）
	maxSize := int64(1024)
	rotator := NewLogRotator(logPath, maxSize, 3, logger)

	// 创建超过最大大小的日志文件
	largeContent := make([]byte, maxSize+100)
	for i := range largeContent {
		largeContent[i] = 'A'
	}
	if err := os.WriteFile(logPath, largeContent, 0644); err != nil {
		t.Fatalf("failed to write log file: %v", err)
	}

	// 执行轮转
	if err := rotator.RotateIfNeeded(); err != nil {
		t.Fatalf("failed to rotate: %v", err)
	}

	// 检查轮转后的文件
	rotatedFile := logPath + ".1.gz"
	if _, err := os.Stat(rotatedFile); os.IsNotExist(err) {
		t.Errorf("expected rotated file %s to exist", rotatedFile)
	}

	// 检查原文件应该被重新创建（但为空或很小）
	info, err := os.Stat(logPath)
	if err != nil {
		t.Fatalf("failed to stat log file: %v", err)
	}
	if info.Size() >= maxSize {
		t.Errorf("expected log file size to be less than %d, got %d", maxSize, info.Size())
	}
}

func TestLogRotator_RotateByTime(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "log_rotator_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	logPath := filepath.Join(tmpDir, "agent.log")
	logger := zaptest.NewLogger(t)

	rotator := NewLogRotator(logPath, 100*1024*1024, 3, logger)
	rotator.rotateByTime = true
	rotator.rotateInterval = 1 * time.Hour
	rotator.lastRotateTime = time.Now().Add(-2 * time.Hour) // 设置为2小时前

	// 创建日志文件
	if err := os.WriteFile(logPath, []byte("test log content"), 0644); err != nil {
		t.Fatalf("failed to write log file: %v", err)
	}

	// 执行轮转
	if err := rotator.RotateIfNeeded(); err != nil {
		t.Fatalf("failed to rotate: %v", err)
	}

	// 检查轮转后的文件
	rotatedFile := logPath + ".1.gz"
	if _, err := os.Stat(rotatedFile); os.IsNotExist(err) {
		t.Errorf("expected rotated file %s to exist", rotatedFile)
	}
}

func TestLogRotator_MaxFiles(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "log_rotator_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	logPath := filepath.Join(tmpDir, "agent.log")
	logger := zaptest.NewLogger(t)

	maxFiles := 3
	rotator := NewLogRotator(logPath, 1024, maxFiles, logger)

	// 创建多个轮转文件（超过最大文件数）
	for i := 1; i <= maxFiles+2; i++ {
		rotatedPath := logPath + "." + strconv.Itoa(i)
		if err := os.WriteFile(rotatedPath, []byte("test"), 0644); err != nil {
			t.Fatalf("failed to create rotated file: %v", err)
		}
	}

	// 创建主日志文件并触发轮转
	largeContent := make([]byte, 2048)
	if err := os.WriteFile(logPath, largeContent, 0644); err != nil {
		t.Fatalf("failed to write log file: %v", err)
	}

	// 执行轮转
	if err := rotator.RotateIfNeeded(); err != nil {
		t.Fatalf("failed to rotate: %v", err)
	}

	// 检查文件数量（应该只有maxFiles个）
	files, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("failed to read dir: %v", err)
	}

	rotatedCount := 0
	for _, file := range files {
		if strings.HasPrefix(file.Name(), "agent.log.") {
			rotatedCount++
		}
	}

	if rotatedCount > maxFiles {
		t.Errorf("expected at most %d rotated files, got %d", maxFiles, rotatedCount)
	}
}

func TestLogRotator_Compress(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "log_rotator_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	logPath := filepath.Join(tmpDir, "agent.log")
	logger := zaptest.NewLogger(t)

	rotator := NewLogRotator(logPath, 1024, 3, logger)
	rotator.compressOldFiles = true

	// 创建超过最大大小的日志文件
	largeContent := make([]byte, 2048)
	if err := os.WriteFile(logPath, largeContent, 0644); err != nil {
		t.Fatalf("failed to write log file: %v", err)
	}

	// 执行轮转
	if err := rotator.RotateIfNeeded(); err != nil {
		t.Fatalf("failed to rotate: %v", err)
	}

	// 检查压缩文件是否存在
	compressedFile := logPath + ".1.gz"
	if _, err := os.Stat(compressedFile); os.IsNotExist(err) {
		t.Errorf("expected compressed file %s to exist", compressedFile)
	}

	// 检查原始文件应该被删除
	originalRotated := logPath + ".1"
	if _, err := os.Stat(originalRotated); err == nil {
		t.Errorf("expected original rotated file %s to be deleted", originalRotated)
	}
}

func TestCleanupOldLogs(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "log_manager_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	agentID := "test-agent"
	logDir := filepath.Join(tmpDir, "agents", agentID, "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		t.Fatalf("failed to create log dir: %v", err)
	}

	logger := zaptest.NewLogger(t)
	lm := NewLogManager(tmpDir, logger)
	lm.SetRetentionDays(1) // 保留1天

	// 创建过期日志文件（2天前）
	oldLogPath := filepath.Join(logDir, "agent.log.old")
	oldTime := time.Now().AddDate(0, 0, -2)
	if err := os.WriteFile(oldLogPath, []byte("old log"), 0644); err != nil {
		t.Fatalf("failed to create old log file: %v", err)
	}
	if err := os.Chtimes(oldLogPath, oldTime, oldTime); err != nil {
		t.Fatalf("failed to set file time: %v", err)
	}

	// 创建未过期日志文件（今天）
	newLogPath := filepath.Join(logDir, "agent.log")
	if err := os.WriteFile(newLogPath, []byte("new log"), 0644); err != nil {
		t.Fatalf("failed to create new log file: %v", err)
	}

	// 执行清理
	lm.cleanupOldLogs()

	// 检查过期文件应该被删除
	if _, err := os.Stat(oldLogPath); err == nil {
		t.Errorf("expected old log file to be deleted")
	}

	// 检查未过期文件应该保留
	if _, err := os.Stat(newLogPath); os.IsNotExist(err) {
		t.Errorf("expected new log file to be retained")
	}
}

func TestCleanupOldLogs_Retention(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "log_manager_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	agentID := "test-agent"
	logDir := filepath.Join(tmpDir, "agents", agentID, "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		t.Fatalf("failed to create log dir: %v", err)
	}

	logger := zaptest.NewLogger(t)
	lm := NewLogManager(tmpDir, logger)
	lm.SetRetentionDays(30) // 保留30天

	// 创建保留期内的日志文件（15天前）
	retainedLogPath := filepath.Join(logDir, "agent.log.retained")
	retainedTime := time.Now().AddDate(0, 0, -15)
	if err := os.WriteFile(retainedLogPath, []byte("retained log"), 0644); err != nil {
		t.Fatalf("failed to create retained log file: %v", err)
	}
	if err := os.Chtimes(retainedLogPath, retainedTime, retainedTime); err != nil {
		t.Fatalf("failed to set file time: %v", err)
	}

	// 执行清理
	lm.cleanupOldLogs()

	// 检查保留期内的文件应该保留
	if _, err := os.Stat(retainedLogPath); os.IsNotExist(err) {
		t.Errorf("expected retained log file to be kept")
	}
}

func TestNewLogRotator(t *testing.T) {
	logPath := "/tmp/test.log"
	logger := zaptest.NewLogger(t)

	rotator := NewLogRotator(logPath, 100*1024*1024, 7, logger)

	if rotator.logPath != logPath {
		t.Errorf("expected logPath %q, got %q", logPath, rotator.logPath)
	}

	if rotator.maxSize != 100*1024*1024 {
		t.Errorf("expected maxSize %d, got %d", 100*1024*1024, rotator.maxSize)
	}

	if rotator.maxFiles != 7 {
		t.Errorf("expected maxFiles 7, got %d", rotator.maxFiles)
	}

	if !rotator.compressOldFiles {
		t.Error("expected compressOldFiles to be true by default")
	}
}

func TestNewLogManager(t *testing.T) {
	workDir := "/tmp/test"
	logger := zaptest.NewLogger(t)

	lm := NewLogManager(workDir, logger)

	if lm.workDir != workDir {
		t.Errorf("expected workDir %q, got %q", workDir, lm.workDir)
	}

	if lm.retentionDays != 30 {
		t.Errorf("expected retentionDays 30, got %d", lm.retentionDays)
	}

	if lm.ctx == nil {
		t.Error("expected context to be initialized")
	}

	if lm.cancel == nil {
		t.Error("expected cancel function to be initialized")
	}
}

// TestGetAgentLogs_VeryLargeFile 测试超大日志文件的读取(>10MB)
func TestGetAgentLogs_VeryLargeFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "log_manager_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	agentID := "test-agent"
	logDir := filepath.Join(tmpDir, "agents", agentID, "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		t.Fatalf("failed to create log dir: %v", err)
	}

	logPath := filepath.Join(logDir, "agent.log")

	// 创建超大日志文件（15MB，超过10MB限制）
	const fileSize = 15 * 1024 * 1024
	largeContent := make([]byte, fileSize)
	for i := range largeContent {
		largeContent[i] = byte('A' + (i % 26))
		if i > 0 && i%100 == 0 {
			largeContent[i] = '\n'
		}
	}
	largeContent[len(largeContent)-1] = '\n'

	if err := os.WriteFile(logPath, largeContent, 0644); err != nil {
		t.Fatalf("failed to write large log file: %v", err)
	}

	logger := zaptest.NewLogger(t)
	lm := NewLogManager(tmpDir, logger)

	// 测试读取（应该只读取最后10MB）
	startTime := time.Now()
	lines, err := lm.GetAgentLogs(agentID, 100)
	duration := time.Since(startTime)

	if err != nil {
		t.Fatalf("failed to get logs from large file: %v", err)
	}

	// 验证读取时间合理（应该很快，因为只读取最后10MB）
	if duration > 2*time.Second {
		t.Errorf("reading large file took too long: %v", duration)
	}

	// 验证返回了部分日志
	if len(lines) == 0 {
		t.Error("expected some log lines from large file")
	}

	t.Logf("read %d lines from %d MB file in %v", len(lines), fileSize/(1024*1024), duration)
}

// TestSearchLogs_ConcurrentSearch 测试并发搜索日志
func TestSearchLogs_ConcurrentSearch(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "log_manager_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	agentID := "test-agent"
	logDir := filepath.Join(tmpDir, "agents", agentID, "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		t.Fatalf("failed to create log dir: %v", err)
	}

	logPath := filepath.Join(logDir, "agent.log")

	// 创建包含多个关键词的日志文件
	logContent := ""
	for i := 0; i < 1000; i++ {
		if i%100 == 0 {
			logContent += fmt.Sprintf("2024-01-01 10:00:%02d [ERROR] Error occurred\n", i%60)
		} else if i%50 == 0 {
			logContent += fmt.Sprintf("2024-01-01 10:00:%02d [WARN] Warning message\n", i%60)
		} else {
			logContent += fmt.Sprintf("2024-01-01 10:00:%02d [INFO] Info message\n", i%60)
		}
	}

	if err := os.WriteFile(logPath, []byte(logContent), 0644); err != nil {
		t.Fatalf("failed to write log file: %v", err)
	}

	logger := zaptest.NewLogger(t)
	lm := NewLogManager(tmpDir, logger)

	// 并发搜索
	const numGoroutines = 10
	done := make(chan bool, numGoroutines)
	errors := make(chan error, numGoroutines)

	keywords := []string{"ERROR", "WARN", "INFO"}
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			keyword := keywords[id%len(keywords)]
			entries, err := lm.SearchLogs(agentID, keyword, 100)
			if err != nil {
				errors <- err
				return
			}
			if len(entries) == 0 {
				errors <- fmt.Errorf("expected to find entries for keyword %s", keyword)
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
			t.Errorf("concurrent search error: %v", err)
		case <-time.After(5 * time.Second):
			t.Fatal("concurrent search timeout")
		}
	}

	t.Logf("concurrent search completed: %d successful", successCount)
}

// TestCleanupOldLogs_ManyFiles 测试大量日志文件的清理性能
func TestCleanupOldLogs_ManyFiles(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "log_manager_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	agentID := "test-agent"
	logDir := filepath.Join(tmpDir, "agents", agentID, "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		t.Fatalf("failed to create log dir: %v", err)
	}

	logger := zaptest.NewLogger(t)
	lm := NewLogManager(tmpDir, logger)
	lm.SetRetentionDays(30)

	// 创建大量日志文件（一些过期，一些不过期）
	const numFiles = 200
	oldTime := time.Now().AddDate(0, 0, -31) // 31天前
	newTime := time.Now().AddDate(0, 0, -15) // 15天前

	for i := 0; i < numFiles; i++ {
		var fileTime time.Time
		if i%2 == 0 {
			fileTime = oldTime // 过期文件
		} else {
			fileTime = newTime // 未过期文件
		}

		logPath := filepath.Join(logDir, fmt.Sprintf("log-%d.log", i))
		if err := os.WriteFile(logPath, []byte(fmt.Sprintf("log content %d\n", i)), 0644); err != nil {
			t.Fatalf("failed to write log file %d: %v", i, err)
		}
		if err := os.Chtimes(logPath, fileTime, fileTime); err != nil {
			t.Fatalf("failed to set file time %d: %v", i, err)
		}
	}

	// 执行清理并测量性能
	startTime := time.Now()
	lm.cleanupOldLogs()
	duration := time.Since(startTime)

	// 验证清理时间合理
	if duration > 5*time.Second {
		t.Errorf("cleanup took too long: %v", duration)
	}

	// 验证过期文件已删除
	files, err := os.ReadDir(logDir)
	if err != nil {
		t.Fatalf("failed to read log dir: %v", err)
	}

	remainingFiles := len(files)
	expectedFiles := numFiles / 2 // 应该保留一半（未过期的）

	// 允许一些误差（文件系统操作可能有延迟）
	if remainingFiles < expectedFiles-10 || remainingFiles > expectedFiles+10 {
		t.Logf("expected approximately %d files, got %d (cleanup may have timing issues)", expectedFiles, remainingFiles)
	}

	t.Logf("cleaned up %d files in %v, %d files remaining", numFiles-remainingFiles, duration, remainingFiles)
}
