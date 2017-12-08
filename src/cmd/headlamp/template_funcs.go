package main

import (
	"db"
	"fmt"
	"strings"

	"github.com/uoregon-libraries/gopkg/tmpl"
)

var localTemplateFuncs = tmpl.FuncMap{
	"BrowseProjectPath":          browseProjectPath,
	"BrowseFolderPath":           browseFolderPath,
	"BrowseContainingFolderPath": browseContainingFolderPath,
	"DownloadFilePath":           downloadFilePath,
	"stripProjectFolder":         stripProjectFolder,
}

// browseProjectPath produces the URL to browse the given project's top-level folder
func browseProjectPath(project *db.Project) string {
	return fmt.Sprintf("/browse/%s", project.Name)
}

func browseFolderPath(folder *db.Folder) string {
	return fmt.Sprintf("/browse/%s/%s", folder.Project.Name, folder.Path)
}

func browseContainingFolderPath(file *db.File) string {
	return fmt.Sprintf("/browse/%s/%s", file.Project.Name, file.ContainingFolder())
}

func downloadFilePath(file *db.File) string {
	return fmt.Sprintf("/download/%s/%s", file.Project.Name, file.PublicPath)
}

// stripProjectFolder takes a string representing a path, and strips out the
// current folder context, if any exists
func stripProjectFolder(f *db.Folder, path string) string {
	if f == nil {
		return path
	}
	path = strings.TrimPrefix(path, f.Path)
	path = strings.TrimPrefix(path, "/") // Just to make sure there's no starting slash
	return path
}
