package main

import (
	"config"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/uoregon-libraries/gopkg/wordutils"
)

var spaces = regexp.MustCompile(`\s+`)

func perrraw(s string) {
	fmt.Fprintln(os.Stderr, s)
}

func perr(s string) {
	s = strings.TrimSpace(s)
	s = spaces.ReplaceAllString(s, " ")
	perrraw(wordutils.Wrap(s, 80))
}
func perrf(s string, args ...interface{}) {
	perr(fmt.Sprintf(s, args...))
}

func usage(msg string) {
	var status = 0
	if msg != "" {
		perr(msg)
		perr("")
		status = 1
	}

	perrf("Usage: %s <settings file>", os.Args[0])

	os.Exit(status)
}

func getCLI() *config.Config {
	if len(os.Args) < 2 {
		usage("You must specify a settings file")
	}
	if len(os.Args) > 2 {
		usage("Too many arguments")
	}

	var c, err = config.Read(os.Args[1])
	if err != nil {
		perrf("Invalid configuration: %s", err)
		os.Exit(1)
	}

	return c
}
