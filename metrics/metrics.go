package metrics

// The metrics package is where all the metric calculations will take place.
// These are the metrics that will be emailed out so that the health of the
// trader bot can be monitored and improved.

import (
	"fmt"
	"reflect"
	"strconv"
	"time"

	"github.com/MSevey/traderbot/api"
)

// Metrics to Get and Track
//
// should focus on amount of coin as well as $$ value since amount of coin is a
// better indicator of how well the trader bot is doing since the $$ value of
// crypto is so volatile
//
// 1) Current Value
//
// 2) Last week's value
//      - % change in balance
//
// 3) Last 24hr value
//      - % change in balance
//
// 4) Number of buys
//      - Highest buy price
//      - Lowest buy price
//
//  5) Number of sells
//      - Highest sell price
//      - Lowest sell price
//
// TODO
//
// 1) persist data, start with writing json to disk

type (
	// Portfolio is the portfolio information pulled from the exchange
	// (currently binance)
	Portfolio struct {
		Assets  []Asset
		Updated time.Time
		Value   float64
	}

	// Asset is the information about a coin held on an exchange (currently
	// binance)
	Asset struct {
		Symbol   string
		Quantity float64
		Value    float64
	}
)

// PortfolioBalance returns the binance portfolio, listing the coins held and
// the quantities and values of each
//
// TODO - this should be moved to Binance file, no need for this to be in
// metrics package
func PortfolioBalance() (Portfolio, error) {
	// Get account info from binance
	client := api.NewBinanceClient()
	accountInfo, err := client.GetAccountInfo()
	if err != nil {
		return Portfolio{}, err
	}
	var p Portfolio
	p.Updated = time.Now()
	for _, asset := range accountInfo.Balances {
		// Check for no zero assets
		free, err := strconv.ParseFloat(asset.Free, 64)
		if err != nil {
			fmt.Println(err)
			continue
		}
		locked, err := strconv.ParseFloat(asset.Locked, 64)
		if err != nil {
			fmt.Println(err)
			continue
		}
		if free+locked == float64(0) {
			continue
		}

		// Get current price
		tp, err := client.GetCoinPrice(asset.Asset + "USDT")
		if err != nil {
			return Portfolio{}, err
		}
		if reflect.DeepEqual(tp, api.TickerPrice{}) {
			// TODO - need to update this to log message and then try to get
			// asset value in BTC
			fmt.Println("No ticker price information")
			continue
		}
		price, err := strconv.ParseFloat(tp.Price, 32)
		if err != nil {
			return Portfolio{}, err
		}

		// Update Portfolio
		qty := free + locked
		value := qty * price
		p.Assets = append(p.Assets, Asset{
			Symbol:   asset.Asset,
			Quantity: qty,
			Value:    value,
		})
		p.Value += value
	}
	return p, err
}
