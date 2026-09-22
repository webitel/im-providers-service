package model

import (
	"net"
	"strings"

	"github.com/webitel/webitel-go-kit/pkg/errors"
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
		return nil, errors.InvalidArgument("not an ip address or cidr block: "+entry,
			errors.WithID("custom.model.parse_allowed_ip"))
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
			return errors.InvalidArgument("allowed_ips: "+err.Error(),
				errors.WithCause(err), errors.WithID("custom.model.validate_allowed_ips"))
		}
	}

	return nil
}
