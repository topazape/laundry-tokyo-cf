package laundrich

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const (
	baseURL    = "https://www.coin-laundry.co.jp/userp/view_interface.php"
	refererURL = "https://www.coin-laundry.co.jp/userp/shop_search"
)

type Client struct {
	httpClient *http.Client
	userAgent  string
	baseURL    string
}

func New(httpClient *http.Client, userAgent string) (*Client, error) {
	if httpClient == nil {
		return nil, fmt.Errorf("httpClient must not be nil")
	}

	if userAgent == "" {
		return nil, fmt.Errorf("userAgent must not be empty")
	}

	return &Client{httpClient: httpClient, userAgent: userAgent, baseURL: baseURL}, nil
}

func (c *Client) FetchShopsRaw(ctx context.Context) ([]byte, error) {
	form := url.Values{
		"className":         {"AilsGoogleMap"},
		"data[PREFECTURES]": {""},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Referer", refererURL)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch shops: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	return body, nil
}
