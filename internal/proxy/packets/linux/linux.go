package linux

import (
	"errors"
	"net"
	"os"
	"os/exec"
	"strings"
	"unsafe"

	"github.com/sekret01/sekret_go_proxy/internal/config"
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
	cfg      *config.Config
	serverIP string
}

func (d *LinuxPacketDevice) CreateTun(name string) error {
	serverIP, err := getIpFromHost(d.cfg.RemoteHost)
	if err != nil {
		return err
	}
	d.serverIP = serverIP
	fd, err := unix.Open(devPath, unix.O_RDWR|unix.O_CLOEXEC, 0)
	if err != nil {
		return err
	}
	var ifreq [unix.IFNAMSIZ + 64]byte
	copy(ifreq[:], name)
	*(*uint16)(unsafe.Pointer(&ifreq[unix.IFNAMSIZ])) = unix.IFF_TUN | unix.IFF_NO_PI

	_, _, errno := unix.Syscall(unix.SYS_IOCTL, uintptr(fd), tunSetIff, uintptr(unsafe.Pointer(&ifreq[0])))
	if errno != 0 {
		unix.Close(fd)
		return nil
	}
	d.file = os.NewFile(uintptr(fd), name)
	if err := d.openVirtualTun(intfName); err != nil {
		d.closeVirtualTun(intfName)
		return err
	}
	d.isActive = true
	return nil
}

func (d *LinuxPacketDevice) ClientConfigure(ipAddr string) error {
	if err := utils.ShRun("ip", "addr", "add", ipAddr+"/32", "dev", intfName); err != nil {
		return err
	}
	gatewayIP, err := getDefaultGateway()
	if err != nil {
		return err
	}
	if err := utils.ShRun("ip", "route", "add", d.serverIP+"/32", "via", gatewayIP); err != nil {
		return err
	}
	if err := utils.ShRun("ip", "route", "add", "0.0.0.0/1", "dev", "tun0"); err != nil {
		return err
	}
	return utils.ShRun("ip", "route", "add", "128.0.0.0/1", "dev", "tun0")
}

func (d *LinuxPacketDevice) ClientUnconfigure() error {
	utils.ShRun("ip", "route", "del", "0.0.0.0/1", "dev", "tun0")
	utils.ShRun("ip", "route", "del", "128.0.0.0/1", "dev", "tun0")
	utils.ShRun("ip", "route", "del", d.serverIP+"/32")
	return nil
}

func (d *LinuxPacketDevice) ServerConfigure() error {
	extIface, err := getDefaultInterface()
	if err != nil {
		return err
	}
	if err := utils.ShRun("sysctl", "-w", "net.ipv4.ip_forward=1"); err != nil {
		return err
	}
	if err := utils.ShRun("iptables", "-A", "FORWARD",
		"-i", "tun0", "-o", extIface, "-j", "ACCEPT"); err != nil {
		return err
	}
	if err := utils.ShRun("iptables", "-A", "FORWARD",
		"-i", extIface, "-o", "tun0",
		"-m", "state", "--state", "RELATED,ESTABLISHED", "-j", "ACCEPT"); err != nil {
		return err
	}
	// MASQUERADE
	if err := utils.ShRun("iptables", "-t", "nat", "-A", "POSTROUTING",
		"-s", "10.0.0.0/24", "-o", extIface, "-j", "MASQUERADE"); err != nil {
		return err
	}
	if err := utils.ShRun("iptables", "-t", "mangle", "-A", "FORWARD",
		"-p", "tcp", "--tcp-flags", "SYN,RST", "SYN",
		"-j", "TCPMSS", "--clamp-mss-to-pmtu"); err != nil {
		return err
	}
	return nil
}

func (d *LinuxPacketDevice) ServerUnconfigure() error {
	extIface, err := getDefaultInterface()
	if err != nil {
		return err
	}
	utils.ShRun("iptables", "-t", "mangle", "-D", "FORWARD",
		"-p", "tcp", "--tcp-flags", "SYN,RST", "SYN",
		"-j", "TCPMSS", "--clamp-mss-to-pmtu")
	utils.ShRun("iptables", "-t", "nat", "-D", "POSTROUTING",
		"-s", "10.0.0.0/24", "-o", extIface, "-j", "MASQUERADE")
	utils.ShRun("iptables", "-D", "FORWARD",
		"-i", extIface, "-o", "tun0",
		"-m", "state", "--state", "RELATED,ESTABLISHED", "-j", "ACCEPT")
	utils.ShRun("iptables", "-D", "FORWARD",
		"-i", "tun0", "-o", extIface, "-j", "ACCEPT")
	utils.ShRun("sysctl", "-w", "net.ipv4.ip_forward=0")
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

func (d *LinuxPacketDevice) ClientClose() error {
	d.ClientUnconfigure()
	d.closeVirtualTun(intfName)
	err := d.file.Close()
	d.isActive = false
	return err
}

func (d *LinuxPacketDevice) ServerClose() error {
	d.ServerUnconfigure()
	d.closeVirtualTun(intfName)
	err := d.file.Close()
	d.isActive = false
	return err
}

func (d *LinuxPacketDevice) openVirtualTun(name string) error {
	if err := utils.ShRun("ip", "link", "set", "dev", name, "up", "mtu", "1400"); err != nil {
		return err
	}
	return nil
}

func (d *LinuxPacketDevice) closeVirtualTun(name string) error {
	utils.ShRun("ip", "addr", "del", "10.0.0.1/24", "dev", name, "2>/dev/null")
	utils.ShRun("ip", "link", "set", "dev", name, "down", "2>/dev/null")
	return nil
}

// Парсит вывод `ip route show default` и достаёт имя интерфейса после "dev".
func getDefaultInterface() (string, error) {
	out, err := exec.Command("ip", "route", "show", "default").Output()
	if err != nil {
		return "", err
	}
	fields := strings.Fields(string(out))
	for i, f := range fields {
		if f == "dev" && i+1 < len(fields) {
			return fields[i+1], nil
		}
	}
	return "", errors.New("default interface not found")
}

func NewLinuxPacketDevice(cfg *config.Config) (core.PacketDevice, error) {
	return &LinuxPacketDevice{
			isActive: false,
			cfg:      cfg,
			serverIP: "",
		},
		nil
}

func init() {
	packets.Registrate("linux", NewLinuxPacketDevice)
}

func getIpFromHost(host string) (string, error) {
	h, _, err := net.SplitHostPort(host)
	if err != nil {
		h = host
	}
	if ip := net.ParseIP(h); ip != nil {
		return ip.String(), nil
	}
	ips, err := net.LookupIP(h)
	if err != nil {
		return "", err
	}
	for _, ip := range ips {
		if ip4 := ip.To4(); ip4 != nil {
			return ip4.String(), nil
		}
	}
	return "", errors.New("Not found any ips")
}

func getDefaultGateway() (string, error) {
	out, err := exec.Command("ip", "route", "show", "default").Output()
	if err != nil {
		return "", err
	}
	fields := strings.Fields(string(out))
	for i, f := range fields {
		if f == "via" && i+1 < len(fields) {
			return fields[i+1], nil
		}
	}
	return "", errors.New("gateway not found")
}
