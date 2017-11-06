package main

import (
	"time"

	"github.com/uoregon-libraries/gopkg/fileutil"
)

type cacher struct {
	daPath   string
	ticker   *time.Ticker
	needStop chan bool
	sigDone  chan bool
}

func newCacher(daPath string) *cacher {
	return &cacher{daPath: daPath, needStop: make(chan bool, 1), sigDone: make(chan bool, 1)}
}

// start kicks off the ticker, refreshing the dark archive inventory list daily
func (c *cacher) start() {
	c.readInventory()
	c.ticker = time.NewTicker(time.Hour * 24)
	select {
	case <-c.ticker.C:
		c.readInventory()
	case <-c.needStop:
		c.ticker.Stop()
		c.sigDone <- true
		return
	}
}

// stop signals the cacher to stop ticking when it can
func (c *cacher) stop() {
	c.needStop <- true
}

// wait runs until the cacher has been stopped successfully
func (c *cacher) wait() {
	select {
	case _ = <-c.sigDone:
		return
	}
}

func (c *cacher) readInventory() {
	fileutil.ReaddirSorted(c.daPath)
}
