package utils

import (
	"encoding/hex"
	"strconv"

	"github.com/sekret01/sekret_go_proxy/internal/core"
)

func RequestIdToString(id core.RequestID) string {
	return hex.EncodeToString(id[:])
}

func ListToString(list []string) string {
	result := ""
	for i, word := range list {
		result += word
		if i < len(list)-1 {
			result += ", "
		}
	}
	return result
}

func BytesToString(data []byte, needLen int) string {
	resultLen := min(needLen, len(data))
	result := strconv.Quote(string(data[:resultLen]))
	if len(data) > resultLen {
		result += "..."
	}
	return result
}

func IntToString(num int) string {
	return strconv.FormatUint(uint64(num), 16)
}

func MessageTypeToHexString(msgType core.MessageType) string {
	return IntToString(int(msgType))
}
