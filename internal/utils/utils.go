package utils

import (
	"encoding/hex"

	"github.com/sekret01/sekret_go_proxy/internal/core"
)

func RequestIdToString(id core.RequestID) string {
	return hex.Dump(id[:])
}

func ListToString(list []string) string {
	result := ""
	for i, word := range list {
		if i < len(list)-1 {
			result += word + ", "
		}
	}
	return result
}
