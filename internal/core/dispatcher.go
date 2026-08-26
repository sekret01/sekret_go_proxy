package core

// Интерфейс диспетчера управления сессиями.
// Работает с парами request-ID и connection.
type Dispatcher interface {
	Register()
	Find()
	Delete()
}
