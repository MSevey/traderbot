package trader

import (
	"container/heap"
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/MSevey/traderBot/api"
	"github.com/sirupsen/logrus"
)

const (
	// trading criteria

	buyBalanceLimit = 5    // buy $5 at a time
	diffLimit       = 0.01 // 1% to start
)

var traderLog = logrus.New()

// Trader is the struct to control some of the functionality
type Trader struct {
	buyAmount     float64 // amount of BTC to buy at a time, set to 0.001 for now (~$6.60 as of 7/6/18)
	dailyHigh     float64 // hight point of the past 24hrs
	dailyLow      float64 // low point of the past 24hrs
	buyLimit      bool    // Has the buy limit been reached
	lastHighPoint float64 // The last price high point to compare against
	numOrders     int     // Number of current active orders

	btcBalance  float64
	bnbBalance  float64
	usdtBalance float64
	minBalance  float64 // in BTC, Set to 25% below starting limit
	canBuyBTC   bool

	// API Limits
	Limits           api.BNBLimits
	LimitsLastUpdate time.Time

	// order structs
	Buyer  *Buyer
	Seller *Seller

	mu sync.Mutex
}

// Buyer is a struct to help control the buying algorithm
type Buyer struct {
	// starting price, price at point of buy or start of program
	bnbBasePrice float64
	btcBasePrice float64
	// price recorded from last api call
	bnbLastPrice float64
	btcLastPrice float64

	orders buyOrderHeap

	mu sync.Mutex
}

// Seller is a struct to help control the selling algorithm
type Seller struct {
	// starting price, price at point of sale or start of program
	bnbBasePrice float64
	btcBasePrice float64
	// price recorded from last api call
	bnbLastPrice float64
	btcLastPrice float64

	mu sync.Mutex
}

// buyOrderHeap is a priority queue and implements heap.Interface and holds orders
type buyOrderHeap []*order

// order contatins the necessary information from a buy order to prioritize sell
// orders
type order struct {
	symbol   string
	price    float64
	quantity float64
	index    int
}

// Required functions for use of heap for buyOrderHeap
func (boh buyOrderHeap) Len() int { return len(boh) }

// Less returns the lesser of two elements
func (boh buyOrderHeap) Less(i, j int) bool { return boh[i].price < boh[j].price }

// Swap swaps two elements from the heap
func (boh buyOrderHeap) Swap(i, j int) {
	boh[i], boh[j] = boh[j], boh[i]
	boh[i].index = i
	boh[j].index = j
}

// Push adds an element to the heap
func (boh *buyOrderHeap) Push(x interface{}) {
	n := len(*boh)
	order := x.(*order)
	order.index = n
	*boh = append(*boh, order)
}

// Pop removes element from the heap
func (boh *buyOrderHeap) Pop() interface{} {
	old := *boh
	n := len(old)
	chunkData := old[n-1]
	chunkData.index = -1 // for safety
	*boh = old[0 : n-1]
	return chunkData
}

// update updates the heap and reorders
func (boh *buyOrderHeap) update(o *order, symbol string, price, quantity float64) {
	o.symbol = symbol
	o.price = price
	o.quantity = quantity
	heap.Fix(boh, o.index)
}

// check pulls out the duplicate error checking code
//
// TODO: Replace with log to file (Logrus)
func check(e error) {
	if e != nil {
		traderLog.Debug(e)
	}
}

// NewTrader returns a pointer to a new Trader
func NewTrader() *Trader {
	t := &Trader{}
	t.Buyer = &Buyer{}
	t.Seller = &Seller{}
	buyOrderHeap := make(buyOrderHeap, 0)
	heap.Init(&buyOrderHeap)
	t.Buyer.orders = buyOrderHeap

	// The API for setting attributes is a little different than the package level
	// exported logger. See Godoc.
	traderLog.Out = os.Stdout

	// You could set this to any `io.Writer` such as a file
	file, err := os.OpenFile("binance.log", os.O_CREATE|os.O_WRONLY, 0666)
	if err == nil {
		traderLog.Out = file
		traderLog.Debug("Binance Trader created")
	} else {
		traderLog.Debug("Failed to create Binance Log")
	}
	return t
}

// TryBNBBuy tries to buy bnb
func (b *Buyer) TryBNBBuy(c *api.Client) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Check price of BNBBTC
	bnbpriceStr := c.GetCoinPrice(api.BNBBTC)
	bnbprice, err := strconv.ParseFloat(bnbpriceStr.Price, 64)
	check(err)

	// Check to make sure base price is set
	if b.bnbBasePrice == 0 {
		b.bnbBasePrice = bnbprice
		return
	}
	// Compare to previous price
	if bnbprice <= b.bnbLastPrice || b.bnbLastPrice == 0 {
		b.bnbLastPrice = bnbprice
		return
	}

	diff := (b.bnbBasePrice - bnbprice) / b.bnbBasePrice
	if diff < diffLimit {
		return
	}

	// Buy
	// TODO
	//  - change to api order
	//  - how to confirm order went through??
	quantity := buyBalanceLimit / b.btcLastPrice * bnbprice
	traderLog.WithFields(logrus.Fields{
		"bnbBasePrice": b.bnbBasePrice,
		"bnbLastPrice": b.bnbLastPrice,
		"bnbprice":     bnbprice,
		"diff":         diff,
		"quantity":     quantity,
	}).Debug("***Buy conditions met***")
}

// TryBTCBuy tries to buy btc
func (b *Buyer) TryBTCBuy(c *api.Client) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Check price of BTCUSDT
	btcpriceStr := c.GetCoinPrice(api.BTCUSDT)
	btcprice, err := strconv.ParseFloat(btcpriceStr.Price, 64)
	check(err)

	// Check to make sure base price is set
	if b.btcBasePrice == 0 {
		b.btcBasePrice = btcprice
		return
	}
	// Compare to previous price
	if btcprice <= b.btcLastPrice || b.btcLastPrice == 0 {
		b.btcLastPrice = btcprice
		return
	}

	diff := (b.btcBasePrice - btcprice) / b.btcBasePrice
	if diff < diffLimit {
		return
	}

	// Buy
	// TODO
	//  - change to api order
	//  - how to confirm order went through??
	quantity := buyBalanceLimit / btcprice
	traderLog.WithFields(logrus.Fields{
		"btcBasePrice": b.btcBasePrice,
		"btcLastPrice": b.btcLastPrice,
		"btcprice":     btcprice,
		"diff":         diff,
		"quantity":     quantity,
	}).Debug("***Buy conditions met***")

	// Add to Heap
	order := &order{
		symbol:   btcpriceStr.Symbol,
		price:    btcprice,
		quantity: quantity,
	}
	heap.Push(&b.orders, order)
	b.orders.update(order, order.symbol, order.price, order.quantity)
}

// TryBTCSell tries to buy btc
func (t *Trader) TryBTCSell(c *api.Client) {
	t.mu.Lock()
	defer t.mu.Unlock()
	// BTC Selling
	// Prioritize selling against previous buy orders, if not orders in heap, track against base and last price
	//

	// Check price of BTCUSDT
	btcpriceStr := c.GetCoinPrice(api.BTCUSDT)
	btcprice, err := strconv.ParseFloat(btcpriceStr.Price, 64)
	check(err)

	// Pop lowest buy order off heap, set price to base price
	t.Buyer.mu.Lock()
	defer t.Buyer.mu.Unlock()
	if len(t.Buyer.orders) != 0 {
		order := heap.Pop(&t.Buyer.orders).(*order)

		// Compare to previous order price
		if btcprice < order.price {
			return
		}

		diff := (btcprice-order.price)/order.price - 1
		if diff < diffLimit {
			return
		}

		// Sell
		// TODO
		//  - change to api order
		//  - how to confirm order went through??
		traderLog.WithFields(logrus.Fields{
			"btcBasePrice": t.Seller.btcBasePrice,
			"btcLastPrice": t.Seller.btcLastPrice,
			"btcprice":     btcprice,
			"diff":         diff,
			"quantity":     order.quantity,
		}).Debug("***Sell conditions met***")

		// if no sell push order back onto heap
		heap.Push(&t.Buyer.orders, order)
		t.Buyer.orders.update(order, order.symbol, order.price, order.quantity)
		return
	}

	// Check to make sure base price is set
	if t.Seller.btcBasePrice == 0 {
		t.Seller.btcBasePrice = btcprice
		return
	}
	// Compare to previous price
	if btcprice >= t.Seller.btcLastPrice {
		t.Seller.btcLastPrice = btcprice
		return
	}

	diff := (btcprice-t.Seller.btcBasePrice)/t.Seller.btcBasePrice - 1
	if diff < diffLimit {
		return
	}

	// Sell
	// TODO
	//  - change to api order
	//  - how to confirm order went through??
	quantity := buyBalanceLimit / t.Seller.btcBasePrice
	traderLog.WithFields(logrus.Fields{
		"btcBasePrice": t.Seller.btcBasePrice,
		"btcLastPrice": t.Seller.btcLastPrice,
		"btcprice":     btcprice,
		"diff":         diff,
		"quantity":     quantity,
	}).Debug("***Sell conditions met***")

}

// UpdateBalances updates the asset and min balance of the Trader
func (t *Trader) UpdateBalances(account api.AccountInfo) {
	t.mu.Lock()
	defer t.mu.Unlock()

	for _, asset := range account.Balances {
		if asset.Asset == "BTC" {
			bal, err := strconv.ParseFloat(asset.Free, 64)
			check(err)
			t.btcBalance = bal
		}
		if asset.Asset == "BNB" {
			bal, err := strconv.ParseFloat(asset.Free, 64)
			check(err)
			t.bnbBalance = bal
		}
		if asset.Asset == "USDT" {
			bal, err := strconv.ParseFloat(asset.Free, 64)
			check(err)
			t.usdtBalance = bal
			if t.usdtBalance > buyBalanceLimit {
				t.canBuyBTC = true
			}
		}
	}

	// Set min balance
	var minBal float64
	var err error
	minBalStr := os.Getenv("binanceMinBalance")
	if minBalStr != "" {
		minBal, err = strconv.ParseFloat(minBalStr, 64)
		check(err)
	}
	if t.minBalance < minBal {
		t.minBalance = minBal
	}
	if t.minBalance < 0.75*t.btcBalance {
		t.minBalance = 0.75 * t.btcBalance
	}
}

// UpdateLimits updates the api limits of the Trader
func (t *Trader) UpdateLimits(info api.ExchangeInfo) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if time.Now().After(t.LimitsLastUpdate.Add(24 * time.Hour)) {
		t.LimitsLastUpdate = time.Now()
		t.Limits = info.BNBLimits
		fmt.Println("Limits Updated")
	}
}

// BtcBalance returns the btcBalance of the trader
func (t *Trader) BtcBalance() float64 {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.btcBalance
}

// BnbBalance returns the bnbBalance of the trader
func (t *Trader) BnbBalance() float64 {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.bnbBalance
}

// CanBuyBTC returns the canBuyBTC value of the trader
func (t *Trader) CanBuyBTC() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.canBuyBTC
}

// MinBalance returns the minBalance of the trader
func (t *Trader) MinBalance() float64 {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.minBalance
}

// UsdtBalance returns the usdtBalance of the trader
func (t *Trader) UsdtBalance() float64 {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.usdtBalance
}
