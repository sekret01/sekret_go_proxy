package core

type PacketDevice interface {
	CreateTun(name string) error
	ClientConfigure(ipAddr string) error
	ServerConfigure() error
	ReadPacket(buf []byte) (int, error)
	WritePacket(pkt []byte) (int, error)
	MTU() int
	ClientClose() error
	ServerClose() error
}
