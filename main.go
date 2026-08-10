//go:build js && wasm

package main

import (
	"context"
	"fmt"
	"laundry-tokyo/internal/ingest"
	"laundry-tokyo/internal/laundrich"
	"laundry-tokyo/internal/monitor"
	"laundry-tokyo/internal/seed"
	"laundry-tokyo/internal/store"
	"strconv"

	"github.com/syumai/workers/cloudflare"
	"github.com/syumai/workers/cloudflare/cron"
	"github.com/syumai/workers/cloudflare/fetch"
)

const (
	cronSeed    = "0 3 * * 1"
	cronMonitor = "*/5 * * * *"
)

func main() {
	cron.ScheduleTask(task)
}

func task(ctx context.Context) error {
	e, err := cron.NewEvent(ctx)
	if err != nil {
		return fmt.Errorf("new event: %w", err)
	}

	switch e.Cron {
	case cronSeed:
		return runSeed(ctx)
	case cronMonitor:
		return runMonitor(ctx)
	default:
		return fmt.Errorf("unknown cron: %q", e.Cron)
	}
}

func runSeed(ctx context.Context) error {
	maxShops, err := strconv.Atoi(cloudflare.Getenv("MAX_SHOPS"))
	if err != nil {
		return fmt.Errorf("parse MAX_SHOPS: %w", err)
	}

	httpClient := fetch.NewClient().HTTPClient(fetch.RedirectModeFollow)

	lc, err := laundrich.New(
		httpClient,
		cloudflare.Getenv("USER_AGENT"),
	)
	if err != nil {
		return fmt.Errorf("new laundrich: %w", err)
	}

	ic, err := ingest.New(
		httpClient,
		cloudflare.Getenv("SHOP_INGEST_URL"),
		cloudflare.Getenv("INGEST_TOKEN"),
	)
	if err != nil {
		return fmt.Errorf("new ingest: %w", err)
	}

	kv, err := store.NewKV()
	if err != nil {
		return fmt.Errorf("new KV: %w", err)
	}

	r2, err := store.NewR2()
	if err != nil {
		return fmt.Errorf("new R2: %w", err)
	}

	pref := cloudflare.Getenv("PREF")
	if pref == "" {
		return fmt.Errorf("PREF must not be empty")
	}

	s := &seed.Seeder{
		Laundrich: lc,
		Ingest:    ic,
		KV:        kv,
		R2:        r2,
		Pref:      pref,
		MaxShops:  maxShops,
	}

	return s.Run(ctx)
}

func runMonitor(ctx context.Context) error {
	spreadSec, err := strconv.Atoi(cloudflare.Getenv("MONITOR_SPREAD_SEC"))
	if err != nil {
		return fmt.Errorf("parse MONITOR_SPREAD_SEC: %w", err)
	}

	concurrency, err := strconv.Atoi(cloudflare.Getenv("MONITOR_CONCURRENCY"))
	if err != nil {
		return fmt.Errorf("parse MONITOR_CONCURRENCY: %w", err)
	}

	httpClient := fetch.NewClient().HTTPClient(fetch.RedirectModeFollow)

	lc, err := laundrich.New(
		httpClient,
		cloudflare.Getenv("USER_AGENT"),
	)
	if err != nil {
		return fmt.Errorf("new laundrich: %w", err)
	}

	ic, err := ingest.New(
		httpClient,
		cloudflare.Getenv("STATUS_INGEST_URL"),
		cloudflare.Getenv("INGEST_TOKEN"),
	)
	if err != nil {
		return fmt.Errorf("new ingest: %w", err)
	}

	kv, err := store.NewKV()
	if err != nil {
		return fmt.Errorf("new KV: %w", err)
	}

	r2, err := store.NewR2()
	if err != nil {
		return fmt.Errorf("new R2: %w", err)
	}

	m := &monitor.Monitor{
		Laundrich:   lc,
		Ingest:      ic,
		KV:          kv,
		R2:          r2,
		SpreadSec:   spreadSec,
		Concurrency: concurrency,
	}

	return m.Run(ctx)
}
