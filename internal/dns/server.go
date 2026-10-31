package dns

import (
	"log"
	"sync"
	"time"

	"github.com/miekg/dns"
)

type IPDomainMap struct {
	sync.RWMutex
	data map[string]string
}

func NewIPDomainMap() *IPDomainMap {
	return &IPDomainMap{
		data: make(map[string]string),
	}
}

func (m *IPDomainMap) Set(ip, domain string) {
	m.Lock()
	defer m.Unlock()
	m.data[ip] = domain
}

func (m *IPDomainMap) Get(ip string) (string, bool) {
	m.RLock()
	defer m.RUnlock()
	domain, ok := m.data[ip]
	return domain, ok
}

type DNSServer struct {
	addr     string
	upstream string
	ipMap    *IPDomainMap
	server   *dns.Server
}

func NewDNSServer(listenAddr string, upstreamAddr string, ipMap *IPDomainMap) *DNSServer {
	return &DNSServer{
		addr:     listenAddr,
		upstream: upstreamAddr,
		ipMap:    ipMap,
	}
}

func (s *DNSServer) Start() error {
	// Инициализируем UDP-сервер
	s.server = &dns.Server{Addr: s.addr, Net: "udp"}
	dns.HandleFunc(".", s.handleDNSRequest)

	log.Printf("[DNS] Starting DNS server on %s, upstream: %s", s.addr, s.upstream)
	return s.server.ListenAndServe()
}

func (s *DNSServer) Stop() {
	if s.server != nil {
		s.server.Shutdown()
	}
}

func (s *DNSServer) handleDNSRequest(w dns.ResponseWriter, r *dns.Msg) {
	client := new(dns.Client)
	client.Timeout = 2 * time.Second

	resp, _, err := client.Exchange(r, s.upstream)
	if err != nil {
		log.Printf("[DNS] Error querying upstream: %v", err)
		dns.HandleFailed(w, r)
		return
	}

	if resp != nil && len(resp.Answer) > 0 {
		domain := r.Question[0].Name
		if len(domain) > 0 && domain[len(domain)-1] == '.' {
			domain = domain[:len(domain)-1]
		}

		for _, ans := range resp.Answer {
			if aRecord, ok := ans.(*dns.A); ok {
				ip := aRecord.A.String()
				s.ipMap.Set(ip, domain)
				log.Printf("[DNS] Mapped IP %s -> %s", ip, domain)
			}
		}
	}

	w.WriteMsg(resp)
}
