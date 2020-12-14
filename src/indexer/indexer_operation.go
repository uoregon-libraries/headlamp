package indexer

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/uoregon-libraries/gopkg/logger"
	"github.com/uoregon-libraries/headlamp/src/db"
)

// indexerOperation wraps an Indexer with a single operation's context so we
// can separate the overall Indexer setup / config from a single transactioned
// indexing job
type indexerOperation struct {
	*Indexer
	op *db.Operation
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
		i.index(inventory, index, record)
	}

	return i.op.Operation.Err()
}

// index takes the inventory record to create all the folders (real and
// collapsed) in the database, and then parses the file record data to index
// the file.  If any database errors occur, the operation halts and the first
// such error is returned.
func (i *indexerOperation) index(inventory *db.Inventory, index int, record []byte) (err error) {
	// Get the inventory record split up and processed
	var ir *inventoryRecord
	ir, err = parseInventoryRecord(record, inventory.Path)
	if err != nil {
		return fmt.Errorf("unable to parse record #%d (inventory %q): %s", index, inventory.Path, err)
	}

	// Skip headers / empty records
	if ir == nil {
		return nil
	}

	var pp *parsedPath
	pp, err = parsePath(ir.fullPath, i.c.PathFormat)
	if err != nil {
		return fmt.Errorf("unable to parse paths in record #%d (inventory %q): %s", index, inventory.Path, err)
	}

	var category *category
	category, err = i.findOrCreateCategory(pp.categoryName)
	if err != nil {
		return err
	}

	var lastFolder *db.Folder
	lastFolder, err = i.indexPaths(category, ir.fullPath)
	if err != nil {
		return err
	}

	var fr = &fileRecord{ir, pp}
	return i.indexFile(inventory, category, lastFolder, fr)
}

// findOrCreateCategory takes the given category name and indexes it if it
// hasn't already been indexed
func (i *indexerOperation) findOrCreateCategory(cName string) (*category, error) {
	i.Lock()
	defer i.Unlock()

	if i.categories[cName] == nil {
		var c, err = i.op.FindOrCreateCategory(cName)
		if err != nil {
			return nil, fmt.Errorf("couldn't create category %q: %s", cName, err)
		}
		i.categories[cName] = &category{
			Category:    c,
			folders:     make(map[string]*db.Folder),
			realFolders: make(map[string]*db.RealFolder),
		}
	}

	return i.categories[cName], nil
}

// indexPaths processes each of the file's folders so we can index them in
// order to know that for a given collapsed folder, there are a given set of
// real folders.  Returns the last Folder record created so the file indexer
// can reuse the work done here.
func (i *indexerOperation) indexPaths(c *category, fullPath string) (lastPublicFolder *db.Folder, err error) {
	var pathParts = strings.Split(fullPath, string(os.PathSeparator))
	var pfLen = len(i.c.PathFormat)
	var publicFolder *db.Folder
	var realFolder *db.RealFolder
	var curPath string

	for index, part := range pathParts[:len(pathParts)-1] {
		var level = index - pfLen
		curPath = filepath.Join(curPath, part)

		// Since we can't expose files in the ignored/archive date directories, we
		// don't try to index anything here
		if level < 0 {
			continue
		}

		var pp, err = parsePath(curPath, i.c.PathFormat)
		if err != nil {
			return nil, err
		}

		// Index the public folder first
		publicFolder = c.folders[pp.publicPath]
		if publicFolder == nil {
			publicFolder, err = i.op.FindOrCreateFolder(c.Category, lastPublicFolder, pp.publicPath)
			if err != nil {
				return nil, fmt.Errorf("couldn't build folder %q: %s", pp.publicPath, err)
			}
			if level <= 2 {
				c.folders[pp.publicPath] = publicFolder
			}
		}
		lastPublicFolder = publicFolder

		// Index the real folder
		realFolder = c.realFolders[curPath]
		if realFolder == nil {
			realFolder, err = i.op.FindOrCreateRealFolder(lastPublicFolder, curPath)
			if err != nil {
				return nil, fmt.Errorf("couldn't build real folder %q: %s", curPath, err)
			}
			if level <= 2 {
				c.realFolders[curPath] = realFolder
			}
		}
	}

	return lastPublicFolder, err
}

func (i *indexerOperation) indexFile(inv *db.Inventory, c *category, folder *db.Folder, fr *fileRecord) error {
	var f = c.buildFile(inv, folder, fr)
	i.op.Files.Save(f)
	if i.op.Operation.Err() != nil {
		return fmt.Errorf("couldn't store file %#v: %s", f, i.op.Operation.Err())
	}
	return nil
}
