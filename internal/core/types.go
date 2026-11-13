package core

import "net"

// TYPES

// Уникальный ID запроса
type RequestID [16]byte

// Тип протокола
type ProtocolType uint8

// Тип сообщения в туннеле
type MessageType uint8

// CONSTS

// Байт для идентификации пакета
const MagicByte byte = 0x53

// Ответ для запроса HTTPS CONNECT
const HttpConnectResponse = "HTTP/1.1 200 Connection Established\r\n\r\n"

// Размеры фрейма
const (
	MAX_PAYLOAD_SIZE = 16 << 20
)

// Типы протоколов
const (
	ProtoTest    ProtocolType = 127
	ProtoUnknown ProtocolType = 0
	ProtoHTTP    ProtocolType = 1
	ProtoSOCKS5  ProtocolType = 2
)

// Типы сообщений в туннеле
const (
	MsgConnect MessageType = 1
	MsgData    MessageType = 2
	MsgClose   MessageType = 3
	MsgError   MessageType = 4
	MsgPacket  MessageType = 5
)

const (
	BYTE    int = 1
	BYTE_8  int = BYTE * 8
	BYTE_16 int = BYTE * 16
	BYTE_32 int = BYTE * 32
	BYTE_64 int = BYTE * 64

	KBYTE    int = BYTE * 1024
	KBYTE_8  int = KBYTE * 8
	KBYTE_16 int = KBYTE * 16
	KBYTE_32 int = KBYTE * 32
	KBYTE_64 int = KBYTE * 64

	MBYTE int = KBYTE * 1024
)

// STRUCTS

// Структура для хранения информации о подключении
type ConnWrapper struct {
	ID        RequestID
	Conn      net.Conn
	ProtoType ProtocolType // Тип протокола (HTTP, SOCKS5)
	IsTunnel  bool         // нужно ли держать открытым (HTTP false)
}

type MessageData struct {
	MsgType   MessageType
	RequestId RequestID
	Data      []byte
}

type ConnectData struct {
	Host string
}
