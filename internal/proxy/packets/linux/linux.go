package linux

import (
	"os"
	"unsafe"

	"github.com/sekret01/sekret_go_proxy/internal/core"
	"github.com/sekret01/sekret_go_proxy/internal/proxy/packets"
	"golang.org/x/sys/unix"
)

const (
	devPath   = "/dev/net/tun"
	intfName  = "tun0"
	tunSetIff = unix.TUNSETIFF
)

type LinuxPacketDevice struct {
	isActive bool
	file     *os.File
}

func (d *LinuxPacketDevice) CreateTun(name string) error {
	fd, err := unix.Open(devPath, unix.O_RDWR|unix.O_CLOEXEC, 0)
	if err != nil {
		return err
	}
	var ifreq [unix.IFNAMSIZE + 64]byte // IFNAMSIZ
	copy(ifreq[:], name)
	*(*uint16)(unsafe.Pointer(&ifreq[unix.IFNAMSIZE])) = unix.IFF_TUN | unix.IFF_NO_PI

	_, _, errno := unix.Syscall(unix.SYS_IOCTL, uintptr(fd), tunSetIff, uintptr(unsafe.Pointer(&ifreq[0])))
	if errno != 0 {
		unix.Close(fd)
		return nil
	}
	d.file = os.NewFile(uintptr(fd), name)
	d.isActive = true
	return nil
}

func (d *LinuxPacketDevice) ReadPacket(buf []byte) (int, error) {
	if !d.isActive {
		return 0, nil
	}
	n, err := d.file.Read(buf)
	return n, err
}

func (d *LinuxPacketDevice) WritePacket(pkt []byte) (int, error) {
	if !d.isActive {
		return 0, nil
	}
	n, err := d.file.Write(pkt)
	return n, err
}

func (d *LinuxPacketDevice) MTU() int {
	return 1400
}

func (d *LinuxPacketDevice) Close() error {
	err := d.file.Close()
	d.isActive = false
	return err
}

func NewLinuxPacketDevice() (core.PacketDevice, error) {
	return &LinuxPacketDevice{
			isActive: false,
		},
		nil
}

func init() {
	packets.Registrate("linux", NewLinuxPacketDevice)
}
