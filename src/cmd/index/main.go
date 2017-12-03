package main

import (
	"db"
	"indexer"
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

	catchInterrupts(func() {
		runner.stop()
	})
	runner.run()
}
