package agent

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

// LogRotator 日志轮转器
type LogRotator struct {
	logPath          string
	maxSize          int64
	maxFiles         int
	rotateByTime     bool
	rotateInterval   time.Duration
	compressOldFiles bool
	lastRotateTime   time.Time
	mu               sync.Mutex
	logger           *zap.Logger
}

// NewLogRotator 创建新的日志轮转器
func NewLogRotator(logPath string, maxSize int64, maxFiles int, logger *zap.Logger) *LogRotator {
	return &LogRotator{
		logPath:          logPath,
		maxSize:          maxSize,
		maxFiles:         maxFiles,
		rotateByTime:     false,
		rotateInterval:   24 * time.Hour,
		compressOldFiles: true,
		lastRotateTime:   time.Now(),
		logger:           logger,
	}
}

// RotateIfNeeded 检查是否需要轮转，如果需要则执行轮转
func (lr *LogRotator) RotateIfNeeded() error {
	lr.mu.Lock()
	defer lr.mu.Unlock()

	// 检查文件是否存在
	info, err := os.Stat(lr.logPath)
	if err != nil {
		if os.IsNotExist(err) {
			// 文件不存在，不需要轮转
			return nil
		}
		return fmt.Errorf("failed to stat log file: %w", err)
	}

	// 检查文件大小
	shouldRotate := false
	if info.Size() >= lr.maxSize {
		shouldRotate = true
		lr.logger.Debug("log rotation triggered by size",
			zap.String("log_path", lr.logPath),
			zap.Int64("size", info.Size()),
			zap.Int64("max_size", lr.maxSize))
	}

	// 检查时间间隔
	if lr.rotateByTime {
		if time.Since(lr.lastRotateTime) >= lr.rotateInterval {
			shouldRotate = true
			lr.logger.Debug("log rotation triggered by time",
				zap.String("log_path", lr.logPath),
				zap.Duration("interval", time.Since(lr.lastRotateTime)))
		}
	}

	if !shouldRotate {
		return nil
	}

	// 执行轮转
	return lr.rotate()
}

// rotate 执行日志轮转
func (lr *LogRotator) rotate() error {
	// 重命名现有文件
	rotatedFiles := make([]string, 0)
	for i := lr.maxFiles - 1; i >= 1; i-- {
		oldPath := fmt.Sprintf("%s.%d", lr.logPath, i)
		newPath := fmt.Sprintf("%s.%d", lr.logPath, i+1)

		// 检查压缩文件
		oldPathGz := oldPath + ".gz"
		newPathGz := newPath + ".gz"

		// 移动压缩文件
		if _, err := os.Stat(oldPathGz); err == nil {
			if err := os.Rename(oldPathGz, newPathGz); err != nil && !os.IsNotExist(err) {
				lr.logger.Warn("failed to rename compressed log file",
					zap.String("old", oldPathGz),
					zap.String("new", newPathGz),
					zap.Error(err))
			}
		}

		// 移动未压缩文件
		if _, err := os.Stat(oldPath); err == nil {
			if err := os.Rename(oldPath, newPath); err != nil && !os.IsNotExist(err) {
				lr.logger.Warn("failed to rename log file",
					zap.String("old", oldPath),
					zap.String("new", newPath),
					zap.Error(err))
			} else {
				rotatedFiles = append(rotatedFiles, newPath)
			}
		}
	}

	// 重命名当前日志文件为 .1
	if err := os.Rename(lr.logPath, fmt.Sprintf("%s.1", lr.logPath)); err != nil {
		return fmt.Errorf("failed to rename current log file: %w", err)
	}

	// 压缩旧文件
	if lr.compressOldFiles {
		for _, filePath := range rotatedFiles {
			// 只压缩 .1 文件，其他文件应该已经被压缩
			if strings.HasSuffix(filePath, ".1") {
				if err := lr.compressFile(filePath); err != nil {
					lr.logger.Warn("failed to compress log file",
						zap.String("file", filePath),
						zap.Error(err))
				}
			}
		}
		// 压缩新创建的 .1 文件
		newRotatedFile := fmt.Sprintf("%s.1", lr.logPath)
		if err := lr.compressFile(newRotatedFile); err != nil {
			lr.logger.Warn("failed to compress new rotated log file",
				zap.String("file", newRotatedFile),
				zap.Error(err))
		}
	}

	// 删除超过最大文件数的旧文件
	lr.cleanupOldFiles()

	// 更新最后轮转时间
	lr.lastRotateTime = time.Now()

	lr.logger.Info("log rotation completed",
		zap.String("log_path", lr.logPath))

	return nil
}

// compressFile 压缩文件
func (lr *LogRotator) compressFile(filePath string) error {
	// 打开源文件
	srcFile, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file for compression: %w", err)
	}
	defer srcFile.Close()

	// 创建压缩文件
	dstFile, err := os.Create(filePath + ".gz")
	if err != nil {
		return fmt.Errorf("failed to create compressed file: %w", err)
	}
	defer dstFile.Close()

	// 创建 gzip writer
	gzWriter := gzip.NewWriter(dstFile)
	defer gzWriter.Close()

	// 复制数据
	if _, err := io.Copy(gzWriter, srcFile); err != nil {
		return fmt.Errorf("failed to compress file: %w", err)
	}

	// 删除原始文件
	if err := os.Remove(filePath); err != nil {
		return fmt.Errorf("failed to remove original file: %w", err)
	}

	return nil
}

// cleanupOldFiles 清理超过最大文件数的旧文件
func (lr *LogRotator) cleanupOldFiles() {
	logDir := filepath.Dir(lr.logPath)
	logBase := filepath.Base(lr.logPath)

	// 读取目录中的所有日志文件
	files, err := os.ReadDir(logDir)
	if err != nil {
		lr.logger.Warn("failed to read log directory",
			zap.String("dir", logDir),
			zap.Error(err))
		return
	}

	// 收集所有轮转的日志文件
	rotatedFiles := make([]string, 0)
	for _, file := range files {
		name := file.Name()
		// 匹配 agent.log.N 或 agent.log.N.gz 格式
		if strings.HasPrefix(name, logBase+".") {
			rotatedFiles = append(rotatedFiles, filepath.Join(logDir, name))
		}
	}

	// 按文件名排序（数字越大越新）
	sort.Slice(rotatedFiles, func(i, j int) bool {
		// 提取文件编号进行比较
		numI := extractFileNumber(rotatedFiles[i])
		numJ := extractFileNumber(rotatedFiles[j])
		return numI > numJ
	})

	// 删除超过最大文件数的文件
	for i := lr.maxFiles; i < len(rotatedFiles); i++ {
		if err := os.Remove(rotatedFiles[i]); err != nil {
			lr.logger.Warn("failed to remove old log file",
				zap.String("file", rotatedFiles[i]),
				zap.Error(err))
		} else {
			lr.logger.Debug("removed old log file",
				zap.String("file", rotatedFiles[i]))
		}
	}
}

// extractFileNumber 从文件名中提取编号
func extractFileNumber(filePath string) int {
	base := filepath.Base(filePath)
	// 移除 .gz 后缀
	base = strings.TrimSuffix(base, ".gz")
	// 提取数字部分
	parts := strings.Split(base, ".")
	if len(parts) < 2 {
		return 0
	}
	var num int
	fmt.Sscanf(parts[len(parts)-1], "%d", &num)
	return num
}

// LogEntry 日志条目
type LogEntry struct {
	LineNumber int
	Content    string
	Timestamp  time.Time
}

// LogManager 日志管理器
type LogManager struct {
	workDir       string
	retentionDays int
	logger        *zap.Logger
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
}

// NewLogManager 创建新的日志管理器
func NewLogManager(workDir string, logger *zap.Logger) *LogManager {
	ctx, cancel := context.WithCancel(context.Background())
	return &LogManager{
		workDir:       workDir,
		retentionDays: 30,
		logger:        logger,
		ctx:           ctx,
		cancel:        cancel,
	}
}

// GetAgentLogs 获取Agent最近N行日志
func (lm *LogManager) GetAgentLogs(agentID string, lines int) ([]string, error) {
	logPath := fmt.Sprintf("%s/agents/%s/logs/agent.log", lm.workDir, agentID)

	// 检查文件是否存在
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		return []string{}, nil
	}

	// 打开文件
	file, err := os.Open(logPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}
	defer file.Close()

	// 获取文件大小
	stat, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat log file: %w", err)
	}

	// 如果文件为空，返回空切片
	if stat.Size() == 0 {
		return []string{}, nil
	}

	// 限制读取大小（避免读取过大文件）
	const maxReadSize = 10 * 1024 * 1024 // 10MB
	readSize := stat.Size()
	startPos := int64(0)
	if readSize > maxReadSize {
		startPos = readSize - maxReadSize
	}

	// 移动到读取起始位置
	if _, err := file.Seek(startPos, io.SeekStart); err != nil {
		return nil, fmt.Errorf("failed to seek: %w", err)
	}

	// 读取内容
	content, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read log file: %w", err)
	}

	// 按行分割
	allLines := strings.Split(string(content), "\n")

	// 如果从中间开始读取，跳过第一行（可能不完整）
	if startPos > 0 && len(allLines) > 0 {
		allLines = allLines[1:]
	}

	// 过滤空行并取最后N行
	result := make([]string, 0)
	for i := len(allLines) - 1; i >= 0 && len(result) < lines; i-- {
		line := strings.TrimSpace(allLines[i])
		if line != "" {
			result = append([]string{line}, result...)
		}
	}

	return result, nil
}

// SearchLogs 搜索日志中的关键词
func (lm *LogManager) SearchLogs(agentID string, keyword string, limit int) ([]LogEntry, error) {
	logPath := fmt.Sprintf("%s/agents/%s/logs/agent.log", lm.workDir, agentID)

	// 检查文件是否存在
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		return []LogEntry{}, nil
	}

	// 打开文件
	file, err := os.Open(logPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}
	defer file.Close()

	// 读取文件内容（限制读取大小，避免读取过大文件）
	const maxReadSize = 10 * 1024 * 1024 // 10MB
	stat, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat log file: %w", err)
	}

	readSize := stat.Size()
	if readSize > maxReadSize {
		// 只读取最后10MB
		if _, err := file.Seek(-maxReadSize, io.SeekEnd); err != nil {
			return nil, fmt.Errorf("failed to seek: %w", err)
		}
		readSize = maxReadSize
	} else {
		// 从文件开头读取
		if _, err := file.Seek(0, io.SeekStart); err != nil {
			return nil, fmt.Errorf("failed to seek: %w", err)
		}
	}

	content, err := io.ReadAll(io.LimitReader(file, readSize))
	if err != nil {
		return nil, fmt.Errorf("failed to read log file: %w", err)
	}

	// 按行分割
	lines := strings.Split(string(content), "\n")
	results := make([]LogEntry, 0)
	lineNumber := 0

	// 搜索包含关键词的行
	for _, line := range lines {
		lineNumber++
		if strings.Contains(line, keyword) {
			results = append(results, LogEntry{
				LineNumber: lineNumber,
				Content:    line,
				Timestamp:  time.Now(), // 如果日志包含时间戳，可以解析后填充
			})
			if len(results) >= limit {
				break
			}
		}
	}

	// 按行号倒序排列（最新的在前）
	sort.Slice(results, func(i, j int) bool {
		return results[i].LineNumber > results[j].LineNumber
	})

	return results, nil
}

// StartCleanupTask 启动日志清理任务
func (lm *LogManager) StartCleanupTask() {
	lm.wg.Add(1)
	go lm.cleanupLoop()
	lm.logger.Info("log cleanup task started",
		zap.Int("retention_days", lm.retentionDays))
}

// StopCleanupTask 停止日志清理任务
func (lm *LogManager) StopCleanupTask() {
	lm.cancel()
	lm.wg.Wait()
	lm.logger.Info("log cleanup task stopped")
}

// cleanupLoop 清理循环
func (lm *LogManager) cleanupLoop() {
	defer lm.wg.Done()

	// 计算下次清理时间（凌晨2点）
	now := time.Now()
	nextCleanup := time.Date(now.Year(), now.Month(), now.Day(), 2, 0, 0, 0, now.Location())
	if nextCleanup.Before(now) {
		nextCleanup = nextCleanup.Add(24 * time.Hour)
	}

	// 首次等待到凌晨2点
	waitDuration := nextCleanup.Sub(now)
	lm.logger.Info("log cleanup task will run at",
		zap.Time("next_cleanup", nextCleanup),
		zap.Duration("wait", waitDuration))

	select {
	case <-time.After(waitDuration):
		// 执行首次清理
		lm.cleanupOldLogs()
	case <-lm.ctx.Done():
		return
	}

	// 之后每天执行一次
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-lm.ctx.Done():
			return
		case <-ticker.C:
			lm.cleanupOldLogs()
		}
	}
}

// cleanupOldLogs 清理过期日志
func (lm *LogManager) cleanupOldLogs() {
	agentsLogDir := fmt.Sprintf("%s/agents", lm.workDir)

	// 检查目录是否存在
	if _, err := os.Stat(agentsLogDir); os.IsNotExist(err) {
		return
	}

	// 读取所有Agent目录
	agents, err := os.ReadDir(agentsLogDir)
	if err != nil {
		lm.logger.Warn("failed to read agents directory",
			zap.String("dir", agentsLogDir),
			zap.Error(err))
		return
	}

	var deletedFiles int
	var freedSpace int64
	cutoffTime := time.Now().AddDate(0, 0, -lm.retentionDays)

	// 遍历每个Agent的日志目录
	for _, agentDir := range agents {
		if !agentDir.IsDir() {
			continue
		}

		logsDir := filepath.Join(agentsLogDir, agentDir.Name(), "logs")
		if _, err := os.Stat(logsDir); os.IsNotExist(err) {
			continue
		}

		// 读取日志文件
		logFiles, err := os.ReadDir(logsDir)
		if err != nil {
			lm.logger.Warn("failed to read logs directory",
				zap.String("dir", logsDir),
				zap.Error(err))
			continue
		}

		// 检查每个日志文件
		for _, logFile := range logFiles {
			if logFile.IsDir() {
				continue
			}

			filePath := filepath.Join(logsDir, logFile.Name())
			info, err := logFile.Info()
			if err != nil {
				continue
			}

			// 检查文件修改时间
			if info.ModTime().Before(cutoffTime) {
				// 删除过期文件
				if err := os.Remove(filePath); err != nil {
					lm.logger.Warn("failed to delete old log file",
						zap.String("file", filePath),
						zap.Error(err))
				} else {
					deletedFiles++
					freedSpace += info.Size()
					lm.logger.Debug("deleted old log file",
						zap.String("file", filePath),
						zap.Time("mod_time", info.ModTime()))
				}
			}
		}
	}

	if deletedFiles > 0 {
		lm.logger.Info("log cleanup completed",
			zap.Int("deleted_files", deletedFiles),
			zap.Int64("freed_space_bytes", freedSpace),
			zap.Int64("freed_space_mb", freedSpace/(1024*1024)))
	}
}

// SetRetentionDays 设置日志保留天数
func (lm *LogManager) SetRetentionDays(days int) {
	lm.retentionDays = days
}
