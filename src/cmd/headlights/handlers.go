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
var home, browse, empty *tmpl.Template

func initTemplates(webroot string) {
	webutil.Webroot = webroot
	root = tmpl.Root("layout", "templates/")
	root.Funcs(tmpl.DefaultTemplateFunctions)
	root.Funcs(webutil.FuncMap)
	root.Funcs(localTemplateFuncs)
	root.MustReadPartials("layout.go.html")
	home = root.Clone().MustBuild("home.go.html")
	browse = root.Clone().MustBuild("browse.go.html")
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

func browseHandler(w http.ResponseWriter, r *http.Request) {
	// We need to at least have a project name after /browse; the rest of the URL
	// path is optional but would represent the folder structure if present
	var parts = strings.Split(r.URL.Path, "/")
	if len(parts) < 2 || parts[2] == "" {
		_400(w, "Invalid request")
		logger.Debugf("invalid request")
		return
	}
	var pName = parts[2]
	var folderPath = filepath.Join(parts[3:]...)

	// We're doing a lot, so let's grab a single operation for all this lovely work
	var op = dbh.Operation()

	var project, err = op.FindProjectByName(pName)
	if err != nil {
		logger.Errorf("Error trying to read project %q from the database: %s", pName, err)
		_500(w, fmt.Sprintf("Error trying to find project %q.  Try again or contact support.", pName))
		return
	}
	if project == nil {
		_404(w, fmt.Sprintf("Project %q not found", pName))
		return
	}

	var folder *db.Folder
	if folderPath != "" {
		folder, err = op.FindFolderByPath(project, folderPath)
		if err != nil {
			logger.Errorf("Error trying to read folder %q (in project %q) from the database: %s", folderPath, pName, err)
			_500(w, fmt.Sprintf("Error trying to find folder %q.  Try again or contact support.", folderPath))
			return
		}
		if folder == nil {
			_404(w, fmt.Sprintf("Folder %q not found", folderPath))
			return
		}
	}

	var folders []*db.Folder
	folders, err = op.GetFolders(project, folder)
	if err != nil {
		logger.Errorf("Error trying to read folders under %q (in project %q) from the database: %s",
			folderPath, pName, err)
		_500(w, fmt.Sprintf("Error trying to read folder %q.  Try again or contact support.", folderPath))
		return
	}

	var files []*db.File
	var totalFileCount uint64
	var tooManyFiles = false
	files, totalFileCount, err = op.GetFiles(project, folder, maxFiles+1)
	if err != nil {
		logger.Errorf("Error trying to read files under %q (in project %q) from the database: %s",
			folderPath, pName, err)
		_500(w, fmt.Sprintf("Error trying to read folder %q.  Try again or contact support.", folderPath))
		return
	}
	if len(files) > maxFiles {
		files = files[:maxFiles]
		tooManyFiles = true
	}

	err = browse.Execute(w, vars{
		"Title":        fmt.Sprintf("Headlights: Browsing %s", project.Name),
		"Project":      project,
		"Folder":       folder,
		"Subfolders":   folders,
		"Files":        files,
		"TooManyFiles": tooManyFiles,
		"MaxFiles":     maxFiles,
		"TotalFiles":   totalFileCount,
	})
	if err != nil {
		logger.Errorf("Unable to render home template: %s", err)
	}
}
