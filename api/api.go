package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/brasilcep/brasilcep-webservice/database"
	"github.com/brasilcep/brasilcep-webservice/zipcodes"
	"github.com/dgraph-io/badger/v4"
)

func Listen(port int) {
	http.HandleFunc("/cep/", buscarCEP)
	http.HandleFunc("/healthcheck", health)

	http.HandleFunc("/debug/list", list)
	http.HandleFunc("/debug/count", count)
	http.HandleFunc("/debug/stats", stats)

	log.Println(fmt.Sprintf("Listening on port :%d", port))
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}

func buscarCEP(w http.ResponseWriter, r *http.Request) {
	cep := strings.TrimPrefix(r.URL.Path, "/cep/")
	cep = strings.ReplaceAll(cep, "-", "")

	if cep == "" {
		http.Error(w, "CEP não informado", http.StatusBadRequest)
		return
	}

	var endereco zipcodes.CEPCompleto

	db := database.GetDB()

	err := db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("cep:" + cep))
		if err != nil {
			return err
		}

		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, &endereco)
		})
	})

	if err == badger.ErrKeyNotFound {
		http.Error(w, "CEP não encontrado", http.StatusNotFound)
		return
	}

	if err != nil {
		http.Error(w, "Erro ao buscar CEP", http.StatusInternalServerError)
		log.Printf("Erro: %v", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(endereco)
}

var (
	Version = "dev"
	Commit  = "none"
	Repo    = "unknown"
)

func health(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)

	w.Header().Set("Content-Type", "application/json")

	resp := struct {
		Status  string `json:"status"`
		Version string `json:"version"`
		Commit  string `json:"commit"`
		Repo    string `json:"repo"`
	}{
		Status:  "ok",
		Version: Version,
		Commit:  Commit,
		Repo:    Repo,
	}

	json.NewEncoder(w).Encode(resp)
}

func list(w http.ResponseWriter, r *http.Request) {
	limit := 100                          // Limite por página
	prefix := r.URL.Query().Get("prefix") // Ex: ?prefix=01310

	db := database.GetDB()

	var ceps []map[string]interface{}

	err := db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 10
		it := txn.NewIterator(opts)
		defer it.Close()

		searchPrefix := []byte("cep:")
		if prefix != "" {
			searchPrefix = []byte("cep:" + prefix)
		}

		count := 0
		for it.Seek(searchPrefix); it.ValidForPrefix(searchPrefix) && count < limit; it.Next() {
			item := it.Item()
			key := string(item.Key())

			err := item.Value(func(val []byte) error {
				var data map[string]interface{}
				if err := json.Unmarshal(val, &data); err != nil {
					return err
				}
				data["_key"] = strings.TrimPrefix(key, "cep:")
				ceps = append(ceps, data)
				return nil
			})

			if err != nil {
				log.Printf("Erro ao ler valor: %v", err)
			}

			count++
		}
		return nil
	})

	if err != nil {
		http.Error(w, "Erro ao listar CEPs", http.StatusInternalServerError)
		log.Printf("Erro: %v", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"total": len(ceps),
		"data":  ceps,
	})
}

func count(w http.ResponseWriter, r *http.Request) {
	db := database.GetDB()

	count := 0

	err := db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false // Não precisa dos valores, só contar
		it := txn.NewIterator(opts)
		defer it.Close()

		prefix := []byte("cep:")
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			count++
		}
		return nil
	})

	if err != nil {
		http.Error(w, "Erro ao contar CEPs", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"total_ceps": count,
	})
}

func stats(w http.ResponseWriter, r *http.Request) {
	db := database.GetDB()

	stats := make(map[string]interface{})

	// Estatísticas por UF
	ufs := make(map[string]int)
	totalCEPs := 0

	err := db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		it := txn.NewIterator(opts)
		defer it.Close()

		prefix := []byte("cep:")
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()

			err := item.Value(func(val []byte) error {
				var cep zipcodes.CEPCompleto
				if err := json.Unmarshal(val, &cep); err != nil {
					return err
				}
				ufs[cep.UF]++
				totalCEPs++
				return nil
			})

			if err != nil {
				log.Printf("Erro: %v", err)
			}
		}
		return nil
	})

	if err != nil {
		http.Error(w, "Erro ao gerar estatísticas", http.StatusInternalServerError)
		return
	}

	stats["total_ceps"] = totalCEPs
	stats["por_uf"] = ufs

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}
