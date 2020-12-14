// Package indexer handles scanning and indexing dark-archive inventories and files
package indexer

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/uoregon-libraries/gopkg/logger"
	"github.com/uoregon-libraries/headlamp/src/config"
	"github.com/uoregon-libraries/headlamp/src/db"
)

// category wraps db.Category, extending it with a cache of the top- and
// second-level folders so we avoid the majority of DB hits without holding
// onto absurd quantites of data in cases where the folder structure is
// unusually deep
type category struct {
	*db.Category

	folders     map[string]*db.Folder
	realFolders map[string]*db.RealFolder
}

func (c *category) buildFile(i *db.Inventory, f *db.Folder, r *fileRecord) *db.File {
	var fid = 0
	if f != nil {
		fid = f.ID
	}

	var _, fname = filepath.Split(r.fullPath)
	return &db.File{
		Category:    c.Category,
		CategoryID:  c.Category.ID,
		Inventory:   i,
		InventoryID: i.ID,
		Folder:      f,
		FolderID:    fid,
		Depth:       strings.Count(r.publicPath, string(os.PathSeparator)),
		ArchiveDate: r.archiveDate,
		Checksum:    r.checksum,
		Filesize:    r.filesize,
		FullPath:    r.fullPath,
		PublicPath:  r.publicPath,
		Name:        fname,
	}
}

// States an indexer can be in
const (
	iStateStopped int32 = iota
	iStateRunning
	iStateStopping
)

// Indexer controls how we find inventory files, which part(s) of the path are
// skipped, and which part of the path defines a category name.
type Indexer struct {
	sync.Mutex
	dbh *db.Database
	c   *config.Config

	// categories keeps a cache of all categories, keyed by the name, to avoid
	// millions of unnecessary lookups in the db
	categories map[string]*category

	// seenInventoryFiles caches the files we've processed in the past so we
	// don't hit the DB each time we're looking at a new inventory file
	seenInventoryFiles map[string]bool

	// state is set via async calls to tell us what the indexer is currently
	// doing (if anything).  This allows running an indexing operation in the
	// background and waiting for it to finish while also being able to request
	// it to stop at the next opportunity.
	state int32
}

// New sets up a scanner for use in indexing dark-archive file data
func New(dbh *db.Database, conf *config.Config) *Indexer {
	return &Indexer{dbh: dbh, c: conf, categories: make(map[string]*category)}
}

// Index searches for inventory files not previously seen and indexes the files
// described therein
func (i *Indexer) Index() error {
	// It's not an error if we're already running, but we don't want to start again
	if i.getState() == iStateRunning {
		return nil
	}

	logger.Infof("Starting indexer.Index()")
	defer logger.Infof("indexer.Index() complete")

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

// findInventoryFiles gathers a list of files matching the Indexer's
// InventoryPattern that haven't been modified in at least an hour
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
		var info, err = os.Stat(fname)
		if err != nil {
			logger.Errorf("Skipping %q: could not stat: %s", fname, err)
			continue
		}
		if time.Since(info.ModTime()) < time.Hour {
			logger.Debugf("Skipping %q: modified too recently (%s)", fname, info.ModTime())
			continue
		}

		files = append(files, fname)
	}

	return files, nil
}

func (i *Indexer) seenInventoryFile(fname string) bool {
	return i.seenInventoryFiles[fname]
}
