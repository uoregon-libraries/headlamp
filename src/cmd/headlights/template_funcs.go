package main

import (
	"fmt"

	"github.com/uoregon-libraries/gopkg/tmpl"
)

var localTemplateFuncs = tmpl.FuncMap{
	"BrowseProjectPath": browseProjectPath,
}

// browseProjectPath produces the URL to browse the given project's top-level folder
func browseProjectPath(projectID int) string {
	return fmt.Sprintf("/browse/%d", projectID)
}
