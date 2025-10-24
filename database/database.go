package database

import (
	"fmt"
	"time"

	"github.com/brasilcep/brasilcep-webservice/logger"
	"github.com/dgraph-io/badger/v4"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var db *badger.DB

type BadgerLogger struct {
	logger *logger.Logger
}

func (b *BadgerLogger) Errorf(format string, args ...interface{}) {
	b.logger.Error(fmt.Sprintf(format, args...))
}

func (b *BadgerLogger) Warningf(format string, args ...interface{}) {
	b.logger.Warn(fmt.Sprintf(format, args...))
}

func (b *BadgerLogger) Infof(format string, args ...interface{}) {
	b.logger.Info(fmt.Sprintf(format, args...))
}

func (b *BadgerLogger) Debugf(format string, args ...interface{}) {
	b.logger.Debug(fmt.Sprintf(format, args...))
}

func NewDatabase(conf *viper.Viper, logger *logger.Logger) error {
	path := conf.GetString("db.path")
	opts := badger.DefaultOptions(path)
	opts.Logger = &BadgerLogger{logger: logger}
	opts.Compression = 1
	opts.IndexCacheSize = 100 << 20 // 100MB de cache para índices

	logger.Info("Initializing database", zap.String("path", path))

	var err error
	db, err = badger.Open(opts)
	if err != nil {
		logger.Error("Failed to open database", zap.Error(err))
		return err
	}

	logger.Info("Database successfully opened")

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
				logger.Error("Error while iterating over database", zap.Error(err))
				continue
			}

			logger.Debug("Database statistics", zap.Int("total_ceps", count), zap.Int64("database_size_bytes", dbSize))
		}
	}()

	go func() {
		logger.Info("Starting garbage collector")
		garbageCollector(db, logger)
		logger.Info("Garbage collector stopped")
	}()

	return nil
}

func GetDB() *badger.DB {
	return db
}

func CloseDatabase() {
	if db != nil {
		db.Close()
	}
}

func garbageCollector(db *badger.DB, logger *logger.Logger) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
	again:
		err := db.RunValueLogGC(0.5)
		if err == nil {
			logger.Info("Garbage collection completed successfully")
			goto again
		}
	}
}
