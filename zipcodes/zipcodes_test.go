package zipcodes

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/brasilcep/api/logger"
	badger "github.com/dgraph-io/badger/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/text/encoding/charmap"
)

func setupTestDB(t *testing.T) (*badger.DB, func()) {
	tmpDir, err := os.MkdirTemp("", "badger-test-*")
	require.NoError(t, err)

	opts := badger.DefaultOptions(tmpDir).WithLoggingLevel(badger.ERROR)
	testDB, err := badger.Open(opts)
	require.NoError(t, err)

	cleanup := func() {
		testDB.Close()
		os.RemoveAll(tmpDir)
	}

	return testDB, cleanup
}

func setupImporter(t *testing.T) (*ZipCodeImporter, func()) {
	testDB, cleanup := setupTestDB(t)

	// Reset global state
	localities = make(map[string]*Localidade)
	districts = make(map[string]*Bairro)
	seenCEPs = make(map[string]bool)
	db = testDB

	testLogger := logger.NewLogger("info")
	importer := &ZipCodeImporter{
		db:     testDB,
		logger: testLogger,
	}

	return importer, cleanup
}

func TestNormalizeCEP(t *testing.T) {
	importer, cleanup := setupImporter(t)
	defer cleanup()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "valid CEP with hyphen",
			input:    "12345-678",
			expected: "12345678",
		},
		{
			name:     "valid CEP without hyphen",
			input:    "12345678",
			expected: "12345678",
		},
		{
			name:     "CEP with spaces",
			input:    "12 345-678",
			expected: "12345678",
		},
		{
			name:     "CEP with dots",
			input:    "12.345.678",
			expected: "12345678",
		},
		{
			name:     "invalid CEP too short",
			input:    "1234567",
			expected: "",
		},
		{
			name:     "invalid CEP too long",
			input:    "123456789",
			expected: "",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "only letters",
			input:    "abcdefgh",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := importer.normalizeCEP(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestWriteCEPIfNew(t *testing.T) {
	importer, cleanup := setupImporter(t)
	defer cleanup()

	t.Run("write new CEP successfully", func(t *testing.T) {
		seenCEPs = make(map[string]bool)
		wb := importer.db.NewWriteBatch()
		defer wb.Cancel()

		cepData := CEPCompleto{
			CEP:        "12345678",
			Logradouro: "Rua Teste",
			Bairro:     "Centro",
			Cidade:     "São Paulo",
			UF:         "SP",
			TipoOrigem: "logradouro",
		}

		err := importer.writeCEPIfNew(wb, "12345678", cepData)
		assert.NoError(t, err)
		assert.True(t, seenCEPs["12345678"])

		err = wb.Flush()
		assert.NoError(t, err)

		err = importer.db.View(func(txn *badger.Txn) error {
			item, err := txn.Get([]byte("cep:12345678"))
			if err != nil {
				return err
			}

			return item.Value(func(val []byte) error {
				var retrieved CEPCompleto
				err := json.Unmarshal(val, &retrieved)
				assert.NoError(t, err)
				assert.Equal(t, cepData.CEP, retrieved.CEP)
				assert.Equal(t, cepData.Logradouro, retrieved.Logradouro)
				assert.Equal(t, cepData.Cidade, retrieved.Cidade)
				return nil
			})
		})
		assert.NoError(t, err)
	})

	t.Run("skip already seen CEP", func(t *testing.T) {
		seenCEPs = make(map[string]bool)
		seenCEPs["99999999"] = true

		wb := importer.db.NewWriteBatch()
		defer wb.Cancel()

		cepData := CEPCompleto{
			CEP:        "99999999",
			Logradouro: "Rua Nova",
		}

		err := importer.writeCEPIfNew(wb, "99999999", cepData)
		assert.NoError(t, err)
	})

	t.Run("return error for empty CEP", func(t *testing.T) {
		seenCEPs = make(map[string]bool)
		wb := importer.db.NewWriteBatch()
		defer wb.Cancel()

		cepData := CEPCompleto{
			Logradouro: "Rua Teste",
		}

		err := importer.writeCEPIfNew(wb, "", cepData)
		assert.Error(t, err)
		assert.Equal(t, "empty cep", err.Error())
	})
}

func TestLoadLocalities(t *testing.T) {
	importer, cleanup := setupImporter(t)
	defer cleanup()

	tmpDir, err := os.MkdirTemp("", "localities-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	t.Run("load valid localities file", func(t *testing.T) {
		localities = make(map[string]*Localidade)

		localitiesFile := filepath.Join(tmpDir, "LOG_LOCALIDADE.TXT")
		content := "001@SP@São Paulo@01000-000@1@M@@SP@3550308\n" +
			"002@RJ@Rio de Janeiro@20000-000@1@M@@RJ@3304557\n" +
			"003@MG@Belo Horizonte@30000-000@1@M@@BH@3106200\n"

		// Encode content to ISO-8859-1
		encoder := charmap.ISO8859_1.NewEncoder()
		encodedContent, err := encoder.String(content)
		require.NoError(t, err)

		err = os.WriteFile(localitiesFile, []byte(encodedContent), 0644)
		require.NoError(t, err)

		err = importer.loadLocalities(localitiesFile)
		assert.NoError(t, err)
		assert.Equal(t, 3, len(localities))

		sp := localities["001"]
		assert.NotNil(t, sp)
		assert.Equal(t, "SP", sp.UF)
		assert.Equal(t, "São Paulo", sp.Nome)
		assert.Equal(t, "01000000", sp.CEP)
		assert.Equal(t, "3550308", sp.CodigoIBGE)
	})

	t.Run("skip invalid lines", func(t *testing.T) {
		localities = make(map[string]*Localidade)

		localitiesFile := filepath.Join(tmpDir, "LOG_LOCALIDADE_INVALID.TXT")
		content := "001@SP@São Paulo@01000-000@1@M@@SP@3550308\n" +
			"invalid line\n" +
			"002@RJ\n" +
			"003@MG@Belo Horizonte@30000-000@1@M@@BH@3106200\n"

		// Encode content to ISO-8859-1
		encoder := charmap.ISO8859_1.NewEncoder()
		encodedContent, err := encoder.String(content)
		require.NoError(t, err)

		err = os.WriteFile(localitiesFile, []byte(encodedContent), 0644)
		require.NoError(t, err)

		err = importer.loadLocalities(localitiesFile)
		assert.NoError(t, err)
		assert.Equal(t, 2, len(localities))
	})

	t.Run("return error for non-existent file", func(t *testing.T) {
		localities = make(map[string]*Localidade)
		err := importer.loadLocalities("/non/existent/file.txt")
		assert.Error(t, err)
	})
}

func TestLoadDistricts(t *testing.T) {
	importer, cleanup := setupImporter(t)
	defer cleanup()

	tmpDir, err := os.MkdirTemp("", "districts-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	t.Run("load valid districts file", func(t *testing.T) {
		districts = make(map[string]*Bairro)

		districtsFile := filepath.Join(tmpDir, "LOG_BAIRRO.TXT")
		content := "001@SP@001@Centro@Ctr\n" +
			"002@SP@001@Jardins@Jds\n" +
			"003@RJ@002@Copacabana@Copa\n"

		err := os.WriteFile(districtsFile, []byte(content), 0644)
		require.NoError(t, err)

		err = importer.loadDistricts(districtsFile)
		assert.NoError(t, err)
		assert.Equal(t, 3, len(districts))

		centro := districts["001"]
		assert.NotNil(t, centro)
		assert.Equal(t, "SP", centro.UF)
		assert.Equal(t, "001", centro.CodigoLocalidade)
		assert.Equal(t, "Centro", centro.Nome)
		assert.Equal(t, "Ctr", centro.NomeAbreviado)
	})

	t.Run("skip invalid lines", func(t *testing.T) {
		districts = make(map[string]*Bairro)

		districtsFile := filepath.Join(tmpDir, "LOG_BAIRRO_INVALID.TXT")
		content := "001@SP@001@Centro@Ctr\n" +
			"invalid\n" +
			"002@SP\n"

		err := os.WriteFile(districtsFile, []byte(content), 0644)
		require.NoError(t, err)

		err = importer.loadDistricts(districtsFile)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(districts))
	})

	t.Run("return error for non-existent file", func(t *testing.T) {
		districts = make(map[string]*Bairro)
		err := importer.loadDistricts("/non/existent/file.txt")
		assert.Error(t, err)
	})
}

func TestImportLocalityCEPs(t *testing.T) {
	importer, cleanup := setupImporter(t)
	defer cleanup()

	t.Run("import locality CEPs successfully", func(t *testing.T) {
		localities = make(map[string]*Localidade)
		seenCEPs = make(map[string]bool)

		localities["001"] = &Localidade{
			Codigo:     "001",
			UF:         "SP",
			Nome:       "São Paulo",
			CEP:        "01000000",
			CodigoIBGE: "3550308",
		}
		localities["002"] = &Localidade{
			Codigo:     "002",
			UF:         "RJ",
			Nome:       "Rio de Janeiro",
			CEP:        "20000000",
			CodigoIBGE: "3304557",
		}

		err := importer.importLocalityCEPs()
		assert.NoError(t, err)
		assert.Equal(t, 2, len(seenCEPs))

		err = importer.db.View(func(txn *badger.Txn) error {
			item, err := txn.Get([]byte("cep:01000000"))
			if err != nil {
				return err
			}

			return item.Value(func(val []byte) error {
				var cep CEPCompleto
				err := json.Unmarshal(val, &cep)
				assert.NoError(t, err)
				assert.Equal(t, "01000000", cep.CEP)
				assert.Equal(t, "São Paulo", cep.Cidade)
				assert.Equal(t, "SP", cep.UF)
				assert.Equal(t, "localidade", cep.TipoOrigem)
				assert.Empty(t, cep.Logradouro)
				return nil
			})
		})
		assert.NoError(t, err)
	})

	t.Run("skip localities without CEP", func(t *testing.T) {
		localities = make(map[string]*Localidade)
		seenCEPs = make(map[string]bool)

		localities["001"] = &Localidade{
			Codigo: "001",
			UF:     "SP",
			Nome:   "São Paulo",
			CEP:    "",
		}

		err := importer.importLocalityCEPs()
		assert.NoError(t, err)
		assert.Equal(t, 0, len(seenCEPs))
	})
}

func TestImportStreets(t *testing.T) {
	importer, cleanup := setupImporter(t)
	defer cleanup()

	tmpDir, err := os.MkdirTemp("", "streets-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	t.Run("import streets with type usage", func(t *testing.T) {
		localities = make(map[string]*Localidade)
		districts = make(map[string]*Bairro)
		seenCEPs = make(map[string]bool)

		localities["001"] = &Localidade{
			Codigo:     "001",
			UF:         "SP",
			Nome:       "São Paulo",
			CodigoIBGE: "3550308",
		}
		districts["001"] = &Bairro{
			Codigo: "001",
			Nome:   "Centro",
		}

		streetsFile := filepath.Join(tmpDir, "LOG_LOGRADOURO_SP.TXT")
		content := "001@SP@001@001@001@Paulista@apto 10@01310-100@Avenida@S@Av Paulista\n"

		err := os.WriteFile(streetsFile, []byte(content), 0644)
		require.NoError(t, err)

		count, err := importer.importStreets(streetsFile)
		assert.NoError(t, err)
		assert.Equal(t, 1, count)

		err = importer.db.View(func(txn *badger.Txn) error {
			item, err := txn.Get([]byte("cep:01310100"))
			if err != nil {
				return err
			}

			return item.Value(func(val []byte) error {
				var cep CEPCompleto
				err := json.Unmarshal(val, &cep)
				assert.NoError(t, err)
				assert.Equal(t, "01310100", cep.CEP)
				assert.Equal(t, "Avenida Paulista", cep.Logradouro)
				assert.Equal(t, "apto 10", cep.Complemento)
				assert.Equal(t, "Centro", cep.Bairro)
				assert.Equal(t, "São Paulo", cep.Cidade)
				assert.Equal(t, "Avenida", cep.TipoLogradouro)
				assert.Equal(t, "logradouro", cep.TipoOrigem)
				return nil
			})
		})
		assert.NoError(t, err)
	})

	t.Run("import streets without type usage", func(t *testing.T) {
		localities = make(map[string]*Localidade)
		districts = make(map[string]*Bairro)
		seenCEPs = make(map[string]bool)

		localities["001"] = &Localidade{
			Codigo: "001",
			UF:     "SP",
			Nome:   "São Paulo",
		}

		streetsFile := filepath.Join(tmpDir, "LOG_LOGRADOURO_NO_TYPE.TXT")
		content := "002@SP@001@@@XV de Novembro@@01013-001@Rua@N@R XV Nov\n"

		err := os.WriteFile(streetsFile, []byte(content), 0644)
		require.NoError(t, err)

		count, err := importer.importStreets(streetsFile)
		assert.NoError(t, err)
		assert.Equal(t, 1, count)

		err = importer.db.View(func(txn *badger.Txn) error {
			item, err := txn.Get([]byte("cep:01013001"))
			if err != nil {
				return err
			}

			return item.Value(func(val []byte) error {
				var cep CEPCompleto
				err := json.Unmarshal(val, &cep)
				assert.NoError(t, err)
				assert.Equal(t, "XV de Novembro", cep.Logradouro)
				return nil
			})
		})
		assert.NoError(t, err)
	})

	t.Run("skip streets without CEP", func(t *testing.T) {
		seenCEPs = make(map[string]bool)

		streetsFile := filepath.Join(tmpDir, "LOG_LOGRADOURO_NO_CEP.TXT")
		content := "003@SP@001@@@Teste Rua@@@@Rua@S@\n"

		err := os.WriteFile(streetsFile, []byte(content), 0644)
		require.NoError(t, err)

		count, err := importer.importStreets(streetsFile)
		assert.NoError(t, err)
		assert.Equal(t, 0, count)
	})
}

func TestImportLargeUsers(t *testing.T) {
	importer, cleanup := setupImporter(t)
	defer cleanup()

	tmpDir, err := os.MkdirTemp("", "largeusers-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	t.Run("import large users successfully", func(t *testing.T) {
		localities = make(map[string]*Localidade)
		districts = make(map[string]*Bairro)
		seenCEPs = make(map[string]bool)

		localities["001"] = &Localidade{
			Codigo:     "001",
			UF:         "SP",
			Nome:       "São Paulo",
			CodigoIBGE: "3550308",
		}
		districts["001"] = &Bairro{
			Codigo: "001",
			Nome:   "Centro",
		}

		largeUsersFile := filepath.Join(tmpDir, "LOG_GRANDE_USUARIO.TXT")
		content := "001@SP@001@001@@Empresa XYZ@Rua da Empresa 100@01234-567@Emp XYZ\n"

		err := os.WriteFile(largeUsersFile, []byte(content), 0644)
		require.NoError(t, err)

		count, err := importer.importLargeUsers(largeUsersFile)
		assert.NoError(t, err)
		assert.Equal(t, 1, count)

		err = importer.db.View(func(txn *badger.Txn) error {
			item, err := txn.Get([]byte("cep:01234567"))
			if err != nil {
				return err
			}

			return item.Value(func(val []byte) error {
				var cep CEPCompleto
				err := json.Unmarshal(val, &cep)
				assert.NoError(t, err)
				assert.Equal(t, "01234567", cep.CEP)
				assert.Equal(t, "Rua da Empresa 100", cep.Logradouro)
				assert.Equal(t, "Centro", cep.Bairro)
				assert.Equal(t, "grande_usuario", cep.TipoOrigem)
				assert.Equal(t, "Empresa XYZ", cep.NomeOrigem)
				return nil
			})
		})
		assert.NoError(t, err)
	})
}

func TestImportOperationalUnits(t *testing.T) {
	importer, cleanup := setupImporter(t)
	defer cleanup()

	tmpDir, err := os.MkdirTemp("", "uop-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	t.Run("import operational units successfully", func(t *testing.T) {
		localities = make(map[string]*Localidade)
		districts = make(map[string]*Bairro)
		seenCEPs = make(map[string]bool)

		localities["001"] = &Localidade{
			Codigo:     "001",
			UF:         "RJ",
			Nome:       "Rio de Janeiro",
			CodigoIBGE: "3304557",
		}
		districts["002"] = &Bairro{
			Codigo: "002",
			Nome:   "Copacabana",
		}

		uopFile := filepath.Join(tmpDir, "LOG_UNID_OPER.TXT")
		content := "001@RJ@001@002@@Agência Central@Av Atlântica 1000@22070-000@CP@Ag Ctr\n"

		// Encode content to ISO-8859-1
		encoder := charmap.ISO8859_1.NewEncoder()
		encodedContent, err := encoder.String(content)
		require.NoError(t, err)

		err = os.WriteFile(uopFile, []byte(encodedContent), 0644)
		require.NoError(t, err)

		count, err := importer.importOperationalUnits(uopFile)
		assert.NoError(t, err)
		assert.Equal(t, 1, count)

		err = importer.db.View(func(txn *badger.Txn) error {
			item, err := txn.Get([]byte("cep:22070000"))
			if err != nil {
				return err
			}

			return item.Value(func(val []byte) error {
				var cep CEPCompleto
				err := json.Unmarshal(val, &cep)
				assert.NoError(t, err)
				assert.Equal(t, "22070000", cep.CEP)
				assert.Equal(t, "Av Atlântica 1000", cep.Logradouro)
				assert.Equal(t, "Copacabana", cep.Bairro)
				assert.Equal(t, "unid_oper", cep.TipoOrigem)
				assert.Equal(t, "Agência Central", cep.NomeOrigem)
				return nil
			})
		})
		assert.NoError(t, err)
	})
}

func TestImportCPC(t *testing.T) {
	importer, cleanup := setupImporter(t)
	defer cleanup()

	tmpDir, err := os.MkdirTemp("", "cpc-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	t.Run("import CPC successfully", func(t *testing.T) {
		localities = make(map[string]*Localidade)
		seenCEPs = make(map[string]bool)

		localities["001"] = &Localidade{
			Codigo:     "001",
			UF:         "MG",
			Nome:       "Belo Horizonte",
			CodigoIBGE: "3106200",
		}

		cpcFile := filepath.Join(tmpDir, "LOG_CPC.TXT")
		content := "001@MG@001@CPC Savassi@Rua Pernambuco 1000@30130-150\n"

		err := os.WriteFile(cpcFile, []byte(content), 0644)
		require.NoError(t, err)

		count, err := importer.importCPC(cpcFile)
		assert.NoError(t, err)
		assert.Equal(t, 1, count)

		err = importer.db.View(func(txn *badger.Txn) error {
			item, err := txn.Get([]byte("cep:30130150"))
			if err != nil {
				return err
			}

			return item.Value(func(val []byte) error {
				var cep CEPCompleto
				err := json.Unmarshal(val, &cep)
				assert.NoError(t, err)
				assert.Equal(t, "30130150", cep.CEP)
				assert.Equal(t, "Rua Pernambuco 1000", cep.Logradouro)
				assert.Equal(t, "Belo Horizonte", cep.Cidade)
				assert.Equal(t, "cpc", cep.TipoOrigem)
				assert.Equal(t, "CPC Savassi", cep.NomeOrigem)
				assert.Empty(t, cep.Bairro)
				return nil
			})
		})
		assert.NoError(t, err)
	})
}
