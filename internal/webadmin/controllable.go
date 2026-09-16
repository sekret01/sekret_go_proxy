package webadmin

type Controllable interface {
	Start() error
	Stop() error
	Status() bool
	GetInfo() string
	Users() []string
}
