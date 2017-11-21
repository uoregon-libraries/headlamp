package main

import (
	"db"
	"indexer"

	"github.com/uoregon-libraries/gopkg/logger"
)

func main() {
	var config = getCLI()
	var dbh = db.New()
	var err = dbh.InTransaction(func(op *db.Operation) {
		var i = indexer.New(op, config)
		i.Index()
	})
	if err != nil {
		logger.Fatalf("Error trying to index files: %s", err)
	}
}
