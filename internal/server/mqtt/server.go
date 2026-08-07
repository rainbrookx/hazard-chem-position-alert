// Package mqtt 提供内嵌 Mochi MQTT Broker 的极简启动封装
package mqtt

import (
	"fmt"
	"log/slog"
	"sync"

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

// ErrCh 返回运行时错误通道，供调用方监听
func (s *Server) ErrCh() <-chan error {
	return s.errCh
}

// Close 安全关闭 MQTT 服务器（幂等）
func (s *Server) Close() {
	s.closeOnce.Do(func() {
		if s.server != nil {
			if err := s.server.Close(); err != nil {
				slog.Error("退出 MQTT 错误", "error", err)
			}
		}
		slog.Info("MQTT Broker 已关闭")
	})
}

// Start 初始化并启动 MQTT Broker（非阻塞）。
// 返回启动阶段的错误（如端口占用）；运行时错误通过 ErrCh() 上报。
func (s *Server) Start() error {
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
		if err := s.server.Serve(); err != nil {
			s.errCh <- fmt.Errorf("MQTT Broker Serve 错误: %w", err)
		}
	}()

	slog.Info("MQTT Broker 已启动", "address", s.cfg.Address)
	return nil
}

// setupSubscriptions 注册默认订阅逻辑（可扩展）
func (s *Server) setupSubscriptions() error {
	if err := s.server.Subscribe("test", 0, handler.CallbackFn); err != nil {
		return err
	}
	return nil
}
