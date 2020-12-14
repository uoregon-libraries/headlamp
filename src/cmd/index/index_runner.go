package main

import (
	"time"

	"github.com/uoregon-libraries/gopkg/logger"
	"github.com/uoregon-libraries/headlamp/src/indexer"
)

type runner struct {
	indexer  *indexer.Indexer
	ticker   *time.Ticker
	needStop chan bool
	sigDone  chan bool
}

// start kicks off the ticker, refreshing the dark archive inventory list regularly
func (r *runner) run() {
	r.ticker = time.NewTicker(time.Minute * 15)
	var reindex = func() {
		var err = r.indexer.Index()
		if err != nil {
			logger.Criticalf("Unable to reindex dark archive files: %s", err)
		}
	}
	go reindex()

	for {
		select {
		case <-r.ticker.C:
			go reindex()
		case <-r.needStop:
			r.ticker.Stop()
			r.indexer.Stop()
			r.indexer.Wait()
			return
		}
	}
}

// stop signals the cacher to stop ticking when it can
func (r *runner) stop() {
	r.needStop <- true
}
