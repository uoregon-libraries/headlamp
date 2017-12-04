package db

import (
	"database/sql"
	"fmt"
	"os"
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

// InTransaction connects to the database and starts a transaction, used by all
// other Database calls, runs the callback function, then ends the transaction,
// returning the error (if any occurs)
func (db *Database) InTransaction(cb func(*Operation) error) error {
	var magicOp = db.dbh.Operation()
	var op = &Operation{
		Operation:   magicOp,
		Files:       magicOp.OperationTable(db.mtFiles),
		Folders:     magicOp.OperationTable(db.mtFolders),
		Inventories: magicOp.OperationTable(db.mtInventories),
		Projects:    magicOp.OperationTable(db.mtProjects),
	}

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

// FindOrCreateProject stores (or finds) the project by the given name and
// returns it.  If there are any database errors, they're returned and Project
// will be undefined.
func (op *Operation) FindOrCreateProject(name string) (*Project, error) {
	var project = &Project{op: op}
	var ok = op.Projects.Select().Where("name = ?", name).First(project)
	if !ok {
		project.Name = name
		op.Projects.Save(project)
	}
	return project, op.Operation.Err()
}

// FindOrCreateFolder centralizes the creation and DB-save operation for folders
func FindOrCreateFolder(p *Project, f *Folder, path string) (*Folder, error) {
	var parts = strings.Split(path, string(os.PathSeparator))
	var parentFolderID = 0
	if f != nil {
		parentFolderID = f.ID
	}
	var newFolder Folder
	var sel = p.op.Folders.Select()
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
	p.op.Folders.Save(&newFolder)
	return &newFolder, p.op.Operation.Err()
}
