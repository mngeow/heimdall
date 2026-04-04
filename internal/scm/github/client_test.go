package github

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func TestVerifyWebhookSignature(t *testing.T) {
	client := &Client{
		webhookSecret: "test-secret",
	}

	// Test with valid signature
	payload := []byte(`{"action":"opened"}`)
	// Calculate expected signature
	expectedMAC := hmac.New(sha256.New, []byte("test-secret"))
	expectedMAC.Write(payload)
	expectedSig := "sha256=" + hex.EncodeToString(expectedMAC.Sum(nil))

	err := client.VerifyWebhookSignature(payload, expectedSig)
	if err != nil {
		t.Errorf("expected valid signature, got error: %v", err)
	}

	// Test with invalid signature
	err = client.VerifyWebhookSignature(payload, "sha256=invalid")
	if err == nil {
		t.Error("expected error for invalid signature")
	}

	// Test with missing signature header
	err = client.VerifyWebhookSignature(payload, "")
	if err == nil {
		t.Error("expected error for missing signature")
	}
}
