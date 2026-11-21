package main

import (
	"errors"
	"testing"
)

func TestValidateDestinationURL(t *testing.T) {
	originalLookup := lookupHost
	defer func() { lookupHost = originalLookup }()

	publicIP := []string{"93.184.216.34"}

	tests := []struct {
		name    string
		rawURL  string
		lookup  func(string) ([]string, error)
		invalid bool
	}{
		{"https public URL is valid", "https://example.com", nil, false},
		{"http public URL is valid", "http://example.com", nil, false},
		{"javascript scheme rejected", "javascript:alert(1)", nil, true},
		{"ftp scheme rejected", "ftp://example.com", nil, true},
		{"file scheme rejected", "file://localhost/etc/passwd", nil, true},
		{"localhost rejected", "http://localhost", nil, true},
		{"localhost subdomain rejected", "http://api.localhost", nil, true},
		{"dot-local rejected", "http://printer.local", nil, true},
		{"loopback v4 rejected", "http://127.0.0.1", nil, true},
		{"private 10/8 rejected", "http://10.0.0.1", nil, true},
		{"private 172.16/12 rejected", "http://172.16.0.1", nil, true},
		{"private 192.168/16 rejected", "http://192.168.1.1", nil, true},
		{"link-local metadata rejected", "http://169.254.169.254", nil, true},
		{"ipv6 loopback rejected", "http://[::1]", nil, true},
		{"ipv4-mapped loopback rejected", "http://[::ffff:127.0.0.1]", nil, true},
		{"hostname resolving to metadata rejected", "http://innocent.example.com", func(string) ([]string, error) {
			return []string{"169.254.169.254"}, nil
		}, true},
		{"hostname resolving to private rejected", "http://innocent.example.com", func(string) ([]string, error) {
			return []string{"10.0.0.5"}, nil
		}, true},
		{"hostname resolving to public accepted", "http://innocent.example.com", func(string) ([]string, error) {
			return publicIP, nil
		}, false},
		{"mixed public then private rejected", "http://innocent.example.com", func(string) ([]string, error) {
			return []string{"93.184.216.34", "10.0.0.5"}, nil
		}, true},
		{"unresolvable hostname accepted", "http://unresolvable.example.com", func(string) ([]string, error) {
			return nil, errors.New("no such host")
		}, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			lookupHost = originalLookup
			if tc.lookup != nil {
				lookupHost = tc.lookup
			}

			if _, invalid := validateDestinationURL(tc.rawURL); invalid != tc.invalid {
				t.Fatalf("validateDestinationURL(%q) invalid = %v, want %v", tc.rawURL, invalid, tc.invalid)
			}
		})
	}
}
