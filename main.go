package main

import (
	"flag"

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

	var (
		listen   = flag.Bool("listen", true, "start the HTTP server")
		populate = flag.Bool("populate", true, "populate the database with DNE data")
		download = flag.Bool("download", false, "download DNE data from Brasil Correios (requires a valid login)")
	)

	flag.Parse()

	if *listen {
		api := api.NewAPI(config, logger)
		api.Listen()
	} else if *populate {
		zipcodesImporter := zipcodes.NewZipCodeImporter(logger)
		zipcodesImporter.PopulateZipcodes("./dne")
	} else if *download {

	}
}
