package trader

import (
	"container/heap"
	"errors"
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

	buyBalanceLimit = 5      // buy $5 at a time
	diffLimit       = 0.0001 // .01% to start
)

var (
	errBasePriceNotSet = errors.New("base price not set")
	errUpdateLastPrice = errors.New("updating last price")
)

// Trader is the struct to control some of the functionality
type Trader struct {
	buyAmount     float64 // amount of BTC to buy at a time, set to 0.001 for now (~$6.60 as of 7/6/18)
	dailyHigh     float64 // hight point of the past 24hrs
	dailyLow      float64 // low point of the past 24hrs
	buyLimit      bool    // Has the buy limit been reached
	lastHighPoint float64 // The last price high point to compare against
	numOrders     int     // Number of current active orders

	// counters to help with refining buy and sell algos so they are making the
	// most and not executing too quickly
	//
	// TODO - will need to tie these into a method that adjust the difference
	// target
	numberOfBTCBuys  int
	numberOfBNBBuys  int
	numberOfBTCSells int

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

	log *logrus.Logger

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
func (t *Trader) check(e error) {
	if e != nil {
		t.log.Debug(e)
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

	// Init logger
	t.log = logrus.New()
	t.log.SetLevel(logrus.DebugLevel)

	// You could set this to any `io.Writer` such as a file
	file, err := os.OpenFile("./trader/trader.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err == nil {
		t.log.Out = file
		t.log.Debug("Trader created")
	} else {
		t.log.Debug("Failed to create Trader Log")
	}
	return t
}

// TryBNBBuy tries to buy bnb
func (t *Trader) TryBNBBuy(c *api.Client) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.Buyer.mu.Lock()
	defer t.Buyer.mu.Unlock()

	// Check price of BNBBTC
	bnbpriceStr := c.GetCoinPrice(api.BNBBTC)
	bnbprice, err := strconv.ParseFloat(bnbpriceStr.Price, 64)
	t.check(err)

	// Check to make sure base price is set
	if t.Buyer.bnbBasePrice == 0 {
		t.Buyer.bnbBasePrice = bnbprice
		return
	}
	// Compare to previous price
	if bnbprice <= t.Buyer.bnbLastPrice || t.Buyer.bnbLastPrice == 0 {
		t.Buyer.bnbLastPrice = bnbprice
		return
	}

	diff := (t.Buyer.bnbBasePrice - bnbprice) / t.Buyer.bnbBasePrice
	if diff < diffLimit {
		return
	}

	// Buy at last price
	// TODO
	//  - change to api order
	//  - how to confirm order went through??
	quantity := buyBalanceLimit / t.Buyer.btcLastPrice / bnbprice
	t.log.WithFields(logrus.Fields{
		"bnbBasePrice":    t.Buyer.bnbBasePrice,
		"bnbLastPrice":    t.Buyer.bnbLastPrice,
		"bnbprice":        bnbprice,
		"diff":            diff,
		"quantity":        quantity,
		"buyBalanceLimit": buyBalanceLimit,
		"btcLastPrice":    t.Buyer.btcLastPrice,
	}).Debug("***BNB Buy conditions met***")

	t.numberOfBNBBuys++
	fmt.Println("Number of BNB Buys", t.numberOfBNBBuys)

	// Update Base Price
	t.Buyer.bnbBasePrice = t.Buyer.bnbLastPrice
}

// TryBTCBuy tries to buy btc
func (t *Trader) TryBTCBuy(c *api.Client, btcPrice float64, btcSymbol string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.Buyer.mu.Lock()
	defer t.Buyer.mu.Unlock()

	diff := (t.Buyer.btcBasePrice - btcPrice) / t.Buyer.btcBasePrice
	if diff < diffLimit {
		return
	}

	// Buy at last price
	// TODO
	//  - change to api order
	//  - how to confirm order went through??
	//  - should by at t.Buyer.btcLastPrice not btcPrice
	quantity := buyBalanceLimit / btcPrice
	t.log.WithFields(logrus.Fields{
		"btcBasePrice": t.Buyer.btcBasePrice,
		"btcLastPrice": t.Buyer.btcLastPrice,
		"btcPrice":     btcPrice,
		"diff":         diff,
		"quantity":     quantity,
	}).Debug("***BTC Buy conditions met***")

	t.numberOfBTCBuys++
	fmt.Println("Number of BTC Buys", t.numberOfBTCBuys)

	// Add to Heap
	order := &order{
		symbol:   btcSymbol,
		price:    t.Buyer.btcLastPrice,
		quantity: quantity,
	}
	heap.Push(&t.Buyer.orders, order)
	t.Buyer.orders.update(order, order.symbol, order.price, order.quantity)

	// update Base price
	t.Buyer.btcBasePrice = t.Buyer.btcLastPrice
}

// TryBTCSell tries to buy btc
func (t *Trader) TryBTCSell(c *api.Client, btcprice float64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.Seller.mu.Lock()
	defer t.Seller.mu.Unlock()
	// BTC Selling
	// Prioritize selling against previous buy orders, if not orders in heap, track against base and last price
	//

	// Pop lowest buy order off heap, set price to base price
	t.Buyer.mu.Lock()
	defer t.Buyer.mu.Unlock()
	if len(t.Buyer.orders) != 0 {
		t.log.Info("Order from Heap")
		order := heap.Pop(&t.Buyer.orders).(*order)
		t.log.WithFields(logrus.Fields{
			"order price": order.price,
			"btcprice":    btcprice,
		}).Debug("BTC Sell info")

		// Compare to previous order price
		if btcprice < order.price {
			return
		}

		diff := (btcprice-order.price)/order.price - 1
		if diff < diffLimit {
			// if no sell push order back onto heap
			heap.Push(&t.Buyer.orders, order)
			t.Buyer.orders.update(order, order.symbol, order.price, order.quantity)
			return
		}

		// Sell at last price
		// TODO
		//  - change to api order
		//  - how to confirm order went through??
		//  - should sell at t.Seller.btcLastPrice not btcprice
		t.log.WithFields(logrus.Fields{
			"btcBasePrice": t.Seller.btcBasePrice,
			"btcLastPrice": t.Seller.btcLastPrice,
			"btcprice":     btcprice,
			"diff":         diff,
			"quantity":     order.quantity,
		}).Debug("***BTC Sell conditions met***")

		t.numberOfBTCSells++
		fmt.Println("Number of BTC Sells", t.numberOfBTCSells)

		// if no sell push order back onto heap
		if false {
			heap.Push(&t.Buyer.orders, order)
			t.Buyer.orders.update(order, order.symbol, order.price, order.quantity)
		}
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
	t.log.Info("Sell based on Prices")
	quantity := buyBalanceLimit / t.Seller.btcBasePrice
	t.log.WithFields(logrus.Fields{
		"btcBasePrice": t.Seller.btcBasePrice,
		"btcLastPrice": t.Seller.btcLastPrice,
		"btcprice":     btcprice,
		"diff":         diff,
		"quantity":     quantity,
	}).Debug("***BTC Sell conditions met***")
	t.numberOfBTCSells++
	fmt.Println("Number of BTC Sells", t.numberOfBTCSells)

	// Reset base price
	t.Seller.btcBasePrice = t.Seller.btcLastPrice

}

// UpdateBalances updates the asset and min balance of the Trader
func (t *Trader) UpdateBalances(account api.AccountInfo) {
	t.mu.Lock()
	defer t.mu.Unlock()

	for _, asset := range account.Balances {
		if asset.Asset == "BTC" {
			bal, err := strconv.ParseFloat(asset.Free, 64)
			t.check(err)
			t.btcBalance = bal
		}
		if asset.Asset == "BNB" {
			bal, err := strconv.ParseFloat(asset.Free, 64)
			t.check(err)
			t.bnbBalance = bal
		}
		if asset.Asset == "USDT" {
			bal, err := strconv.ParseFloat(asset.Free, 64)
			t.check(err)
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
		t.check(err)
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
		t.log.Debug("Limits Updated")
	}
}

// UpdateBTCBuyerPrices updates the base and last price of the Buyer
func (t *Trader) UpdateBTCBuyerPrices(c *api.Client) (float64, string, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.Buyer.mu.Lock()
	defer t.Buyer.mu.Unlock()

	// Check price of BTCUSDT
	btcpriceStr := c.GetCoinPrice(api.BTCUSDT)
	btcprice, err := strconv.ParseFloat(btcpriceStr.Price, 64)
	t.check(err)

	// Check to make sure base price is set
	if t.Buyer.btcBasePrice == 0 {
		t.Buyer.btcBasePrice = btcprice
		return btcprice, btcpriceStr.Symbol, errBasePriceNotSet
	}
	// Compare to previous price
	if btcprice <= t.Buyer.btcLastPrice || t.Buyer.btcLastPrice == 0 {
		t.Buyer.btcLastPrice = btcprice
		return btcprice, btcpriceStr.Symbol, errUpdateLastPrice
	}

	return btcprice, btcpriceStr.Symbol, nil
}

// UpdateBTCSellerPrices updates the base and last price of the Seller
func (t *Trader) UpdateBTCSellerPrices(c *api.Client) (float64, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.Seller.mu.Lock()
	defer t.Seller.mu.Unlock()

	// Check price of BTCUSDT
	btcpriceStr := c.GetCoinPrice(api.BTCUSDT)
	btcprice, err := strconv.ParseFloat(btcpriceStr.Price, 64)
	t.check(err)

	// Check to make sure base price is set
	if t.Seller.btcBasePrice == 0 {
		t.Seller.btcBasePrice = btcprice
		return btcprice, errBasePriceNotSet
	}
	// Compare to previous price
	if btcprice >= t.Seller.btcLastPrice {
		t.Seller.btcLastPrice = btcprice
		return btcprice, errUpdateLastPrice
	}

	return btcprice, nil
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
