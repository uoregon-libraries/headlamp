package main

import "net/http"

func _400(w http.ResponseWriter, r *http.Request, msg string) {
	w.WriteHeader(http.StatusBadRequest)
	setAlert(w, r, msg)
	empty.Render(w, r, vars{"Title": "Invalid Request"})
}

func _404(w http.ResponseWriter, r *http.Request, msg string) {
	w.WriteHeader(http.StatusNotFound)
	setAlert(w, r, msg)
	empty.Render(w, r, vars{"Title": "Not Found"})
}

func _500(w http.ResponseWriter, r *http.Request, msg string) {
	w.WriteHeader(http.StatusInternalServerError)
	setAlert(w, r, msg)
	empty.Render(w, r, vars{"Title": "Error"})
}
