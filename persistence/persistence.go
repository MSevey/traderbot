package persistence

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/boltdb/bolt"
)

const (
	// DefaultDatabaseFile is the default name of the database file
	DefaultDatabaseFile = "traderbot.db"

	// DefaultDatabaseDir is the default directory where the database file will be stored
	DefaultDatabaseDir = "traderbot_data"

	// DefaultTimeout is the default timeout for database operations
	DefaultTimeout = 1 * time.Second
)

var (
	// ErrBucketNotFound is returned when a bucket does not exist
	ErrBucketNotFound = errors.New("bucket not found")

	// ErrKeyNotFound is returned when a key does not exist in a bucket
	ErrKeyNotFound = errors.New("key not found")

	// ErrNilDB is returned when the database is nil
	ErrNilDB = errors.New("database is nil")

	// ErrEmptyKey is returned when an empty key is provided
	ErrEmptyKey = errors.New("empty key")

	// ErrEmptyBucket is returned when an empty bucket name is provided
	ErrEmptyBucket = errors.New("empty bucket name")
)

// DB is a wrapper around a BoltDB database
type DB struct {
	*bolt.DB
}

// OpenDatabase opens a BoltDB database at the specified path
func OpenDatabase(dbPath string) (*DB, error) {
	// Create directory if it doesn't exist
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}

	// Open the database
	boltDB, err := bolt.Open(dbPath, 0600, &bolt.Options{Timeout: DefaultTimeout})
	if err != nil {
		return nil, err
	}

	return &DB{boltDB}, nil
}

// DefaultDatabasePath returns the default path for the database file
func DefaultDatabasePath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}
	return filepath.Join(homeDir, DefaultDatabaseDir, DefaultDatabaseFile)
}

// CreateBucketIfNotExists creates a bucket if it doesn't exist
func (db *DB) CreateBucketIfNotExists(bucketName string) error {
	if db == nil || db.DB == nil {
		return ErrNilDB
	}
	if bucketName == "" {
		return ErrEmptyBucket
	}

	return db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(bucketName))
		return err
	})
}

// Put stores a key-value pair in the specified bucket
func (db *DB) Put(bucketName, key string, value interface{}) error {
	if db == nil || db.DB == nil {
		return ErrNilDB
	}
	if bucketName == "" {
		return ErrEmptyBucket
	}
	if key == "" {
		return ErrEmptyKey
	}

	// Convert value to JSON
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	// Store in database
	return db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketName))
		if bucket == nil {
			return ErrBucketNotFound
		}
		return bucket.Put([]byte(key), data)
	})
}

// Get retrieves a value from the specified bucket
func (db *DB) Get(bucketName, key string, value interface{}) error {
	if db == nil || db.DB == nil {
		return ErrNilDB
	}
	if bucketName == "" {
		return ErrEmptyBucket
	}
	if key == "" {
		return ErrEmptyKey
	}

	var data []byte
	err := db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketName))
		if bucket == nil {
			return ErrBucketNotFound
		}
		data = bucket.Get([]byte(key))
		if data == nil {
			return ErrKeyNotFound
		}
		return nil
	})
	if err != nil {
		return err
	}

	// Unmarshal data into value
	return json.Unmarshal(data, value)
}

// Delete removes a key-value pair from the specified bucket
func (db *DB) Delete(bucketName, key string) error {
	if db == nil || db.DB == nil {
		return ErrNilDB
	}
	if bucketName == "" {
		return ErrEmptyBucket
	}
	if key == "" {
		return ErrEmptyKey
	}

	return db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketName))
		if bucket == nil {
			return ErrBucketNotFound
		}
		return bucket.Delete([]byte(key))
	})
}

// Close closes the database
func (db *DB) Close() error {
	if db == nil || db.DB == nil {
		return ErrNilDB
	}
	return db.DB.Close()
}