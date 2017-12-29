package main

import (
	"db"
	"net/http"
	"net/mail"
	"strconv"

	"github.com/uoregon-libraries/gopkg/logger"
	"github.com/uoregon-libraries/gopkg/webutil"
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

	// Grab saved email list if any is present - we ignore the error here because
	// (a) this isn't critical functionality, and (b) if something were wrong
	// with the session, it probably already went wrong when trying to grab the
	// bulk file queue
	var emails, _ = s.GetString("emails")

	var files []*db.File
	files, err = q.Files()
	if err != nil {
		logger.Errorf("Unable to load files from the database: %s", err)
		_500(w, r, "Unable to load your bulk download queue.  Try again or contact support.")
		return
	}

	var totalFilesize int64
	for _, f := range files {
		totalFilesize += f.Filesize
	}

	bulk.Render(w, r, vars{
		"Title":         "Headlamp: Bulk Download",
		"Files":         files,
		"TotalFilesize": totalFilesize,
		"Emails":        emails,
	})
}

func bulkCreateArchiveHandler(w http.ResponseWriter, r *http.Request) {
	// Grab the session data that holds our queue
	var s = sessionManager.Load(r)
	var q = NewBulkFileQueue()
	var err = s.GetObject("Queue", q)
	if err != nil {
		logger.Errorf("Unable to load user's bulk file queue: %s", err)
		_500(w, r, "Unable to load your bulk download queue.  Try again or contact support.")
		return
	}

	// Store email address list for future use, even if the request is otherwise invalid
	var emails = r.FormValue("emails")
	if emails != "" {
		s.PutString(w, "emails", emails)
	}

	// Pull files so we have their paths
	var files []*db.File
	files, err = q.Files()
	if err != nil {
		logger.Errorf("Unable to load files from the database: %s", err)
		_500(w, r, "Unable to load your bulk download queue.  Try again or contact support.")
		return
	}

	if len(files) == 0 {
		setAlert(w, r, "You don't have any files to archive!")
		http.Redirect(w, r, viewBulkQueuePath(), http.StatusTemporaryRedirect)
		return
	}

	if emails == "" {
		setAlert(w, r, "You must enter at least one valid notification email address")
		http.Redirect(w, r, viewBulkQueuePath(), http.StatusTemporaryRedirect)
		return
	}

	var addrs []*mail.Address
	addrs, err = mail.ParseAddressList(emails)
	if err != nil {
		setAlert(w, r, "One or more addresses are invalid - please re-enter and try again")
		http.Redirect(w, r, viewBulkQueuePath(), http.StatusTemporaryRedirect)
		return
	}

	err = dbh.Operation().QueueArchiveJob(addrs, files)
	if err != nil {
		logger.Errorf("Error trying to queue new archive: %s", err)
		setAlert(w, r, "Unable to queue the archive creation.  Please try again or contact support.")
		http.Redirect(w, r, viewBulkQueuePath(), http.StatusTemporaryRedirect)
		return
	}

	s.Remove(w, "Queue")
	setInfo(w, r, "Your archive is now being generated, and your bulk file queue has been emptied.")
	http.Redirect(w, r, webutil.Webroot, http.StatusTemporaryRedirect)
}
