package model

import (
	"net"
	"strings"
)

// ParseAllowedIP accepts both a bare address and a CIDR block, so an operator
// can write either 10.0.0.7 or 10.0.0.0/8 in the gate configuration.
func ParseAllowedIP(entry string) (*net.IPNet, error) {
	entry = strings.TrimSpace(entry)

	if strings.Contains(entry, "/") {
		_, network, err := net.ParseCIDR(entry)
		if err != nil {
			return nil, err
		}

		return network, nil
	}

	ip := net.ParseIP(entry)
	if ip == nil {
		return nil, &FieldError{Field: "allowed_ips", Reason: "not an ip address or cidr block: " + entry}
	}

	bits := 8 * net.IPv6len
	if v4 := ip.To4(); v4 != nil {
		ip, bits = v4, 8*net.IPv4len
	}

	return &net.IPNet{IP: ip, Mask: net.CIDRMask(bits, bits)}, nil
}

func ValidateAllowedIPs(entries []string) error {
	for _, entry := range entries {
		if strings.TrimSpace(entry) == "" {
			continue
		}

		if _, err := ParseAllowedIP(entry); err != nil {
			if fe, ok := err.(*FieldError); ok {
				return fe
			}

			return &FieldError{Field: "allowed_ips", Reason: err.Error()}
		}
	}

	return nil
}
