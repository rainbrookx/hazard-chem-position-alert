package engine

import (
	"time"

	alertv1 "github.com/rainbrookx/hazard-chem-position-alert/api/grpc/alert/v1"
	"github.com/rainbrookx/hazard-chem-position-alert/internal/model"
	"gorm.io/gorm"
)

// AlertStore 告警持久化存储，封装 GORM 操作。
type AlertStore struct {
	db *gorm.DB
}

// NewAlertStore 创建告警存储实例并自动迁移。
func NewAlertStore(db *gorm.DB) *AlertStore {
	return &AlertStore{db: db}
}

// AutoMigrate 自动迁移数据库表结构。
func (as *AlertStore) AutoMigrate() error {
	return as.db.AutoMigrate(&model.AlertRecord{})
}

// Insert 插入一条告警记录。
func (as *AlertStore) Insert(record *model.AlertRecord) error {
	return as.db.Create(record).Error
}

// QueryResult 查询结果。
type QueryResult struct {
	Records []model.AlertRecord
	Total   int64
}

// QueryHistory 查询历史告警记录（支持类型/时间/区域过滤 + 分页）。
func (as *AlertStore) QueryHistory(types []alertv1.AlertType, startTimeMs, endTimeMs int64, zoneID string, limit, offset int) (*QueryResult, error) {
	if limit <= 0 || limit > 1000 {
		limit = 100
	}

	tx := as.db.Model(&model.AlertRecord{})

	if len(types) > 0 {
		typeInts := make([]int32, len(types))
		for i, t := range types {
			typeInts[i] = int32(t)
		}
		tx = tx.Where("alert_type IN ?", typeInts)
	}
	if startTimeMs > 0 {
		tx = tx.Where("trigger_time_ms >= ?", startTimeMs)
	}
	if endTimeMs > 0 {
		tx = tx.Where("trigger_time_ms <= ?", endTimeMs)
	}
	if zoneID != "" {
		tx = tx.Where("zone_id = ?", zoneID)
	}

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, err
	}

	var records []model.AlertRecord
	if err := tx.Order("trigger_time_ms DESC").Limit(limit).Offset(offset).Find(&records).Error; err != nil {
		return nil, err
	}

	return &QueryResult{
		Records: records,
		Total:   total,
	}, nil
}

// ListActive 查询最近一段时间内的告警记录（模拟活跃告警）。
func (as *AlertStore) ListActive(since time.Duration, types []alertv1.AlertType) ([]model.AlertRecord, error) {
	sinceMs := time.Now().Add(-since).UnixMilli()

	tx := as.db.Model(&model.AlertRecord{}).Where("trigger_time_ms >= ?", sinceMs)

	if len(types) > 0 {
		typeInts := make([]int32, len(types))
		for i, t := range types {
			typeInts[i] = int32(t)
		}
		tx = tx.Where("alert_type IN ?", typeInts)
	}

	var records []model.AlertRecord
	if err := tx.Order("trigger_time_ms DESC").Find(&records).Error; err != nil {
		return nil, err
	}
	return records, nil
}

// Close 关闭数据库连接。
func (as *AlertStore) Close() error {
	sqlDB, err := as.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
