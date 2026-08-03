package seed

import (
	"encoding/json"
	"laundry-tokyo/internal/laundrich"
	"reflect"
	"testing"
	"time"
)

func TestFilterShopIDs(t *testing.T) {
	shops := []laundrich.Shop{
		{ShopID: "1", Pref: "東京都", IsIoTEnabled: true},
		{ShopID: "2", Pref: "東京都", IsIoTEnabled: false},
		{ShopID: "3", Pref: "愛知県", IsIoTEnabled: true},
		{ShopID: "4", Pref: "東京都", IsIoTEnabled: true},
		{ShopID: "5", Pref: "東京都", IsIoTEnabled: true},
	}

	tests := []struct {
		name     string
		shops    []laundrich.Shop
		pref     string
		maxShops int
		want     []string
	}{
		{
			name:     "iot enabled in pref",
			shops:    shops,
			pref:     "東京都",
			maxShops: 0,
			want:     []string{"1", "4", "5"},
		},
		{
			name:     "maxShops truncates after filtering",
			shops:    shops,
			pref:     "東京都",
			maxShops: 2,
			want:     []string{"1", "4"},
		},
		{
			name:     "maxShops larger than matches",
			shops:    shops,
			pref:     "東京都",
			maxShops: 10,
			want:     []string{"1", "4", "5"},
		},
		{
			name:     "no match returns empty slice not nil",
			shops:    shops,
			pref:     "大阪府",
			maxShops: 0,
			want:     []string{},
		},
		{
			name:     "empty input",
			shops:    nil,
			pref:     "東京都",
			maxShops: 0,
			want:     []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filterShopIDs(tt.shops, tt.pref, tt.maxShops)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("filterShopIDs() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestToRows(t *testing.T) {
	fetchedAt := time.Date(2026, 8, 1, 3, 0, 0, 0, time.UTC)

	shops := []laundrich.Shop{
		{
			ShopID:        "10025962",
			Name:          "店A",
			NameKana:      "てんえー",
			Pref:          "東京都",
			City:          "新宿区",
			Address:       "西新宿1-1",
			Postal:        "1600023",
			Lat:           35.69,
			Lng:           139.69,
			IsIoTEnabled:  true,
			Facilities:    []string{"1", "16"},
			BusinessHours: "24時間",
			ClosedDays:    "年中無休",
		},
		{
			ShopID:     "2",
			Facilities: []string{},
		},
	}

	rows := toStreamShops(shops, fetchedAt)

	if got, want := len(rows), 2; got != want {
		t.Fatalf("len(rows) = %d, want %d", got, want)
	}

	// marshal 結果で schema 3.3(snake_case・RFC3339)との一致を検証する
	got, err := json.Marshal(rows[0])
	if err != nil {
		t.Fatalf("marshal row: %v", err)
	}

	want := `{"shop_id":"10025962","name":"店A","name_kana":"てんえー","pref":"東京都","city":"新宿区","address":"西新宿1-1","postal":"1600023","lat":35.69,"lng":139.69,"is_iot_enabled":true,"facilities":["1","16"],"business_hours":"24時間","closed_days":"年中無休","fetched_at":"2026-08-01T03:00:00Z"}`
	if string(got) != want {
		t.Errorf("row JSON = %s, want %s", got, want)
	}

	// 全行が同じ snapshot 識別子を持つ
	row1, ok := rows[1].(streamShop)
	if !ok {
		t.Fatalf("rows[1] is %T, want streamShop", rows[1])
	}

	if !row1.FetchedAt.Equal(fetchedAt) {
		t.Errorf("rows[1].FetchedAt = %v, want %v", row1.FetchedAt, fetchedAt)
	}
}

func TestRawToNDJSON(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			// キー順が辞書順でない入力が原文のまま出れば RawMessage が効いている
			name: "preserves element bytes as-is",
			in:   `[{"b":1,"a":2},{"x":"y"}]`,
			want: "{\"b\":1,\"a\":2}\n{\"x\":\"y\"}\n",
		},
		{
			name: "empty array",
			in:   `[]`,
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := rawToNDJSON([]byte(tt.in))
			if err != nil {
				t.Fatalf("rawToNDJSON() error = %v", err)
			}

			if string(got) != tt.want {
				t.Errorf("rawToNDJSON() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRawToNDJSON_Invalid(t *testing.T) {
	tests := []struct {
		name string
		in   string
	}{
		{name: "not an array", in: `{"a":1}`},
		{name: "HTML error page", in: "<html>error</html>"},
		{name: "empty", in: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := rawToNDJSON([]byte(tt.in)); err == nil {
				t.Error("want error, got nil")
			}
		})
	}
}
