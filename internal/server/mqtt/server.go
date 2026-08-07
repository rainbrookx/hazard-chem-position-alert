// Package mqtt 提供内嵌 Mochi MQTT Broker 的极简启动封装
package mqtt

import (
	"fmt"
	"log/slog"
	"sync"

	mqtt "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/hooks/auth"
	"github.com/mochi-mqtt/server/v2/listeners"
	"github.com/mochi-mqtt/server/v2/packets"

	"github.com/rainbrookx/hazard-chem-position-alert/internal/infrastructure/config"
)

// Server 封装 MQTT Broker
type Server struct {
	server    *mqtt.Server
	cfg       config.MochiMQTTConfig
	errCh     chan error
	closeOnce sync.Once

	topic           string
	positionHandler func(cl *mqtt.Client, sub packets.Subscription, pk packets.Packet)
}

// New 创建 MQTT 服务实例（不启动）
func New(cfg config.MochiMQTTConfig, topic string) *Server {
	opts := &mqtt.Options{InlineClient: true}
	s := mqtt.New(opts)

	return &Server{
		server: s,
		cfg:    cfg,
		topic:  topic,
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
func (s *Server) Start() error {
	if err := s.server.AddHook(new(auth.AllowHook), nil); err != nil {
		return fmt.Errorf("添加认证钩子失败: %w", err)
	}

	tcp := listeners.NewTCP(listeners.Config{Address: s.cfg.Address})
	if err := s.server.AddListener(tcp); err != nil {
		return fmt.Errorf("添加 TCP 监听器失败: %w", err)
	}

	if err := s.setupSubscriptions(); err != nil {
		return fmt.Errorf("订阅主题失败: %w", err)
	}

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

// setupSubscriptions 订阅清洗定位数据 topic (QoS 0).
func (s *Server) setupSubscriptions() error {
	topic := s.topic
	if topic == "" {
		topic = "position/cleaned"
	}

	if err := s.server.Subscribe(topic, 0, func(cl *mqtt.Client, sub packets.Subscription, pk packets.Packet) {
		if s.positionHandler != nil {
			s.positionHandler(cl, sub, pk)
		}
	}); err != nil {
		return err
	}
	slog.Info("MQTT 主题已订阅", "topic", topic)
	return nil
}

// SetPositionHandler 设置定位数据处理回调。
func (s *Server) SetPositionHandler(handler func(cl *mqtt.Client, sub packets.Subscription, pk packets.Packet)) {
	s.positionHandler = handler
}
