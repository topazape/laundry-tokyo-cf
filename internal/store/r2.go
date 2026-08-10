//go:build js && wasm

package store

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"time"

	"github.com/syumai/workers/cloudflare/r2"
)

type R2 struct {
	bucket *r2.Bucket
}

func NewR2() (*R2, error) {
	b, err := r2.NewBucket("BUCKET")
	if err != nil {
		return nil, fmt.Errorf("open R2 bucket: %w", err)
	}

	return &R2{bucket: b}, nil
}

func (s *R2) PutRawShops(fetchedAt time.Time, data []byte) error {
	key := fmt.Sprintf(
		"shop/dt=%s/shops.ndjson",
		fetchedAt.UTC().Format(time.DateOnly),
	)

	if _, err := s.bucket.Put(key, io.NopCloser(bytes.NewReader(data)), nil); err != nil {
		return fmt.Errorf("put %s: %w", key, err)
	}

	return nil
}

func (s *R2) PutRawStatuses(fetchedAt time.Time, data []byte) error {
	t := fetchedAt.UTC()
	key := fmt.Sprintf(
		"status/dt=%s/statuses-%s.ndjson.gz",
		t.Format(time.DateOnly),
		t.Format("150405"),
	)

	var buf bytes.Buffer

	zw := gzip.NewWriter(&buf)
	if _, err := zw.Write(data); err != nil {
		return fmt.Errorf("gzip %s: %w", key, err)
	}

	if err := zw.Close(); err != nil {
		return fmt.Errorf("gzip %s: %w", key, err)
	}

	if _, err := s.bucket.Put(key, io.NopCloser(bytes.NewReader(buf.Bytes())), nil); err != nil {
		return fmt.Errorf("put %s: %w", key, err)
	}

	return nil
}
