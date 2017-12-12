package main

import (
	"net/http"

	"github.com/gorilla/sessions"
	"github.com/uoregon-libraries/gopkg/logger"
)

var sessionStore *sessions.CookieStore

// getSession attempts to retrieve session data from the given request.  If nil
// is returned, the caller should abort as an error has already been sent to
// the browser.
func getSession(w http.ResponseWriter, r *http.Request) *sessions.Session {
	var session, err = sessionStore.Get(r, "headlamp")
	if err != nil {
		logger.Errorf("Unable to create a session: %s", err)
		_500(w, "Error trying to retrieve session data.  Try again or contact support.")
		return nil
	}
	return session
}
