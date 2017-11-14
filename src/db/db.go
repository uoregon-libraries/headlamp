package db

import (
	"database/sql"
	"fmt"
	"sync"

	"github.com/Nerdmaster/magicsql"
	_ "github.com/mattn/go-sqlite3" // database/sql requires "side-effect" packages be loaded
	"github.com/uoregon-libraries/gopkg/logger"
)

var dbh *magicsql.DB
var dbhMutex sync.Mutex
var mtFiles = magicsql.Table("files", &File{})
var mtFolders = magicsql.Table("folders", &Folder{})
var mtProjects = magicsql.Table("projects", &Project{})
var mtInventories = magicsql.Table("inventories", &Inventory{})

// Database encapsulates transactions and magicsql data types
type Database struct {
	sync.Mutex
	Operation   *magicsql.Operation
	Files       *magicsql.OperationTable
	Folders     *magicsql.OperationTable
	Inventories *magicsql.OperationTable
	Projects    *magicsql.OperationTable
}

// Operation is meaningless and just wraps Database.  It's here to keep users
// from doing something like Database.FindOrCreateProject() when an operation /
// transaction hasn't been set up.
type Operation struct {
	*Database
}

// New sets up a database connection and returns a usable Database
func New() *Database {
	dbhMutex.Lock()
	if dbh == nil {
		var _db, err = sql.Open("sqlite3", "db/da.db")
		if err != nil {
			logger.Fatalf("Unable to open database: %s", err)
		}
		dbh = magicsql.Wrap(_db)
	}
	dbhMutex.Unlock()

	return &Database{}
}

// InTransaction connects to the database and starts a transaction, used by all
// other Database calls, runs the callback function, then ends the transaction,
// returning the error (if any occurs)
func (db *Database) InTransaction(cb func(*Operation)) error {
	db.Lock()
	defer db.Unlock()

	if db.Operation != nil {
		return fmt.Errorf("cannot wrap a transaction when a previous operation is still pending")
	}
	db.Operation = dbh.Operation()
	db.Files = db.Operation.OperationTable(mtFiles)
	db.Folders = db.Operation.OperationTable(mtFolders)
	db.Inventories = db.Operation.OperationTable(mtInventories)
	db.Projects = db.Operation.OperationTable(mtProjects)

	db.Operation.BeginTransaction()
	cb(&Operation{db})
	db.Operation.EndTransaction()

	var err = db.Operation.Err()
	db.Operation = nil
	return err
}

// DeleteAll destroys all files and projects from the database in order to
// prepare for a fresh data load
func (op *Operation) DeleteAll() error {
	op.Operation.Exec("DELETE FROM files")
	op.Operation.Exec("DELETE FROM folders")
	op.Operation.Exec("DELETE FROM inventories")
	op.Operation.Exec("DELETE FROM projects")
	return op.Operation.Err()
}

// FindOrCreateProject stores (or finds) the project by the given name and
// returns it.  If there are any database errors, they're returned and Project
// will be undefined.
func (op *Operation) FindOrCreateProject(name string) (*Project, error) {
	var project = &Project{database: op.Database}
	var ok = op.Projects.Select().Where("name = ?", name).First(project)
	if !ok {
		project.Name = name
		op.Projects.Save(project)
	}
	return project, op.Operation.Err()
}
