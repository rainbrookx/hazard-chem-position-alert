package engine

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/rainbrookx/hazard-chem-position-alert/internal/model"
)

// FenceStore 围栏配置缓存，管理外部同步和本地创建的围栏。
// 本地围栏优先级高于外部围栏（同 zone_id 时本地覆盖外部）。
type FenceStore struct {
	mu          sync.RWMutex
	external    map[string]model.Geofence // SOURCE_EXTERNAL 围栏
	local       map[string]model.Geofence // SOURCE_LOCAL 围栏
	fenceClient FenceSyncClient           // gRPC 客户端
}

// FenceSyncClient 围栏同步 gRPC 客户端接口（便于测试 mock）。
type FenceSyncClient interface {
	PullExternalFences(ctx context.Context) ([]model.Geofence, int64, error)
	NotifyLocalChange(ctx context.Context, zoneID string, version int64) error
	Close() error
}

// NewFenceStore 创建围栏配置缓存。
func NewFenceStore(client FenceSyncClient) *FenceStore {
	return &FenceStore{
		external:    make(map[string]model.Geofence),
		local:       make(map[string]model.Geofence),
		fenceClient: client,
	}
}

// LoadFromExternal 从外部加载围栏配置，合并到缓存。
// 外部围栏不覆盖同 zone_id 的本地围栏（SOURCE_LOCAL 优先级更高）。
func (fs *FenceStore) LoadFromExternal(fences []model.Geofence) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	for _, f := range fences {
		f.Source = model.FenceSourceExternal
		// 不覆盖本地围栏
		if _, exists := fs.local[f.ZoneID]; exists {
			slog.Debug("外部围栏因本地围栏存在被跳过", "zone_id", f.ZoneID)
			continue
		}
		fs.external[f.ZoneID] = f
	}
	slog.Info("外部围栏配置已加载", "count", len(fences))
}

// AddLocal 添加或更新本地围栏。
func (fs *FenceStore) AddLocal(fence model.Geofence) {
	fence.Source = model.FenceSourceLocal
	fs.mu.Lock()
	defer fs.mu.Unlock()

	// 从外部缓存中移除同 zone_id 的围栏（本地优先）
	delete(fs.external, fence.ZoneID)
	fs.local[fence.ZoneID] = fence
	slog.Info("本地围栏已添加", "zone_id", fence.ZoneID)
}

// RemoveLocal 移除本地围栏。
func (fs *FenceStore) RemoveLocal(zoneID string) {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	delete(fs.local, zoneID)
	slog.Info("本地围栏已移除", "zone_id", zoneID)
}

// GetAll 返回所有合并后的围栏列表（本地优先）。
func (fs *FenceStore) GetAll() []model.Geofence {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	result := make([]model.Geofence, 0, len(fs.local)+len(fs.external))
	for _, f := range fs.local {
		result = append(result, f)
	}
	for zoneID, f := range fs.external {
		if _, ok := fs.local[zoneID]; !ok {
			result = append(result, f)
		}
	}
	return result
}

// HasFences 检查是否有任何围栏配置。
func (fs *FenceStore) HasFences() bool {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	return len(fs.local) > 0 || len(fs.external) > 0
}

// GetActive 返回所有启用且有效的围栏（type 为 FORBIDDEN/RESTRICTED 且 is_active）。
func (fs *FenceStore) GetActive() []model.Geofence {
	all := fs.GetAll()
	active := make([]model.Geofence, 0, len(all))
	for _, f := range all {
		if f.IsActive && f.Type != model.FenceTypeUnspecified && len(f.Vertices) >= 3 {
			active = append(active, f)
		}
	}
	return active
}

// GetByID 根据 zone_id 获取围栏。
func (fs *FenceStore) GetByID(zoneID string) (model.Geofence, bool) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	if f, ok := fs.local[zoneID]; ok {
		return f, true
	}
	if f, ok := fs.external[zoneID]; ok {
		return f, true
	}
	return model.Geofence{}, false
}

// PullFromDataPreprocessor 通过 gRPC 从 Data Preprocessor 拉取外部围栏。
func (fs *FenceStore) PullFromDataPreprocessor(ctx context.Context) error {
	if fs.fenceClient == nil {
		return fmt.Errorf("fence client is nil, cannot pull fences")
	}

	fences, version, err := fs.fenceClient.PullExternalFences(ctx)
	if err != nil {
		return fmt.Errorf("pull external fences failed: %w", err)
	}

	fs.LoadFromExternal(fences)
	slog.Info("从 Data Preprocessor 拉取围栏完成", "count", len(fences), "version", version)
	return nil
}

// NotifyDataPreprocessor 通知 Data Preprocessor 本地围栏变更。
func (fs *FenceStore) NotifyDataPreprocessor(ctx context.Context, zoneID string) error {
	if fs.fenceClient == nil {
		return fmt.Errorf("fence client is nil, cannot notify")
	}
	return fs.fenceClient.NotifyLocalChange(ctx, zoneID, time.Now().UnixMilli())
}
