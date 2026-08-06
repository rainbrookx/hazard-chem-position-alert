// Package mqtt 提供内嵌 Mochi MQTT Broker 的极简启动封装
package mqtt

import (
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"

	mqtt "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/hooks/auth"
	"github.com/mochi-mqtt/server/v2/listeners"
	"github.com/rainbrookx/hazard-chem-position-alert/internal/infrastructure/config"
	"github.com/rainbrookx/hazard-chem-position-alert/internal/server/mqtt/handler"
)

// Server 封装 MQTT Broker
type Server struct {
	server    *mqtt.Server
	cfg       config.MochiMQTTConfig
	errCh     chan error
	closeOnce sync.Once // 保证 Close 只执行一次
}

// New 创建 MQTT 服务实例（不启动）
func New(cfg config.MochiMQTTConfig) *Server {
	opts := &mqtt.Options{InlineClient: true}
	s := mqtt.New(opts)

	return &Server{
		server: s,
		cfg:    cfg,
		errCh:  make(chan error, 1),
	}
}

// Close 安全关闭 MQTT 服务器（幂等）
func (s *Server) Close() {
	s.closeOnce.Do(func() {
		if s.server != nil {
			if err := s.server.Close(); err != nil {
				slog.Error("退出 MQTT 错误", err)
			}
		}
	})
}

// Run 启动 MQTT Broker 并阻塞，直到收到 SIGINT/SIGTERM 信号或发生致命错误。
// 无论以何种方式退出（包括错误返回），都会通过 defer 调用 Close 释放资源。
func (s *Server) Run() error {
	defer s.Close() // 保证所有退出路径都会关闭资源

	// 添加认证钩子
	if err := s.server.AddHook(new(auth.AllowHook), nil); err != nil {
		return fmt.Errorf("添加认证钩子失败: %w", err)
	}

	// 添加 TCP 监听器
	tcp := listeners.NewTCP(listeners.Config{Address: s.cfg.Address})
	if err := s.server.AddListener(tcp); err != nil {
		return fmt.Errorf("添加 TCP 监听器失败: %w", err)
	}

	// 设置默认订阅
	if err := s.setupSubscriptions(); err != nil {
		return fmt.Errorf("订阅主题失败: %w", err)
	}

	// 在独立 goroutine 中启动 Broker，捕获启动错误和运行时 panic
	go func() {
		defer func() {
			if r := recover(); r != nil {
				s.errCh <- fmt.Errorf("MQTT Broker 运行时 panic: %v", r)
			}
		}()
		// Serve() 方法在成功启动后立即返回 nil（非阻塞），
		// 若启动失败（如端口占用）则立即返回错误。
		if err := s.server.Serve(); err != nil {
			s.errCh <- fmt.Errorf("MQTT Broker Serve 错误: %w", err)
		}
	}()

	slog.Info("MQTT Broker 已启动完成")

	// 设置系统信号监听
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// 阻塞等待：信号 或 运行时错误
	select {
	case sig := <-sigCh:
		slog.Info("收到信号，正在关闭 MQTT Broker", "signal", sig)
	case err := <-s.errCh:
		slog.Error("MQTT Broker 运行时错误", "error", err)
		// 错误发生后，defer 中的 Close 会执行清理
	}

	// 注意：不再显式调用 s.server.Close()，由 defer 完成
	return nil
}

// setupSubscriptions 注册默认订阅逻辑（可扩展）
func (s *Server) setupSubscriptions() error {
	// 示例订阅，可继续添加
	if err := s.server.Subscribe("test", 0, handler.CallbackFn); err != nil {
		return err
	}
	// 可增加更多订阅...
	return nil
}
