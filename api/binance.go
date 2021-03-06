package api

// this file contains the code for interacting with the Binance exchange API.
//
// NOTE: All api calls have a weight of 1 unless otherwise specified

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

	// BNBUSDT is the binance symbol for the BNB/USDT market to get ticker price
	// Price of 1 BNB in USDT
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

// ExchangeInfo is the information returned about the exchange from the Binance
// exchange api endpoint
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

// TickerPrice is the symbol and price of a coin on the Binance exchange
type TickerPrice struct {
	Symbol string `json:"symbol"`
	Price  string `json:"price"`
}

// Stats24hr are the stats from the last 24 hours of a coin on the binance
// exchange
type Stats24hr struct {
	Symbol             string `json:"symbol"`
	PriceChange        string `json:"priceChange"`
	PriceChangePercent string `json:"priceChangePercent"`
	WeightedAvgPrice   string `json:"weightedAvgPrice"`
	PrevClosePrice     string `json:"prevClosePrice"`
	LastPrice          string `json:"lastPrice"`
}

// Orders is a list of orders from the Binance exchange
type Orders struct {
	Orders []Order
}

// Order contains the information about an order on the binance exchange
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

// AccountInfo is the information about a Binance exchange account
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

// Asset is the information about a coin currently held on the Binance exchange
type Asset struct {
	Asset  string `json:"asset"`
	Free   string `json:"free"`
	Locked string `json:"locked"`
}

// ServerTime is the current time of the Binance exchange server
type ServerTime struct {
	ServerTime int64 `json:"serverTime"`
}

// OrderRespHeader is information about an order submission
type OrderRespHeader struct {
	Symbol          string `json:"symbol"`
	OrderID         int    `json:"orderId"`
	ClientOrderID   string `json:"clientOrderId"`
	TransactionTime int64  `json:"transactTime"`
}

// Ack is a type of response from an order submission. It is the information
// that acknowledges that an order was submitted
type Ack struct {
	OrderRespHeader
}

// Result is a type of response from an order submission. It is the information
// about the result of the order submission
type Result struct {
	OrderRespHeader
	Price               string `json:"price"`
	OrigQty             string `json:"origQty"`
	ExecutedQty         string `json:"executedQty"`
	CummulativeQuoteQty string `json:"cummulativeQuoteQty"`
	Status              string `json:"status"`
	TimeInForce         string `json:"timeInForce"`
	Type                string `json:"type"`
	Side                string `json:"side"`
}

// Full is a type of response from an order submission. It is the all the
// available information about an order submission
type Full struct {
	Result
	Fills []Fill `json:"fills"`
}

// Fill is the information about how the order was filled
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

// signature creates the signature for signig api calls
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
func (c *Client) Get24hrStats(symbol string) (Stats24hr, error) {
	body, err := c.GetAPI(c.Address + BNB24hrStats + "?symbol=" + symbol)
	if err != nil {
		apiLog.Warn("WARN: error submitting get request:", err)
		return Stats24hr{}, err
	}

	stats := Stats24hr{}
	err = json.Unmarshal(body, &stats)
	if err != nil {
		apiLog.Warn("WARN: error unmarshaling stats:", err)
		return Stats24hr{}, err
	}

	return stats, nil
}

// GetAccountInfo calls the API endpoint that returns account info
//
// Weight = 5
func (c *Client) GetAccountInfo() (AccountInfo, error) {
	timestamp := strconv.FormatInt(time.Now().UnixNano()/int64(time.Millisecond), 10)
	params := fmt.Sprintf("timestamp=%v&recvWindow=%v", timestamp, recvWindow)
	sig := signature(params)
	query := fmt.Sprintf("?%v&signature=%v", params, sig)
	body, err := c.GetSecureAPI(c.Address + BNBAccount + query)
	if err != nil {
		apiLog.Warn("WARN: error submitting get request:", err)
		return AccountInfo{}, err
	}

	// Get Account Info
	account := AccountInfo{}
	err = json.Unmarshal(body, &account)
	if err != nil {
		apiLog.Warn("WARN: error unmarshaling account info:", err)
		return AccountInfo{}, err
	}
	return account, nil
}

// GetAllOrders calls the endpoint that returns all order history fpr a given symbol
//
// Weight 5
func (c *Client) GetAllOrders(symbol string) ([]Order, error) {
	timestamp := strconv.FormatInt(time.Now().UnixNano()/int64(time.Millisecond), 10)
	params := fmt.Sprintf("symbol=%v&timestamp=%v&recvWindow=%v", symbol, timestamp, recvWindow)
	sig := signature(params)
	query := fmt.Sprintf("?%v&signature=%v", params, sig)
	body, err := c.GetSecureAPI(c.Address + BNBAllOrders + query)
	if err != nil {
		apiLog.Warn("WARN: error submitting get request:", err)
		return []Order{}, err
	}

	// Get Open Orders
	orders := []Order{}
	err = json.Unmarshal(body, &orders)
	if err != nil {
		apiLog.Warn("WARN: error unmarshaling orders:", err)
		return []Order{}, err
	}

	return orders, nil
}

// GetBinanceExchangeInfo calls the API endpoint that returns info on the binance
// exchange
func (c *Client) GetBinanceExchangeInfo() (ExchangeInfo, error) {
	body, err := c.GetAPI(c.Address + BNBExchangeInfo)
	if err != nil {
		apiLog.Warn("WARN: error submitting get request:", err)
		return ExchangeInfo{}, err
	}

	// Get ExchangeInfo
	info := ExchangeInfo{}
	err = json.Unmarshal(body, &info)
	if err != nil {
		apiLog.Warn("WARN: error unmarshaling exchange info:", err)
		return ExchangeInfo{}, err
	}
	return info, nil
}

// GetCoinPrice calls the API endpoint that returns the current price for a coin
func (c *Client) GetCoinPrice(symbol string) (TickerPrice, error) {
	body, err := c.GetAPI(c.Address + BNBPrice + "?symbol=" + symbol)
	if err != nil {
		apiLog.Warn("WARN: error submitting get request:", err)
		return TickerPrice{}, err
	}

	price := TickerPrice{}
	err = json.Unmarshal(body, &price)
	if err != nil {
		apiLog.Warn("WARN: error unmarshaling ticker price:", err)
		return TickerPrice{}, err
	}

	return price, nil
}

// GetOpenOrders calls the endpoint that returns all open orders
//
// Weight 1 with symbol, 40 w/o symbol
func (c *Client) GetOpenOrders() ([]Order, error) {
	timestamp := strconv.FormatInt(time.Now().UnixNano()/int64(time.Millisecond), 10)
	params := fmt.Sprintf("timestamp=%v&recvWindow=%v", timestamp, recvWindow)
	sig := signature(params)
	query := fmt.Sprintf("?%v&signature=%v", params, sig)
	body, err := c.GetSecureAPI(c.Address + BNBOpenOrders + query)
	if err != nil {
		apiLog.Warn("WARN: error submitting get request:", err)
		return []Order{}, err
	}

	// Get Open Orders
	orders := []Order{}
	err = json.Unmarshal(body, &orders)
	if err != nil {
		apiLog.Warn("WARN: error unmarshaling ticker price:", err)
		return []Order{}, err
	}

	return orders, nil
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
func (c *Client) PostNewLimitOrder(symbol, side string, quantity float64) (Result, error) {

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
	if err != nil {
		apiLog.Warn("WARN: error submitting post request:", err)
		return Result{}, err
	}

	result := Result{}
	err = json.Unmarshal(body, &result)
	if err != nil {
		apiLog.Warn("WARN: error unmarshaling ticker price:", err)
		return Result{}, err
	}

	return result, nil
}
