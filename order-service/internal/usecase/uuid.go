package usecase

import (
	"crypto/rand"
	"fmt"
)

func cryptoRead(b []byte) (int, error) {
	return rand.Read(b)
}

func formatUUID(b []byte) string {
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%12x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
