package metrics

// The metrics package is where all the metric calculations will take place.
// These are the metrics that will be emailed out so that the health of the
// trader bot can be monitored and improved.

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"time"

	"github.com/MSevey/traderbot/api"
	"gitlab.com/NebulousLabs/Sia/persist"
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

type (
	// Portfolio is the portfolio information pulled from the exchange
	// (currently binance)
	Portfolio struct {
		Assets  []Asset   `json:"assests"`
		Updated time.Time `json:"updated"`
		Value   float64   `json:"value"`
	}

	// Asset is the information about a coin held on an exchange (currently
	// binance)
	Asset struct {
		Symbol   string  `json:"symbol"`
		Quantity float64 `json:"quantity"`
		Value    float64 `json:"value"`
	}

	// PortfolioPerformance contains information about the performance of the
	// portfolio
	PortfolioPerformance struct {
		Assets []AssetPerformance
	}

	// AssetPerformance contains information about the performance of an asset
	AssetPerformance struct {
		Symbol               string
		QtyIncreaseAbs       float64
		QtyIncreasePercent   float64
		ValueIncreaseAbs     float64
		ValueIncreasePercent float64
	}
)

var (
	// metricsDir is the default directory where the trader metric data will be
	// persisted
	//
	// NOTE: this assumes a linux system
	metricsDir = filepath.Join(os.Getenv("HOME"), "tradermetrics")

	// initialBalanceMetadata is the metadata for the persisted file that stores
	// the initial starting balance for the trader
	initialBalanceMetadata = persist.Metadata{
		Header:  "InitialBalance",
		Version: "v1.0.0",
	}

	// initialBalanceFile is the filename for the persisted file containing the
	// initial balance of the trader
	initialBalanceFile = "initialbalance"

	// balanceExtension is the common file extension for the persisted balance
	// files
	balanceExtension = ".bal"
)

// LifeTimePortfolioPerformance calculates the lifetime performance of the
// portfolio
func LifeTimePortfolioPerformance() (PortfolioPerformance, error) {
	// Get initial Balance
	initial, err := initialBalance()
	if err != nil {
		return PortfolioPerformance{}, err
	}
	// Create map of assets for comparision
	initialAssetMap := make(map[string]Asset)
	for _, asset := range initial.Assets {
		if _, ok := initialAssetMap[asset.Symbol]; ok {
			continue
		}
		initialAssetMap[asset.Symbol] = asset
	}

	// Get Current balance
	current, err := PortfolioBalance()
	if err != nil {
		return PortfolioPerformance{}, err
	}

	// Calculate performance
	//
	// NOTE: Ignore any assets in initial balance that aren't in current balance
	var performance PortfolioPerformance
	for _, asset := range current.Assets {
		var ap AssetPerformance
		// Check if asset was part of initial balance
		initialAsset, ok := initialAssetMap[asset.Symbol]
		if ok {
			// Calculate performance
			ap.Symbol = asset.Symbol
			ap.QtyIncreaseAbs = asset.Quantity - initialAsset.Quantity
			ap.QtyIncreasePercent = ((asset.Quantity - initialAsset.Quantity) / initialAsset.Quantity) * 100
			ap.ValueIncreaseAbs = asset.Value - initialAsset.Value
			ap.ValueIncreasePercent = ((asset.Value - initialAsset.Value) / initialAsset.Value) * 100
			// Add to portfolio performance
			performance.Assets = append(performance.Assets, ap)
			continue
		}
		// Calculate performance
		ap.Symbol = asset.Symbol
		ap.QtyIncreaseAbs = asset.Quantity
		ap.QtyIncreasePercent = 100
		ap.ValueIncreaseAbs = asset.Value
		ap.ValueIncreasePercent = 100
		// Add to portfolio performance
		performance.Assets = append(performance.Assets, ap)
	}

	return performance, nil
}

// initialBalance returns the initial balance of the trader that was saved on
// disk. If there is not an initial balance found on disk one will be generated
func initialBalance() (Portfolio, error) {
	filename := filepath.Join(metricsDir, initialBalanceFile+balanceExtension)
	portfolio := Portfolio{}
	err := persist.LoadJSON(initialBalanceMetadata, portfolio, filename)
	if os.IsNotExist(err) {
		// No initial balance found, create an initial balance
		if err = os.MkdirAll(metricsDir, 0777); err != nil {
			return Portfolio{}, err
		}
		portfolio, err = PortfolioBalance()
		err = persist.SaveJSON(initialBalanceMetadata, portfolio, filename)
	}
	if err != nil {
		return Portfolio{}, err
	}
	return portfolio, nil
}

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
