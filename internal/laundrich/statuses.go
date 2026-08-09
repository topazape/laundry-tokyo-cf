package laundrich

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"time"
	"unicode"
)

type statusRecord struct {
	ANKAILSFLG    string `json:"ANKAILSFLG"`    // 不明; 使わない
	ANKCOMFLG     string `json:"ANKCOMFLG"`     // 通信状態; 1: オンライン?; 使わない
	ANKDRUMFLG    string `json:"ANKDRUMFLG"`    // ドラム式フラグ; 1: 多槽(2段式), 0: 単槽; 使わない
	ANKKHKIND     string `json:"ANKKHKIND"`     // 種別; 1: 洗濯機 2: 乾燥機 3: 洗濯乾燥機 4: スニーカー系
	ANKK_GK       string `json:"ANKK_GK"`       // course code(16進); 0F: 洗濯のみ 01: 乾燥 18: 洗濯乾燥; 使わない
	ANKK_GT       string `json:"ANKK_GT"`       // 残り運転時間(分); 3 桁ゼロ埋め, 非稼働時 000
	ANKK_GU       string `json:"ANKK_GU"`       // 稼働区分; U: 使用中 T: 非稼働 E: 利用不可
	ANKMACHINENUM string `json:"ANKMACHINENUM"` // 店内機器番号; 単独では非一意のため、ANKPOL と組み合わせる
	ANKPOL        string `json:"ANKPOL"`        // 槽識別; A1: 上段(単槽機は常に A1), A2: 下段
	ANKSHOPID     string `json:"ANKSHOPID"`     // 店舗 ID
	ANKSTS        string `json:"ANKSTS"`        // 不明; 使わない
	DTMCREATE     string `json:"DTMCREATE"`     // server 側生成時刻; offset なし(JST)・小数 5〜7 桁可変
	KNJMACHNAME   string `json:"KNJMACHNAME"`   // 機器名称; 改行が \r\n or \n で混在; 使わない
	MIXPARTNUM    string `json:"MIXPARTNUM"`    // メーカー型番; 使わない
	NUMCOINLOCK   int    `json:"NUMCOINLOCK"`   // コインロック状態?; 使わない
	NUMDOORLOCK   int    `json:"NUMDOORLOCK"`   // ドアロック状態?; 使わない
}

const (
	statusAvailable    = "available"
	statusInUse        = "in_use"
	statusOutOfService = "out_of_service"
	statusUnknown      = "unknown"
)

func (r statusRecord) status() string {
	switch r.ANKK_GU {
	case "U":
		return statusInUse
	case "T":
		return statusAvailable
	case "E":
		return statusOutOfService
	default:
		return statusUnknown
	}
}

func parseReportedAt(s string) *time.Time {
	jst := time.FixedZone("JST", 9*60*60)

	// 小数秒は layout に書かなくても Parse が秒の直後から自動で読む
	t, err := time.ParseInLocation("2006-01-02T15:04:05", s, jst)
	if err != nil {
		return nil // 欠損許容
	}

	u := t.UTC()

	return &u
}

func parseStatus(r statusRecord) Status {
	var rmin *int
	if v, err := strconv.Atoi(r.ANKK_GT); err == nil {
		rmin = &v
	}

	return Status{
		ShopID:           r.ANKSHOPID,
		MachineID:        r.ANKMACHINENUM + "-" + r.ANKPOL,
		MachineKind:      r.ANKKHKIND,
		Status:           r.status(),
		StatusCode:       r.ANKK_GU,
		RemainingMinutes: rmin,
		CourseCode:       r.ANKK_GK,
		ReportedAt:       parseReportedAt(r.DTMCREATE),
	}
}

type Status struct {
	ShopID           string
	MachineID        string
	MachineKind      string
	Status           string
	StatusCode       string
	RemainingMinutes *int
	CourseCode       string
	ReportedAt       *time.Time
}

func ParseStatuses(data []byte) ([]Status, error) {
	trimmed := bytes.TrimLeftFunc(data, unicode.IsSpace)
	if len(trimmed) == 0 || trimmed[0] != '[' {
		return nil, fmt.Errorf("unexpected response body: not a JSON array")
	}

	var records []statusRecord
	if err := json.Unmarshal(data, &records); err != nil {
		return nil, fmt.Errorf("unmarshal statuses: %w", err)
	}

	statuses := make([]Status, 0, len(records))
	for _, r := range records {
		statuses = append(statuses, parseStatus(r))
	}

	return statuses, nil
}
