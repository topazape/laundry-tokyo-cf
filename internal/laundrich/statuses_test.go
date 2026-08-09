package laundrich

import (
	"os"
	"reflect"
	"testing"
	"time"
)

func TestParseStatus(t *testing.T) {
	tests := []struct {
		name string
		in   statusRecord
		want Status
	}{
		{
			name: "typical available machine",
			in: statusRecord{
				ANKSHOPID:     "11002442",
				ANKMACHINENUM: "01",
				MIXPARTNUM:    "MCW-W7C(N)",
				KNJMACHNAME:   "スニーカーウォッシャー",
				ANKCOMFLG:     "1",
				ANKAILSFLG:    "1",
				ANKKHKIND:     "4",
				ANKDRUMFLG:    "0",
				ANKPOL:        "A1",
				ANKK_GU:       "T",
				ANKK_GT:       "000",
				ANKK_GK:       "00",
				ANKSTS:        "0",
				NUMDOORLOCK:   0,
				NUMCOINLOCK:   0,
				DTMCREATE:     "2026-08-07T12:12:26.4110128",
			},
			want: Status{
				ShopID:           "11002442",
				MachineID:        "01-A1",
				MachineKind:      "4",
				Status:           "available",
				StatusCode:       "T",
				RemainingMinutes: new(0),
				CourseCode:       "00",
				ReportedAt:       new(time.Date(2026, 8, 7, 3, 12, 26, 411012800, time.UTC)),
			},
		},
		{
			name: "in use lower drum with remaining minutes",
			in: statusRecord{
				ANKSHOPID:     "11002442",
				ANKMACHINENUM: "09",
				ANKPOL:        "A2",
				ANKKHKIND:     "2",
				ANKK_GU:       "U",
				ANKK_GT:       "028",
				ANKK_GK:       "01",
				DTMCREATE:     "2026-08-07T12:12:26.42913",
			},
			want: Status{
				ShopID:           "11002442",
				MachineID:        "09-A2",
				MachineKind:      "2",
				Status:           "in_use",
				StatusCode:       "U",
				RemainingMinutes: new(28),
				CourseCode:       "01",
				ReportedAt:       new(time.Date(2026, 8, 7, 3, 12, 26, 429130000, time.UTC)),
			},
		},
		{
			name: "status code E is out of service",
			in: statusRecord{
				ANKSHOPID:     "x",
				ANKMACHINENUM: "01",
				ANKPOL:        "A1",
				ANKK_GU:       "E",
				ANKK_GT:       "000",
				DTMCREATE:     "2026-08-07T12:12:26.4110128",
			},
			want: Status{
				ShopID:           "x",
				MachineID:        "01-A1",
				Status:           "out_of_service",
				StatusCode:       "E",
				RemainingMinutes: new(0),
				ReportedAt:       new(time.Date(2026, 8, 7, 3, 12, 26, 411012800, time.UTC)),
			},
		},
		{
			name: "missing fields yield unknown status and nil pointers",
			in: statusRecord{
				ANKSHOPID:     "x",
				ANKMACHINENUM: "01",
				ANKPOL:        "A1",
			},
			want: Status{
				ShopID:    "x",
				MachineID: "01-A1",
				Status:    "unknown",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseStatus(tt.in)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseStatus() = %+v want %+v", got, tt.want)
			}
		})
	}
}

func TestParseStatuses(t *testing.T) {
	data, err := os.ReadFile("testdata/statuses.json")
	if err != nil {
		t.Fatal(err)
	}

	statuses, err := ParseStatuses(data)
	if err != nil {
		t.Fatalf("ParseStatuses() error = %v", err)
	}

	if got, want := len(statuses), 14; got != want {
		t.Errorf("total = %d, want %d", got, want)
	}

	var inUse, noReportedAt int

	ids := make(map[string]struct{}, len(statuses))

	for _, s := range statuses {
		if s.Status == "in_use" {
			inUse++
		}

		if s.ReportedAt == nil {
			noReportedAt++
		}

		ids[s.MachineID] = struct{}{}
	}

	if got, want := inUse, 1; got != want {
		t.Errorf("in use = %d, want %d", got, want)
	}

	if got, want := noReportedAt, 0; got != want {
		t.Errorf("nil ReportedAt = %d, want %d", got, want)
	}

	if got, want := len(ids), 14; got != want {
		t.Errorf("unique MachineID = %d, want %d", got, want)
	}
}

func TestParseStatuses_InvalidBody(t *testing.T) {
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
			if _, err := ParseStatuses([]byte(tt.in)); err == nil {
				t.Error("want error, got nil")
			}
		})
	}
}
