package api

import (
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

// Client is a helper struct for API calls
type Client struct {
	// API endpoint address
	Address string

	// Timeout is the timeout set for the API
	Timeout time.Duration
}

// check pulls out the duplicate error checking code
//
// TODO: Replace with log to file
func check(e error) {
	if e != nil {
		log.Fatal(e)
	}
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

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return []byte{}, err
	}

	// Setting the header lets remote servers understand what kind of traffic it
	// is receiving. Some sites will even reject empty or generic User-Agent
	// strings.
	req.Header.Set("User-Agent", "trader")

	res, getErr := apiClient.Do(req)
	if getErr != nil {
		return []byte{}, getErr
	}

	return ioutil.ReadAll(res.Body)
}
