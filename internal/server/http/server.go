// Package http HTTP 服务端（供 Web 前端调用）
package http

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rainbrookx/hazard-chem-position-alert/internal/infrastructure/config"
	"gorm.io/gorm"
)

// Server 封装 HTTP 服务
type Server struct {
	server    *http.Server
	cfg       config.HTTPServerConfig
	errCh     chan error
	closeOnce sync.Once
	router    *gin.Engine
}

// Dependencies HTTP 服务所需的外部依赖。
type Dependencies struct {
	DB              *gorm.DB
	JWTSecret       string
	RSAKey          *RSAKeyPair
	UserHandler     *UserHandler
	TerminalHandler *TerminalHandler
	FenceHandler    *FenceHandler
	AlertHandler    *AlertHandler
}

// New 创建 HTTP 服务实例（不启动）
func New(cfg config.HTTPServerConfig, deps *Dependencies) *Server {
	router := gin.Default()

	// 公开路由
	router.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "pong"})
	})

	if deps != nil && deps.UserHandler != nil {
		// 公开认证路由
		router.POST("/api/login", deps.UserHandler.Login)
		router.POST("/api/refresh", deps.UserHandler.Refresh)
		router.GET("/api/public-key", deps.UserHandler.PublicKey)

		// 受保护路由组
		protected := router.Group("/api")
		protected.Use(JWTAuthMiddleware(deps.JWTSecret))
		{
			protected.PUT("/user/password", deps.UserHandler.ChangePassword)

			if deps.TerminalHandler != nil {
				protected.GET("/terminals", deps.TerminalHandler.GetTerminals)
			}
			if deps.FenceHandler != nil {
				protected.GET("/fences", deps.FenceHandler.GetFences)
				protected.POST("/fences", deps.FenceHandler.CreateFence)
				protected.PUT("/fences/:id", deps.FenceHandler.UpdateFence)
				protected.DELETE("/fences/:id", deps.FenceHandler.DeleteFence)
			}
			if deps.AlertHandler != nil {
				protected.GET("/alerts/active", deps.AlertHandler.GetActiveAlerts)
				protected.POST("/alerts/history", deps.AlertHandler.QueryHistory)
			}
		}
	}

	return &Server{
		server: &http.Server{
			Addr:    cfg.Port,
			Handler: router,
		},
		cfg:    cfg,
		errCh:  make(chan error, 1),
		router: router,
	}
}

// ErrCh 返回运行时错误通道，供调用方监听
func (s *Server) ErrCh() <-chan error {
	return s.errCh
}

// Start 启动 HTTP 服务（非阻塞）。
func (s *Server) Start() {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				s.errCh <- fmt.Errorf("HTTP 服务运行时 panic: %v", r)
			}
		}()
		slog.Info("HTTP 服务已启动", "addr", s.cfg.Port)
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.errCh <- fmt.Errorf("HTTP 服务错误: %w", err)
		}
	}()
}

// Close 优雅关闭 HTTP 服务（幂等）。
func (s *Server) Close() {
	s.closeOnce.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.server.Shutdown(ctx); err != nil {
			slog.Error("HTTP 服务关闭错误", "error", err)
		}
		slog.Info("HTTP 服务已关闭")
	})
}
