package main

import (
	"github.com/miekg/dns"
	"net"
	"testing"
)

func question(zone string, qt uint16) *dns.Msg {
	m := new(dns.Msg)
	m.SetQuestion(dns.Fqdn(zone), qt)
	return m
}

func Test_ProxyMsg_NoMapping(t *testing.T) {
	zones = make(map[string]net.IP)
	if msg := ProxyMsg(new(dns.Msg)); msg != nil {
		t.Fatalf("Expected nil when message doesn't contain questions")
	}
	if msg := ProxyMsg(question("foo.tld.", dns.TypeA)); msg != nil {
		t.Fatalf("Expected nil for missing mapping")
	}
}

func Test_ProxyMsg_Empty(t *testing.T) {
	zone := "foo.tld."
	zones = map[string]net.IP{zone: net.ParseIP("127.0.0.1")}

	actual := ProxyMsg(question(zone, dns.TypeAAAA)).Answer

	if len(actual) != 0 {
		t.Fatalf("Expected empty answer for non-A record")
	}
}

func Test_ProxyMsg(t *testing.T) {
	zone := "foo.tld."
	zones = map[string]net.IP{zone: net.ParseIP("127.0.0.1")}

	expected := zones[zone]
	actual := ProxyMsg(question(zone, dns.TypeA)).Answer[0].(*dns.A).A

	if !actual.Equal(expected) {
		t.Fatalf("%s != %s", actual, expected)
	}
}
