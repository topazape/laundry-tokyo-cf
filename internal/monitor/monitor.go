//go:build js && wasm

package monitor

import (
	"bytes"
	"context"
	"fmt"
	"laundry-tokyo/internal/ingest"
	"laundry-tokyo/internal/laundrich"
	"laundry-tokyo/internal/store"
	"log"
	"time"

	"golang.org/x/sync/errgroup"
)

const fetchTimeout = 240 * time.Second

type Monitor struct {
	Laundrich   *laundrich.Client
	Ingest      *ingest.Client
	KV          *store.KV
	R2          *store.R2
	SpreadSec   int
	Concurrency int
}

func (m *Monitor) Run(ctx context.Context) error {
	fetchedAt := time.Now().UTC()

	ids, err := m.KV.GetShopIDs()
	if err != nil {
		return fmt.Errorf("KV.GetShopIDs: %w", err)
	}

	if len(ids) == 0 {
		return fmt.Errorf("no shop ids in KV")
	}

	fetchCtx, cancel := context.WithTimeout(ctx, fetchTimeout)
	defer cancel()

	raws, err := m.fetchAll(fetchCtx, ids)
	if err != nil {
		return fmt.Errorf("fetchAll: %w", err)
	}

	if len(raws) == 0 {
		return fmt.Errorf("all %d shops failed", len(ids))
	}

	var (
		buf      bytes.Buffer
		statuses []laundrich.Status
		ok       int
	)

	for _, raw := range raws {
		s, err := laundrich.ParseStatuses(raw)
		if err != nil {
			continue
		}

		ndjson, err := rawToNDJSON(raw)
		if err != nil {
			continue
		}

		buf.Write(ndjson)

		statuses = append(statuses, s...)
		ok++
	}

	log.Printf("monitor: shops=%d/%d fetched=%d rows=%d elapsed=%s", ok, len(ids), len(raws), len(statuses), time.Since(fetchedAt))

	if err := m.R2.PutRawStatuses(fetchedAt, buf.Bytes()); err != nil {
		return fmt.Errorf("R2.PutRawStatuses: %w", err)
	}

	if err := m.Ingest.Send(ctx, toStreamStatuses(statuses, fetchedAt)); err != nil {
		return fmt.Errorf("Ingest.Send: %w", err)
	}

	log.Printf("monitor: done total=%s", time.Since(fetchedAt))

	return nil
}

func (m *Monitor) fetchAll(ctx context.Context, ids []string) ([][]byte, error) {
	interval := time.Duration(m.SpreadSec) * time.Second / time.Duration(len(ids))

	raws := make([][]byte, len(ids))
	errs := make([]error, len(ids))

	var g errgroup.Group
	g.SetLimit(m.Concurrency)

	dispatched := 0

	for i, id := range ids {
		if ctx.Err() != nil {
			break
		}

		dispatched++

		g.Go(func() error {
			raw, err := m.Laundrich.FetchStatusesRaw(ctx, id)
			if err != nil {
				errs[i] = err

				return nil
			}

			raws[i] = raw

			return nil
		})

		time.Sleep(interval)
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	if dispatched < len(ids) {
		log.Printf("monitor: not dispatched=%d (%v)", len(ids)-dispatched, ctx.Err())
	}

	var (
		failed   int
		firstErr error
	)

	for _, err := range errs {
		if err != nil {
			failed++

			if firstErr == nil {
				firstErr = err
			}
		}
	}

	if failed > 0 {
		log.Printf("monitor: fetch failed=%d first_err=%v", failed, firstErr)
	}

	got := make([][]byte, 0, len(raws))

	for _, raw := range raws {
		if raw != nil {
			got = append(got, raw)
		}
	}

	return got, nil
}
