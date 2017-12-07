package main

import (
	"db"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/uoregon-libraries/gopkg/logger"
	"github.com/uoregon-libraries/gopkg/tmpl"
	"github.com/uoregon-libraries/gopkg/webutil"
)

// maxFiles tells the app how many files to display on at once; if there are
// more than this many, we let the user know to do a different search
const maxFiles = 1000

var root *tmpl.TRoot
var home, browse, search, empty *tmpl.Template

func initTemplates(webroot string) {
	webutil.Webroot = webroot
	root = tmpl.Root("layout", "templates/")
	root.Funcs(tmpl.DefaultTemplateFunctions)
	root.Funcs(webutil.FuncMap)
	root.Funcs(localTemplateFuncs)
	root.MustReadPartials("layout.go.html", "_search_form.go.html")
	home = root.Clone().MustBuild("home.go.html")
	browse = root.Clone().MustBuild("browse.go.html")
	var searchRoot = root.Clone()
	searchRoot.MustReadPartials("_files_table.go.html")
	search = searchRoot.Clone().MustBuild("search.go.html")
	empty = root.Template()
}

type vars map[string]interface{}

func _400(w http.ResponseWriter, msg string) {
	w.WriteHeader(http.StatusBadRequest)
	empty.Execute(w, vars{"Title": "Invalid Request", "Alert": msg})
}

func _404(w http.ResponseWriter, msg string) {
	w.WriteHeader(http.StatusNotFound)
	empty.Execute(w, vars{"Title": "Not Found", "Alert": msg})
}

func _500(w http.ResponseWriter, msg string) {
	w.WriteHeader(http.StatusInternalServerError)
	empty.Execute(w, vars{"Title": "Error", "Alert": msg})
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.String() != "/" {
		_404(w, "Unable to find the requested resource")
		return
	}

	var projects, err = dbh.Operation().AllProjects()
	if err != nil {
		logger.Errorf("Unable to find projects: %s", err)
		_500(w, "Error trying to find project list.  Try again or contact support.")
		return
	}

	err = home.Execute(w, vars{"Title": "Headlights", "Projects": projects})
	if err != nil {
		logger.Errorf("Unable to render home template: %s", err)
	}
}

type browseSearchData struct {
	op         *db.Operation
	pName      string
	project    *db.Project
	folderPath string
	folder     *db.Folder
	hadError   bool
}

// getBrowseSearchData centralizes some of the common things we need to check /
// pull from the database for both browsing and searching:
//
// - Get the current project, if this isn't a top-level search
// - Get the current folder, if one is set
func getBrowseSearchData(w http.ResponseWriter, r *http.Request) browseSearchData {
	var bsd browseSearchData
	var bsde = browseSearchData{hadError: true}
	var parts = strings.Split(r.URL.Path, "/")

	// We're doing a lot, so let's grab a single operation for all this lovely work
	bsd.op = dbh.Operation()

	// This shouldn't happen unless we screw something up elsewhere in the code
	if len(parts) < 1 || parts[1] == "" {
		_400(w, "Invalid request")
		logger.Debugf("invalid request")
		return bsde
	}

	if len(parts) < 2 {
		return bsd
	}

	bsd.pName = parts[2]
	bsd.folderPath = filepath.Join(parts[3:]...)

	// This is acceptable in some situations, so we don't want to explode due to
	// missing project
	if bsd.pName == "" {
		return bsd
	}

	var err error
	bsd.project, err = bsd.op.FindProjectByName(bsd.pName)
	if err != nil {
		logger.Errorf("Error trying to read project %q from the database: %s", bsd.pName, err)
		_500(w, fmt.Sprintf("Error trying to find project %q.  Try again or contact support.", bsd.pName))
		return bsde
	}
	if bsd.project == nil {
		_404(w, fmt.Sprintf("Project %q not found", bsd.pName))
		return bsde
	}

	if bsd.folderPath != "" {
		bsd.folder, err = bsd.op.FindFolderByPath(bsd.project, bsd.folderPath)
		if err != nil {
			logger.Errorf("Error trying to read folder %q (in project %q) from the database: %s",
				bsd.folderPath, bsd.pName, err)
			_500(w, fmt.Sprintf("Error trying to find folder %q.  Try again or contact support.", bsd.folderPath))
			return bsde
		}
		if bsd.folder == nil {
			_404(w, fmt.Sprintf("Folder %q not found", bsd.folderPath))
			return bsde
		}
	}

	return bsd
}

func browseHandler(w http.ResponseWriter, r *http.Request) {
	var bsd = getBrowseSearchData(w, r)
	if bsd.hadError {
		return
	}

	var folders, err = bsd.op.GetFolders(bsd.project, bsd.folder)
	if err != nil {
		logger.Errorf("Error trying to read folders under %q (in project %q) from the database: %s",
			bsd.folderPath, bsd.pName, err)
		_500(w, fmt.Sprintf("Error trying to read folder %q.  Try again or contact support.", bsd.folderPath))
		return
	}

	var files []*db.File
	var totalFileCount uint64
	var tooManyFiles = false
	files, totalFileCount, err = bsd.op.GetFiles(bsd.project, bsd.folder, maxFiles+1)
	if err != nil {
		logger.Errorf("Error trying to read files under %q (in project %q) from the database: %s",
			bsd.folderPath, bsd.pName, err)
		_500(w, fmt.Sprintf("Error trying to read folder %q.  Try again or contact support.", bsd.folderPath))
		return
	}
	if len(files) > maxFiles {
		files = files[:maxFiles]
		tooManyFiles = true
	}

	err = browse.Execute(w, vars{
		"Title":        fmt.Sprintf("Headlights: Browsing %s", bsd.project.Name),
		"Project":      bsd.project,
		"Folder":       bsd.folder,
		"Subfolders":   folders,
		"Files":        files,
		"TooManyFiles": tooManyFiles,
		"MaxFiles":     maxFiles,
		"TotalFiles":   totalFileCount,
	})
	if err != nil {
		logger.Errorf("Unable to render browse template: %s", err)
	}
}

func searchHandler(w http.ResponseWriter, r *http.Request) {
	var term = r.URL.Query().Get("q")
	if term == "" {
		_400(w, "You must provide a search term")
		return
	}

	var bsd = getBrowseSearchData(w, r)
	if bsd.hadError {
		return
	}

	var files, totalFileCount, err = bsd.op.SearchFiles(bsd.project, bsd.folder, term, maxFiles+1)
	if err != nil {
		logger.Errorf("Error trying to read files under %q (in project %q) from the database: %s",
			bsd.folderPath, bsd.pName, err)
		_500(w, fmt.Sprintf("Error trying to read folder %q.  Try again or contact support.", bsd.folderPath))
		return
	}

	var tooManyFiles = false
	if len(files) > maxFiles {
		files = files[:maxFiles]
		tooManyFiles = true
	}

	err = search.Execute(w, vars{
		"Title":        fmt.Sprintf("Headlights: Search"),
		"SearchTerm":   term,
		"Project":      bsd.project,
		"Folder":       bsd.folder,
		"Files":        files,
		"TooManyFiles": tooManyFiles,
		"MaxFiles":     maxFiles,
		"TotalFiles":   totalFileCount,
	})
	if err != nil {
		logger.Errorf("Unable to render browse template: %s", err)
	}
}
