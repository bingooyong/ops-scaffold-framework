//go:build linux

package agent

import (
	"fmt"
	"os"

	"go.uber.org/zap"
)

// getNumFDs 获取进程的文件描述符数量(Linux专用实现)
// 通过读取 /proc/<pid>/fd 目录下的文件数量来获取
func getNumFDs(pid int32, logger *zap.Logger) (int32, error) {
	return getNumFDsFromProcFS(pid)
}

// getNumFDsFromProcFS 从 /proc 文件系统获取文件描述符数量
// Linux 支持 /proc/<pid>/fd 目录,该目录下每个符号链接代表一个打开的文件描述符
func getNumFDsFromProcFS(pid int32) (int32, error) {
	fdDir := fmt.Sprintf("/proc/%d/fd", pid)

	// 读取 /proc/<pid>/fd 目录
	entries, err := os.ReadDir(fdDir)
	if err != nil {
		return 0, fmt.Errorf("failed to read fd directory: %w", err)
	}

	// 文件描述符数量等于目录中的条目数量
	return int32(len(entries)), nil
}

// getNumFDsAlternative 提供一个备用的文件描述符获取方法
// 通过读取 /proc/<pid>/fdinfo 目录
func getNumFDsAlternative(pid int32, logger *zap.Logger) (int32, error) {
	fdinfoDir := fmt.Sprintf("/proc/%d/fdinfo", pid)

	entries, err := os.ReadDir(fdinfoDir)
	if err != nil {
		return 0, fmt.Errorf("failed to read fdinfo directory: %w", err)
	}

	return int32(len(entries)), nil
}

// getFDsWithFallback 尝试多种方法获取文件描述符数量
// 优先使用 /proc/fd,失败后尝试 /proc/fdinfo
func getFDsWithFallback(pid int32, logger *zap.Logger) (int32, error) {
	// 方法1: 使用 /proc/<pid>/fd (最常用)
	numFDs, err := getNumFDsFromProcFS(pid)
	if err == nil {
		return numFDs, nil
	}

	// 记录失败原因
	logger.Debug("procfs fd method failed, trying fdinfo",
		zap.Int32("pid", pid),
		zap.Error(err))

	// 方法2: 使用 /proc/<pid>/fdinfo
	numFDs, err = getNumFDsAlternative(pid, logger)
	if err == nil {
		return numFDs, nil
	}

	// 所有方法都失败
	return 0, fmt.Errorf("all methods failed to get num fds")
}

// getFDLimit 获取进程的文件描述符限制
// 通过读取 /proc/<pid>/limits 文件
func getFDLimit(pid int32) (soft, hard int64, err error) {
	// 这里简化处理,表示未实现详细解析
	// 实际使用中可以解析 /proc/<pid>/limits 文件获取准确值
	return 0, 0, fmt.Errorf("not implemented")
}
