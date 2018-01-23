package main

import (
	"db"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/uoregon-libraries/gopkg/logger"
)

// maxFiles tells the app how many files to display on at once; if there are
// more than this many, we let the user know to do a different search
const maxFiles = 1000

type vars map[string]interface{}

// getPathParts filters the basePath out of the URL and then returns the actual
// app path elements
func getPathParts(r *http.Request) []string {
	var rawPath = r.URL.Path
	var trimmed = strings.TrimPrefix(rawPath, basePath)
	// Make sure there is no preceding slash
	trimmed = strings.TrimPrefix(trimmed, "/")
	return strings.Split(trimmed, "/")
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	var parts = getPathParts(r)
	if len(parts) == 0 || len(parts) == 1 && parts[0] == "" {
		renderHome(w, r)
		return
	}

	_404(w, r, "Unable to find the requested resource")
}

func renderHome(w http.ResponseWriter, r *http.Request) {
	var categories, err = dbh.Operation().AllCategories()
	if err != nil {
		logger.Errorf("Unable to find categories: %s", err)
		_500(w, r, "Error trying to find category list.  Try again or contact support.")
		return
	}

	home.Render(w, r, vars{"Title": "Headlamp", "Categories": categories})
}

type browseSearchData struct {
	op         *db.Operation
	pName      string
	category   *db.Category
	folderPath string
	folder     *db.Folder
	hadError   bool
}

// getBrowseSearchData centralizes some of the common things we need to check /
// pull from the database for both browsing and searching:
//
// - Get the current category, if this isn't a top-level search
// - Get the current folder, if one is set
func getBrowseSearchData(w http.ResponseWriter, r *http.Request) browseSearchData {
	var bsd browseSearchData
	var bsde = browseSearchData{hadError: true}
	var parts = getPathParts(r)

	// We're doing a lot, so let's grab a single operation for all this lovely work
	bsd.op = dbh.Operation()

	if len(parts) < 2 {
		return bsd
	}

	bsd.pName = parts[1]
	bsd.folderPath = filepath.Join(parts[2:]...)

	// This is acceptable in some situations, so we don't want to explode due to
	// missing category
	if bsd.pName == "" {
		return bsd
	}

	var err error
	bsd.category, err = bsd.op.FindCategoryByName(bsd.pName)
	if err != nil {
		logger.Errorf("Error trying to read category %q from the database: %s", bsd.pName, err)
		_500(w, r, fmt.Sprintf("Error trying to find category %q.  Try again or contact support.", bsd.pName))
		return bsde
	}
	if bsd.category == nil {
		_404(w, r, fmt.Sprintf("Category %q not found", bsd.pName))
		return bsde
	}

	if bsd.folderPath != "" {
		bsd.folder, err = bsd.op.FindFolderByPath(bsd.category, bsd.folderPath)
		if err != nil {
			logger.Errorf("Error trying to read folder %q (in category %q) from the database: %s",
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

	var folders, err = bsd.op.GetFolders(bsd.category, bsd.folder)
	if err != nil {
		logger.Errorf("Error trying to read folders under %q (in category %q) from the database: %s",
			bsd.folderPath, bsd.pName, err)
		_500(w, r, fmt.Sprintf("Error trying to read folder %q.  Try again or contact support.", bsd.folderPath))
		return
	}

	var files []*db.File
	var totalFileCount uint64
	files, totalFileCount, err = bsd.op.GetFiles(bsd.category, bsd.folder, maxFiles+1)
	if err != nil {
		logger.Errorf("Error trying to read files under %q (in category %q) from the database: %s",
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
		"Title":        fmt.Sprintf("Headlamp: Browsing %s", bsd.category.Name),
		"Category":     bsd.category,
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
	var files, totalFileCount, err = bsd.op.SearchFiles(bsd.category, bsd.folder, term, maxFiles+1)
	if err != nil {
		logger.Errorf("Error trying to search for files under %q (in category %q) from the database: %s",
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
		"Title":        "Headlamp: File Search",
		"SearchTerm":   term,
		"Category":     bsd.category,
		"Folder":       bsd.folder,
		"Files":        files,
		"TooManyFiles": tooManyFiles,
		"MaxFiles":     maxFiles,
		"TotalFiles":   totalFileCount,
	})
}

func folderSearch(w http.ResponseWriter, r *http.Request, bsd browseSearchData, term string) {
	var folders, totalFolderCount, err = bsd.op.SearchFolders(bsd.category, bsd.folder, term, maxFiles+1)
	if err != nil {
		logger.Errorf("Error trying to search for folders under %q (in category %q) from the database: %s",
			bsd.folderPath, bsd.pName, err)
		_500(w, r, "Error trying to search for folders.  Try again or contact support.")
		return
	}

	var tooManyFolders = false
	if len(folders) > maxFiles {
		folders = folders[:maxFiles]
		tooManyFolders = true
	}

	search.Render(w, r, vars{
		"Title":            "Headlamp: Folder Search",
		"FolderSearchTerm": term,
		"Category":         bsd.category,
		"Folder":           bsd.folder,
		"Folders":          folders,
		"TooManyFolders":   tooManyFolders,
		"MaxFolders":       maxFiles,
		"TotalFolders":     totalFolderCount,
	})
}

func viewRealFoldersHandler(w http.ResponseWriter, r *http.Request) {
	_500(w, r, "Filesystem information not implemented yet")
}
