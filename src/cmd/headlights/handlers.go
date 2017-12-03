package main

import (
	"net/http"

	"github.com/uoregon-libraries/gopkg/logger"
	"github.com/uoregon-libraries/gopkg/tmpl"
	"github.com/uoregon-libraries/gopkg/webutil"
)

var root *tmpl.TRoot
var home, empty *tmpl.Template

func initTemplates(baseURL string) {
	webutil.Webroot = baseURL
	root = tmpl.Root("layout", "templates/")
	root.Funcs(tmpl.DefaultTemplateFunctions)
	root.Funcs(webutil.FuncMap)
	root.MustReadPartials("layout.go.html")
	home = root.Clone().MustBuild("home.go.html")
	empty = root.Template()
}

type vars map[string]string

func homeHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.String() != "/" {
		w.WriteHeader(http.StatusNotFound)
		empty.Execute(w, vars{"Title": "Error", "Alert": "Unable to find the requested resource"})
		return
	}
	var err = home.Execute(w, vars{"Title": "Search - Headlights"})
	if err != nil {
		logger.Errorf("Unable to render home template: %s", err)
	}
}
