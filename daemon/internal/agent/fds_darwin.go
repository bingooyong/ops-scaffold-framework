//go:build darwin

package agent

import (
	"fmt"
	"os/exec"
	"strings"

	"go.uber.org/zap"
)

// getNumFDs 获取进程的文件描述符数量(macOS专用实现)
// 使用 lsof 命令获取准确的文件描述符数量
func getNumFDs(pid int32, logger *zap.Logger) (int32, error) {
	// 使用 lsof -p <pid> 命令获取进程的文件描述符列表
	// lsof 的输出格式:
	// COMMAND  PID  USER   FD   TYPE DEVICE SIZE/OFF   NODE NAME
	// 我们需要统计 FD 列中的有效文件描述符
	cmd := exec.Command("lsof", "-p", fmt.Sprintf("%d", pid))
	output, err := cmd.Output()
	if err != nil {
		// 如果命令执行失败,可能是进程不存在或权限不足
		return 0, fmt.Errorf("lsof command failed: %w", err)
	}

	// 解析输出,统计文件描述符数量
	lines := strings.Split(string(output), "\n")
	fdCount := 0

	// 跳过第一行(表头)
	for i := 1; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}

		// 按空格分割行
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}

		// FD 列是第 4 列(索引为 3)
		// FD 可能是数字(如 "0", "1", "2")或特殊值(如 "cwd", "txt", "mem")
		// 我们只统计数字形式的文件描述符
		fd := fields[3]

		// 检查是否是数字或以数字开头(如 "1u", "2r", "3w")
		if len(fd) > 0 {
			// 提取数字部分
			numPart := ""
			for _, ch := range fd {
				if ch >= '0' && ch <= '9' {
					numPart += string(ch)
				} else {
					break
				}
			}

			// 如果有数字部分,则认为是有效的文件描述符
			if numPart != "" {
				fdCount++
			}
		}
	}

	return int32(fdCount), nil
}

// getNumFDsFromProcFS 尝试从 /proc 文件系统获取文件描述符数量(macOS不支持,仅作为备用)
// macOS 没有 /proc 文件系统,此函数总是返回错误
func getNumFDsFromProcFS(pid int32) (int32, error) {
	return 0, fmt.Errorf("procfs not available on darwin")
}

// getNumFDsAlternative 提供一个备用的文件描述符获取方法
// 使用 /dev/fd 目录(macOS 支持)
func getNumFDsAlternative(pid int32, logger *zap.Logger) (int32, error) {
	// macOS 上每个进程的 /dev/fd 目录映射到当前进程
	// 无法直接通过文件系统访问其他进程的 fd
	// 因此这个方法不适用于监控其他进程
	return 0, fmt.Errorf("alternative method not available for other processes on darwin")
}

// getFDsWithFallback 尝试多种方法获取文件描述符数量
// 优先使用 lsof,失败后尝试其他方法
func getFDsWithFallback(pid int32, logger *zap.Logger) (int32, error) {
	// 方法1: 使用 lsof (最准确)
	numFDs, err := getNumFDs(pid, logger)
	if err == nil {
		return numFDs, nil
	}

	// 记录 lsof 失败的原因
	logger.Debug("lsof method failed, trying alternatives",
		zap.Int32("pid", pid),
		zap.Error(err))

	// 方法2: 尝试其他方法(在macOS上不可用)
	numFDs, err = getNumFDsFromProcFS(pid)
	if err == nil {
		return numFDs, nil
	}

	// 所有方法都失败
	return 0, fmt.Errorf("all methods failed to get num fds")
}
