package services

import (
	"database/sql"
	"errors"
	"time"

	"balance-monitor/internal/database"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var (
	jwtSecret = []byte("your-secret-key-change-in-production") // 在生产环境应该从环境变量读取
)

type Claims struct {
	UserID   int    `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// 检查系统是否已初始化（是否存在用户）
func IsSystemInitialized(db *sql.DB) (bool, error) {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// 初始化系统，创建管理员账户
func InitializeSystem(db *sql.DB, username, password string) error {
	// 检查是否已初始化
	initialized, err := IsSystemInitialized(db)
	if err != nil {
		return err
	}
	if initialized {
		return errors.New("系统已初始化")
	}

	// 验证输入
	if username == "" || password == "" {
		return errors.New("用户名和密码不能为空")
	}
	if len(password) < 6 {
		return errors.New("密码长度至少为6位")
	}

	// 加密密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	// 创建用户
	_, err = db.Exec(`
		INSERT INTO users (username, password_hash, created_at, updated_at)
		VALUES (?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, username, string(hashedPassword))

	return err
}

// 用户登录
func Login(db *sql.DB, username, password string) (string, error) {
	// 查询用户
	var user database.User
	err := db.QueryRow(`
		SELECT id, username, password_hash, created_at, updated_at
		FROM users WHERE username = ?
	`, username).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return "", errors.New("用户名或密码错误")
		}
		return "", err
	}

	// 验证密码
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		return "", errors.New("用户名或密码错误")
	}

	// 生成 JWT Token
	claims := Claims{
		UserID:   user.ID,
		Username: user.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)), // 24小时过期
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "balance-monitor",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// 验证 Token
func ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("无效的 token")
}

// 修改密码
func ChangePassword(db *sql.DB, userID int, oldPassword, newPassword string) error {
	// 查询用户
	var passwordHash string
	err := db.QueryRow("SELECT password_hash FROM users WHERE id = ?", userID).Scan(&passwordHash)
	if err != nil {
		return err
	}

	// 验证旧密码
	err = bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(oldPassword))
	if err != nil {
		return errors.New("原密码错误")
	}

	// 验证新密码
	if len(newPassword) < 6 {
		return errors.New("新密码长度至少为6位")
	}

	// 加密新密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	// 更新密码
	_, err = db.Exec(`
		UPDATE users 
		SET password_hash = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, string(hashedPassword), userID)

	return err
}

