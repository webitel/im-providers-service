package viberbm

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"syscall"
	"time"
)

const maxMediaRedirects = 3

// validateFetchURL requires https before a forwarded media URL hits the network;
// any other scheme is a misparse or a forged SSRF attempt at an internal host.
// https://www.infobip.com/docs/api/channels/viber/viber-business-messages/inbound-message/receive-viber-business-message
func validateFetchURL(link string) error {
	parsed, err := url.Parse(link)
	if err != nil {
		return fmt.Errorf("viber_bm: unparsable media url: %w", err)
	}

	if !strings.EqualFold(parsed.Scheme, "https") || parsed.Host == "" {
		return fmt.Errorf("viber_bm: refusing to fetch %q, https url required", parsed.Redacted())
	}

	return nil
}

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
				return fmt.Errorf("viber_bm: refusing to dial non-public address %q", host)
			}

			return nil
		},
	}

	return &http.Client{
		Timeout:   timeout,
		Transport: &http.Transport{DialContext: dialer.DialContext},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= maxMediaRedirects {
				return fmt.Errorf("viber_bm: too many redirects for %q", req.URL.Redacted())
			}

			if req.URL.Scheme != "https" {
				return fmt.Errorf("viber_bm: refusing to follow a non-https redirect to %q", req.URL.Redacted())
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

	// Carrier-grade NAT range 100.64.0.0/10 is not publicly routable.
	if v4 := ip.To4(); v4 != nil && v4[0] == 100 && v4[1] >= 64 && v4[1] <= 127 {
		return false
	}

	return true
}
