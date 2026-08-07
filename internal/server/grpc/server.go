// Package grpc gRPC 服务端（供其他后端服务调用）
package grpc

import (
	"fmt"
	"log/slog"
	"net"
	"sync"

	"google.golang.org/grpc"

	pb "github.com/rainbrookx/hazard-chem-position-alert/api/grpc/health/v1"
	"github.com/rainbrookx/hazard-chem-position-alert/internal/infrastructure/config"
)

// Server 封装 gRPC 服务
type Server struct {
	server    *grpc.Server
	cfg       config.GRPCServerConfig
	errCh     chan error
	closeOnce sync.Once
}

// New 创建 gRPC 服务实例（不启动），默认注册健康检查服务。
func New(cfg config.GRPCServerConfig) *Server {
	s := grpc.NewServer()

	// 注册健康检查服务（技术验证用）
	pb.RegisterHealthServer(s, &HealthService{})

	return &Server{
		server: s,
		cfg:    cfg,
		errCh:  make(chan error, 1),
	}
}

// GRPCServer 暴露内部的 *grpc.Server，供外部注册服务
func (s *Server) GRPCServer() *grpc.Server {
	return s.server
}

// ErrCh 返回运行时错误通道，供调用方监听
func (s *Server) ErrCh() <-chan error {
	return s.errCh
}

// Start 启动 gRPC 服务（非阻塞）。
// 运行时错误通过 ErrCh() 上报。
func (s *Server) Start() {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				s.errCh <- fmt.Errorf("gRPC 服务运行时 panic: %v", r)
			}
		}()

		lis, err := net.Listen("tcp", s.cfg.Port)
		if err != nil {
			s.errCh <- fmt.Errorf("gRPC 监听失败: %w", err)
			return
		}

		slog.Info("gRPC 服务已启动", "addr", s.cfg.Port)
		if err := s.server.Serve(lis); err != nil {
			s.errCh <- fmt.Errorf("gRPC 服务错误: %w", err)
		}
	}()
}

// Close 优雅关闭 gRPC 服务（幂等）。
func (s *Server) Close() {
	s.closeOnce.Do(func() {
		s.server.GracefulStop()
		slog.Info("gRPC 服务已关闭")
	})
}
