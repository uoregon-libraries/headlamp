// Package indexer handles scanning and indexing dark-archive inventories and files
package indexer

import (
	"bytes"
	"db"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/uoregon-libraries/gopkg/logger"
)

// Indexer controls how we find inventory files, which part(s) of the path are
// skipped, and which part of the path defines a project name.
type Indexer struct {
	op *db.Operation
	c  *Config
}

// New sets up a scanner for use in indexing dark-archive file data
func New(op *db.Operation, conf *Config) *Indexer {
	return &Indexer{op: op, c: conf}
}

// Index searches for inventory files not previously seen and indexes the files
// described therein
func (i *Indexer) Index() error {
	var newFiles, err = i.findNewInventoryFiles()
	if err != nil {
		return err
	}

	for _, fname := range newFiles {
		err = i.indexInventoryFile(fname)
		if err != nil {
			return err
		}
	}

	return nil
}

// findNewInventoryFiles gathers a list of files matching the Indexer's
// InventoryPattern which haven't already been indexed
func (i *Indexer) findNewInventoryFiles() ([]string, error) {
	// Make note of all inventory files we've already processed
	var allInventories, err = i.op.AllInventories()
	if err != nil {
		return nil, err
	}
	var seenFile = make(map[string]bool)
	for _, inv := range allInventories {
		seenFile[inv.Path] = true
	}

	// Find all inventory files on the filesystem, and return the list of those
	// which have never been seen
	var allFiles []string
	logger.Debugf("Searching for files matching %q", i.c.InventoryPattern)
	allFiles, err = filepath.Glob(filepath.Join(i.c.DARoot, i.c.InventoryPattern))
	if err != nil {
		return nil, err
	}
	var newFiles []string
	for _, fname := range allFiles {
		if !seenFile[fname] {
			newFiles = append(newFiles, fname)
		}
	}

	return newFiles, nil
}

type fileRecord struct {
	checksum    string
	filesize    int64
	archiveDate string
	fullPath    string
	publicPath  string
}

var emptyFR fileRecord

// indexInventoryFile stores the given inventory file in the database and then
// crawls through its contents to index the described archive files
func (i *Indexer) indexInventoryFile(fname string) error {
	var data, err = ioutil.ReadFile(fname)
	if err != nil {
		return fmt.Errorf("unable to read inventory file %q: %s", fname, err)
	}

	// We know the inventory file is legit, so we store it in the database, then
	// process its contents
	var relativePath = strings.TrimLeft(strings.Replace(fname, i.c.DARoot, "", 1), "/")

	logger.Debugf("Indexing inventory file %q as %q", fname, relativePath)
	i.op.WriteInventory(&db.Inventory{Path: relativePath})
	var records = bytes.Split(data, []byte("\n"))
	for index, record := range records {
		var fr = i.parseFileRecord(relativePath, index, record)
		if fr == emptyFR {
			continue
		}
	}

	return i.op.Operation.Err()
}

// parseFileRecord gets the important pieces of the file record (from an
// inventory file), performs some validation, and returns the data
func (i *Indexer) parseFileRecord(inventoryFile string, index int, record []byte) fileRecord {
	// These helpers make handling errors and warnings a bit easier
	var logString = func(msg string, args ...interface{}) string {
		var prefix = fmt.Sprintf("Invalid record (inventory %q, record #%d): ", inventoryFile, index)
		return prefix + fmt.Sprintf(msg, args...)
	}
	var Errorf = func(msg string, args ...interface{}) fileRecord {
		logger.Errorf(logString(msg, args...))
		return emptyFR
	}
	var Warnf = func(msg string, args ...interface{}) { logger.Warnf(logString(msg, args...)) }

	// Skip the blank record at the end
	if len(record) == 0 {
		return emptyFR
	}

	// We sometimes have filenames with commas, but the sha and filesize are
	// always safe, so we just split to 3 elements
	var recParts = bytes.SplitN(record, []byte(","), 3)

	// Skip headers
	if index == 0 && bytes.Equal(recParts[0], []byte("sha256sum")) {
		return emptyFR
	}

	// We should always have exactly 3 fields
	if len(recParts) != 3 {
		return Errorf("there must be exactly 3 fields")
	}

	var filesize, err = strconv.ParseInt(string(recParts[1]), 10, 64)
	if err != nil {
		Warnf("invalid filesize value %q", recParts[1])
	}

	// The filename is relative to the inventory file's parent directory
	var relPath = string(recParts[2])
	var fullPath = filepath.Clean(filepath.Join(filepath.Dir(inventoryFile), "..", relPath))

	// Split apart the path so we get the "magic" pieces separately from the rest
	// of the path, which must reflect our "public" path
	var partCount = len(i.c.PathFormat) + 1
	var pathParts = strings.SplitN(fullPath, string(os.PathSeparator), partCount)
	if len(pathParts) != partCount {
		return Errorf("filename %q doesn't have enough parts for the format string %q", fullPath, i.c.PathFormat)
	}
	var publicPath string
	pathParts, publicPath = pathParts[:partCount-1], pathParts[partCount-1]

	// Of the magic folders, only the date is needed here
	var dateDir string
	for index, part := range pathParts {
		if i.c.PathFormat[index] == Date {
			dateDir = part
		}
	}

	// Make sure the date matches our expected format
	var timeFormat = "2006-01-02"
	_, err = time.Parse(timeFormat, dateDir)
	if err != nil {
		return Errorf("archive date directory %q must be formatted as a date (YYYY-MM-DD)", dateDir)
	}

	var checksum = string(recParts[0])
	return fileRecord{checksum: checksum,
		filesize:    filesize,
		archiveDate: dateDir,
		fullPath:    fullPath,
		publicPath:  publicPath,
	}
}
