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
// alert/info data from the session if available
func (t *Template) Render(w http.ResponseWriter, r *http.Request, data vars) {
	var s = sessionManager.Load(r)
	data["Alert"], _ = s.PopString(w, "Alert")
	data["Info"], _ = s.PopString(w, "Info")

	var err = t.Execute(w, data)
	if err != nil {
		logger.Errorf("Unable to render home template: %s", err)
	}
}
