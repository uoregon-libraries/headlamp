package db

import (
	"strings"

	"github.com/Nerdmaster/magicsql"
)

// FSelect wraps common "SELECT" behaviors for both files and folders
type FSelect struct {
	op          *Operation
	sel         magicsql.Select
	category    *Category
	folder      *Folder
	whereFields []string
	whereArgs   []interface{}
	limit       uint64
	tree        bool
}

// FileSelect creates a new FSelect for querying/searching files
func (op *Operation) FileSelect(c *Category, f *Folder) *FSelect {
	return &FSelect{op: op, sel: op.Files.Select(), category: c, folder: f}
}

// FolderSelect creates a new FSelect for querying/searching folders
func (op *Operation) FolderSelect(c *Category, f *Folder) *FSelect {
	return &FSelect{op: op, sel: op.Folders.Select(), category: c, folder: f}
}

// TreeMode defaults to false, but if set to true will recurse through all
// subdirectories instead of limiting the search to the precise category and
// folder passed into the constructor
func (s *FSelect) TreeMode(t bool) *FSelect {
	s.tree = t
	return s
}

// Search adds to the WHERE clause when the SELECT is run
func (s *FSelect) Search(field string, term interface{}) *FSelect {
	s.whereFields = append(s.whereFields, field)
	s.whereArgs = append(s.whereArgs, term)
	return s
}

// Limit sets the maximum rows to return
func (s *FSelect) Limit(l uint64) *FSelect {
	s.limit = l
	return s
}

func (s *FSelect) setCategory(data interface{}) {
	var files []*File
	var folders []*Folder
	switch fList := data.(type) {
	case *[]*File:
		files = *fList
	case *[]*Folder:
		folders = *fList
	}

	// if Category was blank, pull category via IDs
	if s.category == nil {
		s.op.PopulateCategories(files, folders)
		return
	}

	for _, f := range files {
		f.Category = s.category
	}
	for _, f := range folders {
		f.Category = s.category
	}
}

// AllObjects runs the query based on all the data, sending obj to the
// underlying Select's AllObjects function.  Returns the total number of
// objects found via a COUNT query if Limit was set in order to know if more
// objects were available.
func (s *FSelect) AllObjects(data interface{}) (total uint64, err error) {
	if s.category != nil {
		s.whereFields = append(s.whereFields, "category_id = ?")
		s.whereArgs = append(s.whereArgs, s.category.ID)
	}
	if s.tree == false {
		var folderID int
		if s.folder != nil {
			folderID = s.folder.ID
		}
		s.whereFields = append(s.whereFields, "folder_id = ?")
		s.whereArgs = append(s.whereArgs, folderID)
	} else {
		if s.folder != nil {
			s.whereFields = append(s.whereFields, "public_path like ?")
			s.whereArgs = append(s.whereArgs, s.folder.PublicPath+"/%")
		}
	}

	var sel = s.sel.Where(strings.Join(s.whereFields, " AND "), s.whereArgs...)
	sel = sel.Order("depth, LOWER(public_path)")

	var count = sel.Count().RowCount()
	sel.AllObjects(data)

	s.setCategory(data)
	return count, s.op.Operation.Err()
}
