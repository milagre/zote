package container

import (
	"crypto/sha512"
	"encoding/base64"
	"fmt"
)

// rabbitPasswordHashSHA512 is rabbit_password_hashing_sha512 for definitions.json: base64(salt || sha512(salt||password)); salt is 4 raw bytes base64-encoded.
func rabbitPasswordHashSHA512(saltBase64, password string) (string, error) {
	salt, err := base64.StdEncoding.DecodeString(saltBase64)
	if err != nil {
		return "", fmt.Errorf("decoding salt: %w", err)
	}

	sum := sha512.Sum512(append(append([]byte{}, salt...), []byte(password)...))
	combined := append(append([]byte{}, salt...), sum[:]...)

	return base64.StdEncoding.EncodeToString(combined), nil
}
