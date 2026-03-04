// models/uuid.go - UUID生成（内嵌避免外部依赖）
package models

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

// uuid 包的内嵌实现

// UUID represents a UUID
type UUID [16]byte

// New creates a new UUID v4
func New() UUID {
	var u UUID
	_, err := rand.Read(u[:])
	if err != nil {
		// 回退到时间戳方案
		timestamp := time.Now().UnixNano()
		data := fmt.Sprintf("%d%x", timestamp, make([]byte, 8))
		copy(u[:], []byte(data)[:16])
	} else {
		// Set version (4) and variant bits
		u[6] = (u[6] & 0x0f) | 0x40
		u[8] = (u[8] & 0x3f) | 0x80
	}
	return u
}

// String returns the UUID as a string
func (u UUID) String() string {
	buf := make([]byte, 36)
	hex.Encode(buf[0:8], u[0:4])
	buf[8] = '-'
	hex.Encode(buf[9:13], u[4:6])
	buf[13] = '-'
	hex.Encode(buf[14:18], u[6:8])
	buf[18] = '-'
	hex.Encode(buf[19:23], u[8:10])
	buf[23] = '-'
	hex.Encode(buf[24:36], u[10:16])
	return string(buf)
}

// NewUUID 生成新的UUID字符串
func NewUUID() string {
	return New().String()
}
