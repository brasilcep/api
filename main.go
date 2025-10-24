package main

import (
	"github.com/brasilcep/api/api"
	"github.com/brasilcep/api/config"
	"github.com/brasilcep/api/database"
	"github.com/brasilcep/api/logger"
	"github.com/brasilcep/api/zipcodes"
)

func main() {
	config := config.NewConfig()

	log_level := config.GetString("log.level")

	logger := logger.NewLogger(log_level)

	database.NewDatabase(config, logger)

	mode := config.GetString("mode")

	switch mode {
	case "listen":
		api := api.NewAPI(config, logger)
		api.Listen()
	case "seed":
		dnePath := config.GetString("db.raw.path")
		zipcodesImporter := zipcodes.NewZipCodeImporter(logger)
		zipcodesImporter.PopulateZipcodes(dnePath)
	default:
		logger.Fatal("Invalid mode specified")
	}
}
