package container

import (
	"crypto/sha512"
	"encoding/base64"
	"testing"
)

func TestRabbitPasswordHashSHA512_EndToEnd(t *testing.T) {
	salt := []byte{0x01, 0x02, 0x03, 0x04}
	saltB64 := base64.StdEncoding.EncodeToString(salt)
	password := "s3cret"

	got, err := rabbitPasswordHashSHA512(saltB64, password)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sum := sha512.Sum512(append(append([]byte{}, salt...), []byte(password)...))
	want := base64.StdEncoding.EncodeToString(append(append([]byte{}, salt...), sum[:]...))

	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRabbitPasswordHashSHA512_InvalidSalt(t *testing.T) {
	if _, err := rabbitPasswordHashSHA512("not base64!!!", "pw"); err == nil {
		t.Fatal("expected error, got nil")
	}
}
