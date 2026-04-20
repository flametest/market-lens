package data

import (
	"database/sql"

	"github.com/flametest/market-lens/pkg/model"
	"github.com/shopspring/decimal"
)

func dec(s string) decimal.Decimal {
	d, _ := decimal.NewFromString(s)
	return d
}

func scanCandles(rows *sql.Rows) ([]model.Candle, error) {
	var candles []model.Candle
	for rows.Next() {
		var c model.Candle
		var open, high, low, close, vol string
		if err := rows.Scan(&c.Symbol, &c.Interval, &c.Timestamp, &open, &high, &low, &close, &vol, &c.Source); err != nil {
			return nil, err
		}
		c.Open = dec(open)
		c.High = dec(high)
		c.Low = dec(low)
		c.Close = dec(close)
		c.Volume = dec(vol)
		candles = append(candles, c)
	}
	return candles, rows.Err()
}
