package database

import (
	"time"

	"github.com/dgraph-io/badger/v4"
)

type Endereco struct {
	CEP        string `json:"cep"`
	Logradouro string `json:"logradouro"`
	Bairro     string `json:"bairro"`
	Cidade     string `json:"cidade"`
	UF         string `json:"uf"`
}

var db *badger.DB

func InitDatabase(path string) error {
	opts := badger.DefaultOptions(path)
	opts.Logger = nil
	opts.IndexCacheSize = 100 << 20 // 100MB de cache para índices

	var err error
	db, err = badger.Open(opts)

	go runGC(db)
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

func runGC(db *badger.DB) {
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
