package main

import (
	"db"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/uoregon-libraries/gopkg/tmpl"
)

var localTemplateFuncs = tmpl.FuncMap{
	"BreadCrumbs":                breadcrumbs,
	"SearchPath":                 searchPath,
	"BrowseProjectPath":          browseProjectPath,
	"BrowseFolderPath":           browseFolderPath,
	"BrowseContainingFolderPath": browseContainingFolderPath,
	"ViewFilePath":               viewFilePath,
	"DownloadFilePath":           downloadFilePath,
	"Pathify":                    pathify,
	"stripProjectFolder":         stripProjectFolder,
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

func searchPath(project *db.Project, folder *db.Folder) string {
	// We only handle "search/" and beyond, so if there's no project context, we
	// have to force the trailing slash
	if project == nil {
		return joinPaths("search") + "/"
	}
	return joinPaths("search", pathify(project, folder))
}

// browseProjectPath produces the URL to browse the given project's top-level folder
func browseProjectPath(project *db.Project) string {
	return joinPaths("browse", project.Name)
}

func browseFolderPath(folder *db.Folder) string {
	return joinPaths("browse", pathify(folder.Project, folder))
}

func browseContainingFolderPath(file *db.File) string {
	return joinPaths("browse", file.Project.Name, sanitizePath(file.ContainingFolder()))
}

func viewFilePath(file *db.File) string {
	return joinPaths("view", strconv.FormatUint(file.ID, 10))
}

func downloadFilePath(file *db.File) string {
	return joinPaths("download", strconv.FormatUint(file.ID, 10))
}

// stripProjectFolder takes a string representing a path, and strips out the
// current folder context, if any exists
func stripProjectFolder(f *db.Folder, path string) string {
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

// pathify combines the project and folder to create a slash-delimited string
func pathify(project *db.Project, folder *db.Folder) string {
	var p string

	if project == nil {
		return p
	}
	p = project.Name
	if folder != nil {
		p = path.Join(p, sanitizePath(folder.PublicPath))
	}

	return p
}
