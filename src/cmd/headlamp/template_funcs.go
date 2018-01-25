package main

import (
	"db"
	"fmt"
	"html/template"
	"os"
	"path"
	"strconv"
	"strings"
	"version"

	"github.com/uoregon-libraries/gopkg/humanize"
	"github.com/uoregon-libraries/gopkg/tmpl"
)

var localTemplateFuncs = tmpl.FuncMap{
	"BreadCrumbs":                breadcrumbs,
	"SearchPath":                 searchPath,
	"AddToQueueButton":           addToQueueButton,
	"RemoveFromQueueButton":      removeFromQueueButton,
	"ViewBulkQueuePath":          viewBulkQueuePath,
	"BrowseCategoryPath":         browseCategoryPath,
	"BrowseFolderPath":           browseFolderPath,
	"BrowseContainingFolderPath": browseContainingFolderPath,
	"ViewFilePath":               viewFilePath,
	"ViewRealFoldersPath":        viewRealFoldersPath,
	"DownloadFilePath":           downloadFilePath,
	"BulkDownloadCreatePath":     bulkDownloadCreatePath,
	"Pathify":                    pathify,
	"GenericPath":                joinPaths,
	"stripCategoryFolder":        stripCategoryFolder,
	"humanFilesize":              humanFilesize,
	"VersionString":              versionString,
}

// sanitizePath takes a path from a file or folder and makes it
// "web-compatible" by ensuring the path separator is a forward slash no matter
// the OS
func sanitizePath(p string) string {
	return path.Join(strings.Split(p, string(os.PathSeparator))...)
}

// joinPaths combines the app's base path with the parts passed in
func joinPaths(parts ...string) string {
	parts = append([]string{basePath}, parts...)
	return path.Join(parts...)
}

func searchPath(category *db.Category, folder *db.Folder) string {
	// We only handle "search/" and beyond, so if there's no category context, we
	// have to force the trailing slash
	if category == nil {
		return joinPaths("search") + "/"
	}
	return joinPaths("search", pathify(category, folder))
}

func addToQueuePath(file *db.File) string {
	return joinPaths("bulk", "add", strconv.FormatUint(file.ID, 10))
}

func removeFromQueuePath(file *db.File) string {
	return joinPaths("bulk", "remove", strconv.FormatUint(file.ID, 10))
}

func makeButton(val string, classes []string, attrs map[string]string, disabled bool) template.HTML {
	if disabled {
		attrs["disabled"] = "disabled"
	}
	attrs["class"] = strings.Join(classes, " ")

	var attrPairs []string
	for k, v := range attrs {
		attrPairs = append(attrPairs, fmt.Sprintf(`%s="%s"`, k, v))
	}
	return template.HTML(fmt.Sprintf("<button %s>%s</button>", strings.Join(attrPairs, " "), val))
}

func bulkButtonID(add bool, file *db.File) string {
	var prefix string
	if add {
		prefix = "add"
	} else {
		prefix = "remove"
	}
	return fmt.Sprintf("%s-queue-%d", prefix, file.ID)
}

func addToQueueButton(q *BulkFileQueue, file *db.File) template.HTML {
	var classes = []string{"bulk-action", "btn", "btn-success"}
	var attrs = map[string]string{
		"id":                     bulkButtonID(true, file),
		"data-is-remove":         "0",
		"data-action":            addToQueuePath(file),
		"data-toggle-on-success": bulkButtonID(false, file),
	}
	return makeButton("Queue", classes, attrs, q.HasFile(file))
}

func removeFromQueueButton(q *BulkFileQueue, file *db.File) template.HTML {
	var classes = []string{"bulk-action", "btn", "btn-danger"}
	var attrs = map[string]string{
		"id":                     bulkButtonID(false, file),
		"data-is-remove":         "1",
		"data-action":            removeFromQueuePath(file),
		"data-toggle-on-success": bulkButtonID(true, file),
	}
	return makeButton("Remove", classes, attrs, !q.HasFile(file))
}

func viewBulkQueuePath() string {
	return joinPaths("bulk-download")
}

// browseCategoryPath produces the URL to browse the given category's top-level folder
func browseCategoryPath(category *db.Category) string {
	return joinPaths("browse", category.Name)
}

func browseFolderPath(folder *db.Folder) string {
	return joinPaths("browse", pathify(folder.Category, folder))
}

func browseContainingFolderPath(file *db.File) string {
	return joinPaths("browse", file.Category.Name, sanitizePath(file.ContainingFolder()))
}

func viewFilePath(file *db.File) string {
	return joinPaths("view", strconv.FormatUint(file.ID, 10))
}

func viewRealFoldersPath(folder *db.Folder) string {
	return joinPaths("filesystem", pathify(folder.Category, folder))
}

func downloadFilePath(file *db.File) string {
	return joinPaths("download", strconv.FormatUint(file.ID, 10))
}

func bulkDownloadCreatePath() string {
	return joinPaths("bulk", "create")
}

// stripCategoryFolder takes a string representing a path, and strips out the
// current folder context, if any exists
func stripCategoryFolder(f *db.Folder, path string) string {
	path = sanitizePath(path)
	if f == nil {
		return path
	}
	path = strings.TrimPrefix(path, sanitizePath(f.PublicPath))
	path = strings.TrimPrefix(path, "/") // Just to make sure there's no starting slash
	if path == "" {
		path = "."
	}
	return path
}

// pathify combines the category and folder to create a slash-delimited string
func pathify(category *db.Category, folder *db.Folder) string {
	var p string

	if category == nil {
		return p
	}
	p = category.Name
	if folder != nil {
		p = path.Join(p, sanitizePath(folder.PublicPath))
	}

	return p
}

// humanFilesize returns a more meaningful value for filesizes
func humanFilesize(bytes int64) string {
	return humanize.Bytes(bytes)
}

// versionString returns a version number for inclusion on web pages so it's
// clearer what's on staging vs. dev vs. prod, etc.
func versionString() string {
	return fmt.Sprintf("Headlamp v%s", version.Version)
}
