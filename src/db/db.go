package db

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Nerdmaster/magicsql"
	_ "github.com/mattn/go-sqlite3" // database/sql requires "side-effect" packages be loaded
	"github.com/uoregon-libraries/gopkg/logger"
)

// Database encapsulates the database handle and magicsql table definitions
type Database struct {
	dbh           *magicsql.DB
	mtFiles       *magicsql.MagicTable
	mtFolders     *magicsql.MagicTable
	mtProjects    *magicsql.MagicTable
	mtInventories *magicsql.MagicTable
}

// Operation wraps a magicsql Operation with preloaded OperationTable
// definitions for easy querying
type Operation struct {
	Operation   *magicsql.Operation
	Files       *magicsql.OperationTable
	Folders     *magicsql.OperationTable
	Inventories *magicsql.OperationTable
	Projects    *magicsql.OperationTable
}

// New sets up a database connection and returns a usable Database
func New() *Database {
	var _db, err = sql.Open("sqlite3", "db/da.db")
	if err != nil {
		logger.Fatalf("Unable to open database: %s", err)
	}

	return &Database{
		dbh:           magicsql.Wrap(_db),
		mtFiles:       magicsql.Table("files", &File{}),
		mtFolders:     magicsql.Table("folders", &Folder{}),
		mtProjects:    magicsql.Table("projects", &Project{}),
		mtInventories: magicsql.Table("inventories", &Inventory{}),
	}
}

// Operation returns a pre-set Operation for quick tasks that don't warrant a transaction
func (db *Database) Operation() *Operation {
	var magicOp = db.dbh.Operation()
	return &Operation{
		Operation:   magicOp,
		Files:       magicOp.OperationTable(db.mtFiles),
		Folders:     magicOp.OperationTable(db.mtFolders),
		Inventories: magicOp.OperationTable(db.mtInventories),
		Projects:    magicOp.OperationTable(db.mtProjects),
	}
}

// InTransaction connects to the database and starts a transaction, used by all
// other Database calls, runs the callback function, then ends the transaction,
// returning the error (if any occurs)
func (db *Database) InTransaction(cb func(*Operation) error) error {
	var op = db.Operation()
	op.Operation.BeginTransaction()
	var err = cb(op)

	// Make sure we absolutely rollback if an error is returned
	if err != nil {
		op.Operation.Rollback()
		return err
	}

	op.Operation.EndTransaction()
	err = op.Operation.Err()
	if err != nil {
		return fmt.Errorf("database error: %s", err)
	}
	return nil
}

// AllInventories returns all the inventory files which have been indexed
func (op *Operation) AllInventories() ([]*Inventory, error) {
	var inventories []*Inventory
	op.Inventories.Select().AllObjects(&inventories)
	return inventories, op.Operation.Err()
}

// WriteInventory stores the given inventory object in the database
func (op *Operation) WriteInventory(i *Inventory) error {
	op.Inventories.Save(i)
	return op.Operation.Err()
}

// AllProjects returns all projects which have been seen
func (op *Operation) AllProjects() ([]*Project, error) {
	var projects []*Project
	op.Projects.Select().Order("LOWER(name)").AllObjects(&projects)
	return projects, op.Operation.Err()
}

// FindProjectByName returns a project if one exists with the given name, and
// the database error if any occurred
func (op *Operation) FindProjectByName(name string) (*Project, error) {
	var project = &Project{}
	var ok = op.Projects.Select().Where("name = ?", name).First(project)
	if !ok {
		project = nil
	}
	return project, op.Operation.Err()
}

// FindOrCreateProject stores (or finds) the project by the given name and
// returns it.  If there are any database errors, they're returned and Project
// will be undefined.
func (op *Operation) FindOrCreateProject(name string) (*Project, error) {
	var project, err = op.FindProjectByName(name)
	if project == nil && err == nil {
		project = &Project{Name: name}
		op.Projects.Save(project)
	}
	return project, op.Operation.Err()
}

// FindFolderByPath looks for a folder with the given path under the given project
func (op *Operation) FindFolderByPath(p *Project, path string) (*Folder, error) {
	var folder = &Folder{}
	var ok = op.Folders.Select().Where("project_id = ? AND public_path = ?", p.ID, path).First(folder)
	if !ok {
		folder = nil
	}
	return folder, op.Operation.Err()
}

// FindOrCreateFolder centralizes the creation and DB-save operation for folders
func (op *Operation) FindOrCreateFolder(p *Project, f *Folder, path string) (*Folder, error) {
	var parentFolderID = 0
	if f != nil {
		parentFolderID = f.ID
	}
	var folder, err = op.FindFolderByPath(p, path)
	if err != nil {
		return nil, err
	}
	if folder != nil {
		if folder.FolderID != parentFolderID {
			return nil, fmt.Errorf("existing record with different parent found")
		}
		folder.Folder = f
		folder.Project = p
		return folder, nil
	}

	var _, filename = filepath.Split(path)
	var newFolder = Folder{
		Folder:     f,
		FolderID:   parentFolderID,
		Project:    p,
		ProjectID:  p.ID,
		PublicPath: path,
		Name:       filename,
	}
	op.Folders.Save(&newFolder)
	return &newFolder, op.Operation.Err()
}

// GetFolders returns all folders with the given project and parent folder.  A
// parent folder of nil can be used to pull all top-level folders.
func (op *Operation) GetFolders(project *Project, folder *Folder) ([]*Folder, error) {
	var folders []*Folder
	var fid int
	if folder != nil {
		fid = folder.ID
	}
	op.Folders.Select().
		Where("project_id = ? AND folder_id = ?", project.ID, fid).
		Order("LOWER(name)").
		AllObjects(&folders)

	for _, f := range folders {
		f.Folder = folder
		f.Project = project
	}
	return folders, op.Operation.Err()
}

// GetFiles returns all files with the given project and parent folder.  A
// parent folder of nil can be used to pull all top-level files.
func (op *Operation) GetFiles(project *Project, folder *Folder, limit uint64) ([]*File, uint64, error) {
	var files []*File
	var fid int
	if folder != nil {
		fid = folder.ID
	}
	var sel = op.Files.Select().
		Where("project_id = ? AND folder_id = ?", project.ID, fid).
		Order("LOWER(name)")

	var count = sel.Count().RowCount()
	sel.Limit(limit).AllObjects(&files)
	for _, f := range files {
		f.Folder = folder
		f.Project = project
	}
	return files, count, op.Operation.Err()
}

// commonSearchWhere sets up the SQL "WHERE" string and arg slices common to
// searches for files or folders
func (op *Operation) commonSearchSelect(project *Project, folder *Folder) ([]string, []interface{}) {
	var wherePieces []string
	var whereArgs []interface{}
	if project != nil {
		wherePieces = append(wherePieces, "project_id = ?")
		whereArgs = append(whereArgs, project.ID)
	}
	if folder != nil {
		wherePieces = append(wherePieces, "public_path like ?")
		whereArgs = append(whereArgs, folder.PublicPath+"/%")
	}

	return wherePieces, whereArgs
}

// SearchFiles finds all files which are *descendents* of the given
// project/folder and match the term
//
// Note that folder data is *not* filled in on the returns files.  Pulling
// folders from the database is unnecessary since all folder lookups are via
// path, so this reduces the amount of information we pull from the database
// and simplifies the code quite a bit.
func (op *Operation) SearchFiles(project *Project, folder *Folder, term string, limit uint64) ([]*File, uint64, error) {
	var wherePieces, whereArgs = op.commonSearchSelect(project, folder)
	wherePieces = append(wherePieces, "public_path LIKE ?")
	whereArgs = append(whereArgs, term)

	var sel = op.Files.Select()
	sel = sel.Where(strings.Join(wherePieces, " AND "), whereArgs...)
	sel = sel.Order("LOWER(public_path)")

	var count = sel.Count().RowCount()
	var files []*File
	sel.Limit(limit).AllObjects(&files)

	// If project is blank, pull project from db
	if project == nil {
		var projectLookup = make(map[int]*Project)
		var projectList, err = op.AllProjects()
		if err != nil {
			return nil, 0, err
		}
		for _, p := range projectList {
			projectLookup[p.ID] = p
		}
		for _, f := range files {
			f.Project = projectLookup[f.ProjectID]
		}
	} else {
		for _, f := range files {
			f.Project = project
		}
	}

	return files, count, op.Operation.Err()
}

// SearchFolders finds all folders which are *descendents* of the given
// project/folder and match the term
//
// Note that parent folder data is *not* filled in on the returns files.
// Pulling folders from the database is unnecessary since all folder lookups
// are via path, so this reduces the amount of information we pull from the
// database and simplifies the code quite a bit.
func (op *Operation) SearchFolders(project *Project, folder *Folder, term string) ([]*Folder, error) {
	var wherePieces, whereArgs = op.commonSearchSelect(project, folder)
	wherePieces = append(wherePieces, "name LIKE ?")
	whereArgs = append(whereArgs, term)

	var folders []*Folder
	op.Folders.Select().
		Where(strings.Join(wherePieces, " AND "), whereArgs...).
		Order("LOWER(public_path)").
		AllObjects(&folders)

	// If project is blank, pull project from db
	if project == nil {
		var projectLookup = make(map[int]*Project)
		var projectList, err = op.AllProjects()
		if err != nil {
			return nil, err
		}
		for _, p := range projectList {
			projectLookup[p.ID] = p
		}
		for _, f := range folders {
			f.Project = projectLookup[f.ProjectID]
		}
	} else {
		for _, f := range folders {
			f.Project = project
		}
	}

	return folders, op.Operation.Err()
}

// FindFileByID returns the file found by the given ID, or nil if none if
// found.  Any database errors are passed back to the caller.
func (op *Operation) FindFileByID(id uint64) (*File, error) {
	var file = &File{}
	var ok = op.Files.Select().Where("id = ?", id).First(file)
	if !ok {
		file = nil
	}
	return file, op.Operation.Err()
}
