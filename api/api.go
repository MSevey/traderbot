package api

// the api package handles the code the interacts with the external exchange's
// api. The api package file should contain all the generate API handling code.
// Code for each API should then be contained in its own file, ie code for
// dealing with the Binance Exchange is contained in binance.go

import (
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/sirupsen/logrus"
)

// api logger
var apiLog = logrus.New()

// Client is a helper struct for API calls
type Client struct {
	// API endpoint address
	Address string

	// Timeout is the timeout set for the API
	Timeout time.Duration
}

// check pulls out the duplicate error checking code
//
// TODO: this should be removed and errors should be handling at the source
func check(e error) {
	if e != nil {
		apiLog.Debug(e)
	}
}

// InitLogger initializes the logger for the api
//
// NOTE: currently logging right to the terminal
func InitLogger() {
	apiLog.SetLevel(logrus.DebugLevel)
	// // You could set this to any `io.Writer` such as a file
	// file, err := os.OpenFile("api/api.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	// if err == nil {
	// 	apiLog.Out = file
	// } else {
	// 	apiLog.Warn("File Not Created")
	// }

	apiLog.Out = os.Stdout

	apiLog.Info("API file logging")
}

// NewClient creates a new client to be used for API calls
func NewClient(address string, timeout time.Duration) *Client {
	return &Client{
		Address: address,
		Timeout: time.Second * timeout,
	}
}

// GetAPI submits a get request to the intended url endpoint
func (c *Client) GetAPI(url string) ([]byte, error) {
	apiClient := http.Client{
		Timeout: c.Timeout,
	}

	res, err := apiClient.Get(url)
	if err != nil {
		return []byte{}, err
	}

	return ioutil.ReadAll(res.Body)
}

// PostAPI submits a post request to the intended url endpoint
func (c *Client) PostAPI(url string, data url.Values) ([]byte, error) {
	apiClient := http.Client{
		Timeout: c.Timeout,
	}

	res, err := apiClient.PostForm(url, data)
	if err != nil {
		return []byte{}, err
	}

	return ioutil.ReadAll(res.Body)
}

// GetSecureAPI submits a new get request to the intended url endpoint with the
// public api key in the header
func (c *Client) GetSecureAPI(url string) ([]byte, error) {
	apiClient := http.Client{
		Timeout: c.Timeout,
	}

	req, err := http.NewRequest("GET", url, nil)
	check(err)
	req.Header.Add("X-MBX-APIKEY", BNBAPIPubKey)
	res, err := apiClient.Do(req)
	if err != nil {
		return []byte{}, err
	}

	return ioutil.ReadAll(res.Body)
}

// PostSecureAPI submits a new post request to the intended url endpoint with
// the public api key in the header
func (c *Client) PostSecureAPI(url string) ([]byte, error) {
	apiClient := http.Client{
		Timeout: c.Timeout,
	}

	req, err := http.NewRequest("POST", url, nil)
	check(err)
	req.Header.Add("X-MBX-APIKEY", BNBAPIPubKey)
	res, err := apiClient.Do(req)
	if err != nil {
		return []byte{}, err
	}

	return ioutil.ReadAll(res.Body)
}
