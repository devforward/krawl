package fetcher

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/http/httptrace"
	"time"
)

type Redirect struct {
	URL        string
	StatusCode int
}

type Result struct {
	URL           string
	FinalURL      string
	StatusCode    int
	ContentType   string
	ContentLength int64
	Body          []byte
	Headers       http.Header
	Redirects     []Redirect
	DNSTime       time.Duration
	ConnectTime   time.Duration
	TLSTime       time.Duration
	TTFB          time.Duration
	TotalTime     time.Duration
}

func Fetch(url string, timeout time.Duration, userAgent string) (*Result, error) {
	result := &Result{URL: url}

	client := &http.Client{
		Timeout: timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("too many redirects")
			}
			prev := via[len(via)-1]
			result.Redirects = append(result.Redirects, Redirect{
				URL:        prev.URL.String(),
				StatusCode: prev.Response.StatusCode,
			})
			return nil
		},
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	var dnsStart, connStart, tlsStart time.Time
	start := time.Now()

	trace := &httptrace.ClientTrace{
		DNSStart:             func(_ httptrace.DNSStartInfo) { dnsStart = time.Now() },
		DNSDone:              func(_ httptrace.DNSDoneInfo) { result.DNSTime = time.Since(dnsStart) },
		ConnectStart:         func(_, _ string) { connStart = time.Now() },
		ConnectDone:          func(_, _ string, _ error) { result.ConnectTime = time.Since(connStart) },
		TLSHandshakeStart:   func() { tlsStart = time.Now() },
		TLSHandshakeDone:    func(_ tls.ConnectionState, _ error) { result.TLSTime = time.Since(tlsStart) },
		GotFirstResponseByte: func() { result.TTFB = time.Since(start) },
	}
	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching URL: %w", err)
	}
	defer resp.Body.Close()

	result.TotalTime = time.Since(start)
	result.FinalURL = resp.Request.URL.String()
	result.StatusCode = resp.StatusCode
	result.ContentType = resp.Header.Get("Content-Type")
	result.ContentLength = resp.ContentLength
	result.Headers = resp.Header

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading body: %w", err)
	}
	result.Body = body
	if result.ContentLength < 0 {
		result.ContentLength = int64(len(body))
	}

	return result, nil
}
