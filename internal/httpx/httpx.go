// Package httpx provides bounded HTTP requests with SSRF-safe dialing.
// Paid requests retry only on explicit rejection.
// They never retry ambiguous transport or 5xx errors.
package httpx

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var blocked = []netip.Prefix{
	netip.MustParsePrefix("0.0.0.0/8"), netip.MustParsePrefix("100.64.0.0/10"),
	netip.MustParsePrefix("192.0.0.0/24"), netip.MustParsePrefix("192.0.2.0/24"),
	netip.MustParsePrefix("192.88.99.0/24"), netip.MustParsePrefix("198.18.0.0/15"),
	netip.MustParsePrefix("198.51.100.0/24"), netip.MustParsePrefix("203.0.113.0/24"),
	netip.MustParsePrefix("240.0.0.0/4"), netip.MustParsePrefix("2001::/23"),
	netip.MustParsePrefix("2001:db8::/32"), netip.MustParsePrefix("2002::/16"),
}

func publicIP(a netip.Addr) bool {
	a = a.Unmap()
	if !a.IsGlobalUnicast() || a.IsPrivate() || a.IsLoopback() || a.IsLinkLocalUnicast() {
		return false
	}
	// Restrict IPv6 to allocated global unicast; excludes translation/local ranges.
	if a.Is6() && !netip.MustParsePrefix("2000::/3").Contains(a) {
		return false
	}
	for _, p := range blocked {
		if p.Contains(a) {
			return false
		}
	}
	return true
}

// ValidateURL rejects credentials, non-web schemes, and unusual ports.
// The dialer re-checks DNS so redirects and rebinding cannot bypass this.
func ValidateURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || len(raw) > 8192 || u.Hostname() == "" || u.User != nil || u.Opaque != "" || (u.Scheme != "https" && u.Scheme != "http") {
		return errors.New("article URL must be an absolute public HTTP(S) URL without credentials")
	}
	if p := u.Port(); p != "" && p != "80" && p != "443" {
		return errors.New("article URL port must be 80 or 443")
	}
	if strings.Contains(u.Hostname(), "%") {
		return errors.New("scoped addresses are not allowed")
	}
	if a, e := netip.ParseAddr(u.Hostname()); e == nil && !publicIP(a) {
		return errors.New("article URL must use a public address")
	}
	return nil
}

// ValidatePublicURL also resolves the destination before a third-party reader is used.
func ValidatePublicURL(ctx context.Context, raw string) error {
	if err := ValidateURL(raw); err != nil {
		return err
	}
	u, _ := url.Parse(raw)
	ips, err := net.DefaultResolver.LookupNetIP(ctx, "ip", u.Hostname())
	if err != nil {
		return errors.New("destination DNS lookup failed")
	}
	if len(ips) == 0 {
		return errors.New("destination has no addresses")
	}
	for _, ip := range ips {
		if !publicIP(ip) {
			return errors.New("destination resolves to a non-public address")
		}
	}
	return nil
}

func publicDial(ctx context.Context, network, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, errors.New("invalid destination")
	}
	ips, err := net.DefaultResolver.LookupNetIP(ctx, "ip", host)
	if err != nil {
		return nil, errors.New("destination DNS lookup failed")
	}
	if len(ips) == 0 {
		return nil, errors.New("destination has no addresses")
	}
	for _, ip := range ips {
		if !publicIP(ip) {
			return nil, errors.New("destination resolves to a non-public address")
		}
	}
	d := net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}
	for _, ip := range ips {
		c, e := d.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
		if e == nil {
			return c, nil
		}
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
	}
	return nil, errors.New("destination connection failed")
}

func NewClient(timeout time.Duration, publicOnly bool) *http.Client {
	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.Proxy = nil
	tr.ResponseHeaderTimeout = 30 * time.Second
	tr.TLSHandshakeTimeout = 10 * time.Second
	if publicOnly {
		tr.DialContext = publicDial
	}
	return &http.Client{Transport: tr, Timeout: timeout, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if !publicOnly {
			return errors.New("provider redirects are disabled")
		}
		if len(via) >= 5 {
			return errors.New("too many redirects")
		}
		return ValidateURL(req.URL.String())
	}}
}

// Do returns only sanitized errors: provider response bodies may contain secrets.
func Do(client *http.Client, req *http.Request, limit int64, paid bool) ([]byte, error) {
	for attempt := 0; attempt < 3; attempt++ {
		current := req.Clone(req.Context())
		if req.GetBody != nil {
			b, e := req.GetBody()
			if e != nil {
				return nil, errors.New("cannot prepare request")
			}
			current.Body = b
		}
		resp, err := client.Do(current)
		if err != nil {
			if req.Context().Err() != nil {
				return nil, req.Context().Err()
			}
			if paid || attempt == 2 {
				return nil, errors.New("upstream request failed or timed out")
			}
			if err = wait(req.Context(), backoff(attempt)); err != nil {
				return nil, err
			}
			continue
		}
		retry := resp.StatusCode == 429 || (paid && resp.StatusCode == 529) || (!paid && (resp.StatusCode == 408 || resp.StatusCode >= 500))
		if retry && attempt < 2 {
			delay := backoff(attempt)
			if h := resp.Header.Get("Retry-After"); h != "" {
				if d, ok := retryAfter(h); ok {
					if d > 30*time.Second {
						resp.Body.Close()
						return nil, fmt.Errorf("upstream HTTP %d; retry later", resp.StatusCode)
					}
					if d > delay {
						delay = d
					}
				}
			}
			resp.Body.Close()
			if err = wait(req.Context(), delay); err != nil {
				return nil, err
			}
			continue
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			resp.Body.Close()
			return nil, fmt.Errorf("upstream HTTP %d", resp.StatusCode)
		}
		body, e := io.ReadAll(io.LimitReader(resp.Body, limit+1))
		resp.Body.Close()
		if e != nil {
			return nil, errors.New("upstream response incomplete")
		}
		if int64(len(body)) > limit {
			return nil, errors.New("upstream response exceeds size limit")
		}
		if len(body) == 0 {
			return nil, errors.New("upstream response is empty")
		}
		return body, nil
	}
	return nil, errors.New("upstream retries exhausted")
}
func backoff(attempt int) time.Duration {
	base := time.Second * time.Duration(1<<attempt)
	return base/2 + time.Duration(rand.Int64N(int64(base/2)))
}
func retryAfter(v string) (time.Duration, bool) {
	if n, e := strconv.ParseInt(v, 10, 64); e == nil && n >= 0 {
		if n > 30 {
			return 31 * time.Second, true
		}
		return time.Duration(n) * time.Second, true
	}
	if t, e := http.ParseTime(v); e == nil {
		d := time.Until(t)
		if d < 0 {
			d = 0
		}
		return d, true
	}
	return 0, false
}
func wait(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
