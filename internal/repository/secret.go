package repository

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const sealedSecretPrefix = "enc:v1:"

var secretKeyMu sync.Mutex

// SealSecret encrypts small application secrets before they are persisted.
// An explicit TRACKER_ENCRYPTION_KEY is preferred. Local installations get a
// random per-data-directory key with restrictive permissions, so encryption
// remains stable across restarts without silently falling back to plaintext.
// SealSecret 使用本地持久化密钥加密敏感设置值。
func SealSecret(value string) (string, error) {
	if value == "" || strings.HasPrefix(value, sealedSecretPrefix) {
		return value, nil
	}
	key, err := dataEncryptionKey()
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nil, nonce, []byte(value), nil)
	return sealedSecretPrefix + base64.RawStdEncoding.EncodeToString(append(nonce, ciphertext...)), nil
}

// OpenSecret also accepts legacy plaintext values so existing installations
// continue to work and are upgraded the next time the setting is saved.
// OpenSecret 解密已加密的敏感设置值，并兼容旧版明文值。
func OpenSecret(value string) (string, error) {
	if value == "" || !strings.HasPrefix(value, sealedSecretPrefix) {
		return value, nil
	}
	payload, err := base64.RawStdEncoding.DecodeString(strings.TrimPrefix(value, sealedSecretPrefix))
	if err != nil {
		return "", fmt.Errorf("已保存的密钥数据无效")
	}
	key, err := dataEncryptionKey()
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(payload) < gcm.NonceSize() {
		return "", fmt.Errorf("已保存的密钥数据无效")
	}
	plain, err := gcm.Open(nil, payload[:gcm.NonceSize()], payload[gcm.NonceSize():], nil)
	if err != nil {
		return "", fmt.Errorf("无法解密已保存的密钥")
	}
	return string(plain), nil
}

// dataEncryptionKey 在存储层中完成本文件定义的局部处理。
func dataEncryptionKey() ([]byte, error) {
	if configured := strings.TrimSpace(os.Getenv("TRACKER_ENCRYPTION_KEY")); configured != "" {
		sum := sha256.Sum256([]byte(configured))
		return sum[:], nil
	}
	secretKeyMu.Lock()
	defer secretKeyMu.Unlock()
	path := filepath.Join(DataDir(), ".tracker-secret-key")
	if raw, err := os.ReadFile(path); err == nil {
		key, decodeErr := base64.RawStdEncoding.DecodeString(strings.TrimSpace(string(raw)))
		if decodeErr != nil || len(key) != 32 {
			return nil, fmt.Errorf("本地加密密钥无效")
		}
		return key, nil
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, err
	}
	encoded := []byte(base64.RawStdEncoding.EncodeToString(key))
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if os.IsExist(err) {
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil, readErr
		}
		existing, decodeErr := base64.RawStdEncoding.DecodeString(strings.TrimSpace(string(raw)))
		if decodeErr != nil || len(existing) != 32 {
			return nil, fmt.Errorf("本地加密密钥无效")
		}
		return existing, nil
	}
	if err != nil {
		return nil, err
	}
	if _, err = file.Write(encoded); err == nil {
		err = file.Sync()
	}
	closeErr := file.Close()
	if err != nil {
		return nil, err
	}
	if closeErr != nil {
		return nil, closeErr
	}
	return key, nil
}
