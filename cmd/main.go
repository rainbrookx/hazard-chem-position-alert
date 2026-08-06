package main

import (
	"log/slog"

	"github.com/rainbrookx/hazard-chem-position-alert/internal/infrastructure/config"
	"github.com/rainbrookx/hazard-chem-position-alert/internal/server/mqtt"
)

func main() {
	cfg := func() *config.Config {
		cfg, err := config.Load("")
		if err != nil {
			slog.Error("加载配置错误", err)
		}
		return cfg
	}()

	mqttServer := mqtt.New(cfg.MochiMQTT)
	defer func(mqttServer *mqtt.Server) {
		mqttServer.Close()
	}(mqttServer) // 确保最终关闭资源（双重保障）

	if err := mqttServer.Run(); err != nil {
		slog.Error("MQTT 服务异常退出", err)
	}
}
