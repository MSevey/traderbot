package main

import (
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/MSevey/traderBot/api"
	"github.com/MSevey/traderBot/trader"
	"github.com/sirupsen/logrus"
)

// This will be a trader for Binance.
//
// Since Binance offers lower fees when you pay with BNB, store trading balance
// in BNB and buy and sell BTC

// TODO
//
//  FOR TESTING START WITH LOGGING STATS ON WHAT TRIGGERED THE BUY OF SELL,
//  HIGH/LOW PRICES, CONDITION THAT TRIGGERED ORDER, ETC. SUBMIT TEST ORDERS
//
//  LOG PRICE AFTER BUY AND SELL ORDERS TO SEE IF PRICE CONTINUES TO GO
//  UP/DOWN OR IF IT SWITCHED DIRECTIONS
//
// 1) Need to update error checking to be more uptime resilient.  Handling errors better
//		- ex, if account api fails the program panics because CanTrade is false by default
//
// 2) Buying Algorithm (need buying API calls to be working)
//      - buy when price has gone down by 5% or more.  based on currentPrice
//          compared to basePrice.  buy when 5% down and currentPrice > lastPrice
//          which would indicate the price has ended it's current drop
//		- Need to update to account for buys and then price increase and impact on base price
//		- think about how to reset base price (maybe after certain amount of time without any buys)
//		- think about updating buy difference target based on frequency of buys, too many buys close
//			together means price is dropping a lot and not buying at lowest value
//			- Have different limits for BTC and BNB
//
// 3) Selling Algorithm (need selling API calls to be working)
//      - submit sale order for 6% above purchase price after buy triggered
//          and sell the same about of BTC
//		- create heap of buy orders based on purchase price
//			when decided to sell use lowest price in heap and sell that quantity,
//			compare 5% above that price
//		- Heap needs to be used differently.  Create either submit limit order instantly of
//		create go routine to handle selling
//		- think about how to reset base price (maybe after certain amount of time without any sales)
//		- think about updating sell difference target based on frequency of buys, too many sales close
//			together means price is rising a lot and not selling at highest value
//
// 4) Set up log files (need to decide what is worth logging)
//      - create log for each module and main log
//      - log files for continue to grow until manually deleted
//
// 5) set up metric reporting (need trader to be working to determine how to
// calculate metrics)
//      - Daily high and low
//      - Number of buys/sells
//      - % profit for past 24hrs
//      - % profit all time
//      - number of pending sells
//
// 7) Set up to run on remote server/service (not needed until trader is active)
//      - test by just pinging single API endpoint repeatedly to verify it is working,
//          coinmarkercap for instance
//
// 9) Set up database (not needed until ready to run for extended period of
// time, ie 12hrs)
//      - install postgres
//      - log trades (own table)
//      - log daily values (own table)
//
// 10) Work to keep 10 BNB balance
//      - turn profit into BNB until 10 BNB balance
//      - Set constants for Binance trading limits (BTC / BNB), work contants into algos to optimize fees
//
// 11) Start with 10 sec delays between any API call loops, update to check in
// realtime if weight limits have been met.  Look at slice or map for weights
// and timestamps to analyze
//
// 12) Before pushing to github or any online repo, remove commit containing
// email password

const (
	// the following the time intervals that the loops should run
	binanceLoopTime = 2 * time.Second // if running all day set to 10s
	metricsLoopTime = 12 * time.Hour

	bnbBalanceTarget = 10 // set but binance trading levels
)

var log = logrus.New()

func initLogger() {
	log.SetLevel(logrus.DebugLevel)
	// You could set this to any `io.Writer` such as a file
	file, err := os.OpenFile("main.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err == nil {
		log.Out = file
	} else {
		log.Info("Testing Logrus")
	}

	log.Info("Main file logging")
}

func main() {
	initLogger()
	api.InitLogger()

	// Create channel to control go routines
	//
	// TODO: look at importing Nebulous Labs thread repo
	done := make(chan struct{})
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)

	// // Metrics goroutine
	// fmt.Println("go metrics")
	// go metrics.Test()

	// // coinMarketCap API goroutine
	// fmt.Println("go coin")
	// go coinMarketCap(done)

	// binance API goroutine
	go binance(done)

	// // Sending Email
	// //
	// // Build Mail object
	// mail.SendEmail()

	// Listen for crtl c to end
	for {
		select {
		case <-sig:
			close(done)
			return
		default:
		}
	}
}

// check pulls out the duplicate error checking code
//
// TODO: Replace with log to file
func check(e error) {
	if e != nil {
		log.Debug(e)
	}
}

// binance is the current go routine that controlls all the binance calls
func binance(done chan struct{}) {
	// Initialize trader
	t := trader.NewTrader()

	// Create client for binance requests
	binanceClient := api.NewBinanceClient()

	// Get account information
	account := binanceClient.GetAccountInfo()
	if !account.CanTrade {
		log.Warn("Can't Trade!!")
	}

	// Update balances
	t.UpdateBalances(account)

	fmt.Println("btcBalance", t.BtcBalance())
	fmt.Println("bnbBalance", t.BnbBalance())
	fmt.Println("usdtBalance", t.UsdtBalance())
	fmt.Println("minBalance", t.MinBalance())

	for {
		// Ping exchange to get up to date limits
		info := binanceClient.GetBinanceExchangeInfo()
		t.UpdateLimits(info)

		// // Buy BTC (currently inverting for testing)
		// // TODO, need API call info to be updated even when not buying
		// // or else BNB loop wants to buy infinite BNB
		// if !t.CanBuyBTC() {
		// 	t.TryBTCBuy(binanceClient)

		// }

		if t.MinBalance() < t.BtcBalance() {
			// Buy BNB
			// if t.BnbBalance() < bnbBalanceTarget {
			//  t.TryBNBBuy(binanceClient)
			// }
			//Sell BTC
			t.TryBTCSell(binanceClient)
		}

		// Update Balances
		account := binanceClient.GetAccountInfo()
		if !account.CanTrade {
			panic("Can't Trade!!")
		}
		t.UpdateBalances(account)

		select {
		case <-done:
			// persist minBalance
			if err := os.Setenv("binanceMinBalance", strconv.FormatFloat(t.MinBalance(), 'f', -1, 64)); err != nil {
				log.Warn(err)
			}
			// submit all order heap as sell orders
			return
		default:
		}
		time.Sleep(binanceLoopTime)
	}
}

//
func coinMarketCap(done chan struct{}) {
	fmt.Println("coin")
	// Create client for coinmarketcap requests
	coinClient := api.NewCoinCapClient()

	BTC := coinClient.GetTickerInfo(api.BTCID)
	fmt.Println(BTC)

	BNB := coinClient.GetTickerInfo(api.BNBID)
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
