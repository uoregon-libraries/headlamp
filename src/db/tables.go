package db

import (
	"net/mail"
	"path/filepath"
	"strings"
	"time"
)

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

// The ArchiveJob structure maps to archive_jobs, storing RS-separated files and
// comma-separated notification email(s).  The record represents a single
// archive creation request.
type ArchiveJob struct {
	ID                 int `sql:",primary"`
	CreatedAt          time.Time
	NextAttemptAt      time.Time
	NotificationEmails string
	Files              string
	Processed          bool
}

// Emails parses the email addresses as mail.Addr instances and returns them as
// a list of strings, ensuring the strings are split properly in cases where
// the email addressee has a comma in the name, e.g.:
//
//     "John Doe, III" <jdoeiii@example.org>, Alice <alice@example.org>
//
// Errors are ignored, as the database shouldn't get emails in any way other
// than from a pre-validated email list
func (j *ArchiveJob) Emails() []string {
	var eList, _ = mail.ParseAddressList(j.NotificationEmails)
	var sList = make([]string, len(eList))
	for i, e := range eList {
		sList[i] = e.String()
	}
	return sList
}

// FileList splits the files field and returns them as a list
func (j *ArchiveJob) FileList() []string {
	return strings.Split(j.Files, "\x1E")
}
