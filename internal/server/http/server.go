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
)

// Server 封装 HTTP 服务
type Server struct {
	server    *http.Server
	cfg       config.HTTPServerConfig
	errCh     chan error
	closeOnce sync.Once
}

// New 创建 HTTP 服务实例（不启动）
func New(cfg config.HTTPServerConfig) *Server {
	router := gin.Default()

	// --- 注册路由 ---
	router.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "pong"})
	})

	return &Server{
		server: &http.Server{
			Addr:    cfg.Port,
			Handler: router,
		},
		cfg:   cfg,
		errCh: make(chan error, 1),
	}
}

// ErrCh 返回运行时错误通道，供调用方监听
func (s *Server) ErrCh() <-chan error {
	return s.errCh
}

// Start 启动 HTTP 服务（非阻塞）。
// 运行时错误通过 ErrCh() 上报。
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

// Close 优雅关闭 HTTP 服务（幂等），等待现有请求在 5 秒内完成。
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
