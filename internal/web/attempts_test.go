package web

import (
	"net/netip"
	"testing"
)

func TestLimiterKey(t *testing.T) {
	tests := []struct {
		name string
		a, b string
		same bool
	}{
		{"IPv6 same /64", "2001:db8:1:2::1", "2001:db8:1:2:ffff:ffff:ffff:ffff", true},
		{"IPv6 other /64", "2001:db8:1:2::1", "2001:db8:1:3::1", false},
		{"IPv6 zone ignored", "fe80::1%eth0", "fe80::2", true},
		{"IPv4 full address", "192.0.2.1", "192.0.2.2", false},
		{"IPv4-mapped IPv6 is IPv4", "::ffff:192.0.2.1", "192.0.2.1", true},
		{"IPv4-mapped keeps full address", "::ffff:192.0.2.1", "::ffff:192.0.2.2", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ka := limiterKey(netip.MustParseAddr(tt.a))
			kb := limiterKey(netip.MustParseAddr(tt.b))
			if (ka == kb) != tt.same {
				t.Fatalf("limiterKey(%s) = %s, limiterKey(%s) = %s, same = %v, want %v", tt.a, ka, tt.b, kb, ka == kb, tt.same)
			}
		})
	}
	if got := limiterKey(netip.MustParseAddr("2001:db8:1:2:3:4:5:6")); got != netip.MustParseAddr("2001:db8:1:2::") {
		t.Fatalf("IPv6 key = %s, want 2001:db8:1:2::", got)
	}
	if got := limiterKey(netip.MustParseAddr("::ffff:192.0.2.1")); got != netip.MustParseAddr("192.0.2.1") {
		t.Fatalf("mapped key = %s, want 192.0.2.1", got)
	}
	if got := limiterKey(netip.Addr{}); got.IsValid() {
		t.Fatalf("zero Addr key = %s, want the zero Addr", got)
	}
}
