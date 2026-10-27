package httpc

import (
	"context"
	"errors"
	"net"
	"reflect"
	"syscall"
	"testing"
	"time"
)

func TestResolverPreservesBareIPAddresses(t *testing.T) {
	for _, host := range []string{"127.0.0.1", "::1", "2001:db8::1", "fe80::1%lo0"} {
		t.Run(host, func(t *testing.T) {
			resolver := NewResolver(time.Second, 0)
			ips, err := resolver.Lookup(t.Context(), host)
			if err != nil || !reflect.DeepEqual(ips, []string{host}) {
				t.Fatalf("Lookup(%q) = %v, %v; want bare IP", host, ips, err)
			}
			cached, err := resolver.Get(t.Context(), host)
			if err != nil || !reflect.DeepEqual(cached, ips) {
				t.Fatalf("cached addresses = %v, %v", cached, err)
			}
			if address := net.JoinHostPort(cached[0], "443"); address != net.JoinHostPort(host, "443") {
				t.Fatalf("invalid dial address: %q", address)
			}
		})
	}
}

func TestDNSCacheDialContextUsesValidIPv6Address(t *testing.T) {
	refused := errors.New("test blocks connection")
	var addresses []string
	dialer := &net.Dialer{Timeout: time.Second, Control: func(network, address string, _ syscall.RawConn) error {
		addresses = append(addresses, address)
		return refused
	}}
	// Control 拒绝实际连接；缓存地址和最终回退都必须能通过地址解析。
	_, err := DnsCacheDialContext(dialer)(context.Background(), "tcp", "[::1]:443")
	if !errors.Is(err, refused) {
		t.Fatalf("unexpected dial error: %v", err)
	}
	if !reflect.DeepEqual(addresses, []string{"[::1]:443", "[::1]:443"}) {
		t.Fatalf("cached/fallback dial addresses = %v", addresses)
	}
}
