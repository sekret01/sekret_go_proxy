# Архитектура программы

## Компоненты
|компонент|пакет|описание (за что отвечает)|
|-|-|-|
|client|internal/proxy/client|прием запросов от браузеров/приложений, шифровка, направление в следующий узел|
|server|internal/proxy/client|прием запросов от узла, расшифровка, отправление на target-сервер, отправление ответа на узел|

## модули и интерфейсы

### Детектор (internal/core/detector.go)

```go
type Detector interface {
	Detect(data []byte) (ProtocolType, error)
}
```

### Диспетчер (internal/core/dispatcher.go)

```go
type Dispatcher interface {
	Register(id RequestID, requestConn net.Conn, protoType ProtocolType, isTunnel bool)
	Find(id RequestID) (ConnWrapper, bool)
	Delete(id RequestID)
}
```

### Аутентификация (internal/core/auth.go)

```go
type Auth interface {
	ServerHandshake(conn net.Conn) ([]byte, error)
	ClientHandshake(conn net.Conn) ([]byte, error)
}
```

### Шифрование (internal/core/encription.go)

```go
type Encryptor interface {
	Encrypt(data []byte) []byte
	Decrypt(data []byte) []byte
}
```

### Фреймер (internal/core/framer.go)

```go
type Framer interface {
	Frame(data []byte, messageType MessageType, requestId RequestID) ([]byte, error)
	Unframe(data []byte) ([]byte, MessageType, RequestID, error)
	HeaderSize() int
	GetPayloadSize(data []byte) (int, error)
}
```

### Транспорт (internal/core/transport.go)

```go
type Transport interface {
	Listen(addres string) (net.Listener, error)
	Dial(address string) (net.Conn, error)
}
```

