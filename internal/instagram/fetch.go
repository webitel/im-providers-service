package instagram

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"syscall"
	"time"
)

const (
	maxLinkRedirects = 3
	// maxInboundBytes caps a single inbound media download so a malicious or
	// mislabeled Content-Length cannot exhaust memory.
	maxInboundBytes = 32 << 20 // 32 MiB
)

// validateFetchURL gates every URL taken from a webhook payload before it
// reaches the network. Instagram always serves media over https from Meta's
// CDN, so any other scheme is either a misparse or a forged payload trying to
// reach a cluster-internal service (SSRF) — especially dangerous here because
// the download attaches the gate's bearer token.
// https://developers.instagram.com/docs/instagram-api/reference/ig-user/messages
func validateFetchURL(link string) error {
	parsed, err := url.Parse(link)
	if err != nil {
		return fmt.Errorf("instagram: unparsable media url: %w", err)
	}

	if !strings.EqualFold(parsed.Scheme, "https") || parsed.Host == "" {
		return fmt.Errorf("instagram: refusing to fetch %q, https url required", parsed.Redacted())
	}

	return nil
}

// newGuardedClient returns an http.Client that refuses to dial non-public
// addresses and to follow non-https or excessive redirects — blocking SSRF to
// cluster-internal or cloud-metadata endpoints even when the URL host resolves
// to a private IP.
func newGuardedClient(timeout time.Duration) *http.Client {
	dialer := &net.Dialer{
		Timeout: 5 * time.Second,
		Control: func(_, address string, _ syscall.RawConn) error {
			host, _, err := net.SplitHostPort(address)
			if err != nil {
				return err
			}

			ip := net.ParseIP(host)
			if ip == nil || !isPublicIP(ip) {
				return fmt.Errorf("instagram: refusing to dial non-public address %q", host)
			}

			return nil
		},
	}

	return &http.Client{
		Timeout:   timeout,
		Transport: &http.Transport{DialContext: dialer.DialContext},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= maxLinkRedirects {
				return fmt.Errorf("instagram: too many redirects for %q", req.URL.Redacted())
			}

			if req.URL.Scheme != "https" {
				return fmt.Errorf("instagram: refusing to follow a non-https redirect to %q", req.URL.Redacted())
			}

			return nil
		},
	}
}

func isPublicIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified() ||
		ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast() ||
		ip.IsInterfaceLocalMulticast() {
		return false
	}

	// Carrier-grade NAT range (100.64.0.0/10) is not publicly routable.
	if v4 := ip.To4(); v4 != nil && v4[0] == 100 && v4[1] >= 64 && v4[1] <= 127 {
		return false
	}

	return true
}
