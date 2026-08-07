package service

import (
	"context"

	fencev1 "github.com/rainbrookx/hazard-chem-position-alert/api/grpc/fence/v1"
	"github.com/rainbrookx/hazard-chem-position-alert/internal/engine"
	"github.com/rainbrookx/hazard-chem-position-alert/internal/engine/geom"
	"github.com/rainbrookx/hazard-chem-position-alert/internal/model"
	"google.golang.org/grpc"
)

// grpcFenceSyncClient 是 FenceSyncClient 接口的具体 gRPC 实现。
type grpcFenceSyncClient struct {
	client fencev1.FenceSyncServiceClient
	conn   *grpc.ClientConn
}

// NewGRPCFenceSyncClient 创建基于 gRPC 的围栏同步客户端。
func NewGRPCFenceSyncClient(conn *grpc.ClientConn) engine.FenceSyncClient {
	return &grpcFenceSyncClient{
		client: fencev1.NewFenceSyncServiceClient(conn),
		conn:   conn,
	}
}

func (c *grpcFenceSyncClient) PullExternalFences(ctx context.Context) ([]model.Geofence, int64, error) {
	resp, err := c.client.PullAllFences(ctx, &fencev1.PullAllFencesRequest{
		EngineId: "alert-engine",
	})
	if err != nil {
		return nil, 0, err
	}

	fences := make([]model.Geofence, 0, len(resp.Fences))
	for _, pf := range resp.Fences {
		f := convertProtoFenceToModel(pf)
		fences = append(fences, f)
	}
	return fences, resp.Version, nil
}

func (c *grpcFenceSyncClient) NotifyLocalChange(ctx context.Context, zoneID string, version int64) error {
	_, err := c.client.NotifyFenceChange(ctx, &fencev1.NotifyFenceChangeRequest{
		EngineId:      "alert-engine",
		NewVersion:    version,
		ChangedZoneId: zoneID,
	})
	return err
}

func (c *grpcFenceSyncClient) Close() error {
	return c.conn.Close()
}

// convertProtoFenceToModel 将 protobuf Geofence 转换为领域模型 Geofence。
func convertProtoFenceToModel(pf *fencev1.Geofence) model.Geofence {
	vertices := make([]geom.Point, 0, len(pf.Vertices))
	for _, v := range pf.Vertices {
		vertices = append(vertices, geom.Point{X: v.X, Y: v.Y})
	}

	fenceType := model.FenceTypeUnspecified
	switch pf.Type {
	case fencev1.GeofenceType_GEOFENCE_TYPE_FORBIDDEN:
		fenceType = model.FenceTypeForbidden
	case fencev1.GeofenceType_GEOFENCE_TYPE_RESTRICTED:
		fenceType = model.FenceTypeRestricted
	case fencev1.GeofenceType_GEOFENCE_TYPE_SAFE:
		fenceType = model.FenceTypeSafe
	}

	return model.Geofence{
		ZoneID:                    pf.ZoneId,
		Name:                      pf.Name,
		Type:                      fenceType,
		Source:                    model.FenceSourceExternal,
		Vertices:                  vertices,
		MaxPeople:                 pf.MaxPeople,
		MinPeople:                 pf.MinPeople,
		MaxStaySeconds:            pf.MaxStaySeconds,
		StationarySeconds:         pf.StationarySeconds,
		StationaryThresholdMeters: pf.StationaryThresholdMeters,
		StationaryRecoverySeconds: pf.StationaryRecoverySeconds,
		RequiredPersonIDs:         pf.RequiredPersonIds,
		GridCellMeters:            pf.GridCellMeters,
		IsActive:                  pf.IsActive,
	}
}
