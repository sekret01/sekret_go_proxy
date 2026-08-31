package noopgo

import (
	"encoding/binary"

	"github.com/sekret01/sekret_go_proxy/internal/config"
	"github.com/sekret01/sekret_go_proxy/internal/core"
	"github.com/sekret01/sekret_go_proxy/internal/framers"
)

type SimpleFramer struct{}

// Упаковка данных в протокол.
// Заголовок: MagicByte(1) | MessageType(1) | ID(16) | Size(4)
func (n *SimpleFramer) Frame(data []byte, messageType core.MessageType, requestId core.RequestID) ([]byte, error) {
	headerLen := 1 + 1 + 16 + 4
	totalLen := headerLen + +len(data)
	frame := make([]byte, totalLen)
	offset := 0

	frame[offset] = core.MagicByte
	offset += 1
	frame[offset] = byte(messageType)
	offset += 1
	copy(frame[offset:offset+16], requestId[:])
	offset += 16
	binary.BigEndian.PutUint32(frame[offset:offset+4], uint32(len(data)))
	offset += 4
	copy(frame[offset:], data[:])

	return frame, nil
}

// Распаковка данных из протокола
func (n *SimpleFramer) Unframe(data []byte) ([]byte, core.MessageType, core.RequestID, error) {
	headerLen := 1 + 1 + 16 + 4
	if len(data) < headerLen {
		return nil, core.MsgError, core.RequestID{}, core.ErrInsufficientHeaderLensth
	}
	offset := 0
	magicByte := data[offset]
	if magicByte != core.MagicByte {
		return nil, core.MsgError, core.RequestID{}, core.ErrInvalidMagic
	}
	offset += 1
	msgType := core.MessageType(data[offset])
	offset += 1
	requestId := [16]byte(data[offset : offset+16])
	offset += 16
	dataSize := binary.BigEndian.Uint32(data[offset : offset+4])
	offset += 4

	if uint32(len(data)) < dataSize+uint32(headerLen) {
		return nil, core.MsgError, core.RequestID{}, core.ErrInsufficientPayloadLensth
	}

	payload := data[offset:]
	return payload, msgType, requestId, nil
}

// Получение длины для заголовка данного протокола (в байтах)
func (f *SimpleFramer) HeaderSize() int {
	return 22
}

// Попытка получить длину данных payload, указанную в заголовке.
// Если заголовок меньше необходимого - ошибка
func (f *SimpleFramer) GetPayloadSize(data []byte) (int, error) {
	if len(data) < f.HeaderSize() {
		return -1, core.ErrInsufficientHeaderLensth
	}
	offset := 0 + 1 + 1 + 16
	dataSize := binary.BigEndian.Uint32(data[offset : offset+4])
	return int(dataSize), nil
}

func NewSimpleFramer(cfg *config.Config) (core.Framer, error) {
	return &SimpleFramer{}, nil
}

func init() {
	framers.Registrate("simple", NewSimpleFramer)
}
