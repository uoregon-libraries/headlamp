package main

import (
	"indexer"
	"time"

	"github.com/uoregon-libraries/gopkg/logger"
)

type runner struct {
	indexer  *indexer.Indexer
	ticker   *time.Ticker
	needStop chan bool
	sigDone  chan bool
}

// start kicks off the ticker, refreshing the dark archive inventory list regularly
func (r *runner) start() {
	r.ticker = time.NewTicker(time.Hour)
	var reindex = func() {
		var err = r.indexer.Index()
		if err != nil {
			logger.Criticalf("Unable to reindex dark archive files: %s", err)
		}
	}
	reindex()

	select {
	case <-r.ticker.C:
		reindex()
	case <-r.needStop:
		r.ticker.Stop()
		r.sigDone <- true
		return
	}
}

// stop signals the cacher to stop ticking when it can
func (r *runner) stop() {
	r.needStop <- true
}

// wait runs until the cacher has been stopped successfully
func (r *runner) wait() {
	select {
	case _ = <-r.sigDone:
		return
	}
}
