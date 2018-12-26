package main

import (
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/MSevey/traderBot/api"
	"github.com/MSevey/traderBot/metrics"
	"github.com/MSevey/traderBot/trader"
	"github.com/sirupsen/logrus"
)

// This will be a trader for Binance.
//
// Since Binance offers lower fees when you pay with BNB, store trading balance
// in BNB and buy and sell BTC

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
	p, err := metrics.PortfolioBalance()
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(p)

	// // coinMarketCap API goroutine
	// fmt.Println("go coin")
	// go coinMarketCap(done)

	// binance API goroutine
	// go binance(done)

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
	account, err := binanceClient.GetAccountInfo()
	if err != nil {
		log.Warn("Couldn't get account info")
		return
	}
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

		btcPrice, btcSymbol, err := t.UpdateBTCBuyerPrices(binanceClient)
		if err != nil {
			log.Warn("error updating buyer prices:", err)
			continue
		}

		// Buy BTC (currently inverting for testing)
		if !t.CanBuyBTC() {
			t.TryBTCBuy(binanceClient, btcPrice, btcSymbol)

		}

		if t.MinBalance() < t.BtcBalance() {
			btcPrice, err := t.UpdateBTCSellerPrices(binanceClient)
			if err != nil {
				log.Warn("error updating seller prices:", err)
				continue
			}
			// Buy BNB
			if t.BnbBalance() < bnbBalanceTarget {
				t.TryBNBBuy(binanceClient)
			}
			//Sell BTC
			t.TryBTCSell(binanceClient, btcPrice)
		}

		// Update Balances
		account, err := binanceClient.GetAccountInfo()
		if err != nil {
			log.Warn(err)
			continue
		}
		if !account.CanTrade {
			log.Warn("can't trade, CanTrade is false")
			continue
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
