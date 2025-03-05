package metrics

// The metrics package is where all the metric calculations will take place.
// These are the metrics that will be emailed out so that the health of the
// trader bot can be monitored and improved.

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"time"

	"github.com/MSevey/traderbot/api"
	"github.com/MSevey/traderbot/persistence"
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

// Database instance
var db *persistence.DB

// SetDB sets the database instance for the metrics package
func SetDB(database *persistence.DB) error {
	if database == nil {
		return fmt.Errorf("database is nil")
	}
	db = database
	return nil
}

// InitDB initializes the database connection
func InitDB() error {
	var err error
	db, err = persistence.OpenDatabase(persistence.DefaultDatabasePath())
	if err != nil {
		return err
	}
	return nil
}

// CloseDB closes the database connection
func CloseDB() error {
	if db != nil {
		return db.Close()
	}
	return nil
}

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
	// Initialize DB if not already initialized
	if db == nil {
		if err := InitDB(); err != nil {
			return Portfolio{}, err
		}
		defer CloseDB()
	}

	// Try to load initial balance from database
	dbPortfolio, err := db.LoadInitialBalance()
	if err == persistence.ErrKeyNotFound {
		// No initial balance found, create an initial balance
		portfolio, err := PortfolioBalance()
		if err != nil {
			return Portfolio{}, err
		}
		
		// Save to database
		if err = db.SaveInitialBalance(persistence.Portfolio{
			Assets:  convertToDBAssets(portfolio.Assets),
			Updated: portfolio.Updated,
			Value:   portfolio.Value,
		}); err != nil {
			return Portfolio{}, err
		}
		
		return portfolio, nil
	} else if err != nil {
		return Portfolio{}, err
	}
	
	// Convert DB portfolio to metrics portfolio
	return Portfolio{
		Assets:  convertFromDBAssets(dbPortfolio.Assets),
		Updated: dbPortfolio.Updated,
		Value:   dbPortfolio.Value,
	}, nil
}

// convertToDBAssets converts metrics.Asset slice to persistence.Asset slice
func convertToDBAssets(assets []Asset) []persistence.Asset {
	dbAssets := make([]persistence.Asset, len(assets))
	for i, asset := range assets {
		dbAssets[i] = persistence.Asset{
			Symbol:   asset.Symbol,
			Quantity: asset.Quantity,
			Value:    asset.Value,
		}
	}
	return dbAssets
}

// convertFromDBAssets converts persistence.Asset slice to metrics.Asset slice
func convertFromDBAssets(dbAssets []persistence.Asset) []Asset {
	assets := make([]Asset, len(dbAssets))
	for i, dbAsset := range dbAssets {
		assets[i] = Asset{
			Symbol:   dbAsset.Symbol,
			Quantity: dbAsset.Quantity,
			Value:    dbAsset.Value,
		}
	}
	return assets
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