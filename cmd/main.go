package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/rainbrookx/hazard-chem-position-alert/internal/engine"
	"github.com/rainbrookx/hazard-chem-position-alert/internal/infrastructure/config"
	"github.com/rainbrookx/hazard-chem-position-alert/internal/infrastructure/database"
	"github.com/rainbrookx/hazard-chem-position-alert/internal/model"
	"github.com/rainbrookx/hazard-chem-position-alert/internal/server/grpc"
	"github.com/rainbrookx/hazard-chem-position-alert/internal/server/http"
	"github.com/rainbrookx/hazard-chem-position-alert/internal/server/mqtt"
	mqtthandler "github.com/rainbrookx/hazard-chem-position-alert/internal/server/mqtt/handler"
	"github.com/rainbrookx/hazard-chem-position-alert/internal/service"

	alertv1 "github.com/rainbrookx/hazard-chem-position-alert/api/grpc/alert/v1"
	alertqueryv1 "github.com/rainbrookx/hazard-chem-position-alert/api/grpc/alert_query/v1"
	grpclib "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	// 1. 加载配置
	cfg, err := config.Load("")
	if err != nil {
		slog.Error("加载配置失败", "error", err)
		os.Exit(1)
	}

	// 2. 初始化数据库
	db, err := database.InitDatabase(cfg.Database)
	if err != nil {
		slog.Error("数据库初始化失败", "error", err)
		os.Exit(1)
	}

	// 3. AutoMigrate 数据模型
	if err := db.AutoMigrate(&model.User{}, &model.AlertRecord{}); err != nil {
		slog.Error("数据库迁移失败", "error", err)
		os.Exit(1)
	}

	// 3a. 生成默认管理员用户
	if err := http.SeedDefaultUser(db, cfg.AlertEngine.DefaultUsername, cfg.AlertEngine.DefaultPassword); err != nil {
		slog.Error("创建默认用户失败", "error", err)
		os.Exit(1)
	}

	// 3b. 生成 RSA 密钥对
	rsaBits := cfg.AlertEngine.RSAKeyBits
	if rsaBits <= 0 {
		rsaBits = 2048
	}
	rsaKey, err := http.GenerateRSAKeyPair(rsaBits)
	if err != nil {
		slog.Error("生成 RSA 密钥失败", "error", err)
		os.Exit(1)
	}

	// 4. 创建围栏同步 gRPC 客户端（连接 Data Preprocessor）
	dpConn, err := grpclib.NewClient("localhost:9091",
		grpclib.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		slog.Error("连接 Data Preprocessor gRPC 失败", "error", err)
		os.Exit(1)
	}
	fenceClient := service.NewGRPCFenceSyncClient(dpConn)

	// 5. 创建围栏缓存和引擎
	fenceStore := engine.NewFenceStore(fenceClient)

	// 6. 启动时拉取外部围栏
	ctx := context.Background()
	if err := fenceStore.PullFromDataPreprocessor(ctx); err != nil {
		slog.Warn("启动时拉取外部围栏失败（将使用空围栏配置）", "error", err)
	}

	// 7. 创建告警引擎
	eng := engine.New(cfg.AlertEngine, db, fenceStore)

	// 7a. 注册所有 7 种告警规则
	eng.AddRule(engine.NewBoundaryRule(cfg.AlertEngine.DebounceSeconds))
	eng.AddRule(engine.NewStationaryRule())
	eng.AddRule(engine.NewOvercrowdingRule())
	eng.AddRule(engine.NewUnderstaffingRule())
	eng.AddRule(engine.NewLoiteringRule())
	eng.AddRule(engine.NewOneKeyRule(cfg.AlertEngine.OneKeyCooldownSeconds))
	eng.AddRule(engine.NewGatheringRule(cfg.AlertEngine.GridCellMeters))

	// 7b. 创建 gRPC 服务
	alertPushSvc := service.NewAlertPushService()
	alertQuerySvc := service.NewAlertQueryService(eng.AlertStore())

	// 7c. 连接引擎广播到 gRPC 推送服务
	eng.SetBroadcastFn(func(event engine.AlertEvent) {
		alertPushSvc.Broadcast(event)
	})

	// 7d. 创建 HTTP handlers
	jwtSecret := cfg.AlertEngine.JWTSecret
	if jwtSecret == "" {
		jwtSecret = "default-secret-change-in-production"
	}
	userHandler := http.NewUserHandler(db, rsaKey, jwtSecret)
	terminalHandler := http.NewTerminalHandler(eng)
	fenceHandler := http.NewFenceHandler(fenceStore)
	alertHandler := http.NewAlertHandler(eng.AlertStore())

	// 8. 创建服务实例
	mqttServer := mqtt.New(cfg.MochiMQTT, cfg.AlertEngine.MQTTTopic)
	httpServer := http.New(cfg.HTTPServer, &http.Dependencies{
		DB:              db,
		JWTSecret:       jwtSecret,
		RSAKey:          rsaKey,
		UserHandler:     userHandler,
		TerminalHandler: terminalHandler,
		FenceHandler:    fenceHandler,
		AlertHandler:    alertHandler,
	})
	grpcServer := grpc.New(cfg.GRPCServer)

	// 8a. 注册 gRPC 服务
	alertv1.RegisterAlertPushServiceServer(grpcServer.GRPCServer(), alertPushSvc)
	alertqueryv1.RegisterAlertQueryServiceServer(grpcServer.GRPCServer(), alertQuerySvc)

	// 9. 确保退出时关闭所有资源（后进先出）
	defer grpcServer.Close()
	defer httpServer.Close()
	defer mqttServer.Close()
	defer dpConn.Close()

	// 10. 启动各服务
	if err := mqttServer.Start(); err != nil {
		slog.Error("MQTT 服务启动失败", "error", err)
		os.Exit(1)
	}
	httpServer.Start()
	grpcServer.Start()

	// 11. 启动告警引擎
	eng.Start(ctx)

	// 12. 连接 MQTT 定位数据处理器到引擎
	posHandler := mqtthandler.NewPositionHandler(eng)
	mqttServer.SetPositionHandler(posHandler.Handle)

	slog.Info("全部服务已启动",
		"mqtt", cfg.MochiMQTT.Address,
		"http", cfg.HTTPServer.Port,
		"grpc", cfg.GRPCServer.Port,
	)

	// 13. 统一等待信号或运行时错误
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

	// 14. 优雅关闭引擎
	eng.Stop()
}
