package main

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

	"github.com/uoregon-libraries/gopkg/fileutil"
	"github.com/uoregon-libraries/gopkg/logger"
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

// start kicks off the ticker, refreshing the dark archive inventory list regularly
func (c *cacher) start() {
	c.ticker = time.NewTicker(time.Hour)
	c.RefreshData()

	select {
	case <-c.ticker.C:
		c.RefreshData()
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

// RefreshData reads new inventory manifest files and indexes them
func (c *cacher) RefreshData() {
	var database = db.New()
	database.InTransaction(func(op *db.Operation) {
		c.refreshData(op, false)
	})
}

// RebuildInventory reads all the inventory manifest files and indexes them
func (c *cacher) RebuildInventory() {
	var database = db.New()
	database.InTransaction(func(op *db.Operation) {
		c.refreshData(op, true)
	})
}

// isInventoryCSV returns true if the given item is a file with the .csv extension, but
// *not* manifest.csv, since than's an aggregate file
func isInventoryCSV(i os.FileInfo) bool {
	return strings.HasSuffix(i.Name(), ".csv") && i.Name() != "manifest.csv"
}

func (c *cacher) refreshData(op *db.Operation, fullRebuild bool) {
	var err error
	logger.Infof("Refreshing inventory")

	if fullRebuild {
		err = op.DeleteAll()
		if err != nil {
			logger.Criticalf("Cannot clear database: %s", err)
			return
		}
	}

	var pinfos []os.FileInfo
	pinfos, err = fileutil.ReaddirSorted(c.daPath)
	if err != nil {
		logger.Criticalf("Error trying to read the project directory list: %s", err)
		return
	}

	for _, pinfo := range pinfos {
		var pName = pinfo.Name()

		var csvFiles, err = fileutil.FindIf(filepath.Join(c.daPath, pName, "INVENTORY"), isInventoryCSV)
		if err != nil {
			logger.Errorf("Unable to scan for CSV files in %q: %s", pName, err)
			continue
		}

		var project *db.Project
		project, err = op.FindOrCreateProject(pName)
		if err != nil {
			logger.Criticalf("Unable to store project %q: %s", pName, err)
			return
		}

		for _, csvFilename := range csvFiles {
			var indexed, err = project.HasIndexedInventoryFile(csvFilename)
			if err != nil {
				logger.Criticalf("Unable to look for inventory file %q in project %q: %s", csvFilename, pName, err)
				return
			}
			if indexed {
				logger.Debugf("Skipping inventory file %q: already processed", csvFilename)
				continue
			}

			// Store the inventory.  This is okay to do even before storing all the
			// files, since the whole operation is transactioned.
			var inventory = &db.Inventory{
				ProjectID: project.ID,
				Project:   project,
				Filename:  csvFilename,
			}
			op.Inventories.Save(inventory)

			logger.Debugf("Reading inventory for %q (inventory path: %q)", pName, csvFilename)

			if !fileutil.IsFile(csvFilename) {
				logger.Errorf("Skipping scan of %q: unreadable manifest file %q", pName, csvFilename)
				continue
			}

			var data []byte
			data, err = ioutil.ReadFile(csvFilename)
			if err != nil {
				logger.Errorf("Skipping scan of %q: error reading %q: %s", pName, csvFilename, err)
				continue
			}
			for i, record := range bytes.Split(data, []byte("\n")) {
				// Get the details of the record, skipping if it's invalid
				var fr = parseRecord(inventory.Filename, i, record)
				if fr == emptyFR {
					continue
				}

				// Build folder structure for easier lookups
				var folders = strings.Split(fr.filename, string(os.PathSeparator))
				folders = folders[:len(folders)-1]
				var fullPath string
				var parentFolder *db.Folder
				for _, fName := range folders {
					fullPath = filepath.Join(fullPath, fName)
					parentFolder, err = db.FindOrCreateFolder(project, parentFolder, fullPath)
					if err != nil {
						logger.Criticalf("Database error trying to build folder %q: %s", fullPath, err)
						return
					}
				}

				var f = buildFile(inventory, parentFolder, fr)
				var indexed, err = project.HasIndexedFile(f)
				if err != nil {
					logger.Criticalf("Unable to look for file %q in project %q: %s", f, pName, err)
					return
				}
				if indexed {
					logger.Errorf("Invalid record (inventory %q, record #%d): another record "+
						"with the same project id (%d), archive date (%q), and path (%q) exists",
						inventory.Filename, i, project.ID, f.ArchiveDate, f.Path)
					continue
				}
				op.Files.Save(f)
				if op.Operation.Err() != nil {
					logger.Criticalf("Database error trying to store file (%#v): %s", f, op.Operation.Err())
					return
				}
			}
		}
	}

	logger.Infof("Inventory refreshed")
}

type fileRecord struct {
	checksum    string
	filesize    int64
	archiveDate time.Time
	filename    string
}

var emptyFR fileRecord

// parseRecord converts a slice of bytes (presumably from a CSV file) into our
// internal fileRecord structure for use when creating files and folders
func parseRecord(inventoryFile string, recordNum int, record []byte) fileRecord {
	// Skip the trailing newline (or any blank line, really)
	if len(record) == 0 {
		return emptyFR
	}

	// We sometimes have filenames with commas, but the sha and filesize are
	// always safe, so we just split to 3 elements
	var recParts = bytes.SplitN(record, []byte(","), 3)

	// Skip headers
	if recordNum == 0 && bytes.Equal(recParts[0], []byte("sha256sum")) {
		return emptyFR
	}

	// These helpers make handling errors and warnings a bit easier
	var logString = func(msg string, args ...interface{}) string {
		var prefix = fmt.Sprintf("Invalid record (inventory %q, record #%d): ", inventoryFile, recordNum)
		return prefix + fmt.Sprintf(msg, args...)
	}
	var Errorf = func(msg string, args ...interface{}) fileRecord {
		logger.Errorf(logString(msg, args...))
		return emptyFR
	}
	var Warnf = func(msg string, args ...interface{}) { logger.Warnf(logString(msg, args...)) }

	// We should always have exactly 3 records
	if len(recParts) != 3 {
		return Errorf("there must be exactly 3 fields")
	}

	var filesize, err = strconv.ParseInt(string(recParts[1]), 10, 64)
	if err != nil {
		Warnf("invalid filesize value %q", recParts[1])
	}

	var filename = string(recParts[2])
	var pathParts = strings.SplitN(filename, string(os.PathSeparator), 2)
	var relPath = pathParts[1]
	if len(pathParts) != 2 {
		return Errorf("no top-level date directory in filename %q", filename)
	}

	var timeFormat = "2006-01-02"
	var dateDir = pathParts[0]
	var dt time.Time
	dt, err = time.Parse(timeFormat, dateDir)
	if err != nil {
		return Errorf("top-level directory %q must be formatted as a date (YYYY-MM-DD)", dateDir)
	}

	var checksum = string(recParts[0])
	return fileRecord{checksum: checksum, filesize: filesize, archiveDate: dt, filename: relPath}
}

// buildFile cobbles together the inventory, folder, and record data to return
// a ready-to-save db.File
func buildFile(i *db.Inventory, folder *db.Folder, record fileRecord) *db.File {
	var f = folder
	var fid = 0
	if f != nil {
		fid = f.ID
	}
	return &db.File{
		Project:     i.Project,
		ProjectID:   i.Project.ID,
		Inventory:   i,
		InventoryID: i.ID,
		Folder:      f,
		FolderID:    fid,
		ArchiveDate: record.archiveDate,
		Checksum:    record.checksum,
		Filesize:    record.filesize,
		Path:        record.filename,
	}
}
