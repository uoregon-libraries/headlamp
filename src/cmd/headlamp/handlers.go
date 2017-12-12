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
var home, browse, search, empty *Template

func initTemplates(webroot string) {
	webutil.Webroot = webroot
	root = tmpl.Root("layout", "templates/")
	root.Funcs(tmpl.DefaultTemplateFunctions)
	root.Funcs(webutil.FuncMap)
	root.Funcs(localTemplateFuncs)
	root.MustReadPartials("layout.go.html", "_search_form.go.html", "_tables.go.html")
	home = &Template{root.Clone().MustBuild("home.go.html")}
	browse = &Template{root.Clone().MustBuild("browse.go.html")}
	search = &Template{root.Clone().MustBuild("search.go.html")}
	empty = &Template{root.Template()}
}

type vars map[string]interface{}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.String() != "/" {
		_404(w, r, "Unable to find the requested resource")
		return
	}

	renderHome(w, r)
}

func renderHome(w http.ResponseWriter, r *http.Request) {
	var projects, err = dbh.Operation().AllProjects()
	if err != nil {
		logger.Errorf("Unable to find projects: %s", err)
		_500(w, r, "Error trying to find project list.  Try again or contact support.")
		return
	}

	home.Render(w, r, vars{"Title": "Headlamp", "Projects": projects})
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
		_400(w, r, "Invalid request")
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
		_500(w, r, fmt.Sprintf("Error trying to find project %q.  Try again or contact support.", bsd.pName))
		return bsde
	}
	if bsd.project == nil {
		_404(w, r, fmt.Sprintf("Project %q not found", bsd.pName))
		return bsde
	}

	if bsd.folderPath != "" {
		bsd.folder, err = bsd.op.FindFolderByPath(bsd.project, bsd.folderPath)
		if err != nil {
			logger.Errorf("Error trying to read folder %q (in project %q) from the database: %s",
				bsd.folderPath, bsd.pName, err)
			_500(w, r, fmt.Sprintf("Error trying to find folder %q.  Try again or contact support.", bsd.folderPath))
			return bsde
		}
		if bsd.folder == nil {
			_404(w, r, fmt.Sprintf("Folder %q not found", bsd.folderPath))
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
		_500(w, r, fmt.Sprintf("Error trying to read folder %q.  Try again or contact support.", bsd.folderPath))
		return
	}

	var files []*db.File
	var totalFileCount uint64
	files, totalFileCount, err = bsd.op.GetFiles(bsd.project, bsd.folder, maxFiles+1)
	if err != nil {
		logger.Errorf("Error trying to read files under %q (in project %q) from the database: %s",
			bsd.folderPath, bsd.pName, err)
		_500(w, r, fmt.Sprintf("Error trying to read folder %q.  Try again or contact support.", bsd.folderPath))
		return
	}

	var tooManyFiles = false
	if len(files) > maxFiles {
		files = files[:maxFiles]
		tooManyFiles = true
	}

	browse.Render(w, r, vars{
		"Title":        fmt.Sprintf("Headlamp: Browsing %s", bsd.project.Name),
		"Project":      bsd.project,
		"Folder":       bsd.folder,
		"Folders":      folders,
		"Files":        files,
		"TooManyFiles": tooManyFiles,
		"MaxFiles":     maxFiles,
		"TotalFiles":   totalFileCount,
	})
}

func searchHandler(w http.ResponseWriter, r *http.Request) {
	var bsd = getBrowseSearchData(w, r)
	if bsd.hadError {
		return
	}

	var q = r.URL.Query().Get("q")
	var fq = r.URL.Query().Get("fq")
	if q == "" && fq == "" {
		setAlert(w, r, "You must provide a search term")
		w.WriteHeader(http.StatusBadRequest)

		if bsd.pName == "" {
			renderHome(w, r)
			return
		}

		browseHandler(w, r)
		return
	}

	if fq != "" {
		folderSearch(w, r, bsd, fq)
		return
	}
	fileSearch(w, r, bsd, q)
}

func fileSearch(w http.ResponseWriter, r *http.Request, bsd browseSearchData, term string) {
	var files, totalFileCount, err = bsd.op.SearchFiles(bsd.project, bsd.folder, term, maxFiles+1)
	if err != nil {
		logger.Errorf("Error trying to search for files under %q (in project %q) from the database: %s",
			bsd.folderPath, bsd.pName, err)
		_500(w, r, "Error trying to search for folders.  Try again or contact support.")
		return
	}

	var tooManyFiles = false
	if len(files) > maxFiles {
		files = files[:maxFiles]
		tooManyFiles = true
	}

	search.Render(w, r, vars{
		"Title":        fmt.Sprintf("Headlamp: File Search"),
		"SearchTerm":   term,
		"Project":      bsd.project,
		"Folder":       bsd.folder,
		"Files":        files,
		"TooManyFiles": tooManyFiles,
		"MaxFiles":     maxFiles,
		"TotalFiles":   totalFileCount,
	})
}

func folderSearch(w http.ResponseWriter, r *http.Request, bsd browseSearchData, term string) {
	var folders, err = bsd.op.SearchFolders(bsd.project, bsd.folder, term)
	if err != nil {
		logger.Errorf("Error trying to search for folders under %q (in project %q) from the database: %s",
			bsd.folderPath, bsd.pName, err)
		_500(w, r, "Error trying to search for folders.  Try again or contact support.")
		return
	}

	search.Render(w, r, vars{
		"Title":            fmt.Sprintf("Headlamp: Folder Search"),
		"FolderSearchTerm": term,
		"Project":          bsd.project,
		"Folder":           bsd.folder,
		"Folders":          folders,
	})
}
