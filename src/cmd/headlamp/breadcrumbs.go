package main

import (
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"

	"github.com/uoregon-libraries/headlamp/src/db"
)

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

// breadcrumbs displays the category (if any) and each path element of the
// current folder (if any), each as a clickable location for easier navigation
func breadcrumbs(c *db.Category, f *db.Folder) template.HTML {
	if c == nil {
		return template.HTML("")
	}

	var crumbs = &breadCrumbs{}
	crumbs.add(c.Name, browseCategoryPath(c))
	var folderPathParts []string
	if f != nil {
		folderPathParts = strings.Split(f.PublicPath, string(os.PathSeparator))
	}
	var dummyFolder = &db.Folder{Category: c}
	for _, part := range folderPathParts {
		dummyFolder.PublicPath = filepath.Join(dummyFolder.PublicPath, part)
		crumbs.add(part, browseFolderPath(dummyFolder))
	}

	return crumbs.nav()
}
