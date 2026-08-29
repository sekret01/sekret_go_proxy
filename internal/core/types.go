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
)

// STRUCTS

// Структура для хранения информации о подключении
type ConnWrapper struct {
	ID   RequestID
	Conn net.Conn
}
