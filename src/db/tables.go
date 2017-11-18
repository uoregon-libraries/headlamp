package db

import (
	"fmt"
	"os"
	"strings"
)

// Project maps to the projects database table, which represents a "magic"
// dark-archive directory we expose as if it's a top-level directory for
// browsing files
type Project struct {
	database *Database
	ID       int `sql:",primary"`
	Name     string
}

// Inventory maps to the inventories database table, which represents a
// manifest file in an INVENTORY folder
type Inventory struct {
	ID   int    `sql:",primary"`
	Path string // Path is relative to the dark archive root
}

// Folder maps to the folders table, and is effectively a giant folder list for
// a project to allow easier refining of searches
type Folder struct {
	ID        int      `sql:",primary"`
	Project   *Project `sql:"-"`
	Folder    *Folder  `sql:"-"`
	ProjectID int
	FolderID  int
	Path      string
	Name      string
}

// File maps to the files database table, which represents the actual archived
// files described by the inventory files
type File struct {
	ID          int        `sql:",primary"`
	Project     *Project   `sql:"-"`
	Inventory   *Inventory `sql:"-"`
	Folder      *Folder    `sql:"-"`
	ProjectID   int
	InventoryID int
	FolderID    int
	Checksum    string
	Filesize    int64
	FullPath    string
	PublicPath  string
}

// HasIndexedInventoryFile returns true if this project has already seen the given
// inventory file.  Database errors are passed up to the caller.
func (p *Project) HasIndexedInventoryFile(filename string) (bool, error) {
	var inventory = new(Inventory)
	var indexed = p.database.Inventories.Select().Where("filename = ?", filename).First(inventory)
	return indexed, p.database.Operation.Err()
}

// HasIndexedFile returns true if the given file is already in the database.
// Database-level errors are passed up to the caller.
func (p *Project) HasIndexedFile(f *File) (bool, error) {
	var dummy File
	var sel = p.database.Files.Select()
	sel = sel.Where("project_id = ? AND archive_date = ? AND path = ?", p.ID, f.ArchiveDate, f.Path)
	return sel.First(&dummy), p.database.Operation.Err()
}

// FindOrCreateFolder centralizes the creation and DB-save operation for folders
func FindOrCreateFolder(p *Project, f *Folder, path string) (*Folder, error) {
	var parts = strings.Split(path, string(os.PathSeparator))
	var parentFolderID = 0
	if f != nil {
		parentFolderID = f.ID
	}
	var newFolder Folder
	var sel = p.database.Folders.Select()
	sel = sel.Where("project_id = ? AND path = ?", p.ID, path)
	var ok = sel.First(&newFolder)
	if ok {
		if newFolder.FolderID != parentFolderID {
			return nil, fmt.Errorf("existing record with different parent found")
		}
		newFolder.Folder = f
		newFolder.Project = p
		return &newFolder, nil
	}

	newFolder = Folder{
		Folder:    f,
		FolderID:  parentFolderID,
		Project:   p,
		ProjectID: p.ID,
		Path:      path,
		Name:      parts[len(parts)-1],
	}
	p.database.Folders.Save(&newFolder)
	return &newFolder, p.database.Operation.Err()
}
