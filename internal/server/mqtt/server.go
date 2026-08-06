// Package mqtt 提供内嵌 Mochi MQTT Broker 的极简启动封装
package mqtt

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	mqtt "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/hooks/auth"
	"github.com/mochi-mqtt/server/v2/listeners"
	"github.com/rainbrookx/hazard-chem-position-alert/internal/infrastructure/config"
	"github.com/rainbrookx/hazard-chem-position-alert/internal/server/mqtt/handler"
)

// Server 封装 MQTT Broker
type Server struct {
	server *mqtt.Server
	cfg    config.MochiMQTTConfig
}

// New 创建 MQTT 服务实例（不启动）
func New(cfg config.MochiMQTTConfig) *Server {
	opts := &mqtt.Options{InlineClient: true}
	s := mqtt.New(opts)

	return &Server{
		server: s,
		cfg:    cfg,
	}
}

// Run 启动 MQTT Broker 并阻塞，直到收到 SIGINT/SIGTERM 信号或发生致命错误。
// Run 启动 MQTT Broker 并阻塞，直到收到 SIGINT/SIGTERM 信号。
func (s *Server) Run() error {
	if err := s.server.AddHook(new(auth.AllowHook), nil); err != nil {
		return fmt.Errorf("添加认证钩子失败: %w", err)
	}

	// 添加 TCP 监听器
	tcp := listeners.NewTCP(listeners.Config{Address: s.cfg.Address})
	if err := s.server.AddListener(tcp); err != nil {
		return fmt.Errorf("添加 TCP 监听器失败: %w", err)
	}

	// 设置订阅（可扩展）
	if err := s.setupSubscriptions(); err != nil {
		return fmt.Errorf("订阅主题失败: %w", err)
	}

	// 启动broker，Serve启动完成立刻返回nil，不阻塞
	err := s.server.Serve()
	if err != nil {
		return fmt.Errorf("mochi mqtt serve启动失败: %w", err)
	}
	log.Println("MQTT Broker 已启动完成")

	// 自己阻塞等待关闭信号
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// 等待退出信号
	sig := <-sigCh
	log.Printf("收到信号 %v，正在关闭 MQTT Broker...", sig)

	// 优雅关闭broker
	if err := s.server.Close(); err != nil {
		log.Printf("关闭 MQTT 服务器时发生错误: %v", err)
	}

	log.Println("MQTT Broker 已安全关闭")
	return nil
}

// setupSubscriptions 可在此注册默认订阅逻辑
func (s *Server) setupSubscriptions() error {
	// 示例：订阅通配主题
	_ = s.server.Subscribe("test", 0, handler.CallbackFn)
	return nil
}
