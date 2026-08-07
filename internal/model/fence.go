package model

import "github.com/rainbrookx/hazard-chem-position-alert/internal/engine/geom"

// FenceSource 围栏数据来源。
type FenceSource int32

const (
	FenceSourceUnspecified FenceSource = 0
	FenceSourceExternal    FenceSource = 1 // 第三方系统经 Data Preprocessor 同步
	FenceSourceLocal       FenceSource = 2 // 本项目 Web 页面本地新增/修改
)

// FenceType 围栏类型。
type FenceType int32

const (
	FenceTypeUnspecified FenceType = 0
	FenceTypeForbidden   FenceType = 1 // 禁止进入
	FenceTypeRestricted  FenceType = 2 // 受限区域（可进入但触发其他规则）
	FenceTypeSafe        FenceType = 3 // 安全区域（用于缺员/超员判定基准）
)

// Geofence 电子围栏领域模型。
// 来源: (1) Data Preprocessor gRPC 同步 (SOURCE_EXTERNAL), (2) Web 页面本地新增/修改 (SOURCE_LOCAL)。
type Geofence struct {
	ZoneID                    string       `json:"zone_id"`
	Name                      string       `json:"name"`
	Type                      FenceType    `json:"type"`
	Source                    FenceSource  `json:"source"`
	Vertices                  []geom.Point `json:"vertices"` // 多边形顶点（有序）
	MaxPeople                 int32        `json:"max_people"`
	MinPeople                 int32        `json:"min_people"`
	MaxStaySeconds            int32        `json:"max_stay_seconds"`
	StationarySeconds         int32        `json:"stationary_seconds"`
	StationaryThresholdMeters float64      `json:"stationary_threshold_meters"`
	StationaryRecoverySeconds int32        `json:"stationary_recovery_seconds"`
	RequiredPersonIDs         []string     `json:"required_person_ids"`
	GridCellMeters            int32        `json:"grid_cell_meters"`
	IsActive                  bool         `json:"is_active"`
	Version                   int64        `json:"version"` // 配置版本号
}
