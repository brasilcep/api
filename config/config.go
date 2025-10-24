package config

import (
	"strings"

	"github.com/spf13/viper"
)

func NewConfig() *viper.Viper {
	conf := viper.New()
	conf.AutomaticEnv()

	conf.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	conf.SetDefault("mode", "listen") //listen, seed

	conf.SetDefault("api.port", 8080)

	conf.SetDefault("api.prometheus.enable", true)
	conf.SetDefault("api.enable.gzip", true)

	conf.SetDefault("api.gzip.compression.level", 5)

	conf.SetDefault("api.rate_limit.enable", true)
	conf.SetDefault("api.rate_limit.max_allowed_requests_per_window", 100)
	conf.SetDefault("api.rate_limit.requests_burst", 20)
	conf.SetDefault("api.rate_limit.expire_minutes", 15)

	conf.SetDefault("api.cors.allow.origins", []string{"*"})
	conf.SetDefault("api.cors.allow.methods", []string{"GET", "HEAD", "PUT", "PATCH", "POST", "DELETE"})
	conf.SetDefault("api.cors.allow.headers", []string{"Origin", "Content-Type", "Accept", "Authorization"})

	conf.SetDefault("db.path", "./data")
	conf.SetDefault("db.raw.path", "./dne")

	conf.SetDefault("log.format", "json")
	conf.SetDefault("log.level", "info")

	// Config file
	conf.SetConfigName("config") // nome do arquivo sem extensão
	conf.SetConfigType("yaml")   // ou json, toml, etc.
	conf.AddConfigPath(".")      // onde procurar o arquivo

	if err := conf.ReadInConfig(); err != nil {
		// Se não achar o arquivo, apenas logar (não falhar)
		// fmt.Println("No config file found, using defaults and envs")
	}

	return conf
}
