package default_filter

import (
	"testing"

	"github.com/sekret01/sekret_go_proxy/internal/filters"
)

func TestNewDefaultFilter(t *testing.T) {
	f, err := NewDefaultFilter(true, "ru", []string{"google.com"})
	if err != nil {
		t.Fatalf("NewDefaultFilter returned error: %v", err)
	}
	if f == nil {
		t.Fatal("NewDefaultFilter returned nil")
	}
}

func TestNewDefaultFilter_NilAddresses(t *testing.T) {
	f, err := NewDefaultFilter(false, "", nil)
	if err != nil {
		t.Fatalf("NewDefaultFilter returned error: %v", err)
	}
	if f == nil {
		t.Fatal("NewDefaultFilter returned nil")
	}
}

func TestDefaultFilter_Inactive(t *testing.T) {
	f, _ := NewDefaultFilter(false, "ru", []string{"google.com"})
	filter := f.(*DefaultFilter)

	cases := []string{
		"site.ru",
		"google.com",
		"anything.else",
	}
	for _, host := range cases {
		if filter.Check(host) {
			t.Fatalf("Check(%q) = true, want false when inactive", host)
		}
	}
}

func TestDefaultFilter_DomainMatch(t *testing.T) {
	f, _ := NewDefaultFilter(true, "ru", nil)
	filter := f.(*DefaultFilter)

	positives := []string{
		"site.ru",
		"sub.site.ru",
		"a.b.c.ru",
	}
	for _, host := range positives {
		if !filter.Check(host) {
			t.Fatalf("Check(%q) = false, want true (domain suffix .ru)", host)
		}
	}

	negatives := []string{
		"site.ru.com",
		"ru",
		"site.com",
		"notru",
		"",
	}
	for _, host := range negatives {
		if filter.Check(host) {
			t.Fatalf("Check(%q) = true, want false (domain suffix .ru)", host)
		}
	}
}

func TestDefaultFilter_AddressesMatch(t *testing.T) {
	f, _ := NewDefaultFilter(true, "", []string{"google.com", "example.org"})
	filter := f.(*DefaultFilter)

	positives := []string{
		"google.com",
		"www.google.com",
		"example.org",
		"sub.example.org",
	}
	for _, host := range positives {
		if !filter.Check(host) {
			t.Fatalf("Check(%q) = false, want true (address match)", host)
		}
	}

	negatives := []string{
		"google.com.evil.com",
		"notgoogle.com",
		"google.co",
		"example.com",
	}
	for _, host := range negatives {
		if filter.Check(host) {
			t.Fatalf("Check(%q) = true, want false (address match)", host)
		}
	}
}

func TestDefaultFilter_DomainAndAddresses(t *testing.T) {
	f, _ := NewDefaultFilter(true, "ru", []string{"google.com"})
	filter := f.(*DefaultFilter)

	cases := []struct {
		host string
		want bool
	}{
		{"site.ru", true},
		{"sub.site.ru", true},
		{"google.com", true},
		{"www.google.com", true},
		{"site.ru.com", false},
		{"google.com.ru", true},
		{"google.org", false},
		{"example.com", false},
	}

	for _, tc := range cases {
		if got := filter.Check(tc.host); got != tc.want {
			t.Fatalf("Check(%q) = %v, want %v", tc.host, got, tc.want)
		}
	}
}

func TestDefaultFilter_WithPort(t *testing.T) {
	f, _ := NewDefaultFilter(true, "ru", []string{"google.com"})
	filter := f.(*DefaultFilter)

	cases := []struct {
		host string
		want bool
	}{
		{"site.ru:443", true},
		{"google.com:80", true},
		{"www.google.com:8080", true},
		{"site.com:443", false},
		{"google.com.evil.com:80", false},
	}

	for _, tc := range cases {
		if got := filter.Check(tc.host); got != tc.want {
			t.Fatalf("Check(%q) = %v, want %v", tc.host, got, tc.want)
		}
	}
}

func TestDefaultFilter_IPAddress(t *testing.T) {
	f, _ := NewDefaultFilter(true, "", nil)
	filter := f.(*DefaultFilter)

	ips := []string{
		"127.0.0.1",
		"192.168.1.1",
		"8.8.8.8",
		"2001:db8::1",
	}
	for _, host := range ips {
		if filter.Check(host) {
			t.Fatalf("Check(%q) = true, want false (IP address)", host)
		}
	}
}

func TestDefaultFilter_EmptyConfig(t *testing.T) {
	f, _ := NewDefaultFilter(true, "", nil)
	filter := f.(*DefaultFilter)

	cases := []string{
		"site.ru",
		"google.com",
		"anything",
	}
	for _, host := range cases {
		if filter.Check(host) {
			t.Fatalf("Check(%q) = true, want false (empty config)", host)
		}
	}
}

func TestDefaultFilter_EmptyHost(t *testing.T) {
	f, _ := NewDefaultFilter(true, "ru", []string{"google.com"})
	filter := f.(*DefaultFilter)

	if filter.Check("") {
		t.Fatal("Check(\"\") = true, want false")
	}
}

func TestDefaultFilter_Registration(t *testing.T) {
	created, err := filters.NewFilter(
		"default",
		true,
		"ru",
		[]string{"google.com"},
	)
	if err != nil {
		t.Fatalf("filters.Create(\"default\", ...) returned error: %v", err)
	}
	if created == nil {
		t.Fatal("filters.Create returned nil filter")
	}

	if !created.Check("site.ru") {
		t.Fatal("registered filter: Check(\"site.ru\") = false, want true")
	}
	if created.Check("site.com") {
		t.Fatal("registered filter: Check(\"site.com\") = true, want false")
	}
}
