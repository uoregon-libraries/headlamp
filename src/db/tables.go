package db

import "time"

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
	ArchiveDate time.Time
	Checksum    string
	Filesize    int64
	FullPath    string
	PublicPath  string
}
