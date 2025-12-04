package daemon

import (
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"
)

// WaitForSignal 等待退出信号
func (d *Daemon) WaitForSignal() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	sig := <-sigCh
	d.logger.Info("received signal", zap.String("signal", sig.String()))

	// 优雅退出
	d.Stop()
}
