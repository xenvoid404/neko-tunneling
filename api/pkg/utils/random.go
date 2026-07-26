package utils

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/google/uuid"
)

func RandomUsername() string {
	return "trial-" + randomHex(4)
}

func RandomPassword(protocol string) string {
	if protocol == "ssh" {
		return randomHex(4)
	}
	return uuid.New().String()
}

func randomHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Errorf("crypto/rand gagal: %w", err))
	}
	return hex.EncodeToString(b)
}
