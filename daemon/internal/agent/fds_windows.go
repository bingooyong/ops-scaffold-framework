//go:build windows

package agent

import (
	"fmt"

	"go.uber.org/zap"
)

// getNumFDs 获取进程的文件描述符数量(Windows不支持)
// Windows 没有等价的文件描述符概念,使用句柄(handles)
// 此函数返回错误,调用方应忽略此错误
func getNumFDs(pid int32, logger *zap.Logger) (int32, error) {
	return 0, fmt.Errorf("file descriptor counting not implemented on windows")
}

// getNumFDsFromProcFS Windows 不支持 procfs
func getNumFDsFromProcFS(pid int32) (int32, error) {
	return 0, fmt.Errorf("procfs not available on windows")
}

// getNumFDsAlternative Windows 备用方法(未实现)
func getNumFDsAlternative(pid int32, logger *zap.Logger) (int32, error) {
	return 0, fmt.Errorf("alternative method not implemented on windows")
}

// getFDsWithFallback Windows 平台总是返回错误
func getFDsWithFallback(pid int32, logger *zap.Logger) (int32, error) {
	return 0, fmt.Errorf("file descriptor counting not supported on windows")
}

// getNumHandles 获取 Windows 进程的句柄数量(可选实现)
// 注意: 这需要使用 Windows API,暂时未实现
func getNumHandles(pid int32, logger *zap.Logger) (int32, error) {
	return 0, fmt.Errorf("handle counting not implemented yet")
}
