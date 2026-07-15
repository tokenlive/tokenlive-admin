package biz

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"os"
	"time"
)

// defaultSecretKey standard 32-byte fallback key for AES-256-GCM.
// Only used when OAUTH_CLUSTER_AES_KEY is missing from environment.
var defaultSecretKey = []byte("tokenlive_cluster_aes_key_default_32bytes_")

func getClusterAESKey() []byte {
	envKey := os.Getenv("OAUTH_CLUSTER_AES_KEY")
	if envKey == "" {
		return defaultSecretKey
	}
	keyBytes := []byte(envKey)
	if len(keyBytes) < 32 {
		// Pad to 32 bytes if too short
		padded := make([]byte, 32)
		copy(padded, keyBytes)
		return padded
	}
	return keyBytes[:32]
}

type StatePayload struct {
	Verifier  string `json:"v"`
	Timestamp int64  `json:"t"`
}

// EncryptState encrypts the verifier and timestamp into a base64url string.
func EncryptState(verifier string) (string, error) {
	key := getClusterAESKey()
	payload := StatePayload{
		Verifier:  verifier,
		Timestamp: time.Now().Unix(),
	}
	plaintext, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, aesGCM.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	// Seal GCM ciphertext with nonce prefixed
	ciphertext := aesGCM.Seal(nonce, nonce, plaintext, nil)
	return base64.RawURLEncoding.EncodeToString(ciphertext), nil
}

// DecryptState decrypts and validates the state parameter.
func DecryptState(stateStr string, maxAge time.Duration) (string, error) {
	key := getClusterAESKey()
	ciphertext, err := base64.RawURLEncoding.DecodeString(stateStr)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := aesGCM.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	nonce, actualCiphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := aesGCM.Open(nil, nonce, actualCiphertext, nil)
	if err != nil {
		return "", errors.New("failed to decrypt or authenticate state")
	}

	var payload StatePayload
	if err := json.Unmarshal(plaintext, &payload); err != nil {
		return "", err
	}

	// Check if state is expired
	if time.Since(time.Unix(payload.Timestamp, 0)) > maxAge {
		return "", errors.New("state parameter has expired")
	}

	return payload.Verifier, nil
}
