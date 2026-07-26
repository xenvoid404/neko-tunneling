package utils

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/google/uuid"
)

func RandomUsername() string {
	bytes := make([]byte, 4)
	_, _ = rand.Read(bytes)
	return "trial-" + hex.EncodeToString(bytes)
}

func RandomPassword(protocol string) string {
	if protocol == "ssh" {
		bytes := make([]byte, 4)
		_, _ = rand.Read(bytes)
		return hex.EncodeToString(bytes)
	}
	return uuid.New().String()
}
