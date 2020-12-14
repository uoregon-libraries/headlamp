package main

import (
	"github.com/uoregon-libraries/gopkg/interrupts"
	"github.com/uoregon-libraries/headlamp/src/db"
	"github.com/uoregon-libraries/headlamp/src/indexer"
)

func main() {
	var config = getCLI()
	var dbh = db.New()
	var i = indexer.New(dbh, config)
	var runner = &runner{
		indexer:  i,
		needStop: make(chan bool, 1),
		sigDone:  make(chan bool, 1),
	}

	interrupts.TrapIntTerm(func() {
		runner.stop()
	})
	runner.run()
}
