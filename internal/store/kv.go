//go:build js && wasm

package store

import (
	"encoding/json/v2"
	"fmt"
	"time"

	"github.com/syumai/workers/cloudflare/kv"
)

const (
	keyShopIDs = "monitor:shop_ids"
	keyLastRun = "seed:last_run"
)

type KV struct {
	ns *kv.Namespace
}

func NewKV() (*KV, error) {
	ns, err := kv.NewNamespace("KV")
	if err != nil {
		return nil, fmt.Errorf("open KV namespace: %w", err)
	}

	return &KV{ns: ns}, nil
}

func (s *KV) PutShopIDs(ids []string) error {
	data, err := json.Marshal(ids)
	if err != nil {
		return fmt.Errorf("marshal shop ids: %w", err)
	}

	if err := s.ns.PutString(keyShopIDs, string(data), nil); err != nil {
		return fmt.Errorf("put %s: %w", keyShopIDs, err)
	}

	return nil
}

func (s *KV) GetShopIDs() ([]string, error) {
	data, err := s.ns.GetString(keyShopIDs, nil)
	if err != nil {
		return nil, fmt.Errorf("get %s: %w", keyShopIDs, err)
	}

	var ids []string
	if err := json.Unmarshal([]byte(data), &ids); err != nil {
		return nil, fmt.Errorf("unmarshal shop ids: %w", err)
	}

	return ids, nil
}

func (s *KV) PutLastRun(t time.Time) error {
	if err := s.ns.PutString(keyLastRun, t.UTC().Format(time.RFC3339), nil); err != nil {
		return fmt.Errorf("put %s: %w", keyLastRun, err)
	}

	return nil
}
