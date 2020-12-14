package main

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/uoregon-libraries/gopkg/fileutil"
	"github.com/uoregon-libraries/gopkg/logger"
	"github.com/uoregon-libraries/headlamp/src/db"
)

// getFile returns an *os.File retrieved using the id in the last path element,
// or nil if no file was retrieved.  If nil is returned, the caller shouldn't
// render or output anything; 400, 500, and 404 errors will already have been
// sent to the browser.
func getFile(w http.ResponseWriter, r *http.Request) *os.File {
	var fileID uint64
	var err error
	var op = dbh.Operation()

	var parts = getPathParts(r)
	var idString = parts[len(parts)-1]
	fileID, err = strconv.ParseUint(idString, 10, 64)
	if err != nil {
		_400(w, r, "Invalid request")
		return nil
	}

	var file *db.File
	file, err = op.FindFileByID(fileID)
	if err != nil {
		logger.Errorf("Error trying to find file id %d: %s", fileID, err)
		_500(w, r, "Unable to read the specified file's data.  Try again or contact support.")
		return nil
	}

	if file == nil {
		_404(w, r, "Unable to find the requested file.  Try again or contact support.")
		return nil
	}

	var fullPath = filepath.Join(conf.DARoot, file.FullPath)
	if !fileutil.IsFile(fullPath) {
		logger.Errorf("File id %d describes a file I cannot find: %q / %q", file.ID, conf.DARoot, file.FullPath)
		_500(w, r, fmt.Sprintf("Unable to find %q.  Try again or contact support.", file.FullPath))
		return nil
	}

	var fh *os.File
	fh, err = os.Open(fullPath)
	if err != nil {
		logger.Errorf("Error trying to Open file %q: %s", file.FullPath, err)
		_500(w, r, fmt.Sprintf("Unable to open %q.  Try again or contact support.", file.FullPath))
		return nil
	}

	// Get mimetype via a modified version of golang's FileServer code
	var mimeType = mime.TypeByExtension(filepath.Ext(fullPath))
	if mimeType == "" {
		var buf [512]byte
		var n, _ = io.ReadFull(fh, buf[:])
		mimeType = http.DetectContentType(buf[:n])
		var _, err = fh.Seek(0, io.SeekStart)
		if err != nil {
			logger.Errorf("Error trying to Seek() on file %q: %s", file.FullPath, err)
			_500(w, r, fmt.Sprintf("Unable to read %q.  Try again or contact support.", file.FullPath))
			return nil
		}
	}
	w.Header().Set("Content-Type", mimeType)

	return fh
}

func viewFileHandler(w http.ResponseWriter, r *http.Request) {
	var fh = getFile(w, r)
	if fh == nil {
		return
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf("filename=%s", filepath.Base(fh.Name())))
	io.Copy(w, fh)
}

func downloadFileHandler(w http.ResponseWriter, r *http.Request) {
	var fh = getFile(w, r)
	if fh == nil {
		return
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filepath.Base(fh.Name())))
	io.Copy(w, fh)
}
