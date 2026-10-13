package linux

import (
	"fmt"
	"os"
	"unsafe"

	"github.com/sekret01/sekret_go_proxy/internal/core"
	"github.com/sekret01/sekret_go_proxy/internal/proxy/packets"
	"github.com/sekret01/sekret_go_proxy/internal/utils"
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
	if err := d.openVirtualTun(intfName); err != nil {
		fmt.Println("[!][CreateTun]: error in create virtual tun0: " + err.Error())
		d.closeVirtualTun(intfName)
		return err
	}
	if err := d.setRoutes("157.228.153.149", "192.168.0.1"); err != nil { // TODO из конфига / парсить
		fmt.Println("[!][CreateTun]: error in create virtual tun0: " + err.Error())
		d.unsetRoutes("157.228.153.149") // TODO из конфига / парсить
		d.closeVirtualTun(intfName)
		return err
	}
	fd, err := unix.Open(devPath, unix.O_RDWR|unix.O_CLOEXEC, 0)
	if err != nil {
		fmt.Println("[!][CreateTun]: error in open file: " + err.Error())
		return err
	}
	var ifreq [unix.IFNAMSIZ + 64]byte // IFNAMSIZ
	copy(ifreq[:], name)
	*(*uint16)(unsafe.Pointer(&ifreq[unix.IFNAMSIZ])) = unix.IFF_TUN | unix.IFF_NO_PI

	_, _, errno := unix.Syscall(unix.SYS_IOCTL, uintptr(fd), tunSetIff, uintptr(unsafe.Pointer(&ifreq[0])))
	if errno != 0 {
		fmt.Printf("[!][CreateTun]: error in syscal, num: %d\n", errno)
		unix.Close(fd)
		return nil
	}
	d.file = os.NewFile(uintptr(fd), name)
	d.isActive = true
	fmt.Println("[!][CreateTun]: tun0 opend")
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
	d.unsetRoutes("157.228.153.149") // TODO из конфига / парсить
	d.closeVirtualTun(intfName)
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

func (d *LinuxPacketDevice) openVirtualTun(name string) error {
	if err := utils.ShRun("ip", "link", "set", "dev", name, "up", "mtu", "1400"); err != nil {
		return err
	}
	if err := utils.ShRun("ip", "addr", "add", "10.0.0.1/24", "dev", name); err != nil {
		return err
	}
	return nil
}

func (d *LinuxPacketDevice) closeVirtualTun(name string) error {
	if err := utils.ShRun("ip", "addr", "del", "10.0.0.1/24", "dev", name, "2>/dev/null"); err != nil {
		// return err
	}
	if err := utils.ShRun("ip", "link", "set", "dev", name, "down", "2>/dev/null"); err != nil {
		// return err
	}
	return nil
}

func (d *LinuxPacketDevice) setRoutes(serverIP, gatewayIP string) error {
	if err := utils.ShRun("ip", "route", "add", serverIP+"/32", "via", gatewayIP); err != nil {
		return err
	}
	if err := utils.ShRun("ip", "route", "add", "0.0.0.0/1", "dev", "tun0"); err != nil {
		return err
	}
	return utils.ShRun("ip", "route", "add", "128.0.0.0/1", "dev", "tun0")
}

func (d *LinuxPacketDevice) unsetRoutes(serverIP string) error {
	utils.ShRun("ip", "route", "del", "0.0.0.0/1", "dev", "tun0")
	utils.ShRun("ip", "route", "del", "128.0.0.0/1", "dev", "tun0")
	utils.ShRun("ip", "route", "del", serverIP+"/32")
	return nil
}

func init() {
	packets.Registrate("linux", NewLinuxPacketDevice)
}
