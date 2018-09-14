package api

// All api calls have a weight of 1 unless otherwise specified
//
// Binance fees 0.75% per trade if paid with BNB

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"time"
)

// TODO
//
// 1) add weight as a return value from calls to then increment trader details
//
// 2) Look at different order types to help optimize trading algo

const (
	// recvWindow is the allowable difference between the submitted timestamp
	// and the servertime that an API call will be accepted
	recvWindow = 5000
)

var (
	// BNBAPIPubKey is the public api key
	BNBAPIPubKey = os.Getenv("bnbAPIPubKey")

	// BNBAPISecretKey is the secret api key
	BNBAPISecretKey = os.Getenv("bnbAPISecretKey")

	// BNBBTC is the binance symbol for the BNB/BTC market to get ticker price
	// Price of 1 BNB in BTC
	BNBBTC = "BNBBTC"

	// BNBBTC is the binance symbol for the BNB/BTC market to get ticker price
	// Price of 1 BNB in BTC
	BNBUSDT = "BNBUSDT"

	// BTCUSDT is the binance symbol for the BTC/USDT market to get ticker price
	// Price of 1 BTC in USD
	BTCUSDT = "BTCUSDT"

	// BinanceAPI is the base API endpoint for the binance exchange
	BinanceAPI = "https://api.binance.com/api/"

	// BNBExchangeInfo is the exchange info endpoint for the binance API
	BNBExchangeInfo = "v1/exchangeInfo"

	// BNBPrice is the endpoint for the binance API to get the price of a coin,
	// it takes an input parameter of `symbol`
	BNBPrice = "v3/ticker/price"

	// BNB24hrStats 24 hour price change statistics. Careful when accessing this
	// with no symbol.
	BNB24hrStats = "v1/ticker/24hr"

	// BNBAllOrders Get all account orders; active, canceled, or filled. Use
	// Orders struct
	BNBAllOrders = "v3/allOrders"

	// BNBAccount Get current account information.
	BNBAccount = "v3/account"

	// BNBOpenOrders Get all open orders on a symbol. Careful when accessing
	// this with no symbol. Use Orders struct
	BNBOpenOrders = "v3/openOrders"

	// BNBNewOrder Send in a new order.
	BNBNewOrder = "v3/order"

	// BNBTestOrder sends a test order, does not post to market
	BNBTestOrder = BNBNewOrder + "/test"

	// BNBPing test connectivity
	BNBPing = "v1/ping"

	// BNBTime server time
	BNBTime = "v1/time"
)

// ExchangeInfo is the stuct for the Binance exchange api endpoint
type ExchangeInfo struct {
	Timezone string `json:"timezone"`
	ServerTime
	BNBLimits
}

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

// Stats24hr ...
type Stats24hr struct {
	Symbol             string `json:"symbol"`
	PriceChange        string `json:"priceChange"`
	PriceChangePercent string `json:"priceChangePercent"`
	WeightedAvgPrice   string `json:"weightedAvgPrice"`
	PrevClosePrice     string `json:"prevClosePrice"`
	LastPrice          string `json:"lastPrice"`
}

// Orders ..
type Orders struct {
	Orders []Order
}

// Order ..
type Order struct {
	Symbol              string `json:"symbol"`
	OrderID             int    `json:"orderId"`
	ClientOrderID       string `json:"clientOrderId"`
	Price               string `json:"price"`
	OrigQty             string `json:"origQty"`
	ExecutedQty         string `json:"executedQty"`
	CummulativeQuoteQty string `json:"cummulativeQuoteQty"`
	Status              string `json:"status"`
	TimeInForce         string `json:"timeInForce"`
	Type                string `json:"type"`
	Side                string `json:"side"`
	StopPrice           string `json:"stopPrice"`
	IcebergQty          string `json:"icebergQty"`
	Time                int64  `json:"time"`
	UpdateTime          int64  `json:"updateTime"`
	IsWorking           bool   `json:"isWorking"`
}

// AccountInfo ...
type AccountInfo struct {
	MakerCommission  int     `json:"makerCommission"`
	TakerCommission  int     `json:"takerCommission"`
	BuyerCommission  int     `json:"buyerCommission"`
	SellerCommission int     `json:"sellerCommission"`
	CanTrade         bool    `json:"canTrade"`
	CanWithdraw      bool    `json:"canWithdraw"`
	CanDeposit       bool    `json:"canDeposit"`
	UpdateTime       int64   `json:"updateTime"`
	Balances         []Asset `json:"balances"`
}

// Asset ...
type Asset struct {
	Asset  string `json:"asset"`
	Free   string `json:"free"`
	Locked string `json:"locked"`
}

// ServerTime ...
type ServerTime struct {
	ServerTime int64 `json:"serverTime"`
}

// OrderRespHeader ...
type OrderRespHeader struct {
	Symbol          string `json:"symbol"`
	OrderID         int    `json:"orderId"`
	ClientOrderID   string `json:"clientOrderId"`
	TransactionTime int64  `json:"transactTime"`
}

// ACK ...
type ACK struct {
	Response OrderRespHeader
}

// RESULT ...
type RESULT struct {
	Response            OrderRespHeader
	Price               string `json:"price"`
	OrigQty             string `json:"origQty"`
	ExecutedQty         string `json:"executedQty"`
	CummulativeQuoteQty string `json:"cummulativeQuoteQty"`
	Status              string `json:"status"`
	TimeInForce         string `json:"timeInForce"`
	Type                string `json:"type"`
	Side                string `json:"side"`
}

// FULL ...
type FULL struct {
	Response RESULT
	Fills    []Fill `json:"fills"`
}

// Fill ...
type Fill struct {
	Price           string `json:"price"`
	Qty             string `json:"qty"`
	Commission      string `json:"commission"`
	CommissionAsset string `json:"commissionAsset"`
}

// NewBinanceClient creates a new api client for the Binance API
func NewBinanceClient() *Client {
	return NewClient(BinanceAPI, 2)
}

func signature(params string) string {
	// Create a new HMAC by defining the hash type and the key (as byte array)
	h := hmac.New(sha256.New, []byte(BNBAPISecretKey))

	// Write Data to it
	h.Write([]byte(params))

	// Get result and encode as hexadecimal string
	return hex.EncodeToString(h.Sum(nil))
}

// Get24hrStats calls the API endpoint that returns the 24hr statistics on a
// coin
func (c *Client) Get24hrStats(symbol string) Stats24hr {
	body, err := c.GetAPI(c.Address + BNB24hrStats + "?symbol=" + symbol)
	check(err)

	stats := Stats24hr{}
	jsonErr := json.Unmarshal(body, &stats)
	check(jsonErr)

	return stats
}

// GetAccountInfo calls the API endpoint that returns account info
//
// Weight = 5
func (c *Client) GetAccountInfo() AccountInfo {
	timestamp := strconv.FormatInt(time.Now().UnixNano()/int64(time.Millisecond), 10)
	params := fmt.Sprintf("timestamp=%v&recvWindow=%v", timestamp, recvWindow)
	sig := signature(params)
	query := fmt.Sprintf("?%v&signature=%v", params, sig)
	body, err := c.GetSecureAPI(c.Address + BNBAccount + query)
	check(err)

	// Get Account Info
	account := AccountInfo{}
	jsonErr := json.Unmarshal(body, &account)
	check(jsonErr)
	return account
}

// GetAllOrders calls the endpoint that returns all order history fpr a given symbol
//
// Weight 5
func (c *Client) GetAllOrders(symbol string) []Order {
	timestamp := strconv.FormatInt(time.Now().UnixNano()/int64(time.Millisecond), 10)
	params := fmt.Sprintf("symbol=%v&timestamp=%v&recvWindow=%v", symbol, timestamp, recvWindow)
	sig := signature(params)
	query := fmt.Sprintf("?%v&signature=%v", params, sig)
	body, err := c.GetSecureAPI(c.Address + BNBAllOrders + query)
	check(err)

	// Get Open Orders
	orders := []Order{}
	jsonErr := json.Unmarshal(body, &orders)
	check(jsonErr)

	return orders
}

// GetBinanceExchangeInfo calls the API endpoint that returns info on the binance
// exchange
func (c *Client) GetBinanceExchangeInfo() ExchangeInfo {
	body, err := c.GetAPI(c.Address + BNBExchangeInfo)
	check(err)

	// Get ExchangeInfo
	info := ExchangeInfo{}
	jsonErr := json.Unmarshal(body, &info)
	check(jsonErr)
	return info
}

// GetCoinPrice calls the API endpoint that returns the current price for a coin
func (c *Client) GetCoinPrice(symbol string) TickerPrice {
	body, err := c.GetAPI(c.Address + BNBPrice + "?symbol=" + symbol)
	check(err)

	price := TickerPrice{}
	jsonErr := json.Unmarshal(body, &price)
	check(jsonErr)

	return price
}

// GetOpenOrders calls the endpoint that returns all open orders
//
// Weight 1 with symbol, 40 w/o symbol
func (c *Client) GetOpenOrders() []Order {
	timestamp := strconv.FormatInt(time.Now().UnixNano()/int64(time.Millisecond), 10)
	params := fmt.Sprintf("timestamp=%v&recvWindow=%v", timestamp, recvWindow)
	sig := signature(params)
	query := fmt.Sprintf("?%v&signature=%v", params, sig)
	body, err := c.GetSecureAPI(c.Address + BNBOpenOrders + query)
	check(err)

	// Get Open Orders
	orders := []Order{}
	jsonErr := json.Unmarshal(body, &orders)
	check(jsonErr)

	return orders
}

// Order Parameters
// Name				Type		Mandatory	Description
// symbol			STRING		YES
// side				ENUM		YES
// type				ENUM		YES
// timeInForce		ENUM		NO
// price			DECIMAL		NO
// quantity			DECIMAL		YES
// newClientOrderId	STRING		NO			A unique id for the order. Automatically generated if not sent.
// stopPrice		DECIMAL		NO			Used with STOP_LOSS, STOP_LOSS_LIMIT, TAKE_PROFIT, and TAKE_PROFIT_LIMIT orders.
// icebergQty		DECIMAL		NO			Used with LIMIT, STOP_LOSS_LIMIT, and TAKE_PROFIT_LIMIT to create an iceberg order.
// newOrderRespType	ENUM		NO			Set the response JSON. ACK, RESULT, or FULL; MARKET and LIMIT order types default to FULL, all other orders default to ACK.
// recvWindow		LONG		NO
// timestamp		LONG		YES

// Type					Additional mandatory parameters
// LIMIT				timeInForce, quantity, price
// MARKET				quantity
// STOP_LOSS			quantity, stopPrice
// STOP_LOSS_LIMIT		timeInForce, quantity, price, stopPrice
// TAKE_PROFIT			quantity, stopPrice
// TAKE_PROFIT_LIMIT	timeInForce, quantity, price, stopPrice
// LIMIT_MAKER			quantity, price

// PostNewLimitOrder calls the API endpoint to submit a limit order to Binance
func (c *Client) PostNewLimitOrder(symbol, side string, quantity float64) RESULT {

	timestamp := strconv.FormatInt(time.Now().UnixNano()/int64(time.Millisecond), 10)
	values := url.Values{}
	values.Set("symbol", symbol)                                       // Mandatory
	values.Set("side", side)                                           // Mandatory
	values.Set("type", "LIMIT")                                        // Mandatory
	values.Set("timeInForce", "GTC")                                   // Mandatory
	values.Set("quantity", strconv.FormatFloat(quantity, 'f', -1, 64)) // Mandatory
	values.Set("newOrderRespType", "RESULT")                           //
	values.Set("recvWindow", strconv.FormatInt(recvWindow, 10))        //
	values.Set("timestamp", timestamp)                                 // Mandatory
	params := values.Encode()
	sig := signature(params)
	query := fmt.Sprintf("?%v&signature=%v", params, sig)

	body, err := c.PostSecureAPI(c.Address + BNBTestOrder + query)
	check(err)

	result := RESULT{}
	jsonErr := json.Unmarshal(body, &result)
	check(jsonErr)

	return result
}
