package http

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rainbrookx/hazard-chem-position-alert/internal/engine"
	"github.com/rainbrookx/hazard-chem-position-alert/internal/engine/geom"
	"github.com/rainbrookx/hazard-chem-position-alert/internal/model"
)

// FenceHandler 电子围栏管理 HTTP handler。
type FenceHandler struct {
	fenceStore *engine.FenceStore
}

// NewFenceHandler 创建围栏管理处理器。
func NewFenceHandler(fenceStore *engine.FenceStore) *FenceHandler {
	return &FenceHandler{fenceStore: fenceStore}
}

// fenceJSON 围栏 JSON 序列化格式。
type fenceJSON struct {
	ZoneID                    string      `json:"zone_id"`
	Name                      string      `json:"name"`
	Type                      int32       `json:"type"`
	Source                    int32       `json:"source"`
	Vertices                  []pointJSON `json:"vertices"`
	MaxPeople                 int32       `json:"max_people"`
	MinPeople                 int32       `json:"min_people"`
	MaxStaySeconds            int32       `json:"max_stay_seconds"`
	StationarySeconds         int32       `json:"stationary_seconds"`
	StationaryThresholdMeters float64     `json:"stationary_threshold_meters"`
	StationaryRecoverySeconds int32       `json:"stationary_recovery_seconds"`
	RequiredPersonIDs         []string    `json:"required_person_ids"`
	GridCellMeters            int32       `json:"grid_cell_meters"`
	IsActive                  bool        `json:"is_active"`
	Version                   int64       `json:"version"`
}

type pointJSON struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// GetFences GET /api/fences — 返回所有围栏。
func (h *FenceHandler) GetFences(c *gin.Context) {
	fences := h.fenceStore.GetAll()
	result := make([]fenceJSON, 0, len(fences))
	for _, f := range fences {
		result = append(result, modelToFenceJSON(f))
	}
	c.JSON(http.StatusOK, gin.H{"fences": result, "count": len(result)})
}

// CreateFence POST /api/fences — 创建/更新本地围栏。
func (h *FenceHandler) CreateFence(c *gin.Context) {
	var req fenceJSON
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	fence := fenceJSONToModel(req)
	fence.Source = model.FenceSourceLocal
	h.fenceStore.AddLocal(fence)

	// 通知 Data Preprocessor
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = h.fenceStore.NotifyDataPreprocessor(ctx, fence.ZoneID)

	c.JSON(http.StatusOK, gin.H{"message": "fence created", "zone_id": fence.ZoneID})
}

// UpdateFence PUT /api/fences/:id — 更新本地围栏。
func (h *FenceHandler) UpdateFence(c *gin.Context) {
	zoneID := c.Param("id")
	existing, found := h.fenceStore.GetByID(zoneID)
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "fence not found"})
		return
	}
	if existing.Source == model.FenceSourceExternal {
		c.JSON(http.StatusForbidden, gin.H{"error": "cannot modify external fence"})
		return
	}

	var req fenceJSON
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	fence := fenceJSONToModel(req)
	fence.ZoneID = zoneID
	fence.Source = model.FenceSourceLocal
	h.fenceStore.AddLocal(fence)

	c.JSON(http.StatusOK, gin.H{"message": "fence updated"})
}

// DeleteFence DELETE /api/fences/:id — 删除本地围栏。
func (h *FenceHandler) DeleteFence(c *gin.Context) {
	zoneID := c.Param("id")
	existing, found := h.fenceStore.GetByID(zoneID)
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "fence not found"})
		return
	}
	if existing.Source == model.FenceSourceExternal {
		c.JSON(http.StatusForbidden, gin.H{"error": "cannot delete external fence"})
		return
	}

	h.fenceStore.RemoveLocal(zoneID)
	c.JSON(http.StatusOK, gin.H{"message": "fence deleted"})
}

func modelToFenceJSON(f model.Geofence) fenceJSON {
	verts := make([]pointJSON, 0, len(f.Vertices))
	for _, v := range f.Vertices {
		verts = append(verts, pointJSON{X: v.X, Y: v.Y})
	}
	return fenceJSON{
		ZoneID:                    f.ZoneID,
		Name:                      f.Name,
		Type:                      int32(f.Type),
		Source:                    int32(f.Source),
		Vertices:                  verts,
		MaxPeople:                 f.MaxPeople,
		MinPeople:                 f.MinPeople,
		MaxStaySeconds:            f.MaxStaySeconds,
		StationarySeconds:         f.StationarySeconds,
		StationaryThresholdMeters: f.StationaryThresholdMeters,
		StationaryRecoverySeconds: f.StationaryRecoverySeconds,
		RequiredPersonIDs:         f.RequiredPersonIDs,
		GridCellMeters:            f.GridCellMeters,
		IsActive:                  f.IsActive,
		Version:                   f.Version,
	}
}

func fenceJSONToModel(j fenceJSON) model.Geofence {
	verts := make([]geom.Point, 0, len(j.Vertices))
	for _, v := range j.Vertices {
		verts = append(verts, geom.Point{X: v.X, Y: v.Y})
	}
	return model.Geofence{
		ZoneID:                    j.ZoneID,
		Name:                      j.Name,
		Type:                      model.FenceType(j.Type),
		Vertices:                  verts,
		MaxPeople:                 j.MaxPeople,
		MinPeople:                 j.MinPeople,
		MaxStaySeconds:            j.MaxStaySeconds,
		StationarySeconds:         j.StationarySeconds,
		StationaryThresholdMeters: j.StationaryThresholdMeters,
		StationaryRecoverySeconds: j.StationaryRecoverySeconds,
		RequiredPersonIDs:         j.RequiredPersonIDs,
		GridCellMeters:            j.GridCellMeters,
		IsActive:                  j.IsActive,
		Version:                   j.Version,
	}
}
