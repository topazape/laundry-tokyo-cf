package ingest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Pipelines の 5MB / req 制限のため、マージン込みの上限値
const defaultMaxPayload = 4 << 20 // 4MiB

type Client struct {
	httpClient *http.Client
	url        string
	token      string
	maxPayload int
}

func New(httpClient *http.Client, url string, token string) (*Client, error) {
	if httpClient == nil {
		return nil, fmt.Errorf("httpClient must not be nil")
	}

	if url == "" {
		return nil, fmt.Errorf("url must not be empty")
	}

	if token == "" {
		return nil, fmt.Errorf("token must not be empty")
	}

	return &Client{
		httpClient: httpClient,
		url:        url,
		token:      token,
		maxPayload: defaultMaxPayload,
	}, nil
}

func (c *Client) Send(ctx context.Context, rows []any) error {
	var buf bytes.Buffer

	chunk := 0

	for i, row := range rows {
		line, err := json.Marshal(row)
		if err != nil {
			return fmt.Errorf("marshal row %d: %w", i, err)
		}

		if len(line)+1 > c.maxPayload {
			return fmt.Errorf("row %d exceeds max payload (%d bytes)", i, len(line)+1)
		}

		if buf.Len()+len(line)+1 > c.maxPayload {
			chunk++
			if err := c.post(ctx, buf.Bytes()); err != nil {
				return fmt.Errorf("chunk %d: %w", chunk, err)
			}

			buf.Reset()
		}

		buf.Write(line)
		buf.WriteByte('\n')
	}

	if buf.Len() > 0 {
		chunk++
		if err := c.post(ctx, buf.Bytes()); err != nil {
			return fmt.Errorf("chunk %d: %w", chunk, err)
		}
	}

	return nil
}

func (c *Client) post(ctx context.Context, body []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-ndjson")
	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("post: %w", err)
	}

	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, bytes.TrimSpace(msg))
	}

	return nil
}
