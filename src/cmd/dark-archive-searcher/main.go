package main

import (
	"db"
	"indexer"

	"github.com/uoregon-libraries/gopkg/logger"
)

func main() {
	var config = getCLI()
	var dbh = db.New()
	var i = indexer.New(dbh, config)
	var err = i.Index()
	if err != nil {
		logger.Fatalf("Error trying to index files: %s", err)
	}
}
