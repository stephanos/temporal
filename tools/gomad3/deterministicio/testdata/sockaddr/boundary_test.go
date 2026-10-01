package main

import (
	"net"
	"strings"
	"testing"

	sockaddr "github.com/hashicorp/go-sockaddr"
)

func TestInterfaceDiscoveryRefusesService(t *testing.T) {
	addresses, err := sockaddr.GetAllInterfaces()
	if err == nil || !strings.Contains(err.Error(), "gomad: sockaddr interface discovery is unsupported") || addresses != nil {
		t.Fatalf("interface discovery = %v, %v", addresses, err)
	}
}

func TestInterfaceMethodsRefuseService(t *testing.T) {
	address := &sockaddr.IfAddr{Interface: net.Interface{Index: 1, Name: "gomad-fixture"}}
	for _, test := range []struct {
		name   string
		lookup func() ([]net.Addr, error)
	}{{"unicast", address.Addrs}, {"multicast", address.MulticastAddrs}} {
		t.Run(test.name, func(t *testing.T) {
			addresses, err := test.lookup()
			if err == nil || !strings.Contains(err.Error(), "gomad: sockaddr interface discovery is unsupported") || addresses != nil {
				t.Fatalf("interface method = %v, %v", addresses, err)
			}
		})
	}
}

func TestAddressResolutionRefusesService(t *testing.T) {
	for _, test := range []struct{ family, input string }{
		{"ipv4", "localhost:80"},
		{"ipv6", "localhost:80"},
		{"ipv4", "127.0.0.1:http"},
		{"ipv6", "[::1]:http"},
	} {
		t.Run(test.family+"/"+test.input, func(t *testing.T) {
			var err error
			if test.family == "ipv4" {
				_, err = sockaddr.NewIPv4Addr(test.input)
			} else {
				_, err = sockaddr.NewIPv6Addr(test.input)
			}
			if err == nil || !strings.Contains(err.Error(), "gomad: sockaddr address resolution requires a literal IP and numeric port") {
				t.Fatalf("address resolution %q = %v", test.input, err)
			}
		})
	}
}

func TestLiteralAddressesRemainComputational(t *testing.T) {
	for _, test := range []struct {
		input, ip string
		port      sockaddr.IPPort
		mask      int
	}{
		{"192.0.2.1", "192.0.2.1", 0, 32},
		{"192.0.2.1:0", "192.0.2.1", 0, 32},
		{"192.0.2.1:", "192.0.2.1", 0, 32},
		{"192.0.2.1:65535", "192.0.2.1", 65535, 32},
		{"192.0.2.1:00080", "192.0.2.1", 80, 32},
		{"192.0.2.1/24", "192.0.2.1", 0, 24},
		{"192.0.2.1/00ffffff", "192.0.2.1", 0, 32},
		{"::1", "::1", 0, 128},
		{"[::1]", "::1", 0, 128},
		{"[::1]:0", "::1", 0, 128},
		{"[::1]:", "::1", 0, 128},
		{"[::1]:65535", "::1", 65535, 128},
		{"[fe80::1%gomad]:80", "fe80::1", 80, 128},
		{"2001:db8::1/64", "2001:db8::1", 0, 64},
	} {
		t.Run(test.input, func(t *testing.T) {
			address, err := sockaddr.NewIPAddr(test.input)
			if err != nil {
				t.Fatal(err)
			}
			got := struct {
				ip   string
				port sockaddr.IPPort
				mask int
			}{address.NetIP().String(), address.IPPort(), address.Maskbits()}
			want := struct {
				ip   string
				port sockaddr.IPPort
				mask int
			}{test.ip, test.port, test.mask}
			if got != want {
				t.Fatalf("literal address = %#v, want %#v", got, want)
			}
		})
	}
}

func TestInvalidNumericPortsAreRejected(t *testing.T) {
	for _, input := range []string{"192.0.2.1:-1", "192.0.2.1:65536", "[::1]:-1", "[::1]:65536"} {
		if _, err := sockaddr.NewIPAddr(input); err == nil {
			t.Fatalf("invalid numeric port accepted: %q", input)
		}
	}
}
