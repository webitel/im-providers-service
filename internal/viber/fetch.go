package viber

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"syscall"
	"time"
)

const maxLinkRedirects = 3

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
				return fmt.Errorf("viber: refusing to dial non-public address %q", host)
			}

			return nil
		},
	}

	return &http.Client{
		Timeout:   timeout,
		Transport: &http.Transport{DialContext: dialer.DialContext},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= maxLinkRedirects {
				return fmt.Errorf("viber: too many redirects for %q", req.URL.Redacted())
			}
			if req.URL.Scheme != "https" {
				return fmt.Errorf("viber: refusing to follow a non-https redirect to %q", req.URL.Redacted())
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

	if v4 := ip.To4(); v4 != nil && v4[0] == 100 && v4[1] >= 64 && v4[1] <= 127 {
		return false
	}

	return true
}

func (p *viberProvider) probeLink(ctx context.Context, link string) (contentType string, size int64, ok bool) {
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, link, nil)
	if err != nil {
		return "", 0, false
	}

	resp, err := p.linkClient.Do(req)
	if err != nil {
		p.logger.DebugContext(ctx, "viber link probe refused", "url", link, "err", err)
		return "", 0, false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", 0, false
	}

	return normalizeMime(resp.Header.Get("Content-Type")), resp.ContentLength, true
}
