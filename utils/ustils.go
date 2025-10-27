package utils

import (
	"crypto/rand"
	"encoding/hex"
)

func RemoveBOM(data []byte) []byte {
	if len(data) >= 3 && data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF {
		return data[3:]
	}
	return data
}

func GenerateToken() string {
	bytes := make([]byte, 32) // 32 байта = 256 бит
	if _, err := rand.Read(bytes); err != nil {
		panic("не удалось сгенерировать токен: " + err.Error())
	}
	return hex.EncodeToString(bytes)
}
