package main

import "net/http"

func _400(w http.ResponseWriter, msg string) {
	w.WriteHeader(http.StatusBadRequest)
	empty.Execute(w, vars{"Title": "Invalid Request", "Alert": msg})
}

func _404(w http.ResponseWriter, msg string) {
	w.WriteHeader(http.StatusNotFound)
	empty.Execute(w, vars{"Title": "Not Found", "Alert": msg})
}

func _500(w http.ResponseWriter, msg string) {
	w.WriteHeader(http.StatusInternalServerError)
	empty.Execute(w, vars{"Title": "Error", "Alert": msg})
}
