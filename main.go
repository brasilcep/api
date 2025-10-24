package main

import (
	"github.com/brasilcep/brasilcep-webservice/database"
)

func main() {

	database.InitDatabase("../data")

}

// Função auxiliar para popular o banco com dados
// Execute uma vez para carregar os CEPs
// func popularDados() {
// 	dados := []Endereco{
// 		{
// 			CEP:        "01310100",
// 			Logradouro: "Avenida Paulista",
// 			Bairro:     "Bela Vista",
// 			Cidade:     "São Paulo",
// 			UF:         "SP",
// 		},
// 		{
// 			CEP:        "20040020",
// 			Logradouro: "Praça Mauá",
// 			Bairro:     "Centro",
// 			Cidade:     "Rio de Janeiro",
// 			UF:         "RJ",
// 		},
// 		{
// 			CEP:        "30130100",
// 			Logradouro: "Avenida Afonso Pena",
// 			Bairro:     "Centro",
// 			Cidade:     "Belo Horizonte",
// 			UF:         "MG",
// 		},
// 	}

// 	err := db.Update(func(txn *badger.Txn) error {
// 		for _, end := range dados {
// 			valor, err := json.Marshal(end)
// 			if err != nil {
// 				return err
// 			}

// 			chave := []byte("cep:" + end.CEP)
// 			err = txn.Set(chave, valor)
// 			if err != nil {
// 				return err
// 			}
// 		}
// 		return nil
// 	})

// 	if err != nil {
// 		log.Printf("Erro ao popular dados: %v", err)
// 	} else {
// 		log.Println("Dados populados com sucesso!")
// 	}
// }

// // Função para importar CEPs em lote (melhor performance)
// func ImportarCEPsLote(ceps []Endereco) error {
// 	wb := db.NewWriteBatch()
// 	defer wb.Cancel()

// 	for _, end := range ceps {
// 		valor, err := json.Marshal(end)
// 		if err != nil {
// 			return err
// 		}

// 		chave := []byte("cep:" + end.CEP)
// 		if err := wb.Set(chave, valor); err != nil {
// 			return err
// 		}
// 	}

// 	return wb.Flush()
// }
