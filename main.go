package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/MSevey/trader/api"
	"github.com/MSevey/trader/mail"
	"github.com/MSevey/trader/metrics"
)

// This will be a trader for Binance.
//
// Since Binance offers lower fees when you pay with BNB, store trading balance
// in BNB and buy and sell BTC

// TODO
//
// DONE: set up emailing
//
// 2) set up metric reporting (need trader to be working to determine how to calculate metrics)
//      - Daily high and low
//      - Number of buys/sells
//      - % profit for past 24hrs
//      - % profit all time
//      - number of pending sells
//
// 3) Buying Algorithm (need buying API calls to be working)
//      - buy when price has gone down by 5% or more.  based on currentPrice
//          compared to basePrice.  buy when 5% down and currentPrice > lastPrice
//          which would indicate the price has ended it's current drop
//
// DONE: Determine Binance fees
//		- 0.1% per trade, 0.05% if paid with BNB
//
// 5) Selling Algorithm (need selling API calls to be working)
//      - submit sale order for 6% above purchase price after buy triggered
//          and sell the same about of BTC
//
// 5) Set up API call for Binance (Priority)
//		- endpoint to get BTC / BNB price
//		- endpoint for submitting orders
//		- endpoint for getting order history
//		- ping exchange info daily to check limits as to not to exceed them (ie request limit)
//			- current request limit is 1200, target 1080, 1/min from 6am to 6pm, 1/2min from 6pm to 6am
//			- create struct to store current limits, set daily via API call
//
// 6) Set up to run on remote server/service (not needed until trader is active)
//      - test by just pinging single API endpoint repeatedly to verify it is working,
//			coinmarkercap for instance
//
// 7) Start with $5 trades, 10 times, with 1% difference threshold.
//		- with worst case of .01% fees, this should equate to 10% gains, or $0.5
//
// 8) Set up log files (need to decide what is worth logging)
//		- create log for each module and main log
//		- log files for continue to grow until manually deleted
//
// 9) Set up database (not needed until ready to run for extended period of time, ie 12hrs)
// 		- install postgres
// 		- log trades (own table)
//		- log daily values (own table)
//
// 10) Set up environment variables
//		- email password
//		- binance API key

// trader is the struct to control some of the functionality
type trader struct {
	buyAmount     float64 // amount of BTC to buy at a time, set to 0.001 for now (~$6.60 as of 7/6/18)
	basePrice     float64 // price to be used for comparison
	lastPrice     float64 // price recorded from last api call
	currentPrice  float64 // price recorded from current api call
	dailyHigh     float64 // hight point of the past 24hrs
	dailyLow      float64 // low point of the past 24hrs
	buyLimit      bool    // Has the buy limit been reached
	lastHighPoint float64 // The last price high point to compare against
	numOrders     int     // Number of current active orders

	mu sync.Mutex
}

func main() {
	// Create channel to control go routines
	//
	// TODO: look at importing Nebulous Labs thread repo
	done := make(chan struct{})

	// Metrics goroutine
	fmt.Println("go metrics")
	go metrics.Test()

	// coinMarketCap API goroutine
	fmt.Println("go coin")
	go coinMarketCap(done)

	// binance API goroutine
	fmt.Println("go binance")
	go binance(done)

	time.Sleep(4 * time.Second)
	close(done)

	// Sending Email
	//
	// Build Mail object
	mail.SendEmail()
}

// check pulls out the duplicate error checking code
//
// TODO: Replace with log to file
func check(e error) {
	if e != nil {
		log.Fatal(e)
	}
}

// func jsonUnmarshal(data []byte,)

//
func binance(done chan struct{}) {
	fmt.Println("binance")
	// Create client for binance requests
	binanceClient := api.NewBinanceClient()

	fmt.Println("Binance Rate limits")
	rl := binanceClient.GetBinanceExchangeInfo()
	fmt.Println(rl)

	// Get BNB to BTC price
	fmt.Println("Binance BTCUSDT price")
	price := binanceClient.GetCoinPrice(api.BTCUSDT)
	fmt.Println(price.Price)

	// Get BNB to BTC 24hr statistics
	fmt.Println("Binance BTCUSDT 24hr stats")
	stats := binanceClient.Get24hrStats(api.BTCUSDT)
	fmt.Println(stats)

	// Loop to test go routine
	for {
		select {
		case <-done:
			return
		default:
		}
		fmt.Println("binance loop")
		time.Sleep(time.Second)
	}
}

//
func coinMarketCap(done chan struct{}) {
	fmt.Println("coin")
	// Create client for coinmarketcap requests
	coinClient := api.NewClient(api.CoinMarketCapTickerAPI, 2)

	body, err := coinClient.GetAPI(coinClient.Address + strconv.Itoa(api.BTCID) + "/")
	check(err)

	BTC := api.CoinCap{}
	jsonErr := json.Unmarshal(body, &BTC)
	check(jsonErr)

	fmt.Println(BTC)

	body, err = coinClient.GetAPI(coinClient.Address + strconv.Itoa(api.BNBID) + "/")
	check(err)

	BNB := api.CoinCap{}
	jsonErr = json.Unmarshal(body, &BNB)
	check(jsonErr)

	fmt.Println(BNB)

	// Loop to test go routine
	for {
		select {
		case <-done:
			return
		default:
		}
		fmt.Println("coinMarketCap loop")
		time.Sleep(time.Second)
	}
}
