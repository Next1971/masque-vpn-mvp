package clientcore

import (
	"net"
	"testing"
)

func TestDualDialAddrs(t *testing.T) {
	primary := &net.UDPAddr{IP: net.IPv4(1, 2, 3, 4), Port: 4433}
	if got := dualDialAddrs(primary, 0); len(got) != 1 || got[0].Port != 4433 {
		t.Fatalf("no alt: %+v", got)
	}
	got := dualDialAddrs(primary, 2053)
	if len(got) != 2 || got[0].Port != 4433 || got[1].Port != 2053 {
		t.Fatalf("alt 2053: %+v", got)
	}
	if got := dualDialAddrs(primary, 4433); len(got) != 1 {
		t.Fatalf("same port should not race: %+v", got)
	}
}

func TestValidateAltPort(t *testing.T) {
	if err := validateAltPort("h:4433", 0); err != nil {
		t.Fatal(err)
	}
	if err := validateAltPort("h:4433", 2053); err != nil {
		t.Fatal(err)
	}
	if err := validateAltPort("h:4433", 4433); err == nil {
		t.Fatal("same port")
	}
	if err := validateAltPort("h:4433", 70000); err == nil {
		t.Fatal("range")
	}
}
