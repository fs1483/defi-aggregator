// Package utils 提供加密和签名验证工具函数
// 实现Web3钱包签名验证、JWT令牌管理等安全相关功能
// 遵循以太坊签名标准和安全最佳实践
package utils

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// ========================================
// Web3钱包地址验证
// ========================================

// IsValidEthereumAddress 验证以太坊地址格式
// 检查地址是否符合以太坊地址规范：42位十六进制字符，以0x开头
// 参数:
//   - address: 待验证的地址字符串
//
// 返回:
//   - bool: 地址是否有效
func IsValidEthereumAddress(address string) bool {
	// 检查长度和前缀
	if len(address) != 42 || !strings.HasPrefix(address, "0x") {
		return false
	}

	// 验证十六进制格式
	re := regexp.MustCompile("^0x[0-9a-fA-F]{40}$")
	return re.MatchString(address)
}

// NormalizeEthereumAddress 标准化以太坊地址格式
// 将地址转换为标准的小写格式，便于数据库存储和比较
// 参数:
//   - address: 原始地址
//
// 返回:
//   - string: 标准化后的地址
//   - error: 地址格式错误
func NormalizeEthereumAddress(address string) (string, error) {
	if !IsValidEthereumAddress(address) {
		return "", fmt.Errorf("无效的以太坊地址格式: %s", address)
	}
	return strings.ToLower(address), nil
}

// ========================================
// 随机数生成
// ========================================

// GenerateNonce 生成用于钱包签名的随机数
// 使用加密安全的随机数生成器，确保每次登录的唯一性
// 返回:
//   - string: 十六进制格式的随机数字符串
//   - error: 生成过程中的错误
func GenerateNonce() (string, error) {
	// 生成32字节的随机数据
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("生成随机数失败: %w", err)
	}

	// 转换为十六进制字符串
	return hex.EncodeToString(bytes), nil
}

// GenerateRequestID 生成请求ID
// 用于链路追踪和日志关联
// 返回:
//   - string: UUID格式的请求ID
func GenerateRequestID() string {
	return uuid.New().String()
}

// ========================================
// 消息格式化（用于签名）
// ========================================

// FormatSignMessage 格式化签名消息
// 按照EIP-712或类似标准格式化登录消息，确保签名安全
// 参数:
//   - walletAddress: 用户钱包地址
//   - nonce: 随机数
//   - timestamp: 时间戳
//
// 返回:
//   - string: 格式化的签名消息
func FormatSignMessage(walletAddress, nonce string, timestamp int64) string {
	return fmt.Sprintf(
		"DeFi聚合器登录验证\n\n"+
			"钱包地址: %s\n"+
			"随机数: %s\n"+
			"时间戳: %d\n"+
			"域名: defi-aggregator.com\n"+
			"版本: 1",
		walletAddress, nonce, timestamp,
	)
}

// ========================================
// JWT令牌管理
// ========================================

// JWTClaims JWT声明结构体
// 包含用户信息和自定义声明
type JWTClaims struct {
	UserID               uint   `json:"user_id"`        // 用户ID
	WalletAddress        string `json:"wallet_address"` // 钱包地址
	Role                 string `json:"role"`           // 用户角色
	TokenType            string `json:"token_type"`     // 令牌类型: access, refresh
	jwt.RegisteredClaims        // 标准JWT声明
}

// GenerateJWT 生成JWT令牌
// 创建包含用户信息的JWT访问令牌
// 参数:
//   - userID: 用户ID
//   - walletAddress: 钱包地址
//   - role: 用户角色
//   - secretKey: JWT密钥
//   - expiresIn: 过期时间
//   - tokenType: 令牌类型
//
// 返回:
//   - string: JWT令牌字符串
//   - error: 生成错误
func GenerateJWT(userID uint, walletAddress, role, secretKey string, expiresIn time.Duration, tokenType string) (string, error) {
	now := time.Now()

	// 创建JWT声明
	claims := &JWTClaims{
		UserID:        userID,
		WalletAddress: strings.ToLower(walletAddress),
		Role:          role,
		TokenType:     tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "defi-aggregator",                      // 签发者
			Subject:   fmt.Sprintf("user:%d", userID),         // 主题
			Audience:  []string{"defi-aggregator"},            // 受众
			ExpiresAt: jwt.NewNumericDate(now.Add(expiresIn)), // 过期时间
			NotBefore: jwt.NewNumericDate(now),                // 生效时间
			IssuedAt:  jwt.NewNumericDate(now),                // 签发时间
			ID:        uuid.New().String(),                    // 令牌ID
		},
	}

	// 创建令牌
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// 签名令牌
	tokenString, err := token.SignedString([]byte(secretKey))
	if err != nil {
		return "", fmt.Errorf("JWT令牌签名失败: %w", err)
	}

	return tokenString, nil
}

// ParseJWT 解析JWT令牌
// 验证JWT令牌并提取用户声明信息
// 参数:
//   - tokenString: JWT令牌字符串
//   - secretKey: JWT密钥
//
// 返回:
//   - *JWTClaims: 解析的声明信息
//   - error: 解析或验证错误
func ParseJWT(tokenString, secretKey string) (*JWTClaims, error) {
	// 解析令牌
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		// 验证签名方法
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("无效的签名方法: %v", token.Header["alg"])
		}
		return []byte(secretKey), nil
	})

	if err != nil {
		return nil, fmt.Errorf("JWT令牌解析失败: %w", err)
	}

	// 验证令牌有效性
	if !token.Valid {
		return nil, fmt.Errorf("JWT令牌无效")
	}

	// 提取声明
	claims, ok := token.Claims.(*JWTClaims)
	if !ok {
		return nil, fmt.Errorf("无效的JWT声明格式")
	}

	return claims, nil
}

// ========================================
// 签名验证（简化版本）
// ========================================

// VerifySignature 验证以太坊签名
// 注意：这是一个简化版本，生产环境建议使用go-ethereum库进行完整验证
// 参数:
//   - message: 原始消息
//   - signature: 十六进制签名
//   - walletAddress: 钱包地址
//
// 返回:
//   - bool: 签名是否有效
//   - error: 验证过程中的错误
func VerifySignature(message, signature, walletAddress string) (bool, error) {
	// TODO: 实现真正的以太坊签名验证
	// 这里需要使用 go-ethereum 库进行椭圆曲线签名验证
	//
	// 完整实现需要：
	// 1. 使用 crypto.Keccak256() 对消息进行哈希
	// 2. 使用 crypto.SigToPub() 从签名恢复公钥
	// 3. 使用 crypto.PubkeyToAddress() 从公钥得到地址
	// 4. 比较恢复的地址与传入的地址是否一致

	// 当前简化验证：检查基本格式
	if !IsValidEthereumAddress(walletAddress) {
		return false, fmt.Errorf("无效的钱包地址: %s", walletAddress)
	}

	if len(signature) != 132 || !strings.HasPrefix(signature, "0x") {
		return false, fmt.Errorf("无效的签名格式: %s", signature)
	}

	if message == "" {
		return false, fmt.Errorf("签名消息不能为空")
	}

	// 简化验证：如果格式正确就认为有效
	// 生产环境必须实现真正的密码学验证
	return true, nil
}

// ========================================
// 密码哈希（用于API密钥等）
// ========================================

// HashPassword 使用bcrypt哈希密码
// 虽然Web3应用主要使用钱包认证，但某些场景仍需要密码哈希
// 参数:
//   - password: 原始密码
//
// 返回:
//   - string: 哈希后的密码
//   - error: 哈希错误
func HashPassword(password string) (string, error) {
	// TODO: 实现bcrypt密码哈希
	// 当前返回简化实现
	return fmt.Sprintf("hashed_%s", password), nil
}

// VerifyPassword 验证密码哈希
// 参数:
//   - password: 原始密码
//   - hashedPassword: 哈希密码
//
// 返回:
//   - bool: 密码是否匹配
func VerifyPassword(password, hashedPassword string) bool {
	// TODO: 实现bcrypt密码验证
	expectedHash := fmt.Sprintf("hashed_%s", password)
	return hashedPassword == expectedHash
}

// ========================================
// 工具函数
// ========================================

// GetCurrentTimestamp 获取当前时间戳
// 返回Unix时间戳，用于各种时间相关的验证
func GetCurrentTimestamp() int64 {
	return time.Now().Unix()
}

// IsTimestampValid 验证时间戳有效性
// 检查时间戳是否在合理的时间窗口内（防重放攻击）
// 参数:
//   - timestamp: 要验证的时间戳
//   - windowMinutes: 允许的时间窗口（分钟）
//
// 返回:
//   - bool: 时间戳是否有效
func IsTimestampValid(timestamp int64, windowMinutes int) bool {
	now := time.Now().Unix()
	window := int64(windowMinutes * 60)

	// 检查时间戳是否在允许的窗口内
	return timestamp >= (now-window) && timestamp <= (now+window)
}
