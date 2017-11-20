package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/uoregon-libraries/gopkg/fileutil"
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

	perrf("Usage: %s <DA root> <format> <inv pattern>", os.Args[0])
	perr("")
	perr(`
		"DA root" should be the path to the root of the dark archive.  This will be
		stripped from all indexed data in order to avoid problems if the mount
		point to the dark archive changes.
	`)
	perr("")
	perr(`
		"format" should express the path using the keywords "project" and "ignore".
		There must be exactly one occurrence of "project", designating which path
		element specifies the project name.  There can be any number of "ignore"
		elements in the path, each of which are simply ignored in order to form the
		"public" path.  e.g., "project/ignore/ignore" would state that the
		top-level folder is the project name and the next two folders are
		irrelevant.  "ignore/project/ignore" might be used for
		"Volume/project/date" style archives.
	`)
	perr("")
	perr(`
		"inv pattern" should simply be a pattern to find all the inventory files,
		such as "*/*/INVENTORY/*.csv".
	`)
	perr("")
	perr("Example:")
	perrraw(fmt.Sprintf(`    %s /mnt/darkarchive ignore/project/ignore "*/*/INVENTORY/*.csv"`, os.Args[0]))

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
}
