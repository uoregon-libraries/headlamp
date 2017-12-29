package main

import (
	"archive/tar"
	"config"
	"db"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/uoregon-libraries/gopkg/fileutil"
	"github.com/uoregon-libraries/gopkg/logger"
)

// Archiver holds the database handle and config to simplify processing
type Archiver struct {
	conf *config.Config
	dbh  *db.Database
}

// RunNextArchiveJob grabs the longest-waiting job and processes it
func (a *Archiver) RunNextArchiveJob() {
	logger.Debugf("Scanning for pending archive jobs")
	var err = a.dbh.Operation().ProcessArchiveJob(a.processArchiveJob)
	if err != nil {
		logger.Errorf("Unable to get next job: %s", err)
	}
}

func (a *Archiver) processArchiveJob(j *db.ArchiveJob) bool {
	logger.Infof("Processing archive job %d", j.ID)

	var tempFile, err = fileutil.TempFile(a.conf.ArchiveOutputLocation, ".wip-", ".tar")
	if err != nil {
		logger.Errorf("Unable to create temp archive: %s", err)
		return false
	}

	var tw = tar.NewWriter(tempFile)

	for _, fname := range j.FileList() {
		var p = filepath.Join(a.conf.DARoot, fname)
		var fn = strings.Replace(fname, string(os.PathSeparator), "__", -1)
		err = addFileToTar(tw, p, fn)
		if err != nil {
			logger.Errorf("Unable to add %q to archive: %s", fname, err)
			return false
		}
	}

	err = tw.Close()
	if err != nil {
		logger.Errorf("Error closing tar stream %q: %s", tempFile.Name(), err)
		return false
	}

	err = tempFile.Close()
	if err != nil {
		logger.Errorf("Error closing %q: %s", tempFile.Name(), err)
		return false
	}

	logger.Infof("Job %d completed successfully", j.ID)
	return true
}

func addFileToTar(tw *tar.Writer, filePath, flatname string) error {
	var srcFile, err = os.Open(filePath)
	if err != nil {
		return fmt.Errorf("os.Open(%q): %s", filePath, err)
	}

	var info os.FileInfo
	info, err = srcFile.Stat()
	if err != nil {
		return fmt.Errorf("unable to stat %q: %s", filePath, err)
	}

	var header = &tar.Header{Name: flatname, Mode: 0600, Size: info.Size()}
	err = tw.WriteHeader(header)
	if err != nil {
		return fmt.Errorf("writing header for %q: %s", flatname, err)
	}

	_, err = io.Copy(tw, srcFile)
	if err != nil {
		return fmt.Errorf("%q io.Copy(): %s", flatname, err)
	}

	return nil
}
