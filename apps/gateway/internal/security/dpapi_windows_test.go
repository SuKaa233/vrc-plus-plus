package security

import (
	"bytes"
	"testing"
)

func TestDPAPIRoundTrip(t *testing.T) {
	t.Parallel()
	protector := NewProtector()
	plain := []byte("session-secret-for-test")
	cipher, err := protector.Protect(plain)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(cipher, plain) {
		t.Fatal("protected data must differ from plaintext")
	}
	restored, err := protector.Unprotect(cipher)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(restored, plain) {
		t.Fatalf("Unprotect() = %q, want %q", restored, plain)
	}
}
