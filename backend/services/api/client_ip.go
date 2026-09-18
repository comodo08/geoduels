package main

import (
	"errors"
	"net"
	"net/http"
	"strconv"
	"strings"
)

func parseCIDRs(raw string) ([]*net.IPNet, error) {
	var out []*net.IPNet
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if ip := net.ParseIP(part); ip != nil {
			bits := 32
			if ip.To4() == nil {
				bits = 128
			}
			part += "/" + strconv.Itoa(bits)
		}
		_, network, err := net.ParseCIDR(part)
		if err != nil {
			return nil, errors.New("invalid TRUSTED_PROXY_CIDRS entry: " + part)
		}
		out = append(out, network)
	}
	return out, nil
}

func (a *api) clientIP(r *http.Request) string {
	remoteIP := remoteAddrIP(r.RemoteAddr)
	if remoteIP == "" {
		return ""
	}
	if !a.trustsProxy(remoteIP) {
		return remoteIP
	}
	if ip := net.ParseIP(strings.TrimSpace(r.Header.Get("CF-Connecting-IP"))); ip != nil {
		return ip.String()
	}
	if ip := a.forwardedForClientIP(r.Header.Get("X-Forwarded-For")); ip != "" {
		return ip
	}
	if ip := net.ParseIP(strings.TrimSpace(r.Header.Get("X-Real-IP"))); ip != nil {
		return ip.String()
	}
	return remoteIP
}

func (a *api) forwardedForClientIP(raw string) string {
	parts := strings.Split(raw, ",")
	for i := len(parts) - 1; i >= 0; i-- {
		part := strings.TrimSpace(parts[i])
		if part == "" {
			continue
		}
		ip := net.ParseIP(part)
		if ip == nil {
			continue
		}
		ipString := ip.String()
		if !a.trustsProxy(ipString) {
			return ipString
		}
	}
	return ""
}

func (a *api) trustsProxy(ipString string) bool {
	ip := net.ParseIP(ipString)
	if ip == nil {
		return false
	}
	for _, network := range a.trustedProxyCIDRs {
		if network.Contains(ip) {
			return true
		}
	}
	return false
}

func remoteAddrIP(remoteAddr string) string {
	host, _, err := net.SplitHostPort(strings.TrimSpace(remoteAddr))
	if err != nil {
		host = strings.TrimSpace(remoteAddr)
	}
	if ip := net.ParseIP(host); ip != nil {
		return ip.String()
	}
	return ""
}
