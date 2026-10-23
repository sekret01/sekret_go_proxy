# Архитектура программы

## Компоненты
|компонент|пакет|описание (за что отвечает)|
|-|-|-|
|client|internal/proxy/client|прием запросов от браузеров/приложений, шифровка, направление в следующий узел|
|server|internal/proxy/server|прием запросов от узла, расшифровка, отправление на target-сервер, отправление ответа на узел|

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
	SetEncryptor(Encryptor)
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
## Доступные модули для выбора в конфиге

Все модули, добавленные по умолчанию, называются `default` (кроме Encryptor `chacha20` и Auth `composite_handshake`). Все настройки модулей клиента должны совпадать с сервером, иначе обмен и расшифровка данных будет происходить с ошибками.

|пакет модулей|описание|доступные модули по умолчанию|
|-|-|-|
|**transport_type**|тип транспортного уровня для туннеля|tcp, udp|
|**encryptor_type**|тип шифровальщика|default, chacha20|
|**framer_type**|тип фреймера протокола|default|
|**dispatcher_type**|тип диспетчера|default|
|**auth_type**|тип авторизации и аутентификации|default, composite_handshake|
