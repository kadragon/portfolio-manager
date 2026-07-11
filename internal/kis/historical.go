package kis

import (
	"time"

	"github.com/kadragon/portfolio-manager/internal/datex"
	"github.com/kadragon/portfolio-manager/internal/ktime"
)

// HistoricalPoint is one day's close from a period/range price fetch.
type HistoricalPoint struct {
	Date  datex.Date
	Price float64
}

// parseKISDate parses KIS's "YYYYMMDD" date form (stck_bsop_date / xymd) into a datex.Date.
func parseKISDate(s string) (datex.Date, error) {
	t, err := time.ParseInLocation("20060102", s, ktime.KST)
	if err != nil {
		return datex.Date{}, err
	}
	return datex.Date{Time: t}, nil
}
