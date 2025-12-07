package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"

	"github.com/fsnotify/fsnotify"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

// ConfigManager Agent配置文件管理器
// 提供配置读取、验证、更新和热重载功能
type ConfigManager struct {
	// registry Agent注册表，用于获取Agent信息
	registry *AgentRegistry

	// logger 日志记录器
	logger *zap.Logger

	// mu 保护并发访问的读写锁
	mu sync.RWMutex

	// watcher 文件监听器
	watcher *fsnotify.Watcher

	// watching 是否正在监听
	watching bool

	// fileToAgentID 配置文件路径到Agent ID的映射
	fileToAgentID map[string]string

	// agentInstances Agent实例映射，用于发送重载信号
	agentInstances map[string]*AgentInstance
}

// NewConfigManager 创建新的配置管理器
func NewConfigManager(registry *AgentRegistry, logger *zap.Logger) *ConfigManager {
	return &ConfigManager{
		registry:       registry,
		logger:         logger,
		fileToAgentID:  make(map[string]string),
		agentInstances: make(map[string]*AgentInstance),
	}
}

// SetAgentInstance 设置Agent实例（用于发送重载信号）
func (cm *ConfigManager) SetAgentInstance(agentID string, instance *AgentInstance) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.agentInstances[agentID] = instance
}

// RemoveAgentInstance 移除Agent实例
func (cm *ConfigManager) RemoveAgentInstance(agentID string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	delete(cm.agentInstances, agentID)
}

// ReadConfig 读取Agent配置文件
// 自动检测格式（YAML/JSON）并解析为 map[string]interface{}
func (cm *ConfigManager) ReadConfig(agentID string) (map[string]interface{}, error) {
	// 从Registry获取Agent信息
	info := cm.registry.Get(agentID)
	if info == nil {
		return nil, fmt.Errorf("agent not found: %s", agentID)
	}

	// 获取配置文件路径
	configFile := info.ConfigFile
	if configFile == "" {
		return nil, fmt.Errorf("config file not found: agent %s has no config file", agentID)
	}

	// 读取文件内容
	data, err := os.ReadFile(configFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("config file not found: %s", configFile)
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// 检测格式并解析
	config := make(map[string]interface{})
	format := cm.detectFormat(configFile)

	switch format {
	case "yaml":
		if err := yaml.Unmarshal(data, &config); err != nil {
			return nil, fmt.Errorf("failed to parse YAML config: %w", err)
		}
	case "json":
		if err := json.Unmarshal(data, &config); err != nil {
			return nil, fmt.Errorf("failed to parse JSON config: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported config format: %s (file: %s)", format, configFile)
	}

	return config, nil
}

// detectFormat 检测配置文件格式
func (cm *ConfigManager) detectFormat(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".yaml", ".yml":
		return "yaml"
	case ".json":
		return "json"
	default:
		// 尝试根据内容判断
		return "yaml" // 默认使用 YAML
	}
}

// ValidateConfig 验证Agent配置
// 根据Agent类型应用不同的验证规则
func (cm *ConfigManager) ValidateConfig(agentID string, config map[string]interface{}) error {
	// 从Registry获取Agent信息
	info := cm.registry.Get(agentID)
	if info == nil {
		return fmt.Errorf("agent not found: %s", agentID)
	}

	var validationErrors []error

	// 根据Agent类型进行验证
	switch info.Type {
	case TypeFilebeat:
		errors := cm.validateFilebeatConfig(config)
		validationErrors = append(validationErrors, errors...)

	case TypeTelegraf:
		errors := cm.validateTelegrafConfig(config)
		validationErrors = append(validationErrors, errors...)

	case TypeNodeExporter:
		// Node Exporter 通常不使用配置文件，仅检查配置可解析
		// 如果提供了配置，进行基本检查
		if len(config) == 0 {
			// 空配置也是有效的（Node Exporter 可能不使用配置文件）
			return nil
		}

	case TypeCustom:
		// 自定义类型：仅做基本检查
		// 可以后续扩展

	default:
		// 未知类型：跳过验证或仅做基本检查
	}

	// 如果有验证错误，返回聚合错误
	if len(validationErrors) > 0 {
		return errors.Join(validationErrors...)
	}

	return nil
}

// validateFilebeatConfig 验证Filebeat配置
func (cm *ConfigManager) validateFilebeatConfig(config map[string]interface{}) []error {
	var errors []error

	// 检查必需字段 filebeat.inputs
	if inputs, ok := config["filebeat.inputs"]; !ok {
		errors = append(errors, fmt.Errorf("missing required field: filebeat.inputs"))
	} else {
		// 验证 inputs 是数组
		if _, ok := inputs.([]interface{}); !ok {
			errors = append(errors, fmt.Errorf("filebeat.inputs must be an array"))
		}
	}

	// 检查必需字段 output
	if _, ok := config["output"]; !ok {
		errors = append(errors, fmt.Errorf("missing required field: output"))
	}

	return errors
}

// validateTelegrafConfig 验证Telegraf配置
func (cm *ConfigManager) validateTelegrafConfig(config map[string]interface{}) []error {
	var errors []error

	// 检查必需字段 agent
	if _, ok := config["agent"]; !ok {
		errors = append(errors, fmt.Errorf("missing required field: agent"))
	}

	// 检查至少有一个 input 或 output
	hasInput := false
	hasOutput := false

	// 检查 inputs
	if inputs, ok := config["inputs"]; ok {
		if inputsMap, ok := inputs.(map[string]interface{}); ok && len(inputsMap) > 0 {
			hasInput = true
		}
	}

	// 检查 outputs
	if outputs, ok := config["outputs"]; ok {
		if outputsMap, ok := outputs.(map[string]interface{}); ok && len(outputsMap) > 0 {
			hasOutput = true
		}
	}

	if !hasInput && !hasOutput {
		errors = append(errors, fmt.Errorf("telegraf config must have at least one input or output"))
	}

	return errors
}

// UpdateConfig 更新Agent配置
// 使用深度合并和原子性写入
func (cm *ConfigManager) UpdateConfig(agentID string, updates map[string]interface{}) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// 读取当前配置
	currentConfig, err := cm.ReadConfig(agentID)
	if err != nil {
		return fmt.Errorf("failed to read current config: %w", err)
	}

	// 深度合并配置
	mergedConfig := deepMerge(currentConfig, updates)

	// 验证合并后的配置
	if err := cm.ValidateConfig(agentID, mergedConfig); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	// 获取配置文件路径
	info := cm.registry.Get(agentID)
	if info == nil {
		return fmt.Errorf("agent not found: %s", agentID)
	}
	configFile := info.ConfigFile
	if configFile == "" {
		return fmt.Errorf("config file not found: agent %s has no config file", agentID)
	}

	// 检测格式
	format := cm.detectFormat(configFile)

	// 序列化配置
	var data []byte
	switch format {
	case "yaml":
		data, err = yaml.Marshal(mergedConfig)
		if err != nil {
			return fmt.Errorf("failed to marshal YAML: %w", err)
		}
	case "json":
		data, err = json.MarshalIndent(mergedConfig, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
	default:
		return fmt.Errorf("unsupported config format: %s", format)
	}

	// 原子性写入
	tmpFile := configFile + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	// 验证临时文件可读
	if _, err := os.ReadFile(tmpFile); err != nil {
		os.Remove(tmpFile)
		return fmt.Errorf("failed to verify temp file: %w", err)
	}

	// 原子替换
	if err := os.Rename(tmpFile, configFile); err != nil {
		os.Remove(tmpFile)
		return fmt.Errorf("failed to replace config file: %w", err)
	}

	cm.logger.Info("config updated successfully",
		zap.String("agent_id", agentID),
		zap.String("config_file", configFile),
		zap.String("format", format))

	return nil
}

// StartWatching 开始监听配置文件变化
func (cm *ConfigManager) StartWatching(ctx context.Context) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.watching {
		return fmt.Errorf("config watching already started")
	}

	// 创建文件监听器
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		cm.logger.Error("failed to create file watcher", zap.Error(err))
		return fmt.Errorf("failed to create file watcher: %w", err)
	}

	cm.watcher = watcher

	// 获取所有Agent的配置文件路径
	agents := cm.registry.List()
	for _, agent := range agents {
		if agent.ConfigFile == "" {
			continue
		}

		// 获取配置文件所在目录
		configDir := filepath.Dir(agent.ConfigFile)
		// 监听目录（而不是文件），因为某些系统需要监听目录
		if err := watcher.Add(configDir); err != nil {
			cm.logger.Warn("failed to watch config directory",
				zap.String("agent_id", agent.ID),
				zap.String("config_dir", configDir),
				zap.Error(err))
			continue
		}

		// 记录映射关系
		cm.fileToAgentID[agent.ConfigFile] = agent.ID

		cm.logger.Info("watching config file",
			zap.String("agent_id", agent.ID),
			zap.String("config_file", agent.ConfigFile))
	}

	cm.watching = true

	// 在后台goroutine中处理文件变化事件
	go cm.handleFileEvents(ctx)

	return nil
}

// handleFileEvents 处理文件变化事件
func (cm *ConfigManager) handleFileEvents(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			cm.logger.Info("stopping config file watcher")
			return

		case event, ok := <-cm.watcher.Events:
			if !ok {
				return
			}

			// 只处理写入和创建事件
			if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
				cm.handleConfigChange(event.Name)
			}

		case err, ok := <-cm.watcher.Errors:
			if !ok {
				return
			}
			cm.logger.Warn("file watcher error", zap.Error(err))
		}
	}
}

// handleConfigChange 处理配置文件变化
func (cm *ConfigManager) handleConfigChange(filePath string) {
	cm.mu.RLock()
	agentID, found := cm.fileToAgentID[filePath]
	cm.mu.RUnlock()

	if !found {
		// 可能是目录中的其他文件，尝试匹配
		agents := cm.registry.List()
		for _, agent := range agents {
			if agent.ConfigFile == filePath {
				agentID = agent.ID
				found = true
				break
			}
		}
	}

	if !found {
		return
	}

	cm.logger.Info("config file changed, triggering reload",
		zap.String("agent_id", agentID),
		zap.String("config_file", filePath))

	// 触发配置重载
	if err := cm.reloadAgentConfig(agentID); err != nil {
		cm.logger.Warn("failed to reload agent config",
			zap.String("agent_id", agentID),
			zap.Error(err))
	}
}

// reloadAgentConfig 重载Agent配置
// 通过发送SIGHUP信号触发Agent重载
func (cm *ConfigManager) reloadAgentConfig(agentID string) error {
	cm.mu.RLock()
	instance, exists := cm.agentInstances[agentID]
	cm.mu.RUnlock()

	if !exists {
		// 如果实例不存在，尝试从Registry获取信息并发送信号
		info := cm.registry.Get(agentID)
		if info == nil {
			return fmt.Errorf("agent not found: %s", agentID)
		}

		pid := info.GetPID()
		if pid == 0 {
			cm.logger.Debug("agent not running, skipping reload",
				zap.String("agent_id", agentID))
			return nil
		}

		// 发送SIGHUP信号
		process, err := os.FindProcess(pid)
		if err != nil {
			return fmt.Errorf("failed to find process: %w", err)
		}

		if err := process.Signal(syscall.SIGHUP); err != nil {
			return fmt.Errorf("failed to send SIGHUP: %w", err)
		}

		cm.logger.Info("sent SIGHUP to agent",
			zap.String("agent_id", agentID),
			zap.Int("pid", pid))
		return nil
	}

	// 如果实例存在，通过实例发送信号
	info := instance.GetInfo()
	pid := info.GetPID()
	if pid == 0 {
		cm.logger.Debug("agent not running, skipping reload",
			zap.String("agent_id", agentID))
		return nil
	}

	// 获取进程并发送SIGHUP信号
	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("failed to find process: %w", err)
	}

	if err := process.Signal(syscall.SIGHUP); err != nil {
		return fmt.Errorf("failed to send SIGHUP: %w", err)
	}

	cm.logger.Info("sent SIGHUP to agent via instance",
		zap.String("agent_id", agentID),
		zap.Int("pid", pid))

	return nil
}

// StopWatching 停止监听配置文件变化
func (cm *ConfigManager) StopWatching() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if !cm.watching {
		return nil
	}

	if cm.watcher != nil {
		if err := cm.watcher.Close(); err != nil {
			cm.logger.Error("failed to close file watcher", zap.Error(err))
			return fmt.Errorf("failed to close file watcher: %w", err)
		}
		cm.watcher = nil
	}

	cm.watching = false
	cm.fileToAgentID = make(map[string]string)

	cm.logger.Info("config file watching stopped")

	return nil
}

// deepMerge 深度合并两个配置map
func deepMerge(dst, src map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	// 先复制dst的所有内容
	for k, v := range dst {
		result[k] = deepCopyValue(v)
	}

	// 再合并src的内容
	for k, v := range src {
		if existing, exists := result[k]; exists {
			// 如果两个值都是map，递归合并
			if dstMap, ok := existing.(map[string]interface{}); ok {
				if srcMap, ok := v.(map[string]interface{}); ok {
					result[k] = deepMerge(dstMap, srcMap)
					continue
				}
			}
		}
		// 否则直接覆盖
		result[k] = deepCopyValue(v)
	}

	return result
}

// deepCopyValue 深度复制值
func deepCopyValue(v interface{}) interface{} {
	switch val := v.(type) {
	case map[string]interface{}:
		copy := make(map[string]interface{})
		for k, v := range val {
			copy[k] = deepCopyValue(v)
		}
		return copy
	case []interface{}:
		copy := make([]interface{}, len(val))
		for i, v := range val {
			copy[i] = deepCopyValue(v)
		}
		return copy
	default:
		// 对于基本类型，直接返回（Go会复制值）
		return v
	}
}
