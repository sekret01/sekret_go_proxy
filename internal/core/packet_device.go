package core

type PacketDevice interface {
	CreateTun(name string) error
	ReadPacket(buf []byte) (int, error)
	WritePacket(pkt []byte) (int, error)
	MTU() int
	Close() error
}
