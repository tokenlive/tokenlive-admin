package biz

import (
	"os"
	"testing"
	"time"
)

func TestEncryptDecryptState(t *testing.T) {
	// Set mock cluster AES key
	mockKey := "12345678901234567890123456789012"
	os.Setenv("OAUTH_CLUSTER_AES_KEY", mockKey)
	defer os.Unsetenv("OAUTH_CLUSTER_AES_KEY")

	verifier := "xyz123abc456_this_is_a_mock_code_verifier_long_enough_to_be_secure"

	// 1. Encrypt state
	stateStr, err := EncryptState(verifier)
	if err != nil {
		t.Fatalf("failed to encrypt state: %v", err)
	}

	if stateStr == "" {
		t.Fatal("expected non-empty encrypted state")
	}

	// 2. Decrypt state
	decryptedVerifier, err := DecryptState(stateStr, 5*time.Minute)
	if err != nil {
		t.Fatalf("failed to decrypt state: %v", err)
	}

	if decryptedVerifier != verifier {
		t.Errorf("expected verifier %s, got %s", verifier, decryptedVerifier)
	}

	// 3. Test expiration
	_, err = DecryptState(stateStr, -1*time.Second)
	if err == nil {
		t.Error("expected error for expired state, got nil")
	}
}
