package agent

import (
	"os"
	"runtime"
	"testing"

	"github.com/shirou/gopsutil/v3/process"
	"go.uber.org/zap/zaptest"
)

// TestResourceMonitoring_CrossPlatform 测试不同资源指标在当前平台的支持情况
func TestResourceMonitoring_CrossPlatform(t *testing.T) {
	logger := zaptest.NewLogger(t)
	pid := int32(os.Getpid())

	t.Logf("=== 跨平台资源监控测试 ===")
	t.Logf("平台: %s/%s", runtime.GOOS, runtime.GOARCH)
	t.Logf("进程 PID: %d", pid)
	t.Logf("")

	proc, err := process.NewProcess(pid)
	if err != nil {
		t.Fatalf("failed to create process: %v", err)
	}

	// 测试结果统计
	var supported, unsupported, partial int

	// 1. CPU使用率
	t.Run("CPUPercent", func(t *testing.T) {
		cpuPercent, err := proc.CPUPercent()
		if err != nil {
			t.Logf("❌ CPU使用率: 不支持 - %v", err)
			unsupported++
		} else {
			t.Logf("✅ CPU使用率: %.2f%%", cpuPercent)
			supported++
		}
	})

	// 2. 内存信息
	t.Run("MemoryInfo", func(t *testing.T) {
		memInfo, err := proc.MemoryInfo()
		if err != nil {
			t.Logf("❌ 内存信息: 不支持 - %v", err)
			unsupported++
		} else if memInfo != nil {
			t.Logf("✅ 内存信息:")
			t.Logf("   - RSS: %d bytes (%.2f MB)", memInfo.RSS, float64(memInfo.RSS)/1024/1024)
			t.Logf("   - VMS: %d bytes (%.2f MB)", memInfo.VMS, float64(memInfo.VMS)/1024/1024)
			supported++
		}
	})

	// 3. 磁盘I/O
	t.Run("IOCounters", func(t *testing.T) {
		ioCounters, err := proc.IOCounters()
		if err != nil {
			if runtime.GOOS == "darwin" {
				t.Logf("⚠️  磁盘I/O: macOS 不支持 (已知限制) - %v", err)
				partial++
			} else {
				t.Logf("❌ 磁盘I/O: 不支持 - %v", err)
				unsupported++
			}
		} else if ioCounters != nil {
			t.Logf("✅ 磁盘I/O:")
			t.Logf("   - 读取: %d bytes", ioCounters.ReadBytes)
			t.Logf("   - 写入: %d bytes", ioCounters.WriteBytes)
			supported++
		}
	})

	// 4. 文件描述符数量(gopsutil)
	t.Run("NumFDs_gopsutil", func(t *testing.T) {
		numFDs, err := proc.NumFDs()
		if err != nil {
			if runtime.GOOS == "darwin" || runtime.GOOS == "windows" {
				t.Logf("⚠️  文件描述符(gopsutil): %s 不支持 (使用自定义实现) - %v", runtime.GOOS, err)
				partial++
			} else {
				t.Logf("❌ 文件描述符(gopsutil): 不支持 - %v", err)
				unsupported++
			}
		} else {
			t.Logf("✅ 文件描述符(gopsutil): %d", numFDs)
			supported++
		}
	})

	// 5. 文件描述符数量(自定义实现)
	t.Run("NumFDs_custom", func(t *testing.T) {
		numFDs, err := getFDsWithFallback(pid, logger)
		if err != nil {
			if runtime.GOOS == "windows" {
				t.Logf("⚠️  文件描述符(自定义): Windows 不支持 (已知限制) - %v", err)
				partial++
			} else {
				t.Logf("❌ 文件描述符(自定义): 不支持 - %v", err)
				unsupported++
			}
		} else {
			t.Logf("✅ 文件描述符(自定义): %d", numFDs)
			supported++
		}
	})

	// 6. 线程数
	t.Run("NumThreads", func(t *testing.T) {
		numThreads, err := proc.NumThreads()
		if err != nil {
			t.Logf("❌ 线程数: 不支持 - %v", err)
			unsupported++
		} else {
			t.Logf("✅ 线程数: %d", numThreads)
			supported++
		}
	})

	// 7. 打开的文件列表
	t.Run("OpenFiles", func(t *testing.T) {
		openFiles, err := proc.OpenFiles()
		if err != nil {
			if runtime.GOOS == "darwin" {
				t.Logf("⚠️  打开文件列表: macOS 可能需要权限 - %v", err)
				partial++
			} else {
				t.Logf("❌ 打开文件列表: 不支持 - %v", err)
				unsupported++
			}
		} else {
			t.Logf("✅ 打开文件列表: %d 个文件", len(openFiles))
			if len(openFiles) > 0 && len(openFiles) <= 5 {
				for i, f := range openFiles {
					t.Logf("   [%d] %s (fd=%d)", i+1, f.Path, f.Fd)
				}
			}
			supported++
		}
	})

	// 8. 网络连接
	t.Run("Connections", func(t *testing.T) {
		connections, err := proc.Connections()
		if err != nil {
			t.Logf("⚠️  网络连接: 可能需要权限 - %v", err)
			partial++
		} else {
			t.Logf("✅ 网络连接: %d 个连接", len(connections))
			supported++
		}
	})

	// 9. 进程状态
	t.Run("Status", func(t *testing.T) {
		status, err := proc.Status()
		if err != nil {
			t.Logf("❌ 进程状态: 不支持 - %v", err)
			unsupported++
		} else {
			t.Logf("✅ 进程状态: %v", status)
			supported++
		}
	})

	// 10. 创建时间
	t.Run("CreateTime", func(t *testing.T) {
		createTime, err := proc.CreateTime()
		if err != nil {
			t.Logf("❌ 创建时间: 不支持 - %v", err)
			unsupported++
		} else {
			t.Logf("✅ 创建时间: %d", createTime)
			supported++
		}
	})

	// 输出总结
	t.Logf("")
	t.Logf("=== 测试总结 ===")
	t.Logf("✅ 完全支持: %d", supported)
	t.Logf("⚠️  部分支持/已知限制: %d", partial)
	t.Logf("❌ 不支持: %d", unsupported)
	t.Logf("")

	// 平台特定建议
	t.Logf("=== 平台建议 ===")
	switch runtime.GOOS {
	case "darwin":
		t.Logf("macOS 平台:")
		t.Logf("  - 磁盘I/O: 使用 DTrace 或其他工具")
		t.Logf("  - 文件描述符: 已使用 lsof 自定义实现 ✓")
		t.Logf("  - 打开文件列表: 可能需要 root 权限")
	case "linux":
		t.Logf("Linux 平台:")
		t.Logf("  - 所有指标通过 /proc 文件系统获取")
		t.Logf("  - 文件描述符: 已使用 /proc/fd 自定义实现 ✓")
		t.Logf("  - 建议使用自定义实现以获得更好性能")
	case "windows":
		t.Logf("Windows 平台:")
		t.Logf("  - 文件描述符: 不适用(使用句柄)")
		t.Logf("  - 部分指标可能需要 Windows API")
		t.Logf("  - 考虑实现句柄监控")
	}
}

// TestResourceMonitor_PlatformCompatibility 测试资源监控器的平台兼容性
func TestResourceMonitor_PlatformCompatibility(t *testing.T) {
	logger := zaptest.NewLogger(t)
	workDir := t.TempDir()

	multiManager, err := NewMultiAgentManager(workDir, logger)
	if err != nil {
		t.Fatalf("failed to create multi agent manager: %v", err)
	}

	registry := multiManager.GetRegistry()
	monitor := NewResourceMonitor(multiManager, registry, logger)

	// 注册测试Agent
	info, err := registry.Register(
		"test-agent-platform",
		TypeCustom,
		"Test Agent Platform",
		"/bin/sleep",
		"",
		workDir,
		"",
	)
	if err != nil {
		t.Fatalf("failed to register agent: %v", err)
	}

	instance := NewAgentInstance(info, logger)
	multiManager.mu.Lock()
	multiManager.instances["test-agent-platform"] = instance
	multiManager.mu.Unlock()

	// 使用当前进程的PID进行测试(避免启动新进程)
	// 直接设置PID用于测试
	info.mu.Lock()
	info.PID = os.Getpid()
	info.mu.Unlock()

	// 采集资源数据
	dataPoint, err := monitor.collectAgentResources("test-agent-platform")
	if err != nil {
		t.Fatalf("failed to collect agent resources: %v", err)
	}

	t.Logf("=== ResourceMonitor 采集结果 ===")
	t.Logf("平台: %s", runtime.GOOS)
	t.Logf("CPU: %.2f%%", dataPoint.CPU)
	t.Logf("内存 RSS: %d bytes (%.2f MB)", dataPoint.MemoryRSS, float64(dataPoint.MemoryRSS)/1024/1024)
	t.Logf("内存 VMS: %d bytes (%.2f MB)", dataPoint.MemoryVMS, float64(dataPoint.MemoryVMS)/1024/1024)
	t.Logf("磁盘读取: %d bytes", dataPoint.DiskReadBytes)
	t.Logf("磁盘写入: %d bytes", dataPoint.DiskWriteBytes)
	t.Logf("打开文件数: %d", dataPoint.OpenFiles)
	t.Logf("线程数: %d", dataPoint.NumThreads)

	// 验证关键指标
	hasData := false
	if dataPoint.CPU > 0 || dataPoint.MemoryRSS > 0 || dataPoint.OpenFiles > 0 {
		hasData = true
	}

	if !hasData {
		t.Logf("⚠️  警告: 未采集到有效数据,这在某些平台可能正常")
	} else {
		t.Logf("✅ 成功采集到资源数据")
	}
}

// BenchmarkResourceCollection 基准测试资源采集性能
func BenchmarkResourceCollection(b *testing.B) {
	logger := zaptest.NewLogger(b)
	workDir := b.TempDir()

	multiManager, _ := NewMultiAgentManager(workDir, logger)
	registry := multiManager.GetRegistry()
	monitor := NewResourceMonitor(multiManager, registry, logger)

	info, _ := registry.Register(
		"bench-agent",
		TypeCustom,
		"Bench Agent",
		"/bin/sleep",
		"",
		workDir,
		"",
	)

	// 使用当前进程PID
	info.mu.Lock()
	info.PID = os.Getpid()
	info.mu.Unlock()

	instance := NewAgentInstance(info, logger)
	multiManager.mu.Lock()
	multiManager.instances["bench-agent"] = instance
	multiManager.mu.Unlock()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = monitor.collectAgentResources("bench-agent")
	}
}
