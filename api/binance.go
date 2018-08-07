package api

var (
	// BNBBTC is the binance symbol for the BTC/BNB market to get ticker price
	BNBBTC = "BNBBTC"

	// BinanceAPI is the base API endpoint for the binance exchange
	BinanceAPI = "https://api.binance.com/api/"

	// BNBExchangeInfo is the exchange info endpoint for the binance API
	BNBExchangeInfo = "v1/exchangeInfo"

	// BNBPrice is the endpoint for the binance API to get the price of a coin,
	// it takes an input parameter of `symbol`
	BNBPrice = "v3/ticker/price"
)

// BNBLimits are the limits for the Binance exchange API
type BNBLimits struct {
	RateLimits []Limits `json:"rateLimits"`
}

// Limits are the different types of rate limits for the binance exchange
type Limits struct {
	RateLimitType string `json:"rateLimitType"`
	Interval      string `json:"interval"`
	Limit         int    `json:"limit"`
}

// TickerPrice ...
type TickerPrice struct {
	Symbol string `json:"symbol"`
	Price  string `json:"price"`
}
