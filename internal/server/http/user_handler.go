package http

import (
	"encoding/base64"
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/rainbrookx/hazard-chem-position-alert/internal/model"
)

// UserHandler 用户登录/密码管理 HTTP handler。
type UserHandler struct {
	db        *gorm.DB
	rsaKey    *RSAKeyPair
	jwtSecret string
}

// NewUserHandler 创建用户处理器。
func NewUserHandler(db *gorm.DB, rsaKey *RSAKeyPair, jwtSecret string) *UserHandler {
	return &UserHandler{db: db, rsaKey: rsaKey, jwtSecret: jwtSecret}
}

type loginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type tokenResponse struct {
	Token    string `json:"token"`
	Username string `json:"username"`
}

// Login POST /api/login — RSA 解密 → bcrypt 验证 → 签发 JWT。
func (h *UserHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	// RSA base64 解码 + 解密
	cipherBytes, err := base64.StdEncoding.DecodeString(req.Username)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid base64 encoding"})
		return
	}
	usernameBytes, err := h.rsaKey.DecryptOAEP(cipherBytes)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "decrypt failed"})
		return
	}

	cipherBytes, err = base64.StdEncoding.DecodeString(req.Password)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid base64 encoding"})
		return
	}
	passwordBytes, err := h.rsaKey.DecryptOAEP(cipherBytes)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "decrypt failed"})
		return
	}

	username := string(usernameBytes)
	password := string(passwordBytes)

	// 查询用户
	var user model.User
	if err := h.db.Where("username = ?", username).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	// bcrypt 验证
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	// 签发 JWT
	token, err := GenerateToken(username, h.jwtSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "token generation failed"})
		return
	}

	c.JSON(http.StatusOK, tokenResponse{Token: token, Username: username})
}

// Refresh POST /api/refresh — 续签 JWT。
func (h *UserHandler) Refresh(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" || len(authHeader) < 8 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
		return
	}
	tokenStr := authHeader[7:] // 去掉 "Bearer "

	newToken, err := RefreshToken(tokenStr, h.jwtSecret)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "refresh failed"})
		return
	}
	c.JSON(http.StatusOK, tokenResponse{Token: newToken})
}

// PublicKey GET /api/public-key — 返回 RSA 公钥 PEM。
func (h *UserHandler) PublicKey(c *gin.Context) {
	pem, err := h.rsaKey.PublicKeyPEM()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "key generation failed"})
		return
	}
	c.String(http.StatusOK, pem)
}

type changePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required"`
}

// ChangePassword PUT /api/user/password — 修改密码。
func (h *UserHandler) ChangePassword(c *gin.Context) {
	username := c.GetString("username")
	if username == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}

	var req changePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	var user model.User
	if err := h.db.Where("username = ?", username).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.OldPassword)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "old password incorrect"})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), 10)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "password hashing failed"})
		return
	}

	if err := h.db.Model(&user).Update("password_hash", string(hash)).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "update failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "password updated"})
}

// SeedDefaultUser 在首次启动时创建默认管理员用户。
func SeedDefaultUser(db *gorm.DB, username, password string) error {
	var count int64
	db.Model(&model.User{}).Count(&count)
	if count > 0 {
		return nil // 用户已存在
	}

	if username == "" {
		username = "admin"
	}
	if password == "" {
		password = "admin123"
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), 10)
	if err != nil {
		return err
	}

	return db.Create(&model.User{
		Username:     username,
		PasswordHash: string(hash),
	}).Error
}
