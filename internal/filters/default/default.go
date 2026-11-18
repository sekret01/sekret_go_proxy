package default_filter

import (
	"net"
	"strings"

	"github.com/sekret01/sekret_go_proxy/internal/core"
	"github.com/sekret01/sekret_go_proxy/internal/filters"
)

type DefaultFilter struct {
	active    bool
	domain    string
	addresses []string
}

func (f *DefaultFilter) Check(host string) bool {
	if !f.active {
		return false
	}

	h := host
	if ind := strings.LastIndex(host, ":"); ind != -1 {
		h = host[:ind]
	}

	if net.ParseIP(h) != nil {
		return false // TODO нужен ли парсинг IP
	}

	if f.domain != "" {
		if strings.HasSuffix(h, "."+f.domain) {
			return true
		}
	}

	for _, a := range f.addresses {
		if a == h || strings.HasSuffix(h, "."+a) {
			return true
		}
	}
	return false

}

func NewDefaultFilter(
	active bool,
	domain string,
	addresses []string,
) (core.Filter, error) {
	return &DefaultFilter{
		active:    active,
		domain:    domain,
		addresses: addresses,
	}, nil
}

func init() {
	filters.Registrate("default", NewDefaultFilter)
}
