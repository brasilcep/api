package main

import (
	"github.com/brasilcep/brasilcep-webservice/api"
	"github.com/brasilcep/brasilcep-webservice/config"
	"github.com/brasilcep/brasilcep-webservice/database"
	"github.com/brasilcep/brasilcep-webservice/logger"
	"github.com/brasilcep/brasilcep-webservice/zipcodes"
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
	case "populate":
		dnePath := config.GetString("db.raw.path")
		zipcodesImporter := zipcodes.NewZipCodeImporter(logger)
		zipcodesImporter.PopulateZipcodes(dnePath)
	default:
		logger.Fatal("Invalid mode specified")
	}
}
