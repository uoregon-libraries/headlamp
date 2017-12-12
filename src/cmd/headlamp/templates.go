package main

import (
	"net/http"

	"github.com/uoregon-libraries/gopkg/logger"
	"github.com/uoregon-libraries/gopkg/tmpl"
)

// Template wraps the central package to add session-aware rendering
type Template struct {
	*tmpl.Template
}

// Render executes this template, logging errors and automagically pulling
// alert/info data from the session
func (t *Template) Render(w http.ResponseWriter, r *http.Request, data vars) {
	var s = getSession(w, r)
	if s == nil {
		return
	}
	data["Alert"] = s.Values["Alert"]
	s.Values["Alert"] = ""
	data["Info"] = s.Values["Info"]
	s.Values["Info"] = ""
	s.Save(r, w)

	var err = t.Execute(w, data)
	if err != nil {
		logger.Errorf("Unable to render home template: %s", err)
	}
}
