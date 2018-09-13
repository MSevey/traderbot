package api

import (
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
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
// TODO: Replace with log to file (Logrus)
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
	check(err)

	return ioutil.ReadAll(res.Body)
}
