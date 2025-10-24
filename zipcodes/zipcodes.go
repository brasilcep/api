package zipcodes

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/brasilcep/brasilcep-webservice/database"
	"github.com/brasilcep/brasilcep-webservice/logger"
	badger "github.com/dgraph-io/badger/v4"
	"go.uber.org/zap"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
)

// Localidade (LOG_LOCALIDADE.TXT)
type Localidade struct {
	Codigo         string // LOC_NU
	UF             string // UFE_SG
	Nome           string // LOC_NO
	CEP            string // CEP
	Situacao       string // LOC_IN_SIT
	TipoLocalidade string // LOC_IN_TIPO_LOC
	CodigoSub      string // LOC_NU_SUB
	NomeAbreviado  string // LOC_NO_ABREV
	CodigoIBGE     string // MUN_NU
}

// Bairro (LOG_BAIRRO.TXT)
type Bairro struct {
	Codigo           string // BAI_NU
	UF               string // UFE_SG
	CodigoLocalidade string // LOC_NU
	Nome             string // BAI_NO
	NomeAbreviado    string // BAI_NO_ABREV
}

// Logradouro (LOG_LOGRADOURO_XX.TXT)
type Logradouro struct {
	Codigo           string // LOG_NU
	UF               string // UFE_SG
	CodigoLocalidade string // LOC_NU
	CodigoBairroIni  string // BAI_NU_INI
	CodigoBairroFim  string // BAI_NU_FIM
	Nome             string // LOG_NO
	Complemento      string // LOG_COMPLEMENTO
	CEP              string // CEP
	TipoLogradouro   string // TLO_TX
	UsaTipo          string // LOG_STA_TLO
	Abreviatura      string // LOG_NO_ABREV
}

// CPC (LOG_CPC.TXT)
type CPC struct {
	Codigo           string // CPC_NU
	UF               string // UFE_SG
	CodigoLocalidade string // LOC_NU
	Nome             string // CPC_NO
	Endereco         string // CPC_ENDERECO
	CEP              string // CEP
}

// GrandeUsuario (LOG_GRANDE_USUARIO.TXT)
type GrandeUsuario struct {
	Codigo           string // GRU_NU
	UF               string // UFE_SG
	CodigoLocalidade string // LOC_NU
	CodigoBairro     string // BAI_NU
	CodigoLogradouro string // LOG_NU
	Nome             string // GRU_NO
	Endereco         string // GRU_ENDERECO
	CEP              string // CEP
	NomeAbreviado    string // GRU_NO_ABREV
}

type CEPCompleto struct {
	CEP            string `json:"cep"`
	Logradouro     string `json:"logradouro"`
	Complemento    string `json:"complemento,omitempty"`
	Bairro         string `json:"bairro,omitempty"`
	Cidade         string `json:"cidade,omitempty"`
	UF             string `json:"uf,omitempty"`
	CodigoIBGE     string `json:"codigo_ibge,omitempty"`
	TipoLogradouro string `json:"tipo_logradouro,omitempty"`
	TipoOrigem     string `json:"tipo_origem,omitempty"`
	NomeOrigem     string `json:"nome_origem,omitempty"`
}

var (
	localities = make(map[string]*Localidade)
	districts  = make(map[string]*Bairro)
	db         *badger.DB
	seenCEPs   = make(map[string]bool)
)

type ZipCodeImporter struct {
	db     *badger.DB
	logger *logger.Logger
}

var nonDigit = regexp.MustCompile(`\D`)

func NewZipCodeImporter(logger *logger.Logger) *ZipCodeImporter {
	return &ZipCodeImporter{
		db:     database.GetDB(),
		logger: logger,
	}
}

func (i *ZipCodeImporter) PopulateZipcodes(dnePath string) {
	if dnePath == "" {
		i.logger.Error("DNE path is empty")
	}

	i.logger.Info("Starting DNE import...")
	start := time.Now()

	var err error
	db = database.GetDB()
	if db == nil {
		i.logger.Error("BadgerDB not initialized (database.GetDB() returned nil)")
	}

	i.logger.Info("Loading localities...")
	if err := i.loadLocalities(filepath.Join(dnePath, "LOG_LOCALIDADE.TXT")); err != nil {
		i.logger.Warn("Warning loading localities", zap.Error(err))
	}
	i.logger.Info("Localities loaded", zap.Int("count", len(localities)))

	i.logger.Info("Loading districts...")
	if err := i.loadDistricts(filepath.Join(dnePath, "LOG_BAIRRO.TXT")); err != nil {
		i.logger.Warn("Warning loading districts", zap.Error(err))
	}
	i.logger.Info("Districts loaded", zap.Int("count", len(districts)))

	i.logger.Info("Importing locality CEPs (general CEP)...")
	if err := i.importLocalityCEPs(); err != nil {
		i.logger.Warn("Warning while importing localities", zap.Error(err))
	}

	i.logger.Info("Importing streets by state...")
	totalStreets := 0
	for _, uf := range []string{"AC", "AL", "AP", "AM", "BA", "CE", "DF", "ES", "GO", "MA", "MT", "MS", "MG", "PA", "PB", "PR", "PE", "PI", "RJ", "RN", "RS", "RO", "RR", "SC", "SP", "SE", "TO"} {
		filePath := filepath.Join(dnePath, "LOG_LOGRADOURO_"+uf+".TXT")
		ufCount, err := i.importStreets(filePath)
		if err != nil {
			i.logger.Warn("Warning while importing streets for UF "+uf, zap.Error(err))
			continue
		}
		totalStreets += ufCount
	}
	i.logger.Info("Streets imported", zap.Int("count", totalStreets))

	i.logger.Info("Importing large users...")
	countLU, err := i.importLargeUsers(filepath.Join(dnePath, "LOG_GRANDE_USUARIO.TXT"))
	if err != nil {
		i.logger.Warn("Warning while importing large users", zap.Error(err))
	}
	i.logger.Info("Large users imported", zap.Int("count", countLU))

	i.logger.Info("Importing Operational Units (UOP)...")
	countUOP, err := i.importOperationalUnits(filepath.Join(dnePath, "LOG_UNID_OPER.TXT"))
	if err != nil {
		i.logger.Warn("Warning while importing operational units", zap.Error(err))
	}
	i.logger.Info("UOPs imported", zap.Int("count", countUOP))

	i.logger.Info("Importing CPC...")
	countCPC, err := i.importCPC(filepath.Join(dnePath, "LOG_CPC.TXT"))
	if err != nil {
		i.logger.Warn("Warning while importing CPC", zap.Error(err))
	}
	i.logger.Info("CPC imported", zap.Int("count", countCPC))

	elapsed := time.Since(start)
	i.logger.Info("Import completed", zap.Duration("duration", elapsed))
	i.logger.Info("Total CEPs imported (approx)", zap.Int("count", len(seenCEPs)))
}

func (i *ZipCodeImporter) loadLocalities(file string) error {
	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()

	decoder := transform.NewReader(f, charmap.ISO8859_1.NewDecoder())
	reader := csv.NewReader(decoder)
	reader.Comma = '@'
	reader.LazyQuotes = true

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}
		// 0 LOC_NU, 1 UFE_SG, 2 LOC_NO, 3 CEP, 4 LOC_IN_SIT, 5 LOC_IN_TIPO_LOC,
		// 6 LOC_NU_SUB, 7 LOC_NO_ABREV, 8 MUN_NU
		if len(record) < 9 {
			continue
		}
		loc := &Localidade{
			Codigo:         strings.TrimSpace(record[0]),
			UF:             strings.TrimSpace(record[1]),
			Nome:           strings.TrimSpace(record[2]),
			CEP:            i.normalizeCEP(strings.TrimSpace(record[3])),
			Situacao:       strings.TrimSpace(record[4]),
			TipoLocalidade: strings.TrimSpace(record[5]),
			CodigoSub:      strings.TrimSpace(record[6]),
			NomeAbreviado:  strings.TrimSpace(record[7]),
			CodigoIBGE:     strings.TrimSpace(record[8]),
		}
		if loc.Codigo != "" {
			localities[loc.Codigo] = loc
		}
	}
	return nil
}

func (i *ZipCodeImporter) loadDistricts(file string) error {
	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()

	decoder := transform.NewReader(f, charmap.ISO8859_1.NewDecoder())
	reader := csv.NewReader(decoder)
	reader.Comma = '@'
	reader.LazyQuotes = true

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}
		// 0 BAI_NU, 1 UFE_SG, 2 LOC_NU, 3 BAI_NO, 4 BAI_NO_ABREV
		if len(record) < 4 {
			continue
		}
		district := &Bairro{
			Codigo:           strings.TrimSpace(record[0]),
			UF:               strings.TrimSpace(record[1]),
			CodigoLocalidade: strings.TrimSpace(record[2]),
			Nome:             strings.TrimSpace(record[3]),
		}
		if len(record) >= 5 {
			district.NomeAbreviado = strings.TrimSpace(record[4])
		}
		if district.Codigo != "" {
			districts[district.Codigo] = district
		}
	}
	return nil
}

func (i *ZipCodeImporter) importLocalityCEPs() error {
	wb := db.NewWriteBatch()
	defer wb.Cancel()
	batchSize := 5000
	count := 0

	for _, loc := range localities {
		if loc.CEP == "" {
			continue
		}
		cep := i.normalizeCEP(loc.CEP)
		if cep == "" {
			continue
		}
		cepComplete := CEPCompleto{
			CEP:        cep,
			Logradouro: "",
			Bairro:     "",
			Cidade:     loc.Nome,
			UF:         loc.UF,
			CodigoIBGE: loc.CodigoIBGE,
			TipoOrigem: "localidade",
		}
		if err := i.writeCEPIfNew(wb, cep, cepComplete); err != nil {
			i.logger.Warn("Warning while writing locality CEP", zap.String("cep", cep), zap.Error(err))
			continue
		}
		count++
		if count%batchSize == 0 {
			if err := wb.Flush(); err != nil {
				i.logger.Warn("Warning on flush (localities)", zap.Error(err))
			}
			wb = db.NewWriteBatch()
			i.logger.Info("Localities processed", zap.Int("count", count))
		}
	}
	if err := wb.Flush(); err != nil {
		return err
	}
	return nil
}

func (i *ZipCodeImporter) importStreets(file string) (int, error) {
	f, err := os.Open(file)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	decoder := transform.NewReader(f, charmap.ISO8859_1.NewDecoder())
	reader := csv.NewReader(decoder)
	reader.Comma = '@'
	reader.LazyQuotes = true

	wb := db.NewWriteBatch()
	defer wb.Cancel()

	count := 0
	batchSize := 10000

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}
		// 0 LOG_NU, 1 UFE_SG, 2 LOC_NU, 3 BAI_NU_INI, 4 BAI_NU_FIM, 5 LOG_NO,
		// 6 LOG_COMPLEMENTO, 7 CEP, 8 TLO_TX, 9 LOG_STA_TLO, 10 LOG_NO_ABREV
		if len(record) < 8 {
			continue
		}
		cep := i.normalizeCEP(strings.TrimSpace(record[7]))
		if cep == "" {
			continue
		}
		streetType := ""
		if len(record) >= 9 {
			streetType = strings.TrimSpace(record[8])
		}
		useType := ""
		if len(record) >= 10 {
			useType = strings.TrimSpace(record[9])
		}
		districtCode := ""
		if len(record) >= 4 {
			districtCode = strings.TrimSpace(record[3])
		}
		localityCode := ""
		if len(record) >= 3 {
			localityCode = strings.TrimSpace(record[2])
		}
		streetName := strings.TrimSpace(record[5])
		complement := ""
		if len(record) >= 7 {
			complement = strings.TrimSpace(record[6])
		}

		districtName := ""
		if d, ok := districts[districtCode]; ok {
			districtName = d.Nome
		}

		cityName := ""
		ibgeCode := ""
		uf := ""
		if l, ok := localities[localityCode]; ok {
			cityName = l.Nome
			uf = l.UF
			ibgeCode = l.CodigoIBGE
		}

		completeStreet := streetName
		if streetType != "" && (useType == "S" || useType == "s" || useType == "") {
			completeStreet = strings.TrimSpace(streetType + " " + streetName)
		} else if streetType != "" && (useType == "N" || useType == "n") {
			completeStreet = streetName
		}

		cepComplete := CEPCompleto{
			CEP:            cep,
			Logradouro:     completeStreet,
			Complemento:    complement,
			Bairro:         districtName,
			Cidade:         cityName,
			UF:             uf,
			CodigoIBGE:     ibgeCode,
			TipoLogradouro: streetType,
			TipoOrigem:     "logradouro",
		}

		if err := i.writeCEPIfNew(wb, cep, cepComplete); err != nil {
		}

		count++
		if count%batchSize == 0 {
			if err := wb.Flush(); err != nil {
				i.logger.Warn("Warning on flush (streets)", zap.Error(err))
			}
			wb = db.NewWriteBatch()
			i.logger.Info("Processed (streets)", zap.Int("count", count))
		}
	}

	if err := wb.Flush(); err != nil {
		return count, err
	}
	return count, nil
}

func (i *ZipCodeImporter) importLargeUsers(file string) (int, error) {
	f, err := os.Open(file)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	decoder := transform.NewReader(f, charmap.ISO8859_1.NewDecoder())
	reader := csv.NewReader(decoder)
	reader.Comma = '@'
	reader.LazyQuotes = true

	wb := db.NewWriteBatch()
	defer wb.Cancel()

	count := 0
	batchSize := 5000

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}
		// 0 GRU_NU, 1 UFE_SG, 2 LOC_NU, 3 BAI_NU, 4 LOG_NU, 5 GRU_NO, 6 GRU_ENDERECO, 7 CEP, 8 GRU_NO_ABREV
		if len(record) < 8 {
			continue
		}
		cep := i.normalizeCEP(strings.TrimSpace(record[7]))
		if cep == "" {
			continue
		}
		districtCode := strings.TrimSpace(record[3])
		localityCode := strings.TrimSpace(record[2])
		largeUserName := strings.TrimSpace(record[5])
		largeUserAddress := strings.TrimSpace(record[6])

		districtName := ""
		if d, ok := districts[districtCode]; ok {
			districtName = d.Nome
		}
		cityName := ""
		uf := ""
		ibgeCode := ""
		if l, ok := localities[localityCode]; ok {
			cityName = l.Nome
			uf = l.UF
			ibgeCode = l.CodigoIBGE
		}

		cepComplete := CEPCompleto{
			CEP:         cep,
			Logradouro:  largeUserAddress,
			Complemento: "",
			Bairro:      districtName,
			Cidade:      cityName,
			UF:          uf,
			CodigoIBGE:  ibgeCode,
			TipoOrigem:  "grande_usuario",
			NomeOrigem:  largeUserName,
		}

		if err := i.writeCEPIfNew(wb, cep, cepComplete); err != nil {
		}

		count++
		if count%batchSize == 0 {
			if err := wb.Flush(); err != nil {
				i.logger.Warn("Warning on flush (large users)", zap.Error(err))
			}
			wb = db.NewWriteBatch()
			i.logger.Info("Processed (large users)", zap.Int("count", count))
		}
	}

	if err := wb.Flush(); err != nil {
		return count, err
	}
	return count, nil
}

func (i *ZipCodeImporter) importOperationalUnits(file string) (int, error) {
	f, err := os.Open(file)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	decoder := transform.NewReader(f, charmap.ISO8859_1.NewDecoder())
	reader := csv.NewReader(decoder)
	reader.Comma = '@'
	reader.LazyQuotes = true

	wb := db.NewWriteBatch()
	defer wb.Cancel()

	count := 0
	batchSize := 5000

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}
		// 0 UOP_NU, 1 UFE_SG, 2 LOC_NU, 3 BAI_NU, 4 LOG_NU, 5 UOP_NO, 6 UOP_ENDERECO, 7 CEP, 8 UOP_IN_CP, 9 UOP_NO_ABREV
		if len(record) < 8 {
			continue
		}
		cep := i.normalizeCEP(strings.TrimSpace(record[7]))
		if cep == "" {
			continue
		}
		districtCode := strings.TrimSpace(record[3])
		localityCode := strings.TrimSpace(record[2])
		uopName := strings.TrimSpace(record[5])
		uopAddress := strings.TrimSpace(record[6])

		districtName := ""
		if d, ok := districts[districtCode]; ok {
			districtName = d.Nome
		}
		cityName := ""
		uf := ""
		ibgeCode := ""
		if l, ok := localities[localityCode]; ok {
			cityName = l.Nome
			uf = l.UF
			ibgeCode = l.CodigoIBGE
		}

		cepComplete := CEPCompleto{
			CEP:         cep,
			Logradouro:  uopAddress,
			Complemento: "",
			Bairro:      districtName,
			Cidade:      cityName,
			UF:          uf,
			CodigoIBGE:  ibgeCode,
			TipoOrigem:  "unid_oper",
			NomeOrigem:  uopName,
		}

		if err := i.writeCEPIfNew(wb, cep, cepComplete); err != nil {
		}

		count++
		if count%batchSize == 0 {
			if err := wb.Flush(); err != nil {
				i.logger.Warn("Warning on flush (UOP)", zap.Error(err))
			}
			wb = db.NewWriteBatch()
			i.logger.Info("Processed (UOP)", zap.Int("count", count))
		}
	}

	if err := wb.Flush(); err != nil {
		return count, err
	}
	return count, nil
}

func (i *ZipCodeImporter) importCPC(file string) (int, error) {
	f, err := os.Open(file)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	decoder := transform.NewReader(f, charmap.ISO8859_1.NewDecoder())
	reader := csv.NewReader(decoder)
	reader.Comma = '@'
	reader.LazyQuotes = true

	wb := db.NewWriteBatch()
	defer wb.Cancel()

	count := 0
	batchSize := 5000

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}
		// 0 CPC_NU, 1 UFE_SG, 2 LOC_NU, 3 CPC_NO, 4 CPC_ENDERECO, 5 CEP
		if len(record) < 6 {
			continue
		}
		cep := i.normalizeCEP(strings.TrimSpace(record[5]))
		if cep == "" {
			continue
		}
		localityCode := strings.TrimSpace(record[2])
		cpcName := strings.TrimSpace(record[3])
		cpcAddress := strings.TrimSpace(record[4])

		cityName := ""
		uf := ""
		ibgeCode := ""
		if l, ok := localities[localityCode]; ok {
			cityName = l.Nome
			uf = l.UF
			ibgeCode = l.CodigoIBGE
		}

		cepComplete := CEPCompleto{
			CEP:         cep,
			Logradouro:  cpcAddress,
			Complemento: "",
			Bairro:      "",
			Cidade:      cityName,
			UF:          uf,
			CodigoIBGE:  ibgeCode,
			TipoOrigem:  "cpc",
			NomeOrigem:  cpcName,
		}

		if err := i.writeCEPIfNew(wb, cep, cepComplete); err != nil {
		}

		count++
		if count%batchSize == 0 {
			if err := wb.Flush(); err != nil {
				i.logger.Warn("Warning on flush (CPC)", zap.Error(err))
			}
			wb = db.NewWriteBatch()
			i.logger.Info("Processed (CPC)", zap.Int("count", count))
		}
	}

	if err := wb.Flush(); err != nil {
		return count, err
	}
	return count, nil
}

func (i *ZipCodeImporter) normalizeCEP(raw string) string {
	cep := nonDigit.ReplaceAllString(raw, "")
	if len(cep) != 8 {
		return ""
	}
	return cep
}

func (i *ZipCodeImporter) writeCEPIfNew(wb *badger.WriteBatch, cep string, data CEPCompleto) error {
	if cep == "" {
		return fmt.Errorf("empty cep")
	}
	if seenCEPs[cep] {
		return nil
	}
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	key := []byte("cep:" + cep)
	if err := wb.Set(key, jsonData); err != nil {
		return err
	}
	seenCEPs[cep] = true
	return nil
}
