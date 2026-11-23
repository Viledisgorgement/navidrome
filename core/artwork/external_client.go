package artwork

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"
)

// errPrivateAddress is returned when a remote artwork URL resolves to a private,
// loopback, link-local or unspecified address.
var errPrivateAddress = errors.New("remote artwork address is not public")

type proxiedKey struct{}

// externalHTTPClient fetches remote artwork (playlist #EXTALBUMARTURL and agent-supplied
// image URLs). Both are untrusted, so the client refuses to dial non-public addresses.
// The check runs at dial time on the resolved IP, which also covers redirects and DNS
// rebinding. Backport of upstream navidrome 237276ef (GHSA-8hjf-6h34-82hr).
var externalHTTPClient = newExternalHTTPClient(5 * time.Second)

func newExternalHTTPClient(timeout time.Duration) *http.Client {
	dialer := &net.Dialer{Timeout: 30 * time.Second, KeepAlive: 30 * time.Second}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		// A hop going to a configured HTTP(S)_PROXY is the operator's egress policy:
		// dial it directly. Every other dial must reach a public address.
		if ctx.Value(proxiedKey{}) != nil {
			return dialer.DialContext(ctx, network, address)
		}
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, err
		}
		ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
		if err != nil {
			return nil, err
		}
		for _, ip := range ips {
			if isNonPublicIP(ip.IP) {
				return nil, fmt.Errorf("%w: %s resolves to %s", errPrivateAddress, host, ip.IP)
			}
		}
		if len(ips) == 0 {
			return nil, fmt.Errorf("no addresses found for %s", host)
		}
		return dialer.DialContext(ctx, network, net.JoinHostPort(ips[0].IP.String(), port))
	}
	return &http.Client{Timeout: timeout, Transport: &proxyTagger{next: transport, proxy: transport.Proxy}}
}

// proxyTagger marks requests that go through a proxy, so only that hop skips the guard.
// Redirects re-enter RoundTrip, so each hop is tagged on its own.
type proxyTagger struct {
	next  http.RoundTripper
	proxy func(*http.Request) (*url.URL, error)
}

func (t *proxyTagger) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.proxy != nil {
		if p, err := t.proxy(req); err == nil && p != nil {
			req = req.WithContext(context.WithValue(req.Context(), proxiedKey{}, true))
		}
	}
	return t.next.RoundTrip(req)
}

func isNonPublicIP(ip net.IP) bool {
	return ip.IsLoopback() || ip.IsUnspecified() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() || ip.IsMulticast()
}
