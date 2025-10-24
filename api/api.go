package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/brasilcep/brasilcep-webservice/database"
	"github.com/dgraph-io/badger/v4"
)

func listen() {
	http.HandleFunc("/cep/", buscarCEP)
	http.HandleFunc("/healthcheck", health)

	log.Println("Listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func buscarCEP(w http.ResponseWriter, r *http.Request) {
	cep := strings.TrimPrefix(r.URL.Path, "/cep/")
	cep = strings.ReplaceAll(cep, "-", "")

	if cep == "" {
		http.Error(w, "CEP não informado", http.StatusBadRequest)
		return
	}

	var endereco database.Endereco

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
