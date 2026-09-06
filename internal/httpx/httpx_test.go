package httpx

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/netip"
	"strings"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestPublicAddressPolicy(t *testing.T) {
	for _, ip := range []string{"127.0.0.1", "10.0.0.1", "169.254.169.254", "100.100.100.200", "192.0.2.1", "198.18.0.1", "240.1.1.1", "::1", "::ffff:127.0.0.1", "fc00::1", "fe80::1", "64:ff9b::a00:1", "2001:db8::1", "2002:7f00:1::"} {
		if publicIP(netip.MustParseAddr(ip)) {
			t.Errorf("accepted %s", ip)
		}
	}
	for _, ip := range []string{"8.8.8.8", "2606:4700:4700::1111"} {
		if !publicIP(netip.MustParseAddr(ip)) {
			t.Errorf("rejected %s", ip)
		}
	}
	for _, u := range []string{"file:///etc/passwd", "http://user:secret@example.com", "http://127.0.0.1", "http://[::ffff:127.0.0.1]", "https://example.com:8080", "//example.com"} {
		if ValidateURL(u) == nil {
			t.Errorf("accepted %s", u)
		}
	}
}
func TestPaidRequestsNeverRetryAmbiguousFailures(t *testing.T) {
	for _, status := range []int{0, 400, 401, 500, 503} {
		t.Run(string(rune(status+65)), func(t *testing.T) {
			calls := 0
			c := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				calls++
				if status == 0 {
					return nil, errors.New("secret transport detail")
				}
				return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader("secret provider detail")), Header: make(http.Header)}, nil
			})}
			r, _ := http.NewRequest("POST", "https://provider.test", strings.NewReader("body"))
			_, err := Do(c, r, 100, true)
			if calls != 1 || err == nil || strings.Contains(err.Error(), "secret") {
				t.Fatalf("calls=%d error=%v", calls, err)
			}
		})
	}
}
func TestRetryAndLimits(t *testing.T) {
	calls := 0
	c := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		calls++
		status := 429
		if calls == 2 {
			status = 200
		}
		return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader("ok")), Header: make(http.Header)}, nil
	})}
	r, _ := http.NewRequest("POST", "https://provider.test", strings.NewReader("body"))
	b, e := Do(c, r, 2, true)
	if e != nil || string(b) != "ok" || calls != 2 {
		t.Fatalf("%s %v %d", b, e, calls)
	}
	c.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("oversized")), Header: make(http.Header)}, nil
	})
	if _, e = Do(c, r, 2, true); e == nil {
		t.Fatal("accepted oversized response")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if !errors.Is(wait(ctx, time.Hour), context.Canceled) {
		t.Fatal("did not cancel backoff")
	}
}
func TestClientSecurity(t *testing.T) {
	c := NewClient(time.Second, true)
	tr := c.Transport.(*http.Transport)
	if tr.Proxy != nil {
		t.Fatal("proxy enabled")
	}
	r, _ := http.NewRequest("GET", "http://169.254.169.254", nil)
	if c.CheckRedirect(r, nil) == nil {
		t.Fatal("private redirect accepted")
	}
	if _, e := publicDial(context.Background(), "tcp", "127.0.0.1:80"); e == nil {
		t.Fatal("private dial accepted")
	}
	c = NewClient(time.Second, false)
	if c.CheckRedirect(r, nil) == nil {
		t.Fatal("provider redirect accepted")
	}
}
