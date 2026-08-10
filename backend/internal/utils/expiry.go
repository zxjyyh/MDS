package utils

import (
	"fmt"
	"time"
)

type ExpiryInfo struct {
	Status   string `json:"status"`
	DaysLeft int    `json:"daysLeft"`
	Label    string `json:"label"`
	Color    string `json:"color"`
	Type     string `json:"type"`
}

// ExpiryStatus returns expiry status info for a given deadline date
func ExpiryStatus(date *time.Time) ExpiryInfo {
	if date == nil {
		return ExpiryInfo{
			Status:   "normal",
			DaysLeft: 0,
			Label:    "长期有效",
			Color:    "#67c23a",
			Type:     "success",
		}
	}
	now := time.Now().Truncate(24 * time.Hour)
	target := date.Truncate(24 * time.Hour)
	daysLeft := int(target.Sub(now).Hours() / 24)

	switch {
	case daysLeft < 0:
		return ExpiryInfo{Status: "expired", DaysLeft: daysLeft, Label: "已过期", Color: "#f56c6c", Type: "danger"}
	case daysLeft <= 30:
		return ExpiryInfo{Status: "expiring_30", DaysLeft: daysLeft, Label: fmt.Sprintf("%d天后到期", daysLeft), Color: "#e6a23c", Type: "warning"}
	case daysLeft <= 60:
		return ExpiryInfo{Status: "expiring_60", DaysLeft: daysLeft, Label: fmt.Sprintf("%d天后到期", daysLeft), Color: "#e6a23c", Type: "warning"}
	case daysLeft <= 90:
		return ExpiryInfo{Status: "expiring_90", DaysLeft: daysLeft, Label: fmt.Sprintf("%d天后到期", daysLeft), Color: "#e6a23c", Type: "warning"}
	default:
		return ExpiryInfo{Status: "normal", DaysLeft: daysLeft, Label: fmt.Sprintf("%d天后到期", daysLeft), Color: "#67c23a", Type: "success"}
	}
}

// DaysUntil returns number of days until the given date. Nil date = 9999 (infinity).
func DaysUntil(date *time.Time) int {
	if date == nil {
		return 9999
	}
	return int(date.Truncate(24*time.Hour).Sub(time.Now().Truncate(24*time.Hour)).Hours() / 24)
}
