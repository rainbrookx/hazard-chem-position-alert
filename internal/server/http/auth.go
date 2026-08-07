package http

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"strings"
	"time"
)

// JWTClaims JWT payload.
type JWTClaims struct {
	Username string `json:"username"`
	Exp      int64  `json:"exp"`
	Iat      int64  `json:"iat"`
}

// GenerateToken 使用 stdlib crypto/hmac+sha256 生成 JWT token（30 分钟有效期）。
func GenerateToken(username, secret string) (string, error) {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))

	now := time.Now().Unix()
	claims := JWTClaims{
		Username: username,
		Exp:      now + 1800, // 30 min
		Iat:      now,
	}
	claimsJSON, _ := json.Marshal(claims)
	payload := base64.RawURLEncoding.EncodeToString(claimsJSON)

	signingInput := header + "." + payload
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signingInput))
	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	return signingInput + "." + signature, nil
}

// ValidateToken 验证 JWT token 有效性。
func ValidateToken(tokenStr, secret string) (*JWTClaims, error) {
	parts := strings.Split(tokenStr, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid token format")
	}

	signingInput := parts[0] + "." + parts[1]
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signingInput))
	expectedSig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(parts[2]), []byte(expectedSig)) {
		return nil, fmt.Errorf("invalid signature")
	}

	claimsJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid payload: %w", err)
	}

	var claims JWTClaims
	if err := json.Unmarshal(claimsJSON, &claims); err != nil {
		return nil, fmt.Errorf("invalid claims: %w", err)
	}

	if time.Now().Unix() > claims.Exp {
		return nil, fmt.Errorf("token expired")
	}

	return &claims, nil
}

// RefreshToken 签发新 token（在旧 token 未过期时续签）。
func RefreshToken(tokenStr, secret string) (string, error) {
	claims, err := ValidateToken(tokenStr, secret)
	if err != nil {
		return "", fmt.Errorf("cannot refresh: %w", err)
	}
	return GenerateToken(claims.Username, secret)
}

// RSAKeyPair RSA 密钥对（2048 bits）。
type RSAKeyPair struct {
	PrivateKey *rsa.PrivateKey
	PublicKey  *rsa.PublicKey
}

// GenerateRSAKeyPair 生成 RSA 密钥对。
func GenerateRSAKeyPair(bits int) (*RSAKeyPair, error) {
	if bits <= 0 {
		bits = 2048
	}
	privateKey, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return nil, fmt.Errorf("generate RSA key: %w", err)
	}
	return &RSAKeyPair{
		PrivateKey: privateKey,
		PublicKey:  &privateKey.PublicKey,
	}, nil
}

// PublicKeyPEM 返回公钥 PEM 编码字符串。
func (k *RSAKeyPair) PublicKeyPEM() (string, error) {
	pubASN1, err := x509.MarshalPKIXPublicKey(k.PublicKey)
	if err != nil {
		return "", err
	}
	block := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubASN1,
	}
	return string(pem.EncodeToMemory(block)), nil
}

// DecryptOAEP 使用 RSA 私钥解密密文（OAEP SHA-256，与前端 Web Crypto API 一致）。
func (k *RSAKeyPair) DecryptOAEP(ciphertext []byte) ([]byte, error) {
	return rsa.DecryptOAEP(sha256.New(), rand.Reader, k.PrivateKey, ciphertext, nil)
}
