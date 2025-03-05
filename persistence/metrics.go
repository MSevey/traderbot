package persistence

import (
	"time"
)

const (
	// MetricsBucket is the bucket name for metrics-related data
	MetricsBucket = "metrics"

	// InitialBalanceKey is the key for storing the initial balance
	InitialBalanceKey = "initial_balance"
)

// Portfolio represents a portfolio of assets
type Portfolio struct {
	Assets  []Asset   `json:"assets"`
	Updated time.Time `json:"updated"`
	Value   float64   `json:"value"`
}

// Asset represents an asset in a portfolio
type Asset struct {
	Symbol   string  `json:"symbol"`
	Quantity float64 `json:"quantity"`
	Value    float64 `json:"value"`
}

// SaveInitialBalance saves the initial balance to the database
func (db *DB) SaveInitialBalance(portfolio Portfolio) error {
	if db == nil || db.DB == nil {
		return ErrNilDB
	}

	// Create bucket if it doesn't exist
	if err := db.CreateBucketIfNotExists(MetricsBucket); err != nil {
		return err
	}

	// Save to database
	return db.Put(MetricsBucket, InitialBalanceKey, portfolio)
}

// LoadInitialBalance loads the initial balance from the database
func (db *DB) LoadInitialBalance() (Portfolio, error) {
	if db == nil || db.DB == nil {
		return Portfolio{}, ErrNilDB
	}

	// Load from database
	var portfolio Portfolio
	err := db.Get(MetricsBucket, InitialBalanceKey, &portfolio)
	if err != nil {
		return Portfolio{}, err
	}

	return portfolio, nil
}