package main

import (
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

	perrf("Usage: %s <bind address> <webpath> <dark archive path>", os.Args[0])
	perr("")
	perr("Example:")
	perrraw(fmt.Sprintf(`    %s ":8080" "https://foo.bar/subfoo"`, os.Args[0]))

	os.Exit(status)
}

func getCLI() (string, string, string) {
	if len(os.Args) < 4 {
		usage("You must specify all arguments")
	}
	if len(os.Args) > 4 {
		usage("Too many arguments")
	}

	return os.Args[1], os.Args[2], os.Args[3]
}
