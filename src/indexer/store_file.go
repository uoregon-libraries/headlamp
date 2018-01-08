package indexer

import (
	"db"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func (i *indexerOperation) findOrCreateCategory(pName string) (*category, error) {
	// Get or create the category
	i.Lock()
	defer i.Unlock()

	if i.categories[pName] == nil {
		var p, err = i.op.FindOrCreateCategory(pName)
		if err != nil {
			return nil, fmt.Errorf("couldn't create category %q: %s", pName, err)
		}
		i.categories[pName] = &category{Category: p, folders: make(map[string]*db.Folder)}
	}

	return i.categories[pName], nil
}

// processFolderPaths finds or builds each folder in the given list, where each
// element is the child of the prior element.  The first database error is
// returned, aborting whatever folders were still needing to be built, if any.
//
// The first two levels of folders are cached for reuse.
func (i *indexerOperation) processFolderPaths(p *category, folders []string) (*db.Folder, error) {
	var fullPath string
	var folder, parentFolder *db.Folder

	for level, fName := range folders {
		fullPath = filepath.Join(fullPath, fName)
		folder = p.folders[fullPath]
		if folder == nil {
			var err error
			folder, err = i.op.FindOrCreateFolder(p.Category, parentFolder, fullPath)
			if err != nil {
				return nil, fmt.Errorf("couldn't build folder %q: %s", fullPath, err)
			}
			if level <= 2 {
				p.folders[fullPath] = folder
			}
		}
		parentFolder = folder
	}

	// At this point, the parent folder is whatever was last in the list and can
	// be returned for use in creating the file record
	return parentFolder, nil
}

func (p *category) buildFile(i *db.Inventory, f *db.Folder, r fileRecord) *db.File {
	var fid = 0
	if f != nil {
		fid = f.ID
	}

	var _, fname = filepath.Split(r.fullPath)
	return &db.File{
		Category:    p.Category,
		CategoryID:  p.Category.ID,
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

func (i *indexerOperation) storeFile(index int, inventory *db.Inventory, fr fileRecord) error {
	var prj, err = i.findOrCreateCategory(fr.categoryName)
	if err != nil {
		return err
	}

	var pathParts = strings.Split(fr.publicPath, string(os.PathSeparator))
	var pathCount = len(pathParts)
	var parentFolder *db.Folder
	if pathCount > 1 {
		parentFolder, err = i.processFolderPaths(prj, pathParts[:pathCount-1])
		if err != nil {
			return err
		}
	}

	var f = prj.buildFile(inventory, parentFolder, fr)
	i.op.Files.Save(f)
	if i.op.Operation.Err() != nil {
		return fmt.Errorf("couldn't store file %#v: %s", f, i.op.Operation.Err())
	}

	return nil
}
