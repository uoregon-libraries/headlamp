package main

import (
	"db"
	"net/http"
	"net/url"

	"github.com/uoregon-libraries/gopkg/logger"
	"github.com/uoregon-libraries/gopkg/tmpl"
	"github.com/uoregon-libraries/gopkg/webutil"
)

var root *tmpl.TRoot
var home, empty *tmpl.Template

func initTemplates(baseURL *url.URL) {
	webutil.Webroot = baseURL.Path
	root = tmpl.Root("layout", "templates/")
	root.Funcs(tmpl.DefaultTemplateFunctions)
	root.Funcs(webutil.FuncMap)
	root.MustReadPartials("layout.go.html")
	home = root.Clone().MustBuild("home.go.html")
	empty = root.Template()
}

type vars map[string]interface{}

func _500(w http.ResponseWriter, msg string) {
	w.WriteHeader(http.StatusInternalServerError)
	empty.Execute(w, vars{"Title": "Error", "Alert": msg})
}

func getProjects() (projects []*db.Project, err error) {
	err = dbh.InTransaction(func(op *db.Operation) error {
		projects, err = op.AllProjects()
		return err
	})
	return projects, err
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.String() != "/" {
		w.WriteHeader(http.StatusNotFound)
		empty.Execute(w, vars{"Title": "Error", "Alert": "Unable to find the requested resource"})
		return
	}

	var projects, err = getProjects()
	if err != nil {
		logger.Errorf("Unable to find projects: %s", err)
		_500(w, "Error trying to find project list.  Try again or contact support.")
		return
	}

	err = home.Execute(w, vars{"Title": "Headlights", "Projects": projects})
	if err != nil {
		logger.Errorf("Unable to render home template: %s", err)
	}
}
