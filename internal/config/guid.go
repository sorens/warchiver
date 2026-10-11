package config

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
)

// GenerateGUID generates a random UUID v4.
func GenerateGUID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generating GUID: %w", err)
	}
	// Set version 4 and variant bits
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%12x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}

// DeriveSchedule deterministically derives a check hour (0-23) and minute (0-59)
// from a GUID string. This ensures each user has a unique but stable check time.
func DeriveSchedule(guid string) (hour, minute int) {
	// Remove hyphens and decode hex bytes
	clean := strings.ReplaceAll(guid, "-", "")
	b, err := hex.DecodeString(clean)
	if err != nil || len(b) < 8 {
		return 3, 0 // safe default: 3:00 AM
	}
	// First 4 bytes -> hour
	h := uint32(b[0])<<24 | uint32(b[1])<<16 | uint32(b[2])<<8 | uint32(b[3])
	// Next 4 bytes -> minute
	m := uint32(b[4])<<24 | uint32(b[5])<<16 | uint32(b[6])<<8 | uint32(b[7])
	return int(h % 24), int(m % 60)
}
