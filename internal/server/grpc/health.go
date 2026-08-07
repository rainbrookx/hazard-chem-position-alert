package grpc

import (
	"context"

	pb "github.com/rainbrookx/hazard-chem-position-alert/api/grpc/health/v1"
)

// HealthService 健康检查服务实现
type HealthService struct {
	pb.UnimplementedHealthServer
}

// Check 返回服务状态
func (s *HealthService) Check(ctx context.Context, req *pb.HealthCheckRequest) (*pb.HealthCheckResponse, error) {
	return &pb.HealthCheckResponse{
		Status:  "SERVING",
		Service: "hazard-chem-position-alert",
	}, nil
}
