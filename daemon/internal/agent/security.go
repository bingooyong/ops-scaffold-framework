package agent

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	// agentIDPattern Agent ID 格式验证正则表达式
	// 只允许：字母、数字、连字符、下划线，长度 1-100
	agentIDPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,100}$`)

	// dangerousPathPatterns 危险路径模式
	dangerousPathPatterns = []string{
		"..",
		"~",
		"//",
	}
)

// ValidateAgentID 验证 Agent ID 格式
func ValidateAgentID(agentID string) error {
	if agentID == "" {
		return fmt.Errorf("agent ID cannot be empty")
	}

	if len(agentID) > 100 {
		return fmt.Errorf("agent ID too long (max 100 characters)")
	}

	if !agentIDPattern.MatchString(agentID) {
		return fmt.Errorf("agent ID contains invalid characters (only letters, numbers, hyphens, and underscores allowed)")
	}

	return nil
}

// ValidatePath 验证路径安全性
// 检查路径是否包含危险字符，并规范化路径
func ValidatePath(path string, allowedBaseDirs []string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("path cannot be empty")
	}

	// 规范化路径
	cleanPath := filepath.Clean(path)

	// 检查是否包含危险模式
	for _, pattern := range dangerousPathPatterns {
		if strings.Contains(cleanPath, pattern) {
			return "", fmt.Errorf("path contains dangerous pattern: %s", pattern)
		}
	}

	// 如果指定了允许的基础目录，验证路径是否在允许范围内
	if len(allowedBaseDirs) > 0 {
		absPath, err := filepath.Abs(cleanPath)
		if err != nil {
			return "", fmt.Errorf("failed to get absolute path: %w", err)
		}

		allowed := false
		for _, baseDir := range allowedBaseDirs {
			absBaseDir, err := filepath.Abs(baseDir)
			if err != nil {
				continue
			}
			if strings.HasPrefix(absPath, absBaseDir) {
				allowed = true
				break
			}
		}

		if !allowed {
			return "", fmt.Errorf("path is not within allowed directories")
		}
	}

	return cleanPath, nil
}

// ValidateBinaryPath 验证二进制路径
// 确保是绝对路径且在允许的目录中
func ValidateBinaryPath(binaryPath string, allowedDirs []string) error {
	if binaryPath == "" {
		return fmt.Errorf("binary path cannot be empty")
	}

	// 必须是绝对路径
	if !filepath.IsAbs(binaryPath) {
		return fmt.Errorf("binary path must be absolute")
	}

	// 验证路径安全性
	_, err := ValidatePath(binaryPath, allowedDirs)
	return err
}

// SanitizeLogMessage 脱敏日志消息中的敏感信息
func SanitizeLogMessage(message string) string {
	// 脱敏 Token（保留前4位和后4位）
	tokenPattern := regexp.MustCompile(`(token|Token|TOKEN|bearer|Bearer|BEARER)\s*[:=]\s*([a-zA-Z0-9_-]{20,})`)
	message = tokenPattern.ReplaceAllStringFunc(message, func(match string) string {
		parts := regexp.MustCompile(`[:=]\s*`).Split(match, 2)
		if len(parts) == 2 {
			token := parts[1]
			if len(token) > 8 {
				return parts[0] + ": " + token[:4] + "***" + token[len(token)-4:]
			}
			return parts[0] + ": ***"
		}
		return match
	})

	// 脱敏密码字段
	passwordPattern := regexp.MustCompile(`(password|Password|PASSWORD|pwd|PWD)\s*[:=]\s*[^\s,}]+`)
	message = passwordPattern.ReplaceAllString(message, "$1: ***")

	return message
}
