package main

import (
	"db"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/uoregon-libraries/gopkg/tmpl"
)

var localTemplateFuncs = tmpl.FuncMap{
	"BreadCrumbs":                breadcrumbs,
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

// browseProjectPath produces the URL to browse the given project's top-level folder
func browseProjectPath(project *db.Project) string {
	return fmt.Sprintf("/browse/%s", project.Name)
}

func browseFolderPath(folder *db.Folder) string {
	return fmt.Sprintf("/browse/%s", pathify(folder.Project, folder))
}

func browseContainingFolderPath(file *db.File) string {
	return fmt.Sprintf("/browse/%s/%s", file.Project.Name, sanitizePath(file.ContainingFolder()))
}

func viewFilePath(file *db.File) string {
	return fmt.Sprintf("/view/%d", file.ID)
}

func downloadFilePath(file *db.File) string {
	return fmt.Sprintf("/download/%d", file.ID)
}

// stripProjectFolder takes a string representing a path, and strips out the
// current folder context, if any exists
func stripProjectFolder(f *db.Folder, path string) string {
	path = sanitizePath(path)
	if f == nil {
		return path
	}
	path = strings.TrimPrefix(path, sanitizePath(f.Path))
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
		p = path.Join(p, sanitizePath(folder.Path))
	}

	return p
}
