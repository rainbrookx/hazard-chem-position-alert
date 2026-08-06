package main

import (
	"log"

	"github.com/rainbrookx/hazard-chem-position-alert/internal/infrastructure/config"
	"github.com/rainbrookx/hazard-chem-position-alert/internal/server/mqtt"
)

func main() {
	cfg, _ := config.Load("") // 加载配置

	log.Println("cfg.MochiMQTT.Address:", cfg.MochiMQTT.Address)

	srv := mqtt.New(cfg.MochiMQTT)

	// 启动并阻塞，直到收到 Ctrl+C 或发生错误
	if err := srv.Run(); err != nil {
		log.Fatalf("MQTT 服务异常退出: %v", err)
	}
}
