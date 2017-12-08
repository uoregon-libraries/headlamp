package main

import (
	"db"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"

	"github.com/uoregon-libraries/gopkg/tmpl"
)

var localTemplateFuncs = tmpl.FuncMap{
	"BreadCrumbs":                breadcrumbs,
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

type breadCrumb struct {
	label string
	url   string
}

func (c *breadCrumb) li(last bool) string {
	var aria = ""
	if last {
		aria = `aria-current="page"`
	}
	return fmt.Sprintf(`<li><a href="%s"%s>%s</a></li>`, c.url, aria, c.label)
}

type breadCrumbs struct {
	list []*breadCrumb
}

func (c *breadCrumbs) add(label, url string) {
	c.list = append(c.list, &breadCrumb{label: label, url: url})
}

func (c *breadCrumbs) nav() template.HTML {
	var crumbStrings []string
	for i, crumb := range c.list {
		crumbStrings = append(crumbStrings, crumb.li(i == len(c.list)-1))
	}

	var wrapperOpen = `<nav aria-label="Breadcrumb"><ol class="breadcrumb">`
	var wrapperClose = `</ol></nav>`
	return template.HTML(wrapperOpen + strings.Join(crumbStrings, "") + wrapperClose)
}

// breadcrumbs displays the project (if any) and each path element of the
// current folder (if any), each as a clickable location for easier navigation
func breadcrumbs(p *db.Project, f *db.Folder) template.HTML {
	if p == nil {
		return template.HTML("")
	}

	var crumbs = &breadCrumbs{}
	crumbs.add(p.Name, browseProjectPath(p))
	var folderPathParts []string
	if f != nil {
		folderPathParts = strings.Split(f.Path, string(os.PathSeparator))
	}
	var dummyFolder = &db.Folder{Project: p}
	for _, part := range folderPathParts {
		dummyFolder.Path = filepath.Join(dummyFolder.Path, part)
		crumbs.add(part, browseFolderPath(dummyFolder))
	}

	return crumbs.nav()
}
