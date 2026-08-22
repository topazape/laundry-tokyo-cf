package monitor

import (
	"bytes"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"laundry-tokyo/internal/laundrich"
	"time"
)

// stream(status) schema
type streamStatus struct {
	ShopID           string     `json:"shop_id"`
	MachineID        string     `json:"machine_id"`
	MachineKind      string     `json:"machine_kind"`
	Status           string     `json:"status"`
	StatusCode       string     `json:"status_code"`
	RemainingMinutes *int       `json:"remaining_minutes"`
	CourseCode       string     `json:"course_code"`
	ReportedAt       *time.Time `json:"reported_at"`
	FetchedAt        time.Time  `json:"fetched_at"`
}

func toStreamStatuses(statuses []laundrich.Status, fetchedAt time.Time) []any {
	rows := make([]any, 0, len(statuses))
	for _, s := range statuses {
		rows = append(rows, streamStatus{
			ShopID:           s.ShopID,
			MachineID:        s.MachineID,
			MachineKind:      s.MachineKind,
			Status:           s.Status,
			StatusCode:       s.StatusCode,
			RemainingMinutes: s.RemainingMinutes,
			CourseCode:       s.CourseCode,
			ReportedAt:       s.ReportedAt,
			FetchedAt:        fetchedAt,
		})
	}

	return rows
}

func rawToNDJSON(raw []byte) ([]byte, error) {
	var elems []jsontext.Value
	if err := json.Unmarshal(raw, &elems); err != nil {
		return nil, fmt.Errorf("unmarshal raw statuses: %w", err)
	}

	var buf bytes.Buffer
	for _, e := range elems {
		buf.Write(e)
		buf.WriteByte('\n')
	}

	return buf.Bytes(), nil
}
