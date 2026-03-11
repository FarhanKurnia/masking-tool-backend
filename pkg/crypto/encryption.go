package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"os"
)

// EncryptionManager handles password encryption/decryption
type EncryptionManager struct {
	encryptionKey []byte
	keyName       string
}

// NewEncryptionManager creates a new encryption manager with a key from environment
func NewEncryptionManager(keyName string) (*EncryptionManager, error) {
	// Get encryption key from environment variable
	// In production, this should come from a secure key management service (e.g., AWS KMS, HashiCorp Vault)
	keyEnvVar := fmt.Sprintf("ENCRYPTION_KEY_%s", keyName)
	keyString := os.Getenv(keyEnvVar)
	if keyString == "" {
		return nil, fmt.Errorf("encryption key not found in environment: %s", keyEnvVar)
	}

	// Decode the base64-encoded key
	key, err := base64.StdEncoding.DecodeString(keyString)
	if err != nil {
		return nil, fmt.Errorf("failed to decode encryption key: %v", err)
	}

	// Validate key length (must be 16, 24, or 32 bytes for AES)
	if len(key) != 16 && len(key) != 24 && len(key) != 32 {
		return nil, fmt.Errorf("encryption key must be 16, 24, or 32 bytes, got %d", len(key))
	}

	return &EncryptionManager{
		encryptionKey: key,
		keyName:       keyName,
	}, nil
}

// Encrypt encrypts a plaintext string using AES-GCM
func (em *EncryptionManager) Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	block, err := aes.NewCipher(em.encryptionKey)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %v", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %v", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %v", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts a ciphertext string using AES-GCM
func (em *EncryptionManager) Decrypt(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}

	ciphertextBytes, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("failed to decode ciphertext: %v", err)
	}

	block, err := aes.NewCipher(em.encryptionKey)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %v", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %v", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertextBytes) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertextBytes := ciphertextBytes[:nonceSize], ciphertextBytes[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %v", err)
	}

	return string(plaintext), nil
}

// GetKeyHash returns the SHA256 hash of the encryption key
func (em *EncryptionManager) GetKeyHash() string {
	hash := sha256.Sum256(em.encryptionKey)
	return hex.EncodeToString(hash[:])
}

// GetKeyName returns the name of the encryption key
func (em *EncryptionManager) GetKeyName() string {
	return em.keyName
}

// SHA256Hash computes the hex-encoded SHA256 hash of input data
func SHA256Hash(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}
