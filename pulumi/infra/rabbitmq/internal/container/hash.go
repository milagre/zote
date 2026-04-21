package container

import (
	"crypto/sha512"
	"encoding/base64"
	"fmt"
)

// rabbitPasswordHashSHA512 computes a RabbitMQ password hash in the
// "rabbit_password_hashing_sha512" format that `definitions.json` expects.
// The algorithm, per the upstream docs and the legacy pwhash.sh that this
// replaces, is:
//
//  1. decode the base64 salt (4 raw bytes),
//  2. sha512(salt || password),
//  3. base64(salt || digest).
//
// The salt is generated separately (via random.RandomBytes) and lives in
// the parent caller.
func rabbitPasswordHashSHA512(saltBase64, password string) (string, error) {
	salt, err := base64.StdEncoding.DecodeString(saltBase64)
	if err != nil {
		return "", fmt.Errorf("decoding salt: %w", err)
	}

	sum := sha512.Sum512(append(append([]byte{}, salt...), []byte(password)...))
	combined := append(append([]byte{}, salt...), sum[:]...)

	return base64.StdEncoding.EncodeToString(combined), nil
}
