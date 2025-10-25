package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/brasilcep/api/database"
	"github.com/brasilcep/api/logger"
	"github.com/brasilcep/api/zipcodes"
	"github.com/dgraph-io/badger/v4"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

var (
	Version  = "dev"
	Commit   = "none"
	Repo     = "unknown"
	Compiler = "unknown"
)

type API struct {
	config *viper.Viper
	logger *logger.Logger
	echo   *echo.Echo
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func NewAPI(config *viper.Viper, logger *logger.Logger) *API {
	return &API{
		config: config,
		logger: logger,
		echo:   echo.New(),
	}
}

func (api *API) Listen() {
	e := api.echo

	e.Pre(middleware.RemoveTrailingSlash())

	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			defer func() {
				if r := recover(); r != nil {
					api.logger.Error("Recovered from panic", zap.Any("error", r))
					c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal Server Error"})
				}
			}()
			return next(c)
		}
	})

	e.HideBanner = true

	enableGzip := api.config.GetBool("api.enable.gzip")
	if enableGzip {
		gzipCompressionLevel := api.config.GetInt("api.gzip.compression.level")
		api.logger.Debug("Gzip compression enabled")
		e.Use(middleware.GzipWithConfig(middleware.GzipConfig{
			Level: gzipCompressionLevel,
		}))
	} else {
		api.logger.Debug("Gzip compression disabled")
	}

	e.Use(middleware.Recover())
	e.Use(middleware.RequestID())

	enableRateLimit := api.config.GetBool("api.rate_limit.enable")

	if enableRateLimit {
		api.logger.Debug("Rate limiting enabled")
		rateLimit := api.config.GetInt("api.rate_limit.max_allowed_requests_per_window")
		rateLimit_burst := api.config.GetInt("api.rate_limit.requests_burst")
		ratelimit_expire := api.config.GetDuration("api.rate_limit.expire_minutes")

		config := middleware.RateLimiterConfig{
			Skipper: middleware.DefaultSkipper,
			Store: middleware.NewRateLimiterMemoryStoreWithConfig(
				middleware.RateLimiterMemoryStoreConfig{Rate: rate.Limit(rateLimit), Burst: rateLimit_burst, ExpiresIn: ratelimit_expire * time.Minute},
			),
			IdentifierExtractor: func(ctx echo.Context) (string, error) {
				id := ctx.RealIP()
				return id, nil
			},
			ErrorHandler: func(context echo.Context, err error) error {
				return context.JSON(http.StatusForbidden, nil)
			},
			DenyHandler: func(context echo.Context, identifier string, err error) error {
				return context.JSON(http.StatusTooManyRequests, nil)
			},
		}

		e.Use(middleware.RateLimiterWithConfig(config))
	}

	enablePrometheus := api.config.GetBool("api.prometheus.enable")
	if enablePrometheus {
		api.logger.Debug("Prometheus metrics enabled")
		api.EnablePrometheus()
	} else {
		api.logger.Debug("Prometheus metrics disabled")
	}

	corsAllowOrigins := api.config.GetStringSlice("api.cors.allow.origins")
	corsAllowMethods := api.config.GetStringSlice("api.cors.allow.methods")
	corsAllowHeaders := api.config.GetStringSlice("api.cors.allow.headers")

	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: corsAllowOrigins,
		AllowHeaders: corsAllowHeaders,
		AllowMethods: corsAllowMethods,
	}))

	e.GET("/cep/:cep", api.findZipcode)
	e.GET("/healthcheck", api.health)

	if api.logger.Level() <= zap.DebugLevel {
		e.GET("/debug/list", api.list)
		e.GET("/debug/count", api.count)
		e.GET("/debug/stats", api.stats)
	}

	if api.logger.Level() <= zap.InfoLevel {
		api.Motd()
	}

	fmt.Printf("Brasil CEP Webservice - Brazilian Zip Code API")
	fmt.Printf("version: %s\n", Version)
	fmt.Printf("commit: %s\n", Commit)
	fmt.Printf("repo: %s\n", Repo)
	fmt.Printf("compiler: %s\n", Compiler)
	fmt.Printf("\n")

	port := api.config.GetInt("api.port")

	api.logger.Info("Starting HTTP server", zap.Int("port", port))
	api.logger.Info(fmt.Sprintf("Listening on :%d", port))

	e.Logger.Fatal(e.Start(fmt.Sprintf(":%d", port)))
}

func (api *API) findZipcode(c echo.Context) error {
	cep := c.Param("cep")
	cep = strings.ReplaceAll(cep, "-", "")

	if cep == "" {
		return c.JSON(http.StatusBadRequest, ErrorResponse{Error: "CEP não fornecido"})
	}

	c.Response().Header().Set("X-Served-From", "Brasil CEP API")

	var endereco zipcodes.CEPCompleto

	db := database.GetDB()

	if db == nil {
		return c.JSON(http.StatusServiceUnavailable, ErrorResponse{Error: "database not ready"})
	}

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
		return c.JSON(http.StatusNotFound, ErrorResponse{Error: "CEP não encontrado"})
	}

	if err != nil {
		api.logger.Error("Erro ao buscar CEP", zap.String("cep", cep), zap.Error(err))
		return c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Erro ao buscar CEP"})
	}

	return c.JSON(http.StatusOK, endereco)
}

func (api *API) health(c echo.Context) error {
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

	return c.JSON(http.StatusOK, resp)
}

func (api *API) list(c echo.Context) error {
	limit := 100                     // Limite por página
	prefix := c.QueryParam("prefix") // Ex: ?prefix=01310

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
		log.Printf("Erro: %v", err)
		return c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Erro ao listar CEPs"})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"total": len(ceps),
		"data":  ceps,
	})
}

func (api *API) count(c echo.Context) error {
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
		return c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Erro ao contar CEPs"})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"total_ceps": count,
	})
}

func (api *API) stats(c echo.Context) error {
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
		return c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Erro ao gerar estatísticas"})
	}

	stats["total_ceps"] = totalCEPs
	stats["por_uf"] = ufs

	return c.JSON(http.StatusOK, stats)
}
