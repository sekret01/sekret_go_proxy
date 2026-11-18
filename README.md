# SEKRET GO PROXY

> **version:** 1.3.2

Двухуровневый прокси сервер/клиент на Go с собственным протоколом, мультиплексированием через TCP-туннель, веб-админкой и возможностью добавлять свои модули.

## Возможности
- HTTP-проксирование (включая HTTPS через CONNECT)
- TUN-режим (Linux): перехват всего трафика через виртуальный интерфейс (**только Linux**)
- Мультиплексирование запросов по RequestID через один туннель
- Шифрование ChaCha20-Poly1305
- Веб-админка: статус, логи, пользователи, старт/стоп
- Автопереподключение клиента к серверу
- Фильтрация трафика по доменам/IP (main/filter туннели)

## Требования:
- Go: v 1.26+ (для сборки)
- Linux: root / `CAP_NET_ADMIN` для TUN-режима
- Windows: -

## Быстрый старт

### Сборка

```bash
# Linux
chmod +x build/biuld.sh
./build/build.sh

# Windows
build\build.bat
```

### Запуск клиента

```bash
# Linux
cd build/_result_lin
client

# Windows
cd build\\_result_win
client.exe
```

### Запуск сервера

```bash
# Linux
cd build/_result_lin
server

# Windows
cd build\\_result_win
server.exe
```

## Конфигурация

Файлы конфигураций `client.yaml` и `server.yaml` лежат в дерикториях:

```bash
# Linux
build/_result_lin/configs/...

# Windows
build/_result_win/configs/...
```

### Клиент

```yaml
connection: # настройки подключения клиент->севрер и пользователь->клиент
  connection_type: proxy
  localhost: :15985
  remotehost: 
  # ...

filter: # настройка фильтрации
  active: true
  # ...
  server: # настройка сервера-фильтра
    # ...

dns: # настройка локального DNS сервера
  active: true
  # ...

modules: # настройка используемых модулей
  transport_type: tcp
  encryptor_type: default
  # ...

logs: # настройка логирования
  log_level: INFO

admin: # настройка веб-админки
  adminhost: :12345
  authentication: none
  # ...
```

### Сервер

```yaml
connection:
  connection_type: proxy
  localhost: 127.0.0.1:11111
  # ...


modules:
  transport_type: tcp
  encryptor_type: chacha20
  # ...

vpn: # настройка TUN-режима
  subnet: 10.0.0.0/24

# ...
```

Полный список параметров - в `configs/client.example.yaml`, `configs/server.example.yaml`. 
Полное описание полей конфигураций: [Конфигурация](/docs/CONFIGURING.md)

## Веб-админка

После запуска программы доступна по адресу, указаному в конфиге в поле `admin.adminhost` (по умолчанию 0.0.0.0:12345, [::]:12345)

Страницы:
- `/login` - вход
- `/home` - статус туннеля, старт/стоп
- `/logs` - логи программы
- `/users` - список активных клиентов (только для сервера)

Подробнее: [API](/docs/API.md)

## Архитектура (кратко)

[ ] **КЛИЕНТ (БРАУЗЕР/ПРИЛОЖЕНИЕ)**  
[↓] HTTP / CONNECT  
[ ] **ClientProxy / ClientTun**  
[↓] ecnrypt() -> frame() -> mux by RequestID  
[ ] **TCP-туннель**  
[↓] unframe() -> decrypt()  
[ ] **ServerProxy / ServerTun**  
[↓] Diel / NAT  
[ ] **ЦЕЛЕВОЙ ХОСТ**  

Подробнее: [Архитектура](/docs/ARCHITECTURE.md)

## Добавление своего модуля (пример: encryptor)

1. Создать `internal/encryptors/my_encryptor`
2. Реализовать `core.Encryptor`
3. В `init()` вызвать `encryptors.Register("my-encryptor", NewMyEncryptor)`
4. Импортировать пакет в `internal/register/custom.go`
5. Указать `encryptor_type: my-encryptor` в конфиге

Подробнее: [Разработка своих модулей](/docs/DEVELOPMENT.md)

## Ограничения

- SOCKS5 детектируется, но не обрабатывается
- UDP-транспорт не реализован
- Auth — на данный момент доступен только mock (без реальной аутентификации)
- TUN-режим только Linux
- TLS для админки не настроен

## Дополнительная информация 

- [Архитектура](/docs/ARCHITECTURE.md)
- [Протокол](/docs/PROTOCOLS.md)
- [Разработка своих модулей](/docs/DEVELOPMENT.md)
- [API](/docs/API.md)
- [Конфигурация](/docs/CONFIGURING.md)