package daemon

import (
	"net/http"
	_ "net/http/pprof" // 导入 pprof，注册 HTTP handlers
	"runtime"

	"go.uber.org/zap"
)

// startPprofServer 启动 pprof 性能分析服务器
// 仅在开发/调试模式下启用，生产环境应通过配置控制
func (d *Daemon) startPprofServer() {
	// 从配置读取 pprof 端口，默认不启用
	pprofPort := d.config.Daemon.PprofPort
	if pprofPort == "" || pprofPort == "0" {
		d.logger.Debug("pprof server disabled (port not configured)")
		return
	}

	// 设置 GOMAXPROCS（可选，用于性能调优）
	// 默认使用所有 CPU 核心
	if d.config.Daemon.MaxProcs > 0 {
		runtime.GOMAXPROCS(d.config.Daemon.MaxProcs)
		d.logger.Info("set GOMAXPROCS",
			zap.Int("max_procs", d.config.Daemon.MaxProcs))
	}

	// 启动 pprof HTTP 服务器
	pprofAddr := d.config.Daemon.PprofAddress
	if pprofAddr == "" {
		pprofAddr = "127.0.0.1" // 默认只监听本地
	}

	pprofURL := pprofAddr + ":" + pprofPort
	d.logger.Info("starting pprof server",
		zap.String("address", pprofURL),
		zap.String("docs", "https://golang.org/pkg/net/http/pprof/"))

	// 在单独的 goroutine 中启动服务器
	go func() {
		if err := http.ListenAndServe(pprofURL, nil); err != nil {
			d.logger.Warn("pprof server error",
				zap.String("address", pprofURL),
				zap.Error(err))
		}
	}()
}
