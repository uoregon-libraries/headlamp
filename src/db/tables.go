package db

import "path/filepath"

// Project maps to the projects database table, which represents a "magic"
// dark-archive directory we expose as if it's a top-level directory for
// browsing files
type Project struct {
	ID   int `sql:",primary"`
	Name string
}

// Inventory maps to the inventories database table, which represents a
// manifest file in an INVENTORY folder
type Inventory struct {
	ID   int    `sql:",primary"`
	Path string // Path is relative to the dark archive root
}

// Folder maps to the folders table, and is effectively a giant list of our
// collapsed folder structure for a project to allow easier browsing and/or
// refining of searches
type Folder struct {
	ID         int      `sql:",primary"`
	Project    *Project `sql:"-"`
	Folder     *Folder  `sql:"-"`
	ProjectID  int
	FolderID   int
	Depth      int
	Name       string
	PublicPath string
}

// File maps to the files database table, which represents the actual archived
// files described by the inventory files
type File struct {
	ID          uint64     `sql:",primary"`
	Project     *Project   `sql:"-"`
	Inventory   *Inventory `sql:"-"`
	Folder      *Folder    `sql:"-"`
	ProjectID   int
	InventoryID int
	FolderID    int
	Depth       int
	ArchiveDate string
	Checksum    string
	Filesize    int64
	Name        string
	FullPath    string
	PublicPath  string
}

// ContainingFolder returns the path to the file's folder for cases where
// loading the folder data for each file would be an unnecessary task
func (f *File) ContainingFolder() string {
	return filepath.Dir(f.PublicPath)
}
