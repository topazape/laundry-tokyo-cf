//go:build js && wasm

package store

import (
	"bytes"
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
