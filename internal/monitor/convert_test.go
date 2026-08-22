package monitor

import (
	"encoding/json/v2"
	"laundry-tokyo/internal/laundrich"
	"testing"
	"time"
)

func TestToStreamStatuses(t *testing.T) {
	fetchedAt := time.Date(2026, 8, 7, 3, 12, 0, 0, time.UTC)

	statuses := []laundrich.Status{
		{
			ShopID:           "11002442",
			MachineID:        "09-A1",
			MachineKind:      "2",
			Status:           "in_use",
			StatusCode:       "U",
			RemainingMinutes: new(28),
			CourseCode:       "01",
			ReportedAt:       new(time.Date(2026, 8, 7, 3, 12, 26, 423472500, time.UTC)),
		},
		{
			ShopID:    "11002442",
			MachineID: "10-A2",
			Status:    "available",
		},
	}

	rows := toStreamStatuses(statuses, fetchedAt)

	if got, want := len(rows), 2; got != want {
		t.Fatalf("len(rows) = %d, want %d", got, want)
	}

	// marshal 結果で stream(status) schema(snake_case・RFC3339)との一致を検証する
	got, err := json.Marshal(rows[0])
	if err != nil {
		t.Fatalf("marshal row: %v", err)
	}

	want := `{"shop_id":"11002442","machine_id":"09-A1","machine_kind":"2","status":"in_use","status_code":"U","remaining_minutes":28,"course_code":"01","reported_at":"2026-08-07T03:12:26.4234725Z","fetched_at":"2026-08-07T03:12:00Z"}`
	if string(got) != want {
		t.Errorf("row JSON = %s, want %s", got, want)
	}

	// 欠損値は null で出る(stream 側が null 許容である前提)
	got, err = json.Marshal(rows[1])
	if err != nil {
		t.Fatalf("marshal row: %v", err)
	}

	want = `{"shop_id":"11002442","machine_id":"10-A2","machine_kind":"","status":"available","status_code":"","remaining_minutes":null,"course_code":"","reported_at":null,"fetched_at":"2026-08-07T03:12:00Z"}`
	if string(got) != want {
		t.Errorf("row JSON = %s, want %s", got, want)
	}
}

func TestRawToNDJSON(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			// キー順が辞書順でない入力が原文のまま出れば jsontext.Value が効いている
			name: "preserves element bytes as-is",
			in:   `[{"ANKSHOPID":"1","ANKK_GU":"T"},{"ANKSHOPID":"1","ANKK_GU":"U"}]`,
			want: "{\"ANKSHOPID\":\"1\",\"ANKK_GU\":\"T\"}\n{\"ANKSHOPID\":\"1\",\"ANKK_GU\":\"U\"}\n",
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
