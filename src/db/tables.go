package db

import (
	"time"
)

// Project maps to the projects database table, which represents a top-level
// dark-archive directory
type Project struct {
	database *Database
	ID       int `sql:",primary"`
	Name     string
}

// Inventory maps to the inventories database table, which represents a
// CSV file in a project's INVENTORY folder
type Inventory struct {
	ID        int      `sql:",primary"`
	Project   *Project `sql:"-"`
	ProjectID int
	Filename  string
}

// Folder maps to the folders table, and is effectively a giant folder list for
// a project to allow easier refining of searches.  If the dark archive gets
// huge, this won't be efficient, as we really just use it to know which "LIKE"
// queries to fire off.
type Folder struct {
	ID        int `sql:",primary"`
	ProjectID int
	Path      string
}

// File maps to the files database table, which represents the actual archived
// files described by the inventory CSV files
type File struct {
	ID          int        `sql:",primary"`
	Project     *Project   `sql:"-"`
	Inventory   *Inventory `sql:"-"`
	ProjectID   int
	InventoryID int
	ArchiveDate time.Time
	Checksum    string
	Filesize    int64
	Path        string
}

// HasIndexedInventoryFile returns true if this project has already seen the given
// inventory file.  Database errors are passed up to the caller.
func (p *Project) HasIndexedInventoryFile(filename string) (bool, error) {
	var inventory = new(Inventory)
	var indexed = p.database.Inventories.Select().Where("filename = ?", filename).First(inventory)
	return indexed, p.database.Operation.Err()
}

// CreateFolder adds a folder under the project
func (p *Project) CreateFolder(path string) error {
	p.database.Folders.Save(&Folder{
		ProjectID: p.ID,
		Path:      path,
	})
	return p.database.Operation.Err()
}
