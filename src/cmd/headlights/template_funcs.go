package main

import (
	"db"
	"fmt"

	"github.com/uoregon-libraries/gopkg/tmpl"
)

var localTemplateFuncs = tmpl.FuncMap{
	"BrowseProjectPath": browseProjectPath,
	"BrowseFolderPath":  browseFolderPath,
	"DownloadFilePath":  downloadFilePath,
}

// browseProjectPath produces the URL to browse the given project's top-level folder
func browseProjectPath(project *db.Project) string {
	return fmt.Sprintf("/browse/%s", project.Name)
}

func browseFolderPath(folder *db.Folder) string {
	return fmt.Sprintf("/browse/%s/%s", folder.Project.Name, folder.Path)
}

func downloadFilePath(file *db.File) string {
	return fmt.Sprintf("/download/%s/%s", file.Project.Name, file.PublicPath)
}
