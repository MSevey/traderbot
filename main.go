package main

// the main package initializes threads and loops to start and control the
// trader

import (
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/MSevey/traderbot/api"
	"github.com/MSevey/traderbot/mail"
	"github.com/MSevey/traderbot/trader"
	"github.com/sirupsen/logrus"
)

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

	// start trading
	go trade(done)

	// Send Email Summaries
	go emailSummaries(done)

	// Listen for crtl+c to end
	for {
		select {
		case <-sig:
			close(done)
			return
		default:
		}
	}
}

func emailSummaries(done chan struct{}) {
	// Send emails on start up
	if err := mail.EmailLifeTimePerformance(); err != nil {
		log.Warn("couldn't send life time summary email", err)
	}

	// Send emails on intervals
	for {
		// Send Summary once a minute
		lifeTimeSignal := time.After(24 * time.Hour)
		select {
		case <-done:
			return
		case <-lifeTimeSignal:
			if err := mail.EmailLifeTimePerformance(); err != nil {
				log.Warn("couldn't send life time summary email", err)
			}
		}
	}
}

// trade trades on the binance exchange
func trade(done chan struct{}) {
	// Initialize trader
	t := trader.NewTrader()

	// Create client for binance requests
	binanceClient := api.NewBinanceClient()

	// Get account information
	account, err := binanceClient.GetAccountInfo()
	if err != nil {
		log.Warn("Couldn't get account info", err)
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
		info, err := binanceClient.GetBinanceExchangeInfo()
		if err != nil {
			log.Warn("WARN: error getting exchange info:", err)
			continue
		}
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
