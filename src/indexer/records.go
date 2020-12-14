package indexer

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/uoregon-libraries/headlamp/src/config"
)

// inventoryRecord stores the raw data found on a single line of an inventory file
type inventoryRecord struct {
	fullPath string
	filesize int64
	checksum string
}

// parsedPath holds the processed / extracted data created by running a full
// path through the path processor
type parsedPath struct {
	categoryName string
	archiveDate  string
	publicPath   string
}

// a fileRecord holds all the inventory and path data for the construction of a
// db.File record
type fileRecord struct {
	*inventoryRecord
	*parsedPath
}

// parseInventoryRecord splits the three components of the inventory file line,
// performs some validation, translates the full path (since that's relative to
// the inventory file location) and returns the data
func parseInventoryRecord(record []byte, inventoryPath string) (*inventoryRecord, error) {
	// Skip the blank record at the end
	if len(record) == 0 {
		return nil, nil
	}

	// We sometimes have filenames with commas, but the sha and filesize are
	// always safe, so we just split to 3 elements
	var recParts = bytes.SplitN(record, []byte(","), 3)

	// Skip headers
	var checksum = string(recParts[0])
	if checksum == "sha256sum" {
		return nil, nil
	}

	// We should always have exactly 3 fields
	if len(recParts) != 3 {
		return nil, fmt.Errorf("there must be exactly 3 fields")
	}

	var filesizeString = string(recParts[1])
	var filesize, err = strconv.ParseInt(filesizeString, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid filesize value %q", filesizeString)
	}

	// The filename is relative to the inventory file's parent directory
	var relPath = string(recParts[2])
	var fullPath = filepath.Clean(filepath.Join(filepath.Dir(inventoryPath), "..", relPath))

	return &inventoryRecord{fullPath: fullPath, filesize: filesize, checksum: checksum}, nil
}

// parsePath splits apart the full path and processes it against the given path
// tokens to get the category, archive data, and public path
func parsePath(fullPath string, pf []config.PathToken) (*parsedPath, error) {
	var partCount = len(pf) + 1
	var pathParts = strings.SplitN(fullPath, string(os.PathSeparator), partCount)
	if len(pathParts) != partCount {
		return nil, fmt.Errorf("path %q doesn't have enough parts for path format %#v", fullPath, pf)
	}
	var publicPath string
	pathParts, publicPath = pathParts[:partCount-1], pathParts[partCount-1]

	// Pull the date and category name from the collapsed path elements
	var categoryName, dateDir string
	for index, part := range pathParts {
		switch pf[index] {
		case config.Category:
			categoryName = part
		case config.Date:
			dateDir = part
		}
	}

	// Make sure the date matches our expected format
	var timeFormat = "2006-01-02"
	var _, err = time.Parse(timeFormat, dateDir)
	if err != nil {
		return nil, fmt.Errorf("archive date directory %q must be formatted as a date (YYYY-MM-DD)", dateDir)
	}

	return &parsedPath{categoryName: categoryName, archiveDate: dateDir, publicPath: publicPath}, nil
}
