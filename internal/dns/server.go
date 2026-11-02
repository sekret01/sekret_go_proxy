package dns

import (
	"fmt"
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
	fmt.Printf("Set %s -> %s\n", ip, domain)
	m.data[ip] = domain
}

func (m *IPDomainMap) Get(ip string) (string, bool) {
	m.RLock()
	defer m.RUnlock()
	domain, ok := m.data[ip]
	fmt.Printf("Get %s -> %s\n", ip, domain)
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
	fmt.Printf("build dns on addr %s, upstream %s\n", s.addr, s.upstream)
	s.server = &dns.Server{Addr: s.addr, Net: "tcp"}
	dns.HandleFunc(".", s.handleDNSRequest)
	fmt.Printf("Start dns server\n")
	return s.server.ListenAndServe()
}

func (s *DNSServer) Stop() {
	fmt.Printf("Stop dns server\n")
	if s.server != nil {
		fmt.Printf("DNS not nil\n")
		s.server.Shutdown()
	}
}

func (s *DNSServer) handleDNSRequest(w dns.ResponseWriter, r *dns.Msg) {
	fmt.Printf("handleDNSRequest\n")
	client := new(dns.Client)
	client.Timeout = 2 * time.Second
	fmt.Printf("Start handle dns. go to upstream\n")
	resp, _, err := client.Exchange(r, s.upstream)
	if err != nil {
		dns.HandleFailed(w, r)
		fmt.Printf("Error in upstream go^ %s\n", err.Error())
		return
	}

	if resp != nil && len(resp.Answer) > 0 {
		domain := r.Question[0].Name
		fmt.Println("Get domain: " + domain)
		if len(domain) > 0 && domain[len(domain)-1] == '.' {
			domain = domain[:len(domain)-1]
		}
		fmt.Println("process domain: " + domain)

		for _, ans := range resp.Answer {
			if aRecord, ok := ans.(*dns.A); ok {
				ip := aRecord.A.String()
				fmt.Println("Get domain A: " + ip)
				s.ipMap.Set(ip, domain)
				fmt.Println("Get domain A: " + domain)
			}
		}
	}
	fmt.Println("return resp")
	w.WriteMsg(resp)
}
