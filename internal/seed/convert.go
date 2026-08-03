package seed

import (
	"bytes"
	"encoding/json"
	"fmt"
	"laundry-tokyo/internal/laundrich"
	"time"
)

// stream(shop) schema
type streamShop struct {
	ShopID        string    `json:"shop_id"`
	Name          string    `json:"name"`
	NameKana      string    `json:"name_kana"`
	Pref          string    `json:"pref"`
	City          string    `json:"city"`
	Address       string    `json:"address"`
	Postal        string    `json:"postal"`
	Lat           float64   `json:"lat"`
	Lng           float64   `json:"lng"`
	IsIoTEnabled  bool      `json:"is_iot_enabled"`
	Facilities    []string  `json:"facilities"`
	BusinessHours string    `json:"business_hours"`
	ClosedDays    string    `json:"closed_days"`
	FetchedAt     time.Time `json:"fetched_at"` // RFC3339 で marshal される
}

func filterShopIDs(shops []laundrich.Shop, pref string, maxShops int) []string {
	ids := []string{}

	for _, s := range shops {
		if s.IsIoTEnabled && s.Pref == pref {
			ids = append(ids, s.ShopID)
		}
	}

	if maxShops > 0 && len(ids) > maxShops {
		ids = ids[:maxShops]
	}

	return ids
}

func toStreamShops(shops []laundrich.Shop, fetchedAt time.Time) []any {
	rows := make([]any, 0, len(shops))
	for _, s := range shops {
		rows = append(rows, streamShop{
			ShopID:        s.ShopID,
			Name:          s.Name,
			NameKana:      s.NameKana,
			Pref:          s.Pref,
			City:          s.City,
			Address:       s.Address,
			Postal:        s.Postal,
			Lat:           s.Lat,
			Lng:           s.Lng,
			IsIoTEnabled:  s.IsIoTEnabled,
			Facilities:    s.Facilities,
			BusinessHours: s.BusinessHours,
			ClosedDays:    s.ClosedDays,
			FetchedAt:     fetchedAt,
		})
	}

	return rows
}

func rawToNDJSON(raw []byte) ([]byte, error) {
	var elems []json.RawMessage
	if err := json.Unmarshal(raw, &elems); err != nil {
		return nil, fmt.Errorf("unmarshal raw shops: %w", err)
	}

	var buf bytes.Buffer
	for _, e := range elems {
		buf.Write(e)
		buf.WriteByte('\n')
	}

	return buf.Bytes(), nil
}
