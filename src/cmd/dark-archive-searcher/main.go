package main

import (
	"fmt"
	"os"

	"github.com/uoregon-libraries/gopkg/fileutil"
)

func perr(s string) {
	fmt.Fprintf(os.Stderr, s+"\n")
}
func perrf(s string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, s+"\n", args)
}

func usage(msg string) {
	var status = 0
	if msg != "" {
		perr(msg)
		perr("")
		status = 1
	}

	perrf("Usage: %s <path to dark archive>", os.Args[0])

	os.Exit(status)
}

func main() {
	if len(os.Args) != 2 {
		usage("You must specify a path to the dark archive")
	}

	var daPath = os.Args[1]
	if daPath == "" {
		usage("You must specify a path to the dark archive")
	}

	if !fileutil.IsDir(daPath) {
		usage(fmt.Sprintf("%q is not a valid path", daPath))
	}

	var c = newCacher(daPath)
	go c.start()

	catchInterrupts(func() {
		c.stop()
	})

	c.wait()
}
