package main

import (
	"flag"

	"github.com/brasilcep/brasilcep-webservice/api"
	"github.com/brasilcep/brasilcep-webservice/database"
	"github.com/brasilcep/brasilcep-webservice/zipcodes"
)

func main() {
	database.InitDatabase("./data")

	var (
		listen   = flag.Bool("listen", true, "start the HTTP server")
		populate = flag.Bool("populate", true, "populate the database with DNE data")
		download = flag.Bool("download", false, "download DNE data from Brasil Correios (requires a valid login)")
		port     = flag.Int("port", 8080, "port to listen on")
	)
	flag.Parse()

	if *listen {
		api.Listen(*port)
	} else if *populate {
		zipcodes.PopulateZipcodes("./dne")
	} else if *download {

	}
}
