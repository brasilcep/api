package database

import (
	"time"

	"github.com/dgraph-io/badger/v4"
)

var db *badger.DB

func InitDatabase(path string) error {
	opts := badger.DefaultOptions(path)
	opts.Logger = nil
	// opts.Compression = badger.Snappy
	opts.IndexCacheSize = 100 << 20 // 100MB de cache para índices

	var err error
	db, err = badger.Open(opts)

	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			var count int
			var dbSize int64

			err := db.View(func(txn *badger.Txn) error {
				opts := badger.DefaultIteratorOptions
				it := txn.NewIterator(opts)
				defer it.Close()

				for it.Rewind(); it.Valid(); it.Next() {
					item := it.Item()
					count++
					dbSize += int64(item.EstimatedSize())
				}
				return nil
			})

			if err != nil {
				// Log error if needed
				continue
			}

			// Log the count and size
			println("Total CEPs:", count, "Database size (bytes):", dbSize)
		}
	}()

	go garbageCollector(db)

	return err
}

func GetDB() *badger.DB {
	return db
}

func CloseDatabase() {
	if db != nil {
		db.Close()
	}
}

func garbageCollector(db *badger.DB) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
	again:
		err := db.RunValueLogGC(0.5)
		if err == nil {
			goto again
		}
	}
}
