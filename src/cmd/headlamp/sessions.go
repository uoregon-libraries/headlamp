package main

import "net/http"

func setAlert(w http.ResponseWriter, r *http.Request, val string) {
	var s = sessionManager.Load(r)
	s.PutString(w, "Alert", val)
}

func setInfo(w http.ResponseWriter, r *http.Request, val string) {
	var s = sessionManager.Load(r)
	s.PutString(w, "Info", val)
}
