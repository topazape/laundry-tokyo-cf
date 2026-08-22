package laundrich

import (
	"bytes"
	"encoding/json/v2"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"unicode"
)

type shopRecord struct {
	MIXHOLIDAY      string `json:"MIXHOLIDAY"`      // 定休日
	MIXBUSINESSHOUR string `json:"MIXBUSINESSHOUR"` // 営業時間（自由記述）
	NUMFACILITYLIST string `json:"NUMFACILITYLIST"` // 保有設備（カンマ区切り）; null あり
	ANKSHOPID       string `json:"ANKSHOPID"`       // ショップID
	ANKLONGITUDE    string `json:"ANKLONGITUDE"`    // 経度; 要末尾 trim
	ANKPARALLEL     string `json:"ANKPARALLEL"`     // 緯度; 要末尾 trim
	KNJSHOPNAME     string `json:"KNJSHOPNAME"`     // 店名
	KANASHOPNAME    string `json:"KANASHOPNAME"`    // 店名カナ
	ANKPOSTNUM      string `json:"ANKPOSTNUM"`      // 郵便番号
	KNJADDRESS1     string `json:"KNJADDRESS1"`     // 住所（都道府県）
	KNJADDRESS2     string `json:"KNJADDRESS2"`     // 住所（市区町村）
	KNJADDRESS3     string `json:"KNJADDRESS3"`     // 住所
	ANKTELNUM       string `json:"ANKTELNUM"`       // 使わない
	ANKWEBSHOPFLG   string `json:"ANKWEBSHOPFLG"`   // 使わない
	ENABLESHOWIMAGE string `json:"ENABLESHOWIMAGE"` // 使わない
	PAGEID          string `json:"PAGEID"`          // 使わない
}

func parseShop(r shopRecord) Shop {
	lat, err := strconv.ParseFloat(strings.TrimSpace(r.ANKPARALLEL), 64)
	if err != nil {
		lat = 0
	}

	lng, err := strconv.ParseFloat(strings.TrimSpace(r.ANKLONGITUDE), 64)
	if err != nil {
		lng = 0
	}

	facilities := []string{}
	isIoTEnabled := false

	if r.NUMFACILITYLIST != "" {
		facilities = strings.Split(r.NUMFACILITYLIST, ",")
		isIoTEnabled = slices.Contains(facilities, "1")
	}

	return Shop{
		ShopID:        r.ANKSHOPID,
		Name:          r.KNJSHOPNAME,
		NameKana:      r.KANASHOPNAME,
		Pref:          r.KNJADDRESS1,
		City:          r.KNJADDRESS2,
		Address:       r.KNJADDRESS3,
		Postal:        r.ANKPOSTNUM,
		Lat:           lat,
		Lng:           lng,
		IsIoTEnabled:  isIoTEnabled,
		Facilities:    facilities,
		BusinessHours: r.MIXBUSINESSHOUR,
		ClosedDays:    r.MIXHOLIDAY,
	}
}

type Shop struct {
	ShopID        string
	Name          string
	NameKana      string
	Pref          string
	City          string
	Address       string
	Postal        string
	Lat           float64
	Lng           float64
	IsIoTEnabled  bool
	Facilities    []string
	BusinessHours string
	ClosedDays    string
}

func ParseShops(data []byte) ([]Shop, error) {
	trimmed := bytes.TrimLeftFunc(data, unicode.IsSpace)
	if len(trimmed) == 0 || trimmed[0] != '[' {
		return nil, fmt.Errorf("unexpected response body: not a JSON array")
	}

	var records []shopRecord
	if err := json.Unmarshal(data, &records); err != nil {
		return nil, fmt.Errorf("unmarshal shops: %w", err)
	}

	shops := make([]Shop, 0, len(records))
	for _, r := range records {
		shops = append(shops, parseShop(r))
	}

	return shops, nil
}
