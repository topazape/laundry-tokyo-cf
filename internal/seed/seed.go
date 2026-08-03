//go:build js && wasm

package seed

import (
	"context"
	"fmt"
	"laundry-tokyo/internal/ingest"
	"laundry-tokyo/internal/laundrich"
	"laundry-tokyo/internal/store"
	"time"
)

type Seeder struct {
	Laundrich *laundrich.Client
	Ingest    *ingest.Client
	KV        *store.KV
	R2        *store.R2
	Pref      string
	MaxShops  int
}

func (s *Seeder) Run(ctx context.Context) error {
	fetchedAt := time.Now().UTC()

	raw, err := s.Laundrich.FetchShopsRaw(ctx)
	if err != nil {
		return fmt.Errorf("FetchShopsRaw: %w", err)
	}

	ndjson, err := rawToNDJSON(raw)
	if err != nil {
		return fmt.Errorf("rawToNDJSON: %w", err)
	}

	if err := s.R2.PutRawShops(fetchedAt, ndjson); err != nil {
		return fmt.Errorf("R2.PutRawShops: %w", err)
	}

	shops, err := laundrich.ParseShops(raw)
	if err != nil {
		return fmt.Errorf("laundrich.ParseShops: %w", err)
	}

	if err := s.KV.PutShopIDs(filterShopIDs(shops, s.Pref, s.MaxShops)); err != nil {
		return fmt.Errorf("KV.PutShopIDs: %w", err)
	}

	if err := s.Ingest.Send(ctx, toStreamShops(shops, fetchedAt)); err != nil {
		return fmt.Errorf("Ingest.Send: %w", err)
	}

	return s.KV.PutLastRun(fetchedAt)
}
