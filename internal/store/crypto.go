package store

import (
	"crypto/sha256"
	"fmt"
)

func HashPIN(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return fmt.Sprintf("%x", h)
}
