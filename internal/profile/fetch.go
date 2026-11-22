package profile

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const (
	DefaultUserAgent = "mihomo/v1.19.34"
	fetchTimeout     = 30 * time.Second
	downloadTimeout  = 2 * time.Minute
)

type Response struct {
	Body   []byte
	Header http.Header
}

type Fetcher struct {
	ProxyURL func() string
}

// Fetch downloads url. Errors never include the URL: it carries the subscription token.
func (f *Fetcher) Fetch(ctx context.Context, rawURL, userAgent string) (Response, error) {
	resp, err := fetch(ctx, nil, rawURL, userAgent)
	if err == nil {
		return resp, nil
	}
	if f.ProxyURL == nil {
		return Response{}, err
	}
	proxy := f.ProxyURL()
	if proxy == "" {
		return Response{}, err
	}
	proxyURL, perr := url.Parse(proxy)
	if perr != nil {
		return Response{}, err
	}
	viaProxy, perr := fetch(ctx, proxyURL, rawURL, userAgent)
	if perr != nil {
		return Response{}, errors.Join(err, fmt.Errorf("via core: %w", perr))
	}
	return viaProxy, nil
}

func fetch(ctx context.Context, proxy *url.URL, rawURL, userAgent string) (Response, error) {
	ctx, cancel := context.WithTimeout(ctx, fetchTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return Response{}, errors.New("profile: invalid subscription URL")
	}
	req.Header.Set("User-Agent", cmp.Or(userAgent, DefaultUserAgent))

	// System proxy settings are ignored on purpose: the "via core" retry is explicit.
	transport := &http.Transport{}
	if proxy != nil {
		transport.Proxy = http.ProxyURL(proxy)
	}
	defer transport.CloseIdleConnections()
	resp, err := (&http.Client{Transport: transport}).Do(req)
	if err != nil {
		return Response{}, fmt.Errorf("profile: download failed: %w", redact(err))
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Response{}, fmt.Errorf("profile: provider answered %s", resp.Status)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes))
	if err != nil {
		return Response{}, fmt.Errorf("profile: read body: %w", redact(err))
	}
	return Response{Body: body, Header: resp.Header}, nil
}

func redact(err error) error {
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return urlErr.Err
	}
	return err
}

func (f *Fetcher) Download(p Profile) (Profile, []byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), downloadTimeout)
	defer cancel()
	resp, err := f.Fetch(ctx, p.URL, p.UserAgent)
	if err != nil {
		return p, nil, err
	}
	if _, err := Parse(resp.Body); err != nil {
		return p, nil, err
	}
	meta := ParseHeaders(resp.Header)
	if p.ID == "" {
		if p.ID, err = NewID(); err != nil {
			return p, nil, err
		}
		p.Name = cmp.Or(meta.Title, p.Host(), "Subscription")
		if p.UpdateIntervalHours == 0 {
			p.UpdateIntervalHours = meta.UpdateIntervalHours
		}
	}
	p.Info = meta.Info
	p.UpdatedAt = time.Now()
	return p, resp.Body, nil
}
