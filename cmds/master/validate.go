package main

import (
	"net"
	"net/url"
	"strings"
)

var lookupHost = net.LookupHost

func validateDestinationURL(rawURL string) (string, bool) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "redirectsTo should be a valid URL", true
	}

	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return "redirectsTo should use the http or https scheme", true
	}

	host := parsed.Hostname()
	if host == "" {
		return "redirectsTo should be a valid URL", true
	}

	if isBlockedHostname(host) {
		return "redirectsTo should not target an internal address", true
	}

	if ip := net.ParseIP(host); ip != nil {
		if isPrivateOrReservedIP(ip) {
			return "redirectsTo should not target an internal address", true
		}
		return "", false
	}

	addrs, err := lookupHost(host)
	if err != nil {
		return "", false
	}
	for _, addr := range addrs {
		if ip := net.ParseIP(addr); ip != nil && isPrivateOrReservedIP(ip) {
			return "redirectsTo should not target an internal address", true
		}
	}

	return "", false
}

func isBlockedHostname(host string) bool {
	host = strings.ToLower(strings.TrimSuffix(host, "."))
	if host == "localhost" {
		return true
	}

	for _, suffix := range []string{
		".localhost", ".local", ".internal", ".lan", ".home", ".home.arpa", ".corp",
	} {
		if strings.HasSuffix(host, suffix) {
			return true
		}
	}

	return false
}

func isPrivateOrReservedIP(ip net.IP) bool {
	if v4 := ip.To4(); v4 != nil {
		ip = v4
	}

	return ip.IsUnspecified() ||
		ip.IsLoopback() ||
		ip.IsPrivate() ||
		ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() ||
		ip.IsMulticast()
}
