package webadmin

type Controllable interface {
	Start() error
	Stop() error
	GetInfo() string
	Users() []string
}
