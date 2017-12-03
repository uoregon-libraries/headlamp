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
	"sync"
	"sync/atomic"
	"time"

	"github.com/uoregon-libraries/gopkg/logger"
)

// project wraps db.Project, extending it with a cache of the top- and
// second-level folders so we avoid the majority of DB hits without holding
// onto absurd quantites of data in cases where the folder structure is
// unusually deep
type project struct {
	*db.Project

	folders map[string]*db.Folder
}

// States an indexer can be in
const (
	iStateStopped int32 = iota
	iStateRunning
	iStateStopping
)

// Indexer controls how we find inventory files, which part(s) of the path are
// skipped, and which part of the path defines a project name.
type Indexer struct {
	sync.Mutex
	dbh *db.Database
	c   *Config

	// projects keeps a cache of all projects, keyed by the name, to avoid
	// millions of unnecessary lookups in the db
	projects map[string]*project

	// seenInventoryFiles caches the files we've processed in the past so we
	// don't hit the DB each time we're looking at a new inventory file
	seenInventoryFiles map[string]bool

	// state is set via async calls to tell us what the indexer is currently
	// doing (if anything).  This allows running an indexing operation in the
	// background and waiting for it to finish while also being able to request
	// it to stop at the next opportunity.
	state int32
}

// indexerOperation wraps an Indexer with a single operation's context so we
// can separate the overall Indexer setup / config from a single transactioned
// indexing job
type indexerOperation struct {
	*Indexer
	op *db.Operation
}

// New sets up a scanner for use in indexing dark-archive file data
func New(dbh *db.Database, conf *Config) *Indexer {
	return &Indexer{dbh: dbh, c: conf, projects: make(map[string]*project)}
}

// Index searches for inventory files not previously seen and indexes the files
// described therein
func (i *Indexer) Index() error {
	i.setState(iStateRunning)
	defer i.setState(iStateStopped)

	var files, err = i.findInventoryFiles()
	if err != nil {
		return err
	}

	err = i.dbh.InTransaction(func(op *db.Operation) error {
		var iop = &indexerOperation{i, op}
		return iop.findAlreadyIndexedInventoryFiles()
	})
	if err != nil {
		return err
	}

	for _, fname := range files {
		if i.seenInventoryFile(fname) {
			logger.Debugf("Skipping %q; already indexed this file", fname)
			continue
		}

		err = i.dbh.InTransaction(func(op *db.Operation) error {
			var iop = &indexerOperation{i, op}
			return iop.indexInventoryFile(fname)
		})
		if err != nil {
			logger.Errorf("Error processing %q: %s", fname, err)
		}

		if i.getState() == iStateStopping {
			return nil
		}
	}

	return nil
}

// Stop tells the indexer to stop running Index() when it can do so without
// data loss (in between inventory files)
func (i *Indexer) Stop() {
	var cur = i.getState()
	if cur != iStateStopped && cur != iStateStopping {
		i.setState(iStateStopping)
	}
}

// Wait returns after indexing is complete.  This will return immediately if no
// indexing operation is currently happening.
func (i *Indexer) Wait() {
	for {
		if i.getState() == iStateStopped {
			return
		}
		time.Sleep(time.Second)
	}
}

func (i *Indexer) getState() int32 {
	return atomic.LoadInt32(&i.state)
}

func (i *Indexer) setState(state int32) {
	atomic.StoreInt32(&i.state, state)
}

// findInventoryFiles gathers a list of files matching the Indexer's InventoryPattern
func (i *Indexer) findInventoryFiles() ([]string, error) {
	logger.Debugf("Searching for files matching %q (skipping manifest.csv)", i.c.InventoryPattern)
	var allFiles, err = filepath.Glob(filepath.Join(i.c.DARoot, i.c.InventoryPattern))
	if err != nil {
		return nil, err
	}
	var files []string
	for _, fname := range allFiles {
		if strings.HasSuffix(fname, "manifest.csv") {
			logger.Debugf("Skipping manifest file (%q)", fname)
			continue
		}
		files = append(files, fname)
	}

	return files, nil
}

// findAlreadyIndexedInventoryFiles caches the list of inventory files already processed
func (i *indexerOperation) findAlreadyIndexedInventoryFiles() error {
	var allInventories, err = i.op.AllInventories()

	i.Lock()
	defer i.Unlock()
	i.seenInventoryFiles = make(map[string]bool)
	for _, inv := range allInventories {
		// The database indexes everything relative to the dark archive so that the
		// mount point doesn't have to be immutable.  Pretty great, right?  But
		// that means we have to prepend the current root here....
		i.seenInventoryFiles[filepath.Join(i.c.DARoot, inv.Path)] = true
	}
	return err
}

func (i *Indexer) seenInventoryFile(fname string) bool {
	return i.seenInventoryFiles[fname]
}

type fileRecord struct {
	checksum    string
	filesize    int64
	projectName string
	archiveDate string
	fullPath    string
	publicPath  string
}

var emptyFR fileRecord

// indexInventoryFile stores the given inventory file in the database and then
// crawls through its contents to index the described archive files
func (i *indexerOperation) indexInventoryFile(fname string) error {
	var relativePath = strings.TrimLeft(strings.Replace(fname, i.c.DARoot, "", 1), "/")
	logger.Debugf("Indexing inventory file %q as %q", fname, relativePath)

	var data, err = ioutil.ReadFile(fname)
	if err != nil {
		return fmt.Errorf("unable to read inventory file %q: %s", fname, err)
	}

	var inventory = &db.Inventory{Path: relativePath}
	i.op.WriteInventory(inventory)
	var records = bytes.Split(data, []byte("\n"))
	for index, record := range records {
		var fr = i.parseFileRecord(relativePath, index, record)
		if fr == emptyFR {
			continue
		}
		i.storeFile(index, inventory, fr)
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

	// Pull the date and project name from the collapsed path elements
	var projectName, dateDir string
	for index, part := range pathParts {
		switch i.c.PathFormat[index] {
		case Project:
			projectName = part
		case Date:
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
		projectName: projectName,
		archiveDate: dateDir,
		fullPath:    fullPath,
		publicPath:  publicPath,
	}
}
