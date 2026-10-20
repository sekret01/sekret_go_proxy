package server

import (
	"errors"
	"net"
	"sync"
)

type IpPool struct {
	mu   sync.Mutex
	free []string
	used map[string]bool
}

func newIPPool(subnet string) *IpPool {
	_, ipnet, err := net.ParseCIDR(subnet)
	if err != nil {
		panic("bad vpn subnet: " + err.Error())
	}
	// Генерируем все адреса кроме network, broadcast и .1 (сервер)
	pool := &IpPool{used: make(map[string]bool)}
	ip := ipnet.IP.To4()
	for i := 2; i < 255; i++ { // .2 .. .254
		a := make(net.IP, 4)
		copy(a, ip)
		a[3] = byte(i)
		pool.free = append(pool.free, a.String())
	}
	return pool
}

func (p *IpPool) Allocate() (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.free) == 0 {
		return "", errors.New("ip pool exhausted")
	}
	ip := p.free[0]
	p.free = p.free[1:]
	p.used[ip] = true
	return ip, nil
}

func (p *IpPool) Release(ip string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.used[ip] {
		return
	}
	delete(p.used, ip)
	p.free = append(p.free, ip)
}
