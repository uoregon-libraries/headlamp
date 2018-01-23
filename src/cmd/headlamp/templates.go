package main

import (
	"net/http"
	"path/filepath"

	"github.com/uoregon-libraries/gopkg/logger"
	"github.com/uoregon-libraries/gopkg/tmpl"
	"github.com/uoregon-libraries/gopkg/webutil"
)

// Template wraps the central package to add session-aware rendering
type Template struct {
	*tmpl.Template
}

var home, browse, search, bulk, empty *Template

func initTemplates(webroot string) {
	webutil.Webroot = webroot
	var root = tmpl.Root("layout", filepath.Join(conf.Approot, "templates"))

	var t = func(name string) *Template {
		return &Template{root.Clone().MustBuild(name + ".go.html")}
	}

	root.Funcs(tmpl.DefaultTemplateFunctions)
	root.Funcs(webutil.FuncMap)
	root.Funcs(localTemplateFuncs)
	root.MustReadPartials("layout.go.html", "_search_form.go.html", "_tables.go.html")
	home = t("home")
	browse = t("browse")
	search = t("search")
	bulk = t("bulk")
	empty = &Template{root.Template()}
}

// Render executes this template, logging errors and automagically pulling
// alert/info data from the session if available
func (t *Template) Render(w http.ResponseWriter, r *http.Request, data vars) {
	var s = sessionManager.Load(r)
	data["Alert"], _ = s.PopString(w, "Alert")
	data["Info"], _ = s.PopString(w, "Info")
	var q = NewBulkFileQueue()
	var err = s.GetObject("Queue", q)
	if err != nil {
		logger.Errorf("Unable to load user's bulk file queue: %s", err)
	}
	data["Queue"] = q

	err = t.Execute(w, data)
	if err != nil {
		logger.Errorf("Unable to render home template: %s", err)
	}
}
