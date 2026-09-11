# Разработка: добавление своих модулей

## Допустимые модули к добавлению:

|модуль|пакет|
|-|-|
|Auth|internal/auths|
|Dispatcher|internal/dispacthers|
|Encryptor|internal/encryptors|
|Framer|internal/framers|
|Transport|internal/transports|

## Добавление сових модулей

Необходимые действия для добавления модулей:
1. Создать пакет своего модуля в пакете хранения модулей типа
2. Написать структуру, реализующую интерфейс типа из internal/core
3. в функции init() зарегистрировать функцию создания своего модуля
4. Добавить импорт своего модуля в `internal/register/custom.go`
5. В конфигурации указать тип нужного модуля

## Пример создания нового модуля Encryptor

### Создание пакета модуля

Необходимо создать директорию `internal/encryptors/my_encryptor`

### Создание структуры

Необхоидмо реализовать методы интерфейса core.Encryption, написать функцию для создания структуры

### Добавить init()

```go
// internal/encryptors/my_encryptor/my.go
package my_encryption

import ("github.com/sekret01/sekret_go_proxy/internal/encryptors")

// Новая структура
type MyEncryptor struct {...}
// Функция создания структуры
func NewMyEncryptor(...) {...}

func init() {
    encryptors.Reigstrate("my-encryptor", NewMyEncryptor)
}
```

### Добавление импорта

```go
// internal/registrate/custom.go

import (
    // ...
    _ "github.com/sekret01/sekret_go_proxy/internal/encryptors/my_encryptor"
)
```

### Изменение конфигурации

```go
// configs/server.yaml / configs/client.yaml

encryptor_type: my-encryptor
```

## Практический пример

Структура ChaCha20Encryptor (`github.com/sekret01/sekret_go_proxy/internal/encryptors/chacha20/chacha20.go`) является примером нового модуля Encryptor.