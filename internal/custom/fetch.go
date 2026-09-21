package custom

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"syscall"
	"time"

	"github.com/webitel/webitel-go-kit/pkg/errors"

	custommodel "github.com/webitel/im-providers-service/internal/custom/model"
)

const (
	maxLinkRedirects = 3
	fetchTimeout     = 30 * time.Second
)

const (
	clientForErrID = "custom.fetch.client_for"
	getErrID       = "custom.fetch.get"
)

// Inbound files arrive as a URL owned by the external system, and that URL is
// attacker-controlled the moment a secret leaks. Two clients cover the two
// legitimate cases without opening a request-forgery hole:
//
//   - trustedClient fetches from the same host as the configured callback URL.
//     That host is chosen by an administrator, and an on-premise middleware
//     usually lives on a private address, so the public-address rule cannot
//     apply to it.
//   - linkClient fetches anything else and refuses to dial a non-public
//     address, which is what keeps a forged payload from reaching a
//     cluster-internal service.
//
// Neither follows a redirect that leaves the host it started on.
type fetcher struct {
	trusted *http.Client
	public  *http.Client
}

func newFetcher() *fetcher {
	return &fetcher{
		trusted: &http.Client{
			Timeout:       fetchTimeout,
			Transport:     &http.Transport{DialContext: guardedDialer(true).DialContext},
			CheckRedirect: sameHostRedirect,
		},
		public: &http.Client{
			Timeout:       fetchTimeout,
			Transport:     &http.Transport{DialContext: guardedDialer(false).DialContext},
			CheckRedirect: sameHostRedirect,
		},
	}
}

// clientFor picks the client a link is allowed to be fetched with, and reports
// why a link is refused outright.
func (f *fetcher) clientFor(link, callbackURL string) (*http.Client, error) {
	parsed, err := url.Parse(link)
	if err != nil {
		return nil, errors.InvalidArgument("custom: unparsable file url",
			errors.WithCause(err), errors.WithID(clientForErrID))
	}

	switch strings.ToLower(parsed.Scheme) {
	case "http", "https":
	default:
		return nil, errors.InvalidArgument(fmt.Sprintf("custom: refusing to fetch %q, http(s) url required", parsed.Redacted()),
			errors.WithID(clientForErrID))
	}

	if parsed.Host == "" {
		return nil, errors.InvalidArgument("custom: refusing to fetch a url without a host",
			errors.WithID(clientForErrID))
	}

	if sameEndpoint(parsed, callbackURL) {
		return f.trusted, nil
	}

	return f.public, nil
}

func sameEndpoint(link *url.URL, callbackURL string) bool {
	callback, err := url.Parse(callbackURL)
	if err != nil {
		return false
	}

	if callback.Hostname() == "" || !strings.EqualFold(callback.Hostname(), link.Hostname()) {
		return false
	}

	return portOf(callback) == portOf(link)
}

func portOf(u *url.URL) string {
	if p := u.Port(); p != "" {
		return p
	}

	if strings.EqualFold(u.Scheme, "https") {
		return "443"
	}

	return "80"
}

func sameHostRedirect(req *http.Request, via []*http.Request) error {
	if len(via) >= maxLinkRedirects {
		return errors.Forbidden(fmt.Sprintf("custom: too many redirects for %q", req.URL.Redacted()),
			errors.WithID("custom.fetch.same_host_redirect"))
	}

	if origin := via[0].URL; !strings.EqualFold(origin.Hostname(), req.URL.Hostname()) {
		return errors.Forbidden(fmt.Sprintf("custom: refusing to follow a cross-host redirect to %q", req.URL.Redacted()),
			errors.WithID("custom.fetch.same_host_redirect"))
	}

	return nil
}

// guardedDialer refuses, at dial time, the address ranges a partner's
// middleware is never legitimately on and that an SSRF is actually after:
// loopback, the link-local block where cloud metadata lives, the unspecified
// address and multicast. allowPrivate keeps RFC1918 reachable, because an
// on-premise middleware usually sits on a private address — that is the whole
// reason the trusted client exists.
func guardedDialer(allowPrivate bool) *net.Dialer {
	return &net.Dialer{
		Timeout: 5 * time.Second,
		Control: func(_, address string, _ syscall.RawConn) error {
			host, _, err := net.SplitHostPort(address)
			if err != nil {
				return err
			}

			ip := net.ParseIP(host)
			if ip == nil || !dialableIP(ip, allowPrivate) {
				return errors.Forbidden(fmt.Sprintf("custom: refusing to dial %q", host),
					errors.WithID("custom.fetch.guarded_dialer"))
			}

			return nil
		},
	}
}

func dialableIP(ip net.IP, allowPrivate bool) bool {
	if ip.IsLoopback() || ip.IsUnspecified() ||
		ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() ||
		ip.IsMulticast() || ip.IsInterfaceLocalMulticast() {
		return false
	}

	if ip.IsPrivate() {
		return allowPrivate
	}

	// 100.64.0.0/10, the carrier-grade NAT range, is not covered by IsPrivate.
	if v4 := ip.To4(); v4 != nil && v4[0] == 100 && v4[1] >= 64 && v4[1] <= 127 {
		return allowPrivate
	}

	return true
}

func (f *fetcher) get(ctx context.Context, gate *custommodel.CustomGate, link string) (*http.Response, error) {
	httpClient, err := f.clientFor(link, gate.CallbackURL)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, link, nil)
	if err != nil {
		return nil, errors.Internal("custom: build file request",
			errors.WithCause(err), errors.WithID(getErrID))
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, errors.WithID(getErrID))
	}

	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()

		return nil, errors.Unavailable("custom: file download status "+resp.Status,
			errors.WithID(getErrID))
	}

	return resp, nil
}
