package noopgo

import (
	"encoding/binary"

	"github.com/sekret01/sekret_go_proxy/internal/config"
	"github.com/sekret01/sekret_go_proxy/internal/core"
	"github.com/sekret01/sekret_go_proxy/internal/framers"
)

type SimpleFramer struct{}

const (
	MAGIC_BYTE_LEN   = 1
	MESSAGE_TYPE_LEN = 1
	ID_LEN           = 16
	PAYLOAD_SIZE_LEN = 4
	HEADER_SIZE_LEN  = MAGIC_BYTE_LEN + MESSAGE_TYPE_LEN + ID_LEN + PAYLOAD_SIZE_LEN
)

// Упаковка данных в протокол.
// Заголовок: MagicByte(1) | MessageType(1) | ID(16) | Size(4)
func (n *SimpleFramer) Frame(data []byte, messageType core.MessageType, requestId core.RequestID) ([]byte, error) {
	headerLen := HEADER_SIZE_LEN
	totalLen := headerLen + +len(data)
	frame := make([]byte, totalLen)
	offset := 0

	frame[offset] = core.MagicByte
	offset += MAGIC_BYTE_LEN
	frame[offset] = byte(messageType)
	offset += MESSAGE_TYPE_LEN
	copy(frame[offset:offset+ID_LEN], requestId[:])
	offset += ID_LEN
	binary.BigEndian.PutUint32(frame[offset:offset+PAYLOAD_SIZE_LEN], uint32(len(data)))
	offset += PAYLOAD_SIZE_LEN
	copy(frame[offset:], data[:])

	return frame, nil
}

// Распаковка данных из протокола
func (n *SimpleFramer) Unframe(data []byte) ([]byte, core.MessageType, core.RequestID, error) {
	headerLen := HEADER_SIZE_LEN
	if len(data) < headerLen {
		return nil, core.MsgError, core.RequestID{}, core.ErrInsufficientHeaderLensth
	}
	offset := 0
	magicByte := data[offset]
	if magicByte != core.MagicByte {
		return nil, core.MsgError, core.RequestID{}, core.ErrInvalidMagic
	}
	offset += MAGIC_BYTE_LEN
	msgType := core.MessageType(data[offset])
	offset += MESSAGE_TYPE_LEN
	requestId := [ID_LEN]byte(data[offset : offset+ID_LEN])
	offset += ID_LEN
	dataSize := binary.BigEndian.Uint32(data[offset : offset+PAYLOAD_SIZE_LEN])
	offset += PAYLOAD_SIZE_LEN

	if uint32(len(data)) < dataSize+uint32(headerLen) {
		return nil, core.MsgError, core.RequestID{}, core.ErrInsufficientPayloadLensth
	}

	payload := data[offset:]
	return payload, msgType, requestId, nil
}

// Получение длины для заголовка данного протокола (в байтах)
func (f *SimpleFramer) HeaderSize() int {
	return HEADER_SIZE_LEN
}

// Попытка получить длину данных payload, указанную в заголовке.
// Если заголовок меньше необходимого - ошибка
func (f *SimpleFramer) GetPayloadSize(data []byte) (int, error) {
	if len(data) < f.HeaderSize() {
		return -1, core.ErrInsufficientHeaderLensth
	}
	offset := MAGIC_BYTE_LEN + MESSAGE_TYPE_LEN + ID_LEN
	dataSize := binary.BigEndian.Uint32(data[offset : offset+PAYLOAD_SIZE_LEN])
	return int(dataSize), nil
}

func NewSimpleFramer(cfg *config.Config) (core.Framer, error) {
	return &SimpleFramer{}, nil
}

func init() {
	framers.Registrate("simple", NewSimpleFramer)
}
