package main

import (
	"github.com/brasilcep/api/api"
	"github.com/brasilcep/api/config"
	"github.com/brasilcep/api/database"
	"github.com/brasilcep/api/logger"
	"github.com/brasilcep/api/zipcodes"
)

var (
	Version  = "dev"
	Commit   = "none"
	Repo     = "unknown"
	Compiler = "unknown"
)

func main() {
	config := config.NewConfig()

	buildInfo := api.BuildInfo{
		Version:  Version,
		Commit:   Commit,
		Repo:     Repo,
		Compiler: Compiler,
	}

	log_level := config.GetString("log.level")

	logger := logger.NewLogger(log_level)

	database.NewDatabase(config, logger)

	mode := config.GetString("mode")

	switch mode {
	case "listen":
		api := api.NewAPI(config, logger, buildInfo)
		api.Listen()
	case "seed":
		dnePath := config.GetString("db.raw.path")
		zipcodesImporter := zipcodes.NewZipCodeImporter(logger)
		zipcodesImporter.PopulateZipcodes(dnePath)
	default:
		logger.Fatal("Invalid mode specified")
	}
}
