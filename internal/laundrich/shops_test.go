package laundrich

import (
	"os"
	"reflect"
	"testing"
)

func TestParseShop(t *testing.T) {
	tests := []struct {
		name string
		in   shopRecord
		want Shop
	}{
		{
			name: "typical shop with training-space coords",
			in: shopRecord{
				ANKSHOPID:       "10025962",
				KNJSHOPNAME:     "コインランドリーボーテ　甚目寺北店",
				KANASHOPNAME:    "こいんらんどりーぼーて　じもくじきたてん",
				KNJADDRESS1:     "愛知県",
				KNJADDRESS2:     "あま市",
				KNJADDRESS3:     "森3丁目14-5",
				ANKPOSTNUM:      "4901107",
				ANKPARALLEL:     "35.20924333 ",
				ANKLONGITUDE:    "136.8128425 ",
				NUMFACILITYLIST: "1,15,16",
				MIXBUSINESSHOUR: "朝5:00～深夜24:00",
				MIXHOLIDAY:      "年中無休",
			},
			want: Shop{
				ShopID:        "10025962",
				Name:          "コインランドリーボーテ　甚目寺北店",
				NameKana:      "こいんらんどりーぼーて　じもくじきたてん",
				Pref:          "愛知県",
				City:          "あま市",
				Address:       "森3丁目14-5",
				Postal:        "4901107",
				Lat:           35.20924333,
				Lng:           136.8128425,
				IsIoTEnabled:  true,
				Facilities:    []string{"1", "15", "16"},
				BusinessHours: "朝5:00～深夜24:00",
				ClosedDays:    "年中無休",
			},
		},
		{
			name: "facilities without code 1 is not IoT-enabled",
			in: shopRecord{
				ANKSHOPID:       "x",
				NUMFACILITYLIST: "16,21",
			},
			want: Shop{
				ShopID:       "x",
				IsIoTEnabled: false,
				Facilities:   []string{"16", "21"},
			},
		},
		{
			name: "null facilities normalized to empty slice",
			in: shopRecord{
				ANKSHOPID: "x",
			},
			want: Shop{
				ShopID:     "x",
				Facilities: []string{},
			},
		},
		{
			name: "invalid coords fall back to zero",
			in: shopRecord{
				ANKSHOPID:    "x",
				ANKPARALLEL:  "",
				ANKLONGITUDE: "abc",
			},
			want: Shop{
				ShopID:     "x",
				Lat:        0,
				Lng:        0,
				Facilities: []string{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseShop(tt.in)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseShop() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestParseShops(t *testing.T) {
	data, err := os.ReadFile("testdata/shops.json")
	if err != nil {
		t.Fatal(err)
	}

	shops, err := ParseShops(data)
	if err != nil {
		t.Fatalf("ParseShops() error = %v", err)
	}

	// Statistical fingerprint of the 2026-07-31 snapshot.
	if got, want := len(shops), 3987; got != want {
		t.Errorf("total = %d, want %d", got, want)
	}

	var iot, tokyo, zeroCoord int

	for _, s := range shops {
		if s.IsIoTEnabled {
			iot++
		}

		if s.Pref == "東京都" {
			tokyo++
		}

		if s.Lat == 0 && s.Lng == 0 {
			zeroCoord++
		}
	}

	if got, want := iot, 3618; got != want {
		t.Errorf("IoT-enabled = %d, want %d", got, want)
	}

	if got, want := tokyo, 806; got != want {
		t.Errorf("東京都 = %d, want %d", got, want)
	}

	if got, want := zeroCoord, 27; got != want {
		t.Errorf("zero-coord = %d, want %d", got, want)
	}
}

func TestParseShops_InvalidBody(t *testing.T) {
	tests := []struct {
		name string
		in   string
	}{
		{name: "empty", in: ""},
		{name: "whitespace only", in: "    "},
		{name: "JSON object not array", in: `{"a":1}`},
		{name: "HTML error page", in: "<html><body>error</body></html>"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := ParseShops([]byte(tt.in)); err == nil {
				t.Error("want error, got nil")
			}
		})
	}
}
