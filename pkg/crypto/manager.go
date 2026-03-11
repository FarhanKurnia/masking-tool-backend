package crypto

import (
	"fmt"
	"sync"

	"github.com/example/masking-tool-backend/config"
)

// Global encryption manager instance
var (
	encryptionManager *EncryptionManager
	encOnce           sync.Once
)

// Initialize initializes the global encryption manager
func Initialize() error {
	var err error
	encOnce.Do(func() {
		// Initialize with the configured encryption key
		keyName := config.Config.EncryptionKeyName
		encryptionManager, err = NewEncryptionManager(keyName)
		if err != nil {
			// Log but don't fail - passwords can be stored unencrypted for now
			fmt.Printf("Warning: Could not initialize encryption manager: %v. Passwords will not be encrypted.\n", err)
			err = nil // Non-fatal
		}
	})
	return err
}

// GetManager returns the global encryption manager instance
// Returns nil if encryption is not available
func GetManager() *EncryptionManager {
	return encryptionManager
}

// IsAvailable checks if encryption manager is available
func IsAvailable() bool {
	return encryptionManager != nil
}
