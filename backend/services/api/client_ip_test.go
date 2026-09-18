package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientIPIgnoresForwardedHeadersFromUntrustedPeer(t *testing.T) {
	a := &api{}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "198.51.100.20:12345"
	req.Header.Set("CF-Connecting-IP", "203.0.113.1")
	req.Header.Set("X-Forwarded-For", "203.0.113.2")
	req.Header.Set("X-Real-IP", "203.0.113.3")

	if got := a.clientIP(req); got != "198.51.100.20" {
		t.Fatalf("clientIP = %q, want remote address", got)
	}
}

func TestClientIPUsesForwardedForFromTrustedPeer(t *testing.T) {
	trusted, err := parseCIDRs("10.42.0.0/16")
	if err != nil {
		t.Fatal(err)
	}
	a := &api{trustedProxyCIDRs: trusted}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.42.1.10:12345"
	req.Header.Set("X-Forwarded-For", "203.0.113.50")

	if got := a.clientIP(req); got != "203.0.113.50" {
		t.Fatalf("clientIP = %q, want forwarded client", got)
	}
}

func TestClientIPSkipsSpoofedForwardedForBeforeActualClient(t *testing.T) {
	trusted, err := parseCIDRs("10.42.0.0/16")
	if err != nil {
		t.Fatal(err)
	}
	a := &api{trustedProxyCIDRs: trusted}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.42.1.10:12345"
	req.Header.Set("X-Forwarded-For", "192.0.2.222, 203.0.113.50, 10.42.1.10")

	if got := a.clientIP(req); got != "203.0.113.50" {
		t.Fatalf("clientIP = %q, want last untrusted forwarded address", got)
	}
}

func TestParseCIDRsAcceptsSingleIPs(t *testing.T) {
	trusted, err := parseCIDRs("10.42.1.10, 2001:db8::1")
	if err != nil {
		t.Fatal(err)
	}
	a := &api{trustedProxyCIDRs: trusted}

	if !a.trustsProxy("10.42.1.10") {
		t.Fatal("expected IPv4 host to be trusted")
	}
	if !a.trustsProxy("2001:db8::1") {
		t.Fatal("expected IPv6 host to be trusted")
	}
	if a.trustsProxy("10.42.1.11") {
		t.Fatal("did not expect adjacent IPv4 host to be trusted")
	}
}
