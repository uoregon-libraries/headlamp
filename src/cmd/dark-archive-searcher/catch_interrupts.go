package main

import (
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"

	"github.com/uoregon-libraries/gopkg/logger"
)

var isDone int32

func catchInterrupts(quit func()) {
	var sigInt = make(chan os.Signal, 1)
	signal.Notify(sigInt, syscall.SIGINT)
	signal.Notify(sigInt, syscall.SIGTERM)
	go func() {
		for range sigInt {
			if done() {
				logger.Warnf("Force-interrupt detected; shutting down.")
				os.Exit(1)
			}

			logger.Infof("Interrupt detected; attempting to clean up.  Another signal will immediately end the process.")
			atomic.StoreInt32(&isDone, 1)
			quit()
		}
	}()
}

func done() bool {
	return atomic.LoadInt32(&isDone) == 1
}
