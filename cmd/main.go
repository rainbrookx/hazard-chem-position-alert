package main

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/rainbrookx/hazard-chem-position-alert/internal/infrastructure/config"
	"github.com/rainbrookx/hazard-chem-position-alert/internal/server/grpc"
	"github.com/rainbrookx/hazard-chem-position-alert/internal/server/http"
	"github.com/rainbrookx/hazard-chem-position-alert/internal/server/mqtt"
)

func main() {
	// 1. 加载配置
	cfg, err := config.Load("")
	if err != nil {
		slog.Error("加载配置失败", "error", err)
		os.Exit(1)
	}

	// 2. 创建服务实例（不启动）
	mqttServer := mqtt.New(cfg.MochiMQTT)
	httpServer := http.New(cfg.HTTPServer)
	grpcServer := grpc.New(cfg.GRPCServer)

	// 3. 确保退出时关闭所有资源（后进先出）
	defer grpcServer.Close()
	defer httpServer.Close()
	defer mqttServer.Close()

	// 4. 启动各服务
	if err := mqttServer.Start(); err != nil {
		slog.Error("MQTT 服务启动失败", "error", err)
		os.Exit(1)
	}
	httpServer.Start()
	grpcServer.Start()

	slog.Info("全部服务已启动",
		"mqtt", cfg.MochiMQTT.Address,
		"http", cfg.HTTPServer.Port,
		"grpc", cfg.GRPCServer.Port,
	)

	// 5. 统一等待信号或运行时错误
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigCh:
		slog.Info("收到信号，正在优雅关闭所有服务", "signal", sig)
	case err := <-mqttServer.ErrCh():
		slog.Error("MQTT 服务异常退出", "error", err)
	case err := <-httpServer.ErrCh():
		slog.Error("HTTP 服务异常退出", "error", err)
	case err := <-grpcServer.ErrCh():
		slog.Error("gRPC 服务异常退出", "error", err)
	}
}
