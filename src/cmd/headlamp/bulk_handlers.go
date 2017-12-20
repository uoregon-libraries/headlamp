package main

import (
	"db"
	"net/http"
	"strconv"

	"github.com/uoregon-libraries/gopkg/logger"
)

func bulkQueueHandler(w http.ResponseWriter, r *http.Request) {
	// Verify request isn't completely broken
	var parts = getPathParts(r)
	if len(parts) != 3 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Verify we have a valid numeric ID
	var operation = parts[1]
	var fileIDString = parts[2]
	var fileID, err = strconv.ParseUint(fileIDString, 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Verify that the file exists
	var op = dbh.Operation()
	var f *db.File
	f, err = op.FindFileByID(fileID)
	if err != nil {
		logger.Errorf("Unable to look up file id %d: %s", fileID, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if f == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// Grab the session data that holds our queue
	var s = sessionManager.Load(r)
	var q = NewBulkFileQueue()
	err = s.GetObject("Queue", q)
	if err != nil {
		logger.Errorf("Unable to load user's bulk file queue: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Make sure we have a valid operation
	switch operation {
	case "add":
		q.AddFile(f)
	case "remove":
		q.RemoveFile(f)
	default:
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err = s.PutObject(w, "Queue", q)
	if err != nil {
		logger.Errorf("Unable to save user's bulk file queue: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func bulkDownloadHandler(w http.ResponseWriter, r *http.Request) {
	// Grab the session data that holds our queue
	var s = sessionManager.Load(r)
	var q = NewBulkFileQueue()
	var err = s.GetObject("Queue", q)
	if err != nil {
		logger.Errorf("Unable to load user's bulk file queue: %s", err)
		_500(w, r, "Unable to load your bulk download queue.  Try again or contact support.")
		return
	}

	var files []*db.File
	files, err = q.Files()
	if err != nil {
		logger.Errorf("Unable to load files from the database: %s", err)
		_500(w, r, "Unable to load your bulk download queue.  Try again or contact support.")
		return
	}

	bulk.Render(w, r, vars{
		"Title": "Headlamp: Bulk Download",
		"Files": files,
	})
}
