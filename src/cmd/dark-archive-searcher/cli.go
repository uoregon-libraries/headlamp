package main

import (
	"fmt"
	"indexer"
	"os"
	"path/filepath"
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
		"format" should express the path using the keywords "project", "date", and
		"ignore".  There must be exactly one occurrence of "project", designating
		which path element specifies the project name.  There must be one "date" as
		well, which tells us which folder represents the archive date (in
		YYYY-MM-DD format).  There can be any number of "ignore" elements in the
		path, each of which are simply ignored in order to form the "public" path.
		e.g., "project/ignore/date" would state that the top-level folder is the
		project name and the next two folders are collapsed, while the third is
		stored as the archive date.  "ignore/project/date" might be used for
		"Volume/project/date" style archives.
	`)
	perr("")
	perr(`
		"inv pattern" should simply be a pattern to find all the inventory files,
		such as "*/*/INVENTORY/*.csv".  The files should be discoverable by taking
		the path of the inventory file, removing the filename, adding "../" and the
		filename.  e.g., project/date/INVENTORY/foo.csv might describe
		"bar/baz.tiff", which could be found at
		project/date/INVENTORY/../bar/baz.tiff, or project/date/bar/baz.tiff.
	`)
	perr("")
	perr("Example:")
	perrraw(fmt.Sprintf(`    %s /mnt/darkarchive ignore/project/ignore "*/*/INVENTORY/*.csv"`, os.Args[0]))

	os.Exit(status)
}

func getCLI() *indexer.Config {
	var c = &indexer.Config{}
	if len(os.Args) < 4 {
		usage("You must specify all arguments")
	}
	if len(os.Args) > 4 {
		usage("Too many arguments")
	}

	c.DARoot = parseDARoot(os.Args[1])
	c.PathFormat = parsePathFormat(os.Args[2])
	c.InventoryPattern = parseInventoryPattern(os.Args[3])

	return c
}

func parseDARoot(val string) string {
	if val == "" {
		usage("You must specify a path to the dark archive")
	}
	var err error
	val, err = filepath.Abs(val)
	if err != nil {
		usage(fmt.Sprintf("%q is not a valid path: %s", val, err))
	}
	if !fileutil.IsDir(val) {
		usage(fmt.Sprintf("%q is not a valid path: not a directory", val))
	}
	return val
}

func parsePathFormat(val string) []indexer.PathToken {
	var formatParts = strings.Split(val, string(os.PathSeparator))
	var hasProject bool
	var hasDate bool
	var pathFormat []indexer.PathToken
	for _, part := range formatParts {
		switch part {
		case "ignore":
			pathFormat = append(pathFormat, indexer.Ignored)

		case "project":
			if hasProject {
				usage(fmt.Sprintf(`Invalid path format %q: "project" must be specified exactly once`, val))
			}
			hasProject = true
			pathFormat = append(pathFormat, indexer.Project)

		case "date":
			if hasDate {
				usage(fmt.Sprintf(`Invalid path format %q: "date" must be specified exactly once`, val))
			}
			hasDate = true
			pathFormat = append(pathFormat, indexer.Date)

		default:
			usage(fmt.Sprintf("Invalid path format %q: unknown keyword %q", val, part))
		}
	}
	if !hasProject {
		usage(fmt.Sprintf(`Invalid path format %q: "project" must be specified exactly once`, val))
	}
	if !hasDate {
		usage(fmt.Sprintf(`Invalid path format %q: "date" must be specified exactly once`, val))
	}

	return pathFormat
}

func parseInventoryPattern(val string) string {
	// This doesn't have any logic yet, but I think it should get some sanity
	// checking eventually
	return val
}
