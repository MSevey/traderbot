package persistence

import (
	"strconv"
)

const (
	// TraderBucket is the bucket name for trader-related data
	TraderBucket = "trader"

	// MinBalanceKey is the key for storing the minimum balance
	MinBalanceKey = "min_balance"
)

// SaveMinBalance saves the minimum balance to the database
func (db *DB) SaveMinBalance(minBalance float64) error {
	if db == nil || db.DB == nil {
		return ErrNilDB
	}

	// Create bucket if it doesn't exist
	if err := db.CreateBucketIfNotExists(TraderBucket); err != nil {
		return err
	}

	// Convert float to string
	minBalanceStr := strconv.FormatFloat(minBalance, 'f', -1, 64)

	// Save to database
	return db.Put(TraderBucket, MinBalanceKey, minBalanceStr)
}

// LoadMinBalance loads the minimum balance from the database
func (db *DB) LoadMinBalance() (float64, error) {
	if db == nil || db.DB == nil {
		return 0, ErrNilDB
	}

	// Load from database
	var minBalanceStr string
	err := db.Get(TraderBucket, MinBalanceKey, &minBalanceStr)
	if err != nil {
		return 0, err
	}

	// Convert string to float
	minBalance, err := strconv.ParseFloat(minBalanceStr, 64)
	if err != nil {
		return 0, err
	}

	return minBalance, nil
}