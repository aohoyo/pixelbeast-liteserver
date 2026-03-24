package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
)

var (
	ErrInvalidKey      = errors.New("invalid key size")
	ErrInvalidCiphertext = errors.New("invalid ciphertext")
	ErrKeyNotFound     = errors.New("key file not found")
)

// KeySize AES-256 需要 32 字节密钥
const KeySize = 32

// Encrypt 使用 AES-256-GCM 加密数据
func Encrypt(plaintext []byte, key []byte) ([]byte, error) {
	if len(key) != KeySize {
		return nil, ErrInvalidKey
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// 生成随机 nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	// 加密并附加 nonce
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// Decrypt 使用 AES-256-GCM 解密数据
func Decrypt(ciphertext []byte, key []byte) ([]byte, error) {
	if len(key) != KeySize {
		return nil, ErrInvalidKey
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, ErrInvalidCiphertext
	}

	// 提取 nonce 和密文
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

// GenerateKey 生成随机密钥
func GenerateKey() ([]byte, error) {
	key := make([]byte, KeySize)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, err
	}
	return key, nil
}

// DeriveKey 从密码派生密钥
func DeriveKey(password string, salt []byte) []byte {
	h := sha256.New()
	h.Write([]byte(password))
	h.Write(salt)
	return h.Sum(nil)
}

// LoadKey 从文件加载密钥
func LoadKey(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrKeyNotFound
		}
		return nil, err
	}
	
	// 支持 hex 编码的密钥
	key, err := hex.DecodeString(string(data))
	if err != nil {
		return nil, err
	}
	
	if len(key) != KeySize {
		return nil, ErrInvalidKey
	}
	
	return key, nil
}

// SaveKey 保存密钥到文件
func SaveKey(path string, key []byte) error {
	if len(key) != KeySize {
		return ErrInvalidKey
	}
	
	// 确保目录存在
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	
	// 保存为 hex 编码
	data := []byte(hex.EncodeToString(key))
	
	// 权限 600，只有所有者可读写
	return os.WriteFile(path, data, 0600)
}

// EnsureKey 确保密钥文件存在，不存在则创建
func EnsureKey(path string) ([]byte, error) {
	key, err := LoadKey(path)
	if err == nil {
		return key, nil
	}
	
	if errors.Is(err, ErrKeyNotFound) {
		// 生成新密钥
		key, err = GenerateKey()
		if err != nil {
			return nil, err
		}
		
		if err := SaveKey(path, key); err != nil {
			return nil, err
		}
		
		return key, nil
	}
	
	return nil, err
}

// EncryptString 加密字符串，返回 hex 编码的密文
func EncryptString(plaintext string, key []byte) (string, error) {
	ciphertext, err := Encrypt([]byte(plaintext), key)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(ciphertext), nil
}

// DecryptString 解密 hex 编码的密文
func DecryptString(ciphertextHex string, key []byte) (string, error) {
	ciphertext, err := hex.DecodeString(ciphertextHex)
	if err != nil {
		return "", err
	}
	
	plaintext, err := Decrypt(ciphertext, key)
	if err != nil {
		return "", err
	}
	
	return string(plaintext), nil
}