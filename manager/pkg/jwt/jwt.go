package jwt

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	// ErrInvalidToken token无效
	ErrInvalidToken = errors.New("invalid token")
	// ErrExpiredToken token已过期
	ErrExpiredToken = errors.New("token expired")
	// ErrTokenNotYetValid token尚未生效
	ErrTokenNotYetValid = errors.New("token not yet valid")
)

// Claims JWT声明
type Claims struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// Manager JWT管理器
type Manager struct {
	secretKey     []byte
	issuer        string
	expireDuration time.Duration
}

// NewManager 创建JWT管理器
func NewManager(secretKey, issuer string, expireDuration time.Duration) *Manager {
	return &Manager{
		secretKey:      []byte(secretKey),
		issuer:         issuer,
		expireDuration: expireDuration,
	}
}

// GenerateToken 生成JWT token
func (m *Manager) GenerateToken(userID uint, username, role string) (string, error) {
	now := time.Now()

	claims := Claims{
		UserID:   userID,
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.expireDuration)),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secretKey)
}

// ParseToken 解析JWT token
func (m *Manager) ParseToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// 验证签名方法
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return m.secretKey, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		if errors.Is(err, jwt.ErrTokenNotValidYet) {
			return nil, ErrTokenNotYetValid
		}
		return nil, ErrInvalidToken
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, ErrInvalidToken
}

// ValidateToken 验证JWT token
func (m *Manager) ValidateToken(tokenString string) error {
	_, err := m.ParseToken(tokenString)
	return err
}

// RefreshToken 刷新JWT token（生成新token）
func (m *Manager) RefreshToken(tokenString string) (string, error) {
	// 解析旧token
	claims, err := m.ParseToken(tokenString)
	if err != nil {
		// 如果token已过期，检查是否在刷新窗口期内（例如7天内）
		if !errors.Is(err, ErrExpiredToken) {
			return "", err
		}

		// 尝试解析过期的token
		token, parseErr := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
			return m.secretKey, nil
		}, jwt.WithoutClaimsValidation())

		if parseErr != nil {
			return "", ErrInvalidToken
		}

		if parsedClaims, ok := token.Claims.(*Claims); ok {
			// 检查是否在刷新窗口期内（7天）
			if time.Since(parsedClaims.ExpiresAt.Time) > 7*24*time.Hour {
				return "", errors.New("token expired beyond refresh window")
			}
			claims = parsedClaims
		} else {
			return "", ErrInvalidToken
		}
	}

	// 生成新token
	return m.GenerateToken(claims.UserID, claims.Username, claims.Role)
}

// IsAdmin 检查用户是否为管理员
func (c *Claims) IsAdmin() bool {
	return c.Role == "admin"
}

// IsActive 检查token是否在有效期内
func (c *Claims) IsActive() bool {
	now := time.Now()
	return now.After(c.NotBefore.Time) && now.Before(c.ExpiresAt.Time)
}
