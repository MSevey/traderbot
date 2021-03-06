package api

// this file contains the code for interacting with the CoinMarketCap API

import (
	"encoding/json"
	"strconv"
)

var (
	// BTCID is bitcoin's id for the coinmarket cap API endpoint
	BTCID = 1

	// BNBID is binance coin's id for the coinmarket cap API endpoint
	BNBID = 1839

	// CoinMarketCapTickerAPI is the API endpoint for coinMarketCap to get the
	// detailed ticker information for a specific cryptocurrency
	CoinMarketCapTickerAPI = "https://api.coinmarketcap.com/v2/ticker/"
)

// CoinCap is the data struct returned from the coinMarketCap API
type CoinCap struct {
	Data     coinData     `json:"data"`
	Metadata coinMetadata `json:"metadata"`
}

// coinData is the information about the crypto currency returned from the
// coinMarketCap API
type coinData struct {
	ID                int64      `json:"id"`
	Name              string     `json:"name"`
	Symbol            string     `json:"symbol"`
	WebsiteSlug       string     `json:"website_slug"`
	Rank              int64      `json:"rank"`
	CirculatingSupply float64    `json:"circulating_supply"`
	TotalSupply       float64    `json:"total_supply"`
	MaxSupply         float64    `json:"max_supply"`
	LastUpdates       uint64     `json:"last_updated"`
	Quotes            coinQuotes `json:"quotes"`
}

// coinMetadata is the metadata from the coinMarketCap API call
type coinMetadata struct {
	Timestamp uint64 `json:"timestamp"`
	Error     error  `json:"error"`
}

// coinQuotes are the fiat currency quotes for the crypto currency requested
// from the coinMarketCap endpoint
type coinQuotes struct {
	USD quoteData `json:"USD"`
}

// quoteData is the specific quote data for each currency
type quoteData struct {
	Price            float64 `json:"price"`
	Volume24h        float64 `json:"volume_24h"`
	MarketCap        float64 `json:"market_cap"`
	PercentChange1h  float64 `json:"percent_change_1h"`
	PercentChange24h float64 `json:"percent_change_24h"`
	PercentChange7d  float64 `json:"percent_change_7d"`
}

// NewCoinCapClient creates a new api client for the Binance API
func NewCoinCapClient() *Client {
	return NewClient(CoinMarketCapTickerAPI, 2)
}

// GetTickerInfo calls the api endpoints that returns the ticker information for
// a coin
func (c *Client) GetTickerInfo(ticker int) (CoinCap, error) {
	body, err := c.GetAPI(c.Address + strconv.Itoa(ticker) + "/")
	if err != nil {
		apiLog.Warn("WARN: error submitting get request:", err)
		return CoinCap{}, err
	}

	coin := CoinCap{}
	err = json.Unmarshal(body, &coin)
	if err != nil {
		apiLog.Warn("WARN: error unmarshaling coin cap info:", err)
		return CoinCap{}, err
	}

	return coin, nil
}
