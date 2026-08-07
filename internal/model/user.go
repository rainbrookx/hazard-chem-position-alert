package model

// User GORM 持久化模型。
type User struct {
	ID           uint   `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Username     string `gorm:"column:username;unique;not null;size:64" json:"username"`
	PasswordHash string `gorm:"column:password_hash;not null" json:"-"` // bcrypt hash, never serialize to JSON
	CreatedAt    int64  `gorm:"column:created_at;autoCreateTime:milli" json:"created_at"`
	UpdatedAt    int64  `gorm:"column:updated_at;autoUpdateTime:milli" json:"updated_at"`
}

// TableName 指定 GORM 表名。
func (User) TableName() string {
	return "users"
}
