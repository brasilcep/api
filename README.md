<p align="center">
  <img src="./img/logo.svg" alt="Logo" width="400"/>
</p>

<a href="https://github.com/brasilcep/brasilcep-webservice/actions/workflows/cicd.yml"><img src="https://github.com/brasilcep/brasilcep-webservice/actions/workflows/cicd.yml/badge.svg" /></a>

API REST para consulta de CEPs brasileiros, baseada na base oficial dos Correios (DNE) com propósito de hospedar no seu servidor sem necessidade de acesso a terceiros.

Projeto open source, rápido, eficiente e fácil de usar.

## Sumário
- [Sumário](#sumário)
- [Como rodar](#como-rodar)
- [Build local](#build-local)
- [Configurações via ENV](#configurações-via-env)
- [Importação da base DNE](#importação-da-base-dne)
- [Modo seed: importação da base DNE](#modo-seed-importação-da-base-dne)
  - [Estrutura de pastas esperada para a base DNE](#estrutura-de-pastas-esperada-para-a-base-dne)
- [Endpoints da API](#endpoints-da-api)
  - [`GET /cep/:cep`](#get-cepcep)
  - [`GET /healthcheck`](#get-healthcheck)
  - [`GET /debug/list?prefix=XXXXX`](#get-debuglistprefixxxxxx)
  - [`GET /debug/count`](#get-debugcount)
  - [`GET /debug/stats`](#get-debugstats)
- [Configurações da API](#configurações-da-api)
- [Licença](#licença)
- [Contribua](#contribua)

---

## Como rodar

1. Clone o repositório:
     ```sh
     git clone https://github.com/brasilcep/brasilcep-webservice.git
     cd brasilcep-webservice
     ```
2. Instale o Go (>=1.18).
3. Instale as dependências:
     ```sh
     go mod tidy
     ```
4. Execute o serviço:
     ```sh
     go run main.go
     ```
     Ou utilize o Makefile:
     ```sh
     make run
     ```

## Build local

Para compilar o binário localmente:
```sh
make build
```
O binário será gerado em `./bin/brasilcep-webservice`.

## Configurações via ENV

Todas as configurações podem ser definidas via variáveis de ambiente:

- **MODE**: Modo de operação ("listen" para API HTTP ou "seed" para popular dados do DNE na base). Padrão: `listen`
- **API_PORT**: Porta HTTP para escutar. Padrão: `8080`
  
- **API_PROMETHEUS_ENABLE**: Habilita métricas Prometheus. Padrão: `true`
  
- **API_ENABLE_GZIP**: Habilita compressão Gzip. Padrão: `true`
- **API_GZIP_COMPRESSION_LEVEL**: Nível de compressão Gzip (1-9). Padrão: `5`
  
- **API_RATE_LIMIT_ENABLE**: Habilita rate limiting. Padrão: `true`
- **API_RATE_LIMIT_MAX_ALLOWED_REQUESTS_PER_WINDOW**: Máximo de requisições por janela de tempo. Padrão: `100`
- **API_RATE_LIMIT_REQUESTS_BURST**: Burst de requisições permitidas. Padrão: `20`
- **API_RATE_LIMIT_EXPIRE_MINUTES**: Janela de tempo do rate limit (minutos). Padrão: `15`
  
- **API_CORS_ALLOW_ORIGINS**: Origens permitidas no CORS (array, ex: ["*"]). Padrão: `*`
- **API_CORS_ALLOW_METHODS**: Métodos permitidos no CORS (array). Padrão: `GET,HEAD,PUT,PATCH,POST,DELETE`
- **API_CORS_ALLOW_HEADERS**: Headers permitidos no CORS (array). Padrão: `Origin,Content-Type,Accept,Authorization`
  
- **DB_PATH**: Caminho para os arquivos do banco BadgerDB. Padrão: `./data`
- **DB_RAW_PATH**: Caminho para os arquivos originais do DNE. Padrão: `./dne`

- **LOG_FORMAT**: Formato do log ("json" ou "text"). Padrão: `json`
- **LOG_LEVEL**: Nível de log ("debug", "info", "warn", "error"). Padrão: `info`
Exemplo de uso:
```sh
export API_PORT=8081
export LOG_LEVEL=debug
```

## Importação da base DNE

Para importar a base oficial dos Correios (DNE):
1. Baixe os arquivos TXT do DNE e coloque-os na pasta definida em `DB_RAW_PATH` (por padrão, `./dne`).
2. Execute o importador:
     ```go
     import "github.com/brasilcep/brasilcep-webservice/zipcodes"
     zipImporter := zipcodes.NewZipCodeImporter(logger)
     zipImporter.PopulateZipcodes("./dne")
     ```
     Ou execute o modo seed

Arquivos necessários:
- LOG_LOCALIDADE.TXT
- LOG_BAIRRO.TXT
- LOG_LOGRADOURO_XX.TXT (para cada UF)
- LOG_GRANDE_USUARIO.TXT
- LOG_UNID_OPER.TXT
- LOG_CPC.TXT

## Modo seed: importação da base DNE

O modo `seed` serve para importar a base oficial dos Correios (DNE) para o banco local BadgerDB. Basta definir a variável de ambiente `MODE=seed` e executar o projeto:

```sh
export MODE=seed
go run main.go
```
Ou via Makefile:
```sh
MODE=seed make run
```

O caminho da base DNE pode ser configurado via `DB_RAW_PATH` (por padrão, `./dne`).

O importador lê todos os arquivos TXT necessários e popula o banco local. Após o término, o modo pode ser alterado para `listen` para servir a API normalmente.

### Estrutura de pastas esperada para a base DNE

Coloque todos os arquivos TXT do DNE na pasta definida por `DB_RAW_PATH` (por padrão, `./dne`).

Exemplo:
```
./
├── main.go
├── dne/
│   ├── LOG_LOCALIDADE.TXT
│   ├── LOG_BAIRRO.TXT
│   ├── LOG_LOGRADOURO_AC.TXT
│   ├── LOG_LOGRADOURO_AL.TXT
│   ├── LOG_LOGRADOURO_AM.TXT
│   ├── LOG_LOGRADOURO_AP.TXT
│   ├── LOG_LOGRADOURO_BA.TXT
│   ├── LOG_LOGRADOURO_CE.TXT
│   ├── LOG_LOGRADOURO_DF.TXT
│   ├── LOG_LOGRADOURO_ES.TXT
│   ├── LOG_LOGRADOURO_GO.TXT
│   ├── LOG_LOGRADOURO_MA.TXT
│   ├── LOG_LOGRADOURO_MG.TXT
│   ├── LOG_LOGRADOURO_MS.TXT
│   ├── LOG_LOGRADOURO_MT.TXT
│   ├── LOG_LOGRADOURO_PA.TXT
│   ├── LOG_LOGRADOURO_PB.TXT
│   ├── LOG_LOGRADOURO_PE.TXT
│   ├── LOG_LOGRADOURO_PI.TXT
│   ├── LOG_LOGRADOURO_PR.TXT
│   ├── LOG_LOGRADOURO_RJ.TXT
│   ├── LOG_LOGRADOURO_RN.TXT
│   ├── LOG_LOGRADOURO_RO.TXT
│   ├── LOG_LOGRADOURO_RR.TXT
│   ├── LOG_LOGRADOURO_RS.TXT
│   ├── LOG_LOGRADOURO_SC.TXT
│   ├── LOG_LOGRADOURO_SE.TXT
│   ├── LOG_LOGRADOURO_SP.TXT
│   ├── LOG_LOGRADOURO_TO.TXT
│   ├── LOG_GRANDE_USUARIO.TXT
│   ├── LOG_UNID_OPER.TXT
│   ├── LOG_CPC.TXT
│   └── LOG_LOCALIDADE.TXT
```

**Atenção:** Todos os arquivos devem estar codificados em ISO-8859-1, conforme padrão dos Correios.

Após importar, rode o projeto normalmente em modo `listen` para servir a API.

## Endpoints da API

### `GET /cep/:cep`
Consulta um CEP.
- **Exemplo:**
    ```sh
    curl http://localhost:8080/cep/01310930
    ```
- **Resposta:**
    ```json
    {
        "cep": "01310930",
        "logradouro": "Av. Paulista",
        "bairro": "Bela Vista",
        "cidade": "São Paulo",
        "uf": "SP",
        "codigo_ibge": "3550308",
        "tipo_logradouro": "Avenida",
        "tipo_origem": "logradouro",
        "nome_origem": "LOG_LOGRADOURO_SP.TXT"
    }
    ```
- **Erros:**
    - 404: CEP não encontrado
    - 400: CEP não fornecido

### `GET /healthcheck`
Verifica o status do serviço.
- **Exemplo:**
    ```sh
    curl http://localhost:8080/healthcheck
    ```
- **Resposta:**
    ```json
    {
        "status": "ok",
        "version": "dev",
        "commit": "none",
        "repo": "unknown"
    }
    ```

### `GET /debug/list?prefix=XXXXX`
Lista até 100 CEPs, opcionalmente filtrando por prefixo.
- **Exemplo:**
    ```sh
    curl "http://localhost:8080/debug/list?prefix=01310"
    ```
- **Resposta:**
    ```json
    {
        "total": 100,
        "data": [ { ... } ]
    }
    ```

### `GET /debug/count`
Conta o total de CEPs cadastrados.
- **Exemplo:**
    ```sh
    curl http://localhost:8080/debug/count
    ```
- **Resposta:**
    ```json
    {
        "total_ceps": 123456
    }
    ```

### `GET /debug/stats`
Estatísticas dos CEPs por UF.
- **Exemplo:**
    ```sh
    curl http://localhost:8080/debug/stats
    ```
- **Resposta:**
    ```json
    {
        "total_ceps": 123456,
        "por_uf": { "SP": 50000, "RJ": 30000, ... }
    }
    ```

## Configurações da API

- **Gzip:** Ative com `API_ENABLE_GZIP=true` e ajuste o nível com `API_GZIP_COMPRESSION_LEVEL`.
- **Rate Limiting:** Controle requisições com `API_RATE_LIMIT_ENABLE`, `API_RATE_LIMIT_MAX_ALLOWED_REQUESTS_PER_WINDOW`, `API_RATE_LIMIT_REQUESTS_BURST`, `API_RATE_LIMIT_EXPIRE_MINUTES`.
- **CORS:** Configure origens, métodos e headers permitidos com `API_CORS_ALLOW_ORIGINS`, `API_CORS_ALLOW_METHODS`, `API_CORS_ALLOW_HEADERS`.
- **Prometheus:** Ative métricas com `API_PROMETHEUS_ENABLE=true`.

---

## Licença
MIT

## Contribua
Pull requests são bem-vindos! Abra issues para sugestões ou problemas.